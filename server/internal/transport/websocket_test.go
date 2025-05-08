/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package transport_test

import (
	"aws/amazon-gamelift-go-sdk/common"
	"aws/amazon-gamelift-go-sdk/server/internal/mock"
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/websocket"
	"go.uber.org/goleak"

	"aws/amazon-gamelift-go-sdk/server/internal/transport"
)

var retryableErrorTypes = [...]error{&websocket.CloseError{Code: websocket.CloseAbnormalClosure}, errors.New("example propogated error")}

const (
	rawAddr     = "https://example.test"
	testMessage = `{"key": "value"}`
)

func noopCloseHandler(int, string) error {
	return nil
}

func expectCloseTimes(times int, logger *mock.MockILogger, conn *mock.MockConn) {
	logger.
		EXPECT().
		Debugf("Close websocket connection").
		Times(times)
	conn.
		EXPECT().
		Close().
		Times(times)
}

func createMockWebsocket(t *testing.T) (transport.ITransport, *mock.MockDialer, *mock.MockConn, *mock.MockILogger) {
	ctrl := gomock.NewController(t)
	dialer := mock.NewMockDialer(ctrl)
	conn := mock.NewMockConn(ctrl)
	logger := mock.NewMockILogger(ctrl)
	tr := transport.Websocket(logger, dialer)
	return tr, dialer, conn, logger
}

func expectConnectTimes(times int, logger *mock.MockILogger, dialer *mock.MockDialer, conn *mock.MockConn) {
	// GIVEN
	dialer.
		EXPECT().
		Dial(rawAddr, http.Header{"User-Agent": []string{"gamelift-go-sdk/1.0"}}).
		Return(conn, new(http.Response), error(nil)).
		Times(times)
	conn.
		EXPECT().
		CloseHandler().
		Return(noopCloseHandler).
		Times(times)

	// EXPECT
	logger.
		EXPECT().
		Debugf("Connection string: %s", gomock.Any()).
		Times(times)
	conn.
		EXPECT().
		SetCloseHandler(gomock.Any()).
		Times(times)
}

func expectReconnect(logger *mock.MockILogger, dialer *mock.MockDialer, conn *mock.MockConn) {
	expectConnectTimes(2, logger, dialer, conn)
	// The underlying websocket is closed three times; however the websocket transport is closed twice.
	expectCloseTimes(2, logger, conn)
	conn.
		EXPECT().
		Close().
		Times(1)
}

func TestWebsocketWriteRefreshConnection(t *testing.T) {
	// GIVEN
	defer goleak.VerifyNone(t)
	addr, err := url.Parse(rawAddr)
	if err != nil {
		t.Fatalf("parse url: %s", err)
	}
	tr, dialer, conn, logger := createMockWebsocket(t)
	conn.
		EXPECT().
		ReadMessage().
		Return(websocket.TextMessage, []byte(testMessage), error(nil)).
		AnyTimes()

	var connectionsRefreshedWaitGroup sync.WaitGroup
	refreshConnection := func(t *testing.T, tr transport.ITransport) {
		defer connectionsRefreshedWaitGroup.Done()
		err := tr.Connect(addr)
		if err != nil {
			t.Fatalf("websocket connect: %v", err)
		}
	}

	// EXPECT
	expectConnectTimes(101, logger, dialer, conn)
	expectCloseTimes(101, logger, conn)
	conn.EXPECT().WriteMessage(gomock.Any(), gomock.Any()).Times(100)

	// WHEN
	connectionsRefreshedWaitGroup.Add(1)
	refreshConnection(t, tr)
	for i := 0; i < 100; i++ {
		connectionsRefreshedWaitGroup.Add(1)
		go refreshConnection(t, tr)
		err = tr.Write([]byte(testMessage))
		if err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}
	connectionsRefreshedWaitGroup.Wait()
	err = tr.Close()
	if err != nil {
		t.Fatalf("websocket close connection: %v", err)
	}
}

func TestWebsocketTransportRead(t *testing.T) {
	// GIVEN
	defer goleak.VerifyNone(t)
	addr, err := url.Parse(rawAddr)
	if err != nil {
		t.Fatalf("parse url: %s", err)
	}
	tr, dialer, conn, logger := createMockWebsocket(t)
	conn.
		EXPECT().
		ReadMessage().
		Return(websocket.TextMessage, []byte(testMessage), error(nil)).
		AnyTimes()
	var handlerCalled common.AtomicBool
	tr.SetReadHandler(func(data []byte) {
		handlerCalled.Store(true)
		if string(data) != testMessage {
			t.Fatalf("unexpected message: %s", data)
		}
	})

	// EXPECT
	expectConnectTimes(1, logger, dialer, conn)
	expectCloseTimes(1, logger, conn)

	// WHEN
	err = tr.Connect(addr)
	if err != nil {
		t.Fatalf("websocket connect: %v", err)
	}
	time.Sleep(time.Millisecond) // wait for read handler
	err = tr.Close()
	if err != nil {
		t.Fatalf("websocket close connection: %v", err)
	}

	// THEN
	if !handlerCalled.Load() {
		t.Fatalf("handler was not called")
	}
}

