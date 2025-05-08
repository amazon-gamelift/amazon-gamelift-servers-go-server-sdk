/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package transport_test

import (
	"aws/amazon-gamelift-go-sdk/common"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/websocket"
	"go.uber.org/goleak"

	"aws/amazon-gamelift-go-sdk/server/internal/mock"
	"aws/amazon-gamelift-go-sdk/server/internal/transport"
)

const (
	rawAddr     = "https://example.test"
	testMessage = `{"key": "value"}`
)

func TestWebsocketTransportRead(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctrl := gomock.NewController(t)

	logger := mock.NewMockILogger(ctrl)
	dialer := mock.NewMockDialer(ctrl)
	conn := mock.NewMockConn(ctrl)

	dialer.
		EXPECT().
		Dial(rawAddr, http.Header{"User-Agent": []string{"gamelift-go-sdk/1.0"}}).
		Return(conn, new(http.Response), error(nil))

	conn.
		EXPECT().
		CloseHandler().
		Return(noopCloseHandler)

	conn.
		EXPECT().
		SetCloseHandler(gomock.Any())

	conn.
		EXPECT().
		ReadMessage().
		Return(websocket.TextMessage, []byte(testMessage), error(nil)).
		AnyTimes()

	logger.
		EXPECT().
		Debugf("Connection string: %s", gomock.Any())

	logger.
		EXPECT().
		Debugf("Close websocket connection").
		MinTimes(1)

	conn.
		EXPECT().
		Close().
		MinTimes(1)

	tr := transport.Websocket(logger, dialer)

	var handlerCalled common.AtomicBool
	tr.SetReadHandler(func(data []byte) {
		handlerCalled.Store(true)
		if string(data) != testMessage {
			t.Fatalf("unexpected message: %s", data)
		}
	})

	addr, err := url.Parse(rawAddr)
	if err != nil {
		t.Fatalf("parse url: %s", err)
	}

	err = tr.Connect(addr)
	if err != nil {
		t.Fatalf("websocket connect: %v", err)
	}

	time.Sleep(time.Millisecond) // wait for read handler

	err = tr.Close()
	if err != nil {
		t.Fatalf("websocket close connection: %v", err)
	}

	if !handlerCalled.Load() {
		t.Fatalf("handler was not called")
	}
}

func TestWebsocketTransportWrite(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctrl := gomock.NewController(t)

	logger := mock.NewMockILogger(ctrl)
	dialer := mock.NewMockDialer(ctrl)
	conn := mock.NewMockConn(ctrl)

	dialer.
		EXPECT().
		Dial(rawAddr, http.Header{"User-Agent": []string{"gamelift-go-sdk/1.0"}}).
		Return(conn, new(http.Response), error(nil))

	conn.
		EXPECT().
		CloseHandler().
		Return(noopCloseHandler)

	conn.
		EXPECT().
		SetCloseHandler(gomock.Any())

	conn.
		EXPECT().
		WriteMessage(websocket.TextMessage, []byte(testMessage))

	conn.
		EXPECT().
		ReadMessage().
		Return(websocket.TextMessage, []byte(nil), error(nil)).
		AnyTimes()

	logger.
		EXPECT().
		Debugf("Connection string: %s", gomock.Any())

	logger.
		EXPECT().
		Debugf("Close websocket connection").
		MinTimes(1)

	conn.
		EXPECT().
		Close().
		MinTimes(1)

	tr := transport.Websocket(logger, dialer)

	addr, err := url.Parse(rawAddr)
	if err != nil {
		t.Fatalf("parse url: %s", err)
	}

	err = tr.Connect(addr)
	if err != nil {
		t.Fatalf("websocket connect: %v", err)
	}

	err = tr.Write([]byte(testMessage))
	if err != nil {
		t.Fatalf("fall to write to transport: %v", err)
	}

	err = tr.Close()
	if err != nil {
		t.Fatalf("websocket close connectin: %v", err)
	}
}

func noopCloseHandler(int, string) error {
	return nil
}
