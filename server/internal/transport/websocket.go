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

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server/log"

	"github.com/gorilla/websocket"
	"github.com/sethvargo/go-retry"
)

const websocketConnectFailureMessage string = "Connection to the game server process' web socket has failed. If you are using Anywhere without an Amazon GameLift Servers Agent," +
	"please verify that values of ServerParameters in InitSDK() are correct. For example, process ID needs to be " +
	"unique between executions, and the authentication token needs to be correct and unexpired."

type connectionAttempt struct {
	done chan struct{}
	err  error
}

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
	reconnectMtx         sync.Mutex
	connectionAttempt    *connectionAttempt
	connectURL           url.URL

	readHandlerMu sync.RWMutex
	readHandler   ReadHandler

	readRetries int

	connectionId int

	connectRetryInterval       time.Duration
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
		log:                  logger,
		dialer:               dialer,
		connectRetryInterval: common.ConnectRetryInterval,
		disconnectWebsocketTimeout: common.GetEnvDurationOrDefault(
			common.DisconnectWebsocketTimeout,
			common.DisconnectWebsocketTimeoutDefault,
			logger,
		),
	}
}

func (tr *websocketTransport) handleNetworkInterrupt(e error, connectionId int) error {
	if tr.preventAutoReconnect.Load() {
		tr.log.Debugf("Preventing auto-reconnect attempt due to explicit previous call to PreventAutoReconnect()")
		return nil
	}
	tr.log.Warnf("Detected network interruption %s! Reconnecting...", e)
	if reconnectError := tr.connect(nil, connectionId, true); reconnectError != nil {
		tr.log.Errorf("Reconnect failed: %s", reconnectError)
		return reconnectError
	}
	return nil
}

func (tr *websocketTransport) Connect(u *url.URL) error {
	return tr.connect(u, 0, false)
}

func (tr *websocketTransport) connect(u *url.URL, failedConnectionId int, reconnect bool) error {
	for {
		tr.reconnectMtx.Lock()
		if failedConnectionId != 0 && failedConnectionId != tr.connectionId {
			tr.reconnectMtx.Unlock()
			return nil
		}
		if attempt := tr.connectionAttempt; attempt != nil {
			tr.reconnectMtx.Unlock()
			<-attempt.done
			if reconnect {
				return attempt.err
			}
			continue
		}
		if reconnect {
			target := tr.connectURL
			u = &target
		}
		attempt := &connectionAttempt{done: make(chan struct{})}
		tr.connectionAttempt = attempt
		tr.reconnecting.Store(true)
		tr.reconnectMtx.Unlock()

		err := tr.connectOnce(u, attempt)
		tr.reconnectMtx.Lock()
		tr.finishConnectionAttemptLocked(attempt, err)
		tr.reconnectMtx.Unlock()
		return err
	}
}

func (tr *websocketTransport) finishConnectionAttemptLocked(attempt *connectionAttempt, err error) {
	if tr.connectionAttempt == attempt {
		attempt.err = err
		tr.connectionAttempt = nil
		tr.reconnecting.Store(false)
		close(attempt.done)
	}
}

func (tr *websocketTransport) connectOnce(u *url.URL, attempt *connectionAttempt) error {
	tr.writeMtx.Lock()
	defer tr.writeMtx.Unlock()
	tr.reconnectMtx.Lock()
	oldConn := tr.conn
	oldConnectionId := tr.connectionId
	oldCancelConnectionFn := tr.cancelConnectionFn
	tr.reconnectMtx.Unlock()

	tr.log.Debugf("Establishing websocket connection")

	ctx := context.Background()
	// Exponential doubles the interval between retries
	backOff := retry.NewExponential(tr.connectRetryInterval)
	// We are adding two because we skip the first two retries until the initial duration is 4
	backOff = retry.WithMaxRetries(common.ConnectMaxRetries+2, backOff)
	backOff = retry.WithCappedDuration(common.MaxReconnectBackoffDuration, backOff)
	backOff.Next()
	backOff.Next()

	var connectionLifetimeContext context.Context
	var nextConn Conn
	var nextCancelConnectionFn context.CancelFunc
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
		nextConn = conn
		connectionLifetimeContext, nextCancelConnectionFn = context.WithCancel(context.Background())
		return nil
	}); err != nil {
		return err
	}

	connectionId := oldConnectionId + 1
	tr.setCloseHandler(nextConn, connectionId)
	tr.reconnectMtx.Lock()
	tr.conn = nextConn
	tr.connectionId = connectionId
	tr.cancelConnectionFn = nextCancelConnectionFn
	tr.connectURL = *u

	// NOTE: Publish the result before starting the new reader or draining the old socket.
	tr.markConnectionAsRedundant(oldCancelConnectionFn, oldConnectionId)
	tr.isConnected.Store(true)
	tr.finishConnectionAttemptLocked(attempt, nil)
	tr.reconnectMtx.Unlock()
	go tr.readProcess(nextConn, connectionLifetimeContext, connectionId)

	// Close the previous connection
	if err := tr.closeConnectionSafely(oldConn, oldConnectionId); err != nil {
		tr.log.Debugf("websocket %d: Error occurred when trying to close websocket connection: %s", oldConnectionId, err)
	}
	return nil
}

// Reconnect - blocks until ongoing reconnect succeeds or initiates and finishes a new reconnect.
func (tr *websocketTransport) Reconnect() error {
	return tr.connect(nil, 0, true)
}

func (tr *websocketTransport) setCloseHandler(conn Conn, connectionId int) {
	// wraps a default handler that correctly implements the protocol specification.
	currentHandler := conn.CloseHandler()
	conn.SetCloseHandler(func(code int, text string) error {
		tr.log.Debugf("Socket disconnected. Code is %d. Reason is %s", code, text)
		tr.reconnectMtx.Lock()
		if connectionId == tr.connectionId {
			tr.isConnected.Store(false)
		}
		tr.reconnectMtx.Unlock()
		if err := tr.closeConnection(conn, connectionId); err != nil {
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
						if err = tr.handleNetworkInterrupt(err, connectionId); err != nil {
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
	tr.reconnectMtx.Lock()
	if !tr.isConnected.CompareAndSwap(true, false) {
		tr.reconnectMtx.Unlock()
		return nil
	}
	conn, connectionId := tr.conn, tr.connectionId
	tr.reconnectMtx.Unlock()
	return tr.closeConnection(conn, connectionId)
}

func (tr *websocketTransport) PreventAutoReconnect() {
	tr.preventAutoReconnect.Store(true)
}

func (tr *websocketTransport) Write(data []byte) error {
	tr.writeMtx.Lock()
	defer tr.writeMtx.Unlock()
	if !tr.isConnected.Load() {
		return common.NewGameLiftError(common.GameLiftServerNotInitialized, "", "")
	}
	var lastWriteErr error
	for writeRetries := 0; writeRetries < common.MaxReadWriteRetry; writeRetries++ {
		tr.reconnectMtx.Lock()
		conn, connectionId := tr.conn, tr.connectionId
		tr.reconnectMtx.Unlock()
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err == nil || !isAbnormalCloseError(err) {
			return err
		}
		lastWriteErr = err
		if writeRetries == common.ReconnectOnReadWriteFailureNumber {
			tr.writeMtx.Unlock()
			tr.handleNetworkInterrupt(err, connectionId)
			tr.writeMtx.Lock()
		} else {
			tr.log.Debugf("Failed to write message: %v, retrying...", err)
			time.Sleep(time.Second)
		}
	}
	return common.NewGameLiftError(common.WebsocketSendMessageFailure, "Failed write data", lastWriteErr.Error())
}
