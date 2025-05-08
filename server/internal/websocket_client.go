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

	"aws/amazon-gamelift-go-sdk/common"
	"aws/amazon-gamelift-go-sdk/model/message"
	"aws/amazon-gamelift-go-sdk/server/internal/transport"
	"aws/amazon-gamelift-go-sdk/server/log"
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

// SendRequest - sends message to the GameLift server via websocket, answer will be sent to the resp channel.
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
	if err := c.SendMessage(req); err != nil {
		c.sendResponse(r.RequestID, nil, err)
		return err
	}

	return nil
}

// SendMessage - sends message to the GameLift server without waiting for a response.
func (c *websocketClient) SendMessage(msg any) error {
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

// Close closes underlying connections and releases their associated resources.
// All Send calls after Close call will return an error.
func (c *websocketClient) Close() error {
	c.respMtx.Lock()
	for reqID, resp := range c.responses {
		close(resp)
		delete(c.responses, reqID)
	}
	c.respMtx.Unlock()
	return c.iTransport.Close()
}

func (c *websocketClient) getHandlerByAction(action message.MessageAction) (func([]byte), bool) {
	c.handleMtx.RLock()
	defer c.handleMtx.RUnlock()
	handler, ok := c.asyncHandlers[action]
	return handler, ok
}

func (c *websocketClient) readHandler(data []byte) {
	// Try to find Action and RequestId in received data
	var resp message.ResponseMessage
	if err := json.Unmarshal(data, &resp); err != nil {
		c.log.Warnf("Failed %s when try deserialize response", err.Error())
		return
	}

	c.log.Debugf("Received %s for GameLift with status %d. Data is %s", resp.Action, resp.StatusCode, data)

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