func TestWebsocketTransportWrite(t *testing.T) {
	// GIVEN
	defer goleak.VerifyNone(t)
	addr, err := url.Parse(rawAddr)
	if err != nil {
		t.Fatalf("parse url: %s", err)
	}
	tr, dialer, conn, logger := createMockWebsocket(t)
	conn.
		EXPECT().
		ReadMessage().
		Return(websocket.TextMessage, []byte(nil), error(nil)).
		AnyTimes()

	// EXPECT
	expectConnectTimes(1, logger, dialer, conn)
	expectCloseTimes(1, logger, conn)
	conn.
		EXPECT().
		WriteMessage(websocket.TextMessage, []byte(testMessage))

	// WHEN
	err = tr.Connect(addr)
	if err != nil {
		t.Fatalf("websocket connect: %v", err)
	}
	err = tr.Write([]byte(testMessage))
	if err != nil {
		t.Fatalf("fail to write to transport: %v", err)
	}
	err = tr.Close()
	if err != nil {
		t.Fatalf("websocket close connection: %v", err)
	}
}

func TestWebsocketRetryConnection(t *testing.T) {
	// GIVEN
	defer goleak.VerifyNone(t)
	addr, err := url.Parse(rawAddr)
	if err != nil {
		t.Fatalf("parse url: %s", err)
	}
	tr, dialer, conn, logger := createMockWebsocket(t)

	// Mock failed connection attempt
	errorResponse := new(http.Response)
	errorResponse.Body = ioutil.NopCloser(bytes.NewBufferString(""))
	gomock.InOrder(
		dialer.
			EXPECT().
			Dial(rawAddr, http.Header{"User-Agent": []string{"gamelift-go-sdk/1.0"}}).
			Return(conn, errorResponse, errors.New("Test error")),
		dialer.
			EXPECT().
			Dial(rawAddr, http.Header{"User-Agent": []string{"gamelift-go-sdk/1.0"}}).
			Return(conn, new(http.Response), error(nil)),
	)

	// EXPECT
	expectCloseTimes(1, logger, conn)

	// Expect connection with retry 1 time
	conn.
		EXPECT().
		CloseHandler().
		Return(noopCloseHandler)

	conn.
		EXPECT().
		SetCloseHandler(gomock.Any())

	logger.
		EXPECT().
		Debugf("Connection string: %s", gomock.Any())

	logger.
		EXPECT().
		Debugf("Response header is: %v", gomock.Any())

	logger.
		EXPECT().
		Debugf("Response body is: %s", gomock.Any())

	// WHEN
	err = tr.Connect(addr)
	if err != nil {
		t.Fatalf("websocket connect: %v", err)
	}

	err = tr.Close()
	if err != nil {
		t.Fatalf("websocket close connection: %v", err)
	}
}

func TestWebsocketReconnectOnClose_Read(t *testing.T) {
	// GIVEN
	defer goleak.VerifyNone(t)

	for _, readError := range retryableErrorTypes {
		addr, err := url.Parse(rawAddr)
		if err != nil {
			t.Fatalf("parse url: %s", err)
		}
		tr, dialer, conn, logger := createMockWebsocket(t)
		var handlerCalled common.AtomicBool
		tr.SetReadHandler(func(data []byte) {
			handlerCalled.Store(true)
			if string(data) != testMessage {
				t.Fatalf("unexpected message: %s", data)
			}
		})

		// Mock network interrupt on read
		gomock.InOrder(
			conn.EXPECT().ReadMessage().Return(websocket.TextMessage, []byte(testMessage), readError).Times(common.ReconnectOnReadWriteFailureNumber+1),
			conn.
				EXPECT().
				ReadMessage().
				Return(websocket.TextMessage, []byte(testMessage), error(nil)).AnyTimes())

		// EXPECT
		expectReconnect(logger, dialer, conn)
		logger.EXPECT().Debugf("Failed to read message with error %v, retrying...", gomock.Any()).Times(common.ReconnectOnReadWriteFailureNumber)
		logger.EXPECT().Warnf("Detected network interruption %s! Reconnecting...", gomock.Any())

		// WHEN
		err = tr.Connect(addr)
		if err != nil {
			t.Fatalf("websocket connect: %v", err)
		}

		conn.Close()

		// Waits for readProcess to pull a message and calls the handler after retrying for ~2 seconds.
		time.Sleep(10 * time.Second)

		err = tr.Close()
		if err != nil {
			t.Fatalf("websocket close connection: %v", err)
		}

		// THEN
		if !handlerCalled.Load() {
			t.Fatalf("handler was not called")
		}
	}
}

