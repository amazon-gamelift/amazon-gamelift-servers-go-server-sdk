/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package transport

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/server/log"

	"github.com/gorilla/websocket"
	"github.com/sethvargo/go-retry"
)

const websocketConnectFailureMessage string = "Connection to the game server process' web socket has failed. If you are using Anywhere without an Amazon GameLift Servers Agent," +
	"please verify that values of ServerParameters in InitSDK() are correct. For example, process ID needs to be " +
	"unique between executions, and the authentication token needs to be correct and unexpired."

// websocketTransport - implement ITransport interface for websocket connection.
type websocketTransport struct {
	log    log.ILogger
	dialer Dialer

	conn                 Conn
	cancelConnectionFn   context.CancelFunc
	isConnected          common.AtomicBool
	reconnecting         common.AtomicBool
	preventAutoReconnect common.AtomicBool
	writeMtx             sync.Mutex
	connectURL           url.URL

	readHandlerMu sync.RWMutex
	readHandler   ReadHandler

	readRetries  int
	writeRetries int

	connectionId int

	disconnectWebsocketTimeout time.Duration
}

// isAbnormalCloseError returns true if the error is not a CloseError or if it is a CloseError with an unexpected status code
func isAbnormalCloseError(err error) bool {
	return !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) ||
		websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure)
}

// Websocket creates a new instance of the ITransport implementation.
func Websocket(logger log.ILogger, dialer Dialer) ITransport {
	return &websocketTransport{
		log:    logger,
		dialer: dialer,
		disconnectWebsocketTimeout: common.GetEnvDurationOrDefault(
			common.DisconnectWebsocketTimeout,
			common.DisconnectWebsocketTimeoutDefault,
			logger,
		),
	}
}

func (tr *websocketTransport) handleNetworkInterrupt(e error) error {
	if tr.preventAutoReconnect.Load() {
		tr.log.Debugf("Preventing auto-reconnect attempt due to explicit previous call to PreventAutoReconnect()")
		return nil
	}
	tr.log.Warnf("Detected network interruption %s! Reconnecting...", e)
	if reconnectError := tr.Reconnect(); reconnectError != nil {
		tr.log.Errorf("Reconnect failed: %s", reconnectError)
		return reconnectError
	}
	return nil
}

func (tr *websocketTransport) Connect(u *url.URL) error {
	tr.writeMtx.Lock()
	defer tr.writeMtx.Unlock()
	// always set reconnecting to true so other goroutines can check whether a new connection is being set up
	tr.reconnecting.Store(true)
	oldConn := tr.conn
	oldConnectionId := tr.connectionId
	oldCancelConnectionFn := tr.cancelConnectionFn

	tr.log.Debugf("Establishing websocket connection")

	ctx := context.Background()
	// Exponential doubles the interval between retries
	backOff := retry.NewExponential(common.ConnectRetryInterval)
	// We are adding two because we skip the first two retries until the initial duration is 4
	backOff = retry.WithMaxRetries(common.ConnectMaxRetries+2, backOff)
	backOff = retry.WithCappedDuration(common.MaxReconnectBackoffDuration, backOff)
	backOff.Next()
	backOff.Next()

	var connectionLifetimeContext context.Context
	if err := retry.Do(ctx, backOff, func(ctx context.Context) error {
		//nolint:bodyclose // The response body may not contain the entire response and does not need to be closed by the application
		conn, resp, dialErr := tr.dialer.Dial(u.String(), http.Header{"User-Agent": []string{"gamelift-go-sdk/1.0"}})
		if dialErr != nil {
			var reason string
			if resp != nil {
				reason = resp.Status
				b, _ := io.ReadAll(resp.Body)
				tr.log.Debugf("Response header is: %v", resp.Header)
				tr.log.Debugf("Response body is: %s", b)
			}
			return retry.RetryableError(
				common.NewGameLiftError(common.WebsocketConnectFailure,
					"",
					fmt.Sprintf("connection error %s:%s. %s", reason, dialErr.Error(), websocketConnectFailureMessage),
				),
			)
		}
		tr.conn = conn
		connectionLifetimeContext, tr.cancelConnectionFn = context.WithCancel(context.Background())
		return nil
	}); err != nil {
		return err
	}

	tr.setCloseHandler()
	tr.connectURL = *u

	// Marking old connection as redundant must occur before we turn off the reconnection flag
	// Otherwise we have a race condition if the old connection receives an error or normal closure
	// while we have a new, main connection
	tr.markConnectionAsRedundant(oldCancelConnectionFn, oldConnectionId)
	tr.isConnected.Store(true)
	tr.reconnecting.Store(false)

	tr.connectionId++
	go tr.readProcess(tr.conn, connectionLifetimeContext, tr.connectionId)

	// Close the previous connection
	if err := tr.closeConnectionSafely(oldConn, oldConnectionId); err != nil {
		tr.log.Debugf("websocket %d: Error occurred when trying to close websocket connection: %s", oldConnectionId, err)
	}
	return nil
}

// Reconnect - blocks until ongoing reconnect succeeds or initiates and finishes a new reconnect.
func (tr *websocketTransport) Reconnect() error {
	if tr.reconnecting.Swap(true) {
		tr.writeMtx.Lock() // Wait for reconnect to finish
		defer tr.writeMtx.Unlock()
		return nil
	}
	err := tr.Connect(&tr.connectURL)
	tr.reconnecting.Store(false)
	return err
}

func (tr *websocketTransport) setCloseHandler() {
	// wraps a default handler that correctly implements the protocol specification.
	currentHandler := tr.conn.CloseHandler()
	tr.conn.SetCloseHandler(func(code int, text string) error {
		tr.log.Debugf("Socket disconnected. Code is %d. Reason is %s", code, text)
		err := tr.Close()
		if err != nil {
			return err
		}
		return currentHandler(code, text)
	})
}

