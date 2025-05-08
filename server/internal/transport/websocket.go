/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package transport

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"aws/amazon-gamelift-go-sdk/common"
	"aws/amazon-gamelift-go-sdk/server/log"

	"github.com/gorilla/websocket"
)

// websocketTransport - implement ITransport interface for websocket connection.
type websocketTransport struct {
	log    log.ILogger
	dialer Dialer

	conn        Conn
	isConnected common.AtomicBool
	writeMtx    sync.Mutex

	readHandlerMu sync.RWMutex
	readHandler   ReadHandler
}

// Websocket creates a new instance of the ITransport implementation.
func Websocket(logger log.ILogger, dialer Dialer) ITransport {
	return &websocketTransport{
		log:    logger,
		dialer: dialer,
	}
}

func (tr *websocketTransport) Connect(u *url.URL) error {
	if err := tr.Close(); err != nil {
		tr.log.Debugf("Error occurred when try close websocket connection: %s", err)
	}
	tr.log.Debugf("Connection string: %s", u)

	//nolint:bodyclose // The response body may not contain the entire response and does not need to be closed by the application
	conn, resp, err := tr.dialer.Dial(u.String(), http.Header{"User-Agent": []string{"gamelift-go-sdk/1.0"}})
	if err != nil {
		var reason string
		if resp != nil {
			reason = resp.Status
			b, _ := io.ReadAll(resp.Body)
			tr.log.Debugf("Response header is: %v", resp.Header)
			tr.log.Debugf("Response body is: %s", b)
		}
		return common.NewGameLiftError(common.WebsocketConnectFailure,
			"",
			fmt.Sprintf("connection error %s:%s", reason, err.Error()),
		)
	}

	tr.conn = conn
	tr.setCloseHandler()

	tr.isConnected.Store(true)
	go tr.readProcess()
	return nil
}

func (tr *websocketTransport) setCloseHandler() {
	// wraps a default handler that correctly implements the protocol specification.
	currentHandler := tr.conn.CloseHandler()
	tr.conn.SetCloseHandler(func(code int, text string) error {
		tr.log.Debugf("Socket disconnected. Code is %d. Reason is %s", code, text)
		tr.isConnected.Store(false)

		return currentHandler(code, text)
	})
}

func (tr *websocketTransport) readProcess() {
	defer tr.Close()
	for tr.isConnected.Load() {
		// ReadMessage will read all message from the NextReader
		// The returned messageType is either TextMessage or BinaryMessage.
		t, msg, err := tr.conn.ReadMessage()
		if err != nil {
			tr.log.Errorf("Websocket readHandler failed: %v", err)
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

func (tr *websocketTransport) write(data []byte) error {
	tr.writeMtx.Lock() // the gorilla/websocket WriteMessage is not thread safe
	defer tr.writeMtx.Unlock()
	return tr.conn.WriteMessage(websocket.TextMessage, data)
}

func (tr *websocketTransport) Write(data []byte) error {
	if !tr.isConnected.Load() {
		return common.NewGameLiftError(common.GameLiftServerNotInitialized, "", "")
	}
	if err := tr.write(data); err != nil {
		return common.NewGameLiftError(common.WebsocketSendMessageFailure, "Failed write data", err.Error())
	}

	return nil
}
