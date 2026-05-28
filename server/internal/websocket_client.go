/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package internal

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/model/message"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server/internal/transport"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server/log"
)

var (
	initWebsocketOnce sync.Once
	gameliftWebsocket websocketClient
)

// websocketClient - Singleton, implements IWebSocketClient interface.
// Stores all handlers for requests and messages
type websocketClient struct {
	iTransport    transport.ITransport
	log           log.ILogger
	respMtx       sync.Mutex
	handleMtx     sync.RWMutex
	responses     map[string]chan<- common.Outcome
	asyncHandlers map[message.MessageAction]func([]byte)

	// consecutiveTimeouts tracks consecutive HandleRequest timeouts since the last successful response.
	// Reset to 0 on any successful response. When it reaches
	// common.RequestTimeoutReconnectThreshold, a transport reconnect is triggered to recover
	// from half-open WebSocket connections.
	// Accessed atomically via sync/atomic primitives.
	consecutiveTimeouts int32
	// reconnectInFlight ensures only one reconnect is kicked off per streak of timeouts.
	reconnectInFlight common.AtomicBool
}

// GetWebsocketClient - return an implementation of IWebSocketClient.
func GetWebsocketClient(
	iTransport transport.ITransport,
	l log.ILogger,
) IWebSocketClient {
	initWebsocketOnce.Do(func() {
		gameliftWebsocket.init(iTransport, l)
	})

	return &gameliftWebsocket
}

func (c *websocketClient) init(iTransport transport.ITransport, l log.ILogger) {
	c.iTransport = iTransport
	c.log = l
	c.responses = make(map[string]chan<- common.Outcome)
	c.asyncHandlers = make(map[message.MessageAction]func([]byte))
	c.iTransport.SetReadHandler(gameliftWebsocket.readHandler)
}

// Connect creates a websocket connection with the specified address.
// All Send calls before Connect call will return an error.
func (c *websocketClient) Connect(connectURL *url.URL) error {
	if err := c.iTransport.Connect(connectURL); err != nil {
		return err
	}
	c.log.Debugf("Connected to GameLift API Gateway.")

	return nil
}

// SendRequest - sends message to the game server process via websocket, answer will be sent to the resp channel.
func (c *websocketClient) SendRequest(req MessageGetter, resp chan<- common.Outcome) error {
	if resp == nil {
		return common.NewGameLiftError(common.BadRequestException, "", "invalid input parameters")
	}

	r := req.GetMessage()
	if r.RequestID == "" {
		return common.NewGameLiftError(common.BadRequestException, "", "empty RequestID")
	}

	if err := c.storeResponse(r.RequestID, resp); err != nil {
		return err
	}
	if err := c.sendMessage(req); err != nil {
		c.sendResponse(r.RequestID, nil, err)
		return err
	}

	return nil
}

// SendMessage - sends message to the game server process without waiting for a response.
func (c *websocketClient) sendMessage(msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return common.NewGameLiftError(common.ServiceCallFailed, "Failed serialize data", err.Error())
	}
	if err = c.iTransport.Write(data); err != nil {
		return common.NewGameLiftError(common.ServiceCallFailed, "Failed write data", err.Error())
	}
	return nil
}

// AddHandler allows to register an incoming message handler with the specified Action.
func (c *websocketClient) AddHandler(action message.MessageAction, handler func([]byte)) {
	c.handleMtx.Lock()
	defer c.handleMtx.Unlock()
	c.asyncHandlers[action] = handler
}

// CancelRequest allows to cancel request if the request time duration was expire.
func (c *websocketClient) CancelRequest(requestID string) {
	c.sendResponse(requestID, nil, nil)
}