func (tr *websocketTransport) readProcess(connection Conn, context context.Context, connectionId int) {
	defer connection.Close() // No need to close "safely" as this is only hit when the connection fails or is closed
	tr.log.Debugf("read goroutine %d: starting", connectionId)
	for {
		// ReadMessage will read all message from the NextReader
		// The returned messageType is either TextMessage or BinaryMessage.
		// Applications must break out of the application's read loop when this method
		// returns a non-nil error value. Errors returned from this method are
		// permanent. Once this method returns a non-nil error, all subsequent calls to
		// this method return the same error.
		t, msg, err := connection.ReadMessage()

		if err != nil {
			select {
			case <-context.Done():
				// Only short-circuit on errors for best-effort at flushing incoming messages until traffic
				// has been directed elsewhere. This is to avoid a race condition where a new connection
				// has successfully created, but an error or normal closure is received on the old connection.
				tr.log.Debugf("read goroutine %d: connection marked redundant, error handling can be ignored", connectionId)
			default:
				if isAbnormalCloseError(err) {
					if !tr.reconnecting.Load() {
						if !tr.preventAutoReconnect.Load() {
							tr.log.Errorf("read goroutine %d: Websocket readProcess failed: %v", connectionId, err)
						}
						// RefreshConnection can lead to disconnection.
						if err = tr.handleNetworkInterrupt(err); err != nil {
							tr.log.Errorf("read goroutine %d: Failed to handle network interrupt with error %v", connectionId, err)
						}
					} else {
						tr.log.Debugf("read goroutine %d: ongoing connection setup", connectionId)
					}
				}
			}
			// Must break, since we got an error from connection.ReadMessage()
			break
		}

		if t != websocket.TextMessage {
			tr.log.Warnf("read goroutine %d: Unknown Data received. Data type is not a text message", connectionId)
			continue // Skip all non text messages
		}

		if handler := tr.getReadHandler(); handler != nil {
			go handler(msg)
		}
	}
	tr.log.Debugf("read goroutine %d: ending", connectionId)
}

func (tr *websocketTransport) SetReadHandler(handler ReadHandler) {
	tr.readHandlerMu.Lock()
	defer tr.readHandlerMu.Unlock()

	tr.readHandler = handler
}

func (tr *websocketTransport) getReadHandler() ReadHandler {
	tr.readHandlerMu.RLock()
	defer tr.readHandlerMu.RUnlock()

	return tr.readHandler
}

// markConnectionAsRedundant - cancel further error handling on the corresponding connection.
// Only short-circuit on errors for best-effort at flushing incoming messages until traffic
// has been directed elsewhere. This is to avoid a race condition where a new connection
// has successfully created, but an error or normal closure is received on the old connection.
func (tr *websocketTransport) markConnectionAsRedundant(cancelConnectionFn context.CancelFunc, connectionId int) {
	if cancelConnectionFn != nil {
		tr.log.Debugf("websocket %d: Mark websocket connection as redundant due to reconnect", connectionId)
		cancelConnectionFn()
	}
}

// closeConnectionSafely - Close connection when we expect the connection to possibly receive more messages such
// as when we're establishing a newer connection
func (tr *websocketTransport) closeConnectionSafely(conn Conn, connectionId int) error {
	if conn != nil {
		tr.log.Debugf("websocket %d: Wait %v while connection flushes", connectionId, tr.disconnectWebsocketTimeout)
		<-time.After(tr.disconnectWebsocketTimeout)
		return tr.closeConnection(conn, connectionId)
	}
	return nil
}

// closeConnection - Close connection when we don't expect the connection to receive more messages such as when
// we've received an error, close control, or the game server initiates a disconnect.
func (tr *websocketTransport) closeConnection(conn Conn, connectionId int) error {
	tr.log.Debugf("websocket %d: Close websocket connection", connectionId)
	if err := conn.Close(); err != nil {
		return common.NewGameLiftError(common.WebsocketClosingError, "", err.Error())
	}
	return nil
}

func (tr *websocketTransport) Close() error {
	// Set isConnected to false and close connection only if previously isConnected value was true.
	if tr.isConnected.CompareAndSwap(true, false) {
		return tr.closeConnection(tr.conn, tr.connectionId)
	}

	return nil
}

func (tr *websocketTransport) PreventAutoReconnect() {
	tr.preventAutoReconnect.Store(true)
}

func (tr *websocketTransport) Write(data []byte) error {
	tr.writeMtx.Lock()
	if !tr.isConnected.Load() {
		tr.writeMtx.Unlock()
		return common.NewGameLiftError(common.GameLiftServerNotInitialized, "", "")
	}
	tr.writeRetries = 0
	var err error
	for ; tr.writeRetries < common.MaxReadWriteRetry; tr.writeRetries++ {
		if err = tr.conn.WriteMessage(websocket.TextMessage, data); err != nil && isAbnormalCloseError(err) {
			if tr.writeRetries == common.ReconnectOnReadWriteFailureNumber {
				tr.writeMtx.Unlock()
				if err = tr.handleNetworkInterrupt(err); err == nil {
					tr.writeRetries--
				}
				tr.writeMtx.Lock()
			} else {
				tr.log.Debugf("Failed to write message: %v, retrying...", err)
				time.Sleep(time.Second)
			}
		} else {
			tr.writeMtx.Unlock()
			return err
		}
	}
	tr.writeMtx.Unlock()
	return common.NewGameLiftError(common.WebsocketSendMessageFailure, "Failed write data", err.Error())
}
