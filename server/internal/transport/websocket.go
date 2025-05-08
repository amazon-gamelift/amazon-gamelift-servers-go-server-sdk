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

	"github.com/sethvargo/go-retry"

	"aws/amazon-gamelift-go-sdk/common"
	"aws/amazon-gamelift-go-sdk/server/log"

	"github.com/gorilla/websocket"
)

// websocketTransport - implement ITransport interface for websocket connection.
type websocketTransport struct {
	log    log.ILogger
	dialer Dialer

	conn         Conn
	isConnected  common.AtomicBool
	reconnecting common.AtomicBool
	writeMtx     sync.Mutex
	connectURL   url.URL

	readHandlerMu sync.RWMutex
	readHandler   ReadHandler

	readRetries  int
	writeRetries int
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
	}
}

func (tr *websocketTransport) handleNetworkInterrupt(e error) error {
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
	if err := tr.Close(); err != nil {
		tr.log.Debugf("Error occurred when try close websocket connection: %s", err)
	}
	tr.log.Debugf("Connection string: %s", u)

	ctx := context.Background()
	// Exponential doubles the interval between retries
	backOff := retry.NewExponential(common.ConnectRetryInterval)
	// We are adding two because we skip the first two retries until the initial duration is 4
	backOff = retry.WithMaxRetries(common.ConnectMaxRetries+2, backOff)
	backOff = retry.WithCappedDuration(common.MaxReconnectBackoffDuration, backOff)
	backOff.Next()
	backOff.Next()

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
					fmt.Sprintf("connection error %s:%s", reason, dialErr.Error()),
				),
			)
		}
		tr.conn = conn
		return nil
	}); err != nil {
		return err
	}

	tr.setCloseHandler()
	tr.connectURL = *u
	tr.isConnected.Store(true)
	go tr.readProcess()
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
		tr.isConnected.Store(false)
		err := tr.Close()
		if err != nil {
			return err
		}
		return currentHandler(code, text)
	})
}

func (tr *websocketTransport) readProcess() {
	defer tr.Close()
	for tr.isConnected.Load() {
		// ReadMessage will read all message from the NextReader
		// The returned messageType is either TextMessage or BinaryMessage.
		t, msg, err := tr.conn.ReadMessage()
		if err != nil && isAbnormalCloseError(err) {
			if tr.readRetries == common.ReconnectOnReadWriteFailureNumber {
				tr.readRetries++
				if err = tr.handleNetworkInterrupt(err); err == nil {
					continue // Try to read again after reconnecting
				}
			} else if tr.readRetries < common.MaxReadWriteRetry {
				tr.log.Debugf("Failed to read message with error %v, retrying...", err)
				time.Sleep(time.Second)
				tr.readRetries++
				continue
			}
			// Reconnect failed or error not related to connection
			tr.log.Errorf("Websocket readProcess failed: %v", err)
			tr.readRetries = 0
			break
		} else if err == nil {
			tr.readRetries = 0
		} else { // Is expected closure code
			tr.log.Errorf("Websocket readProcess failed: %v", err)
			break
		}

		if t != websocket.TextMessage {
			tr.log.Warnf("Unknown Data received. Data type is not a text message")
			continue // Skip all non text messages
		}

		if handler := tr.getReadHandler(); handler != nil {
			go handler(msg)
		}
	}
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

func (tr *websocketTransport) Close() error {
	// Set isConnected to false and close connection only if previously isConnected value was true.
	if tr.isConnected.CompareAndSwap(true, false) {
		tr.log.Debugf("Close websocket connection")
		if tr.conn != nil {
			if err := tr.conn.Close(); err != nil {
				return common.NewGameLiftError(common.WebsocketClosingError, "", err.Error())
			}
		}
	}

	return nil
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