// NotifyRequestTimeout - increments the consecutive-timeout counter. When the counter reaches
// common.RequestTimeoutReconnectThreshold, an asynchronous Reconnect is initiated on the
// underlying transport. This recovers from half-open WebSocket connections where requests
// enqueue successfully but responses never arrive.
//
// The counter is reset to 0 by any successful response received through readHandler.
// A single reconnect goroutine is in flight at a time to avoid flooding Reconnect() on
// repeated timeouts after the threshold is crossed.
func (c *websocketClient) NotifyRequestTimeout() {
	n := atomic.AddInt32(&c.consecutiveTimeouts, 1)
	if int(n) < common.RequestTimeoutReconnectThreshold {
		return
	}
	// Kick off at most one reconnect per streak of timeouts. Once a reconnect is in flight,
	// the Reconnect() call itself serializes via the transport's writeMtx, and subsequent
	// timeouts stacking up on the dead socket simply bump the counter without doing work.
	if !c.reconnectInFlight.CompareAndSwap(false, true) {
		return
	}
	c.log.Warnf("Reached %d consecutive request timeouts — triggering transport reconnect", n)
	go func() {
		defer c.reconnectInFlight.Store(false)
		if err := c.iTransport.Reconnect(); err != nil {
			c.log.Errorf("Reconnect after consecutive request timeouts failed: %s", err)
			return
		}
		// A successful reconnect resets the streak. Successful round-trips on the new
		// connection will keep it at zero via readHandler; setting it here prevents a
		// lingering stale counter value if no traffic follows immediately.
		atomic.StoreInt32(&c.consecutiveTimeouts, 0)
	}()
}

// Close closes underlying connections and releases their associated resources.
// All Send calls after Close call will return an error.
func (c *websocketClient) Close() error {
	c.respMtx.Lock()
	for reqID, resp := range c.responses {
		close(resp)
		delete(c.responses, reqID)
	}
	c.respMtx.Unlock()
	c.iTransport.PreventAutoReconnect()
	return c.iTransport.Close()
}

func (c *websocketClient) getHandlerByAction(action message.MessageAction) (func([]byte), bool) {
	c.handleMtx.RLock()
	defer c.handleMtx.RUnlock()
	handler, ok := c.asyncHandlers[action]
	return handler, ok
}

func (c *websocketClient) readHandler(data []byte) {
	// Any inbound frame is evidence of transport liveness — reset the consecutive-timeout
	// counter so the next request-timeout streak starts from zero.
	atomic.StoreInt32(&c.consecutiveTimeouts, 0)

	// Try to find Action and RequestId in received data
	var resp message.ResponseMessage
	if err := json.Unmarshal(data, &resp); err != nil {
		c.log.Warnf("Failed %s when try deserialize response", err.Error())
		return
	}

	c.log.Debugf("Received %s for GameLift with status %d.", resp.Action, resp.StatusCode)

	if resp.StatusCode != http.StatusOK && resp.RequestID != "" {
		c.log.Warnf(
			"Received unsuccessful status code %d for request %s with message %q",
			resp.StatusCode,
			resp.RequestID,
			resp.ErrorMessage,
		)
		err := common.NewGameLiftErrorFromStatusCode(resp.StatusCode, resp.ErrorMessage)
		c.sendResponse(resp.RequestID, data, err)
		return
	}

	if handler, ok := c.getHandlerByAction(resp.Action); ok {
		handler(data)
		return
	}

	c.sendResponse(resp.RequestID, data, nil)
}

func (c *websocketClient) storeResponse(requestID string, resp chan<- common.Outcome) error {
	c.respMtx.Lock()
	defer c.respMtx.Unlock()
	if _, ok := c.responses[requestID]; ok {
		c.log.Errorf("Request %s already exists.", requestID)
		return common.NewGameLiftError(common.InternalServiceException, "", "")
	}
	c.responses[requestID] = resp
	return nil
}

func (c *websocketClient) sendResponse(requestID string, data []byte, err error) {
	c.respMtx.Lock()
	defer c.respMtx.Unlock()
	resp, ok := c.responses[requestID]
	if !ok {
		c.log.Debugf("Response received for message with ID: %s", requestID)
		return
	}
	if data != nil {
		resp <- common.Outcome{Data: data, Error: err}
	}
	close(resp)
	delete(c.responses, requestID)
}
