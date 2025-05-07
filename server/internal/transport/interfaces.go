/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

//go:generate mockgen -destination ../mock/conn.go -package=mock . Conn
//go:generate mockgen -destination ../mock/dialer.go -package=mock . Dialer
//go:generate mockgen -destination ../mock/transport.go -package=mock . ITransport
package transport

import (
	"net/http"
	"net/url"
)

// ReadHandler is a callback function that is called when incoming messages are received.
type ReadHandler func([]byte)

// ITransport is the interface that manages input/output operations on the underlying connection.
type ITransport interface {
	// Connect creates a websocket connection with the specified address.
	// All Write calls before Connect call will return an error.
	Connect(url *url.URL) error

	// Write sends message to underlying connection.
	Write([]byte) error

	// SetReadHandler sets a callback function that is called when incoming messages are received.
	SetReadHandler(ReadHandler)

	// Close closes underlying connections and releases their associated resources.
	// All Write calls after Close call will return an error.
	Close() error
}

// Conn is the interface that represents a websocket connection.
type Conn interface {
	// ReadMessage is a method for reading from connection to a buffer.
	ReadMessage() (messageType int, data []byte, err error)

	// WriteMessage is a method for writing the message to connection.
	WriteMessage(messageType int, data []byte) error

	// CloseHandler returns the current close handler.
	CloseHandler() func(code int, text string) error

	// SetCloseHandler sets the handler for close messages received from the peer.
	SetCloseHandler(h func(code int, text string) error)

	// Close closes the network connection without sending or waiting for a close message.
	Close() error
}

// Dialer is the interface that creates a websocket connection.
type Dialer interface {
	Dial(urlStr string, requestHeader http.Header) (Conn, *http.Response, error)
}