func TestWebsocketReconnectOnClose_Write(t *testing.T) {
	// GIVEN
	defer goleak.VerifyNone(t)

	for _, writeError := range retryableErrorTypes {
		addr, err := url.Parse(rawAddr)
		if err != nil {
			t.Fatalf("parse url: %s", err)
		}
		tr, dialer, conn, logger := createMockWebsocket(t)

		// Mock network interrupt on write
		gomock.InOrder(
			conn.
				EXPECT().
				WriteMessage(websocket.TextMessage, []byte(testMessage)).Return(writeError).Times(common.ReconnectOnReadWriteFailureNumber+1),
			conn.
				EXPECT().
				WriteMessage(websocket.TextMessage, []byte(testMessage)).Return(nil))

		// EXPECT
		expectReconnect(logger, dialer, conn)
		conn.
			EXPECT().
			ReadMessage().
			Return(websocket.TextMessage, []byte(nil), error(nil)).
			AnyTimes()
		logger.EXPECT().Warnf("Detected network interruption %s! Reconnecting...", gomock.Any())
		logger.EXPECT().Debugf("Failed to write message: %v, retrying...", gomock.Any()).Times(common.ReconnectOnReadWriteFailureNumber)

		// WHEN
		err = tr.Connect(addr)
		if err != nil {
			t.Fatalf("websocket connect: %v", err)
		}

		conn.Close()

		err = tr.Write([]byte(testMessage))
		if err != nil {
			t.Fatalf("fail to write to transport: %v", err)
		}

		err = tr.Close()
		if err != nil {
			t.Fatalf("websocket close connection: %v", err)
		}
	}
}

func TestWebsocketWriteDoesNotReconnectForNormalClosure(t *testing.T) {
	// GIVEN
	defer goleak.VerifyNone(t)
	addr, err := url.Parse(rawAddr)
	if err != nil {
		t.Fatalf("parse url: %s", err)
	}
	tr, dialer, conn, logger := createMockWebsocket(t)

	// Mock websocket closed on write
	conn.
		EXPECT().
		WriteMessage(websocket.TextMessage, []byte(testMessage)).
		Return(&websocket.CloseError{Code: websocket.CloseNormalClosure})
	conn.
		EXPECT().
		ReadMessage().
		Return(websocket.TextMessage, []byte(nil), error(nil)).
		AnyTimes()

	// EXPECT
	expectConnectTimes(1, logger, dialer, conn)
	expectCloseTimes(1, logger, conn)
	conn.EXPECT().Close().Times(1)

	// WHEN
	err = tr.Connect(addr)
	if err != nil {
		t.Fatalf("websocket connect: %v", err)
	}

	conn.Close()

	err = tr.Write([]byte(testMessage))
	if err == nil {
		t.Fatalf("write should have failed because the connection is closed normally")
	}

	err = tr.Close()
	if err != nil {
		t.Fatalf("websocket close connection: %v", err)
	}
}

func TestWebsocketReadDoesNotReconnectForNormalClosure(t *testing.T) {
	// GIVEN
	defer goleak.VerifyNone(t)
	addr, err := url.Parse(rawAddr)
	if err != nil {
		t.Fatalf("parse url: %s", err)
	}
	tr, dialer, conn, logger := createMockWebsocket(t)

	// Mock websocket closed on read
	conn.
		EXPECT().
		ReadMessage().
		Return(websocket.TextMessage, []byte(testMessage), &websocket.CloseError{Code: websocket.CloseNormalClosure}).
		AnyTimes()

	// EXPECT
	expectConnectTimes(1, logger, dialer, conn)
	expectCloseTimes(1, logger, conn)
	conn.EXPECT().Close().Times(1)
	logger.
		EXPECT().
		Errorf("Websocket readProcess failed: %v", gomock.Any())

	// WHEN
	err = tr.Connect(addr)
	if err != nil {
		t.Fatalf("websocket connect: %v", err)
	}
	conn.Close()
	time.Sleep(10 * time.Second) // wait for read handler

	err = tr.Close()
	if err != nil {
		t.Fatalf("websocket close connection: %v", err)
	}
}

func TestWebsocketConcurrentReconnectConnectsOnce(t *testing.T) {
	// GIVEN
	defer goleak.VerifyNone(t)
	addr, err := url.Parse(rawAddr)
	if err != nil {
		t.Fatalf("parse url: %s", err)
	}
	tr, dialer, conn, logger := createMockWebsocket(t)

	// EXPECT
	conn.
		EXPECT().
		CloseHandler().
		Return(noopCloseHandler).
		Times(2)
	logger.
		EXPECT().
		Debugf("Connection string: %s", gomock.Any()).
		Times(2)
	conn.
		EXPECT().
		SetCloseHandler(gomock.Any()).
		Times(2)
	conn.
		EXPECT().
		ReadMessage().
		Return(websocket.TextMessage, []byte(testMessage), error(nil)).
		AnyTimes()
	dialer.EXPECT().Dial(rawAddr, http.Header{"User-Agent": []string{"gamelift-go-sdk/1.0"}}).
		DoAndReturn(func(string, http.Header) (transport.Conn, *http.Header, error) {
			time.Sleep(time.Millisecond)
			return conn, nil, nil
		}).Times(2)
	expectCloseTimes(2, logger, conn)

	// WHEN
	err = tr.Connect(addr)
	if err != nil {
		t.Fatalf("websocket connect: %v", err)
	}
	go tr.Reconnect()
	err = tr.Reconnect()
	if err != nil {
		t.Fatalf("Reconnect failed: %v", err)
	}
	err = tr.Close()
	if err != nil {
		t.Fatalf("websocket close connection: %v", err)
	}
}
