/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package internal_test

import (
	"bytes"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"go.uber.org/goleak"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/model/message"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/model/request"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server/internal"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server/internal/mock"
)

const rawAddr = "https://example.test"

var testRequest = request.DescribePlayerSessionsRequest{
	Message: message.Message{
		Action:    message.DescribePlayerSessions,
		RequestID: "test-request-id",
	},
	PlayerID:        "test-player-id",
	PlayerSessionID: "test-player-session-id",
	NextToken:       "test-next-token",
	Limit:           1,
}

var testRequestJSON = `{"Action":"DescribePlayerSessions","RequestId":"test-request-id","PlayerId":"test-player-id","PlayerSessionId":"test-player-session-id","NextToken":"test-next-token","Limit":1}`

func TestWebsocketClientSendRequest(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctrl := gomock.NewController(t)

	logger := mock.NewTestLogger(t, ctrl)
	transportMock := mock.NewMockITransport(ctrl)

	c := new(internal.WebsocketClient)
	transportMock.
		EXPECT().
		SetReadHandler(gomock.Not(gomock.Nil())) // we can't compare functions

	c.Init(transportMock, logger)

	addr, err := url.Parse(rawAddr)
	if err != nil {
		t.Fatalf("parse url: %s", err)
	}

	transportMock.
		EXPECT().
		Connect(addr)

	transportMock.
		EXPECT().
		Write([]byte(testRequestJSON))

	transportMock.
		EXPECT().
		PreventAutoReconnect()

	transportMock.
		EXPECT().
		Close()

	if err := c.Connect(addr); err != nil {
		t.Fatal(err)
	}

	req := testRequest
	respCh := make(chan common.Outcome, 1)
	if err := c.SendRequest(req, respCh); err != nil {
		t.Fatal(err)
	}

	const rawResponse = `{
  "Action": "DescribePlayerSessions",
  "RequestId": "test-request-id",
  "NextToken": "test-next-token",
  "PlayerSessions": [
    {
      "PlayerId": "test-player-id",
      "PlayerSessionId": "test-player-session-id",
      "GameSessionId": "",
      "FleetId": "",
      "PlayerData": "",
      "IpAddress": "",
      "Port": 0,
      "CreationTime": 0,
      "TerminationTime": 0,
      "DnsName": ""
    }
  ]
}`
	c.RunReadHandler([]byte(rawResponse))

	if !bytes.Equal((<-respCh).Data, []byte(rawResponse)) {
		t.Fatal("unexpected response")
	}

	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestWebsocketClientHandler(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctrl := gomock.NewController(t)

	logger := mock.NewTestLogger(t, ctrl)
	transportMock := mock.NewMockITransport(ctrl)

	createGameSessionHandler := mock.NewMockMessageHandler(ctrl)
	updateGameSessionHandler := mock.NewMockMessageHandler(ctrl)
	refreshConnectionHandler := mock.NewMockMessageHandler(ctrl)
	terminateProcessHandler := mock.NewMockMessageHandler(ctrl)

	c := new(internal.WebsocketClient)
	transportMock.
		EXPECT().
		SetReadHandler(gomock.Not(gomock.Nil())) // we can't compare functions

	c.Init(transportMock, logger)

	addr, err := url.Parse(rawAddr)
	if err != nil {
		t.Fatalf("parse url: %s", err)
	}

	transportMock.
		EXPECT().
		Connect(addr)

	const (
		createGameSessionRequest = `{"Action": "CreateGameSession"}`
		updateGameSessionRequest = `{"Action": "UpdateGameSession"}`
		refreshConnectionRequest = `{"Action": "RefreshConnection"}`
		terminateProcessRequest  = `{"Action": "TerminateProcess"}`
	)

	createGameSessionHandler.
		EXPECT().
		OnMessage([]byte(createGameSessionRequest))

	updateGameSessionHandler.
		EXPECT().
		OnMessage([]byte(updateGameSessionRequest))

	refreshConnectionHandler.
		EXPECT().
		OnMessage([]byte(refreshConnectionRequest))

	terminateProcessHandler.
		EXPECT().
		OnMessage([]byte(terminateProcessRequest))

	transportMock.
		EXPECT().
		PreventAutoReconnect()

	transportMock.
		EXPECT().
		Close()

	if err := c.Connect(addr); err != nil {
		t.Fatal(err)
	}

	c.AddHandler(message.CreateGameSession, createGameSessionHandler.OnMessage)
	c.AddHandler(message.UpdateGameSession, updateGameSessionHandler.OnMessage)
	c.AddHandler(message.RefreshConnection, refreshConnectionHandler.OnMessage)
	c.AddHandler(message.TerminateProcess, terminateProcessHandler.OnMessage)

	c.RunReadHandler(nil)
	c.RunReadHandler([]byte("invalid json"))
	c.RunReadHandler([]byte(createGameSessionRequest))
	c.RunReadHandler([]byte(updateGameSessionRequest))
	c.RunReadHandler([]byte(refreshConnectionRequest))
	c.RunReadHandler([]byte(terminateProcessRequest))

	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestWebsocketClientHandlerError(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctrl := gomock.NewController(t)

	logger := mock.NewTestLogger(t, ctrl)
	transportMock := mock.NewMockITransport(ctrl)

	c := new(internal.WebsocketClient)
	transportMock.
		EXPECT().
		SetReadHandler(gomock.Not(gomock.Nil())) // we can't compare functions

	c.Init(transportMock, logger)

	transportMock.
		EXPECT().
		Write([]byte(testRequestJSON))

	req := testRequest

	respCh := make(chan common.Outcome, 1)
	if err := c.SendRequest(req, respCh); err != nil {
		t.Fatal(err)
	}

	c.RunReadHandler([]byte(`{
		"Action": null,
		"RequestId": "test-request-id",
		"StatusCode": ` + strconv.Itoa(http.StatusBadRequest) + `,
		"ErrorMessage":"Invalid request: Connect"
	}`))

	result := <-respCh

	expectedError := common.NewGameLiftErrorFromStatusCode(400, "Invalid request: Connect")
	if !reflect.DeepEqual(result.Error, expectedError) {
		t.Fatalf("unexpected error %s, want %s", result.Error, expectedError)
	}
}

// ---------------------------------------------------------------------------
// NotifyRequestTimeout → transport.Reconnect() after
// common.RequestTimeoutReconnectThreshold consecutive timeouts, with reset
// on any inbound frame.
// ---------------------------------------------------------------------------

func TestWebsocketClient_NotifyRequestTimeout_BelowThresholdDoesNotReconnect(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctrl := gomock.NewController(t)
	logger := mock.NewTestLogger(t, ctrl)
	transportMock := mock.NewMockITransport(ctrl)

	c := new(internal.WebsocketClient)
	transportMock.EXPECT().SetReadHandler(gomock.Not(gomock.Nil()))
	c.Init(transportMock, logger)

	// GIVEN: threshold is 2, so one timeout must not trigger Reconnect.
	// WHEN: NotifyRequestTimeout is called once.
	c.NotifyRequestTimeout()

	// THEN: no Reconnect is called (gomock fails if transportMock.Reconnect is invoked),
	// counter is 1, and no reconnect is in flight.
	if got := c.ConsecutiveTimeouts(); got != 1 {
		t.Fatalf("expected consecutiveTimeouts=1 after one timeout, got %d", got)
	}
	if c.ReconnectInFlight() {
		t.Fatal("expected no reconnect in flight below threshold")
	}
}

func TestWebsocketClient_NotifyRequestTimeout_AtThresholdTriggersReconnect(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctrl := gomock.NewController(t)
	logger := mock.NewTestLogger(t, ctrl)
	transportMock := mock.NewMockITransport(ctrl)

	c := new(internal.WebsocketClient)
	transportMock.EXPECT().SetReadHandler(gomock.Not(gomock.Nil()))
	c.Init(transportMock, logger)

	// GIVEN: Reconnect() is expected exactly once when we hit threshold.
	// Signal via a channel so we can deterministically wait for the async goroutine.
	reconnectCalled := make(chan struct{}, 1)
	transportMock.
		EXPECT().
		Reconnect().
		DoAndReturn(func() error {
			reconnectCalled <- struct{}{}
			return nil
		})

	// WHEN: NotifyRequestTimeout is called twice consecutively (hits threshold=2).
	c.NotifyRequestTimeout()
	c.NotifyRequestTimeout()

	// THEN: Reconnect() is invoked on a background goroutine.
	select {
	case <-reconnectCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("Reconnect was not called within 2 seconds after threshold reached")
	}

	// Counter should be reset to 0 by the reconnect goroutine, and the in-flight flag
	// should be cleared via defer. Wait briefly for these to land.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if c.ConsecutiveTimeouts() == 0 && !c.ReconnectInFlight() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("post-reconnect state not reached: consecutiveTimeouts=%d reconnectInFlight=%v",
		c.ConsecutiveTimeouts(), c.ReconnectInFlight())
}

func TestWebsocketClient_NotifyRequestTimeout_InboundFrameResetsCounter(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctrl := gomock.NewController(t)
	logger := mock.NewTestLogger(t, ctrl)
	transportMock := mock.NewMockITransport(ctrl)

	c := new(internal.WebsocketClient)
	transportMock.EXPECT().SetReadHandler(gomock.Not(gomock.Nil()))
	c.Init(transportMock, logger)

	// GIVEN: one timeout has occurred and the counter is at 1.
	c.NotifyRequestTimeout()
	if got := c.ConsecutiveTimeouts(); got != 1 {
		t.Fatalf("setup: expected consecutiveTimeouts=1, got %d", got)
	}

	// WHEN: an inbound frame arrives via readHandler. The payload doesn't need to
	// match any registered handler — any inbound frame resets the counter at the
	// top of readHandler.
	c.RunReadHandler([]byte(`{"Action":"UnknownAction","RequestId":"irrelevant","StatusCode":200}`))

	// THEN: counter is reset to 0. A subsequent single timeout stays under threshold,
	// so Reconnect is still not invoked (gomock fails if it is).
	if got := c.ConsecutiveTimeouts(); got != 0 {
		t.Fatalf("expected consecutiveTimeouts=0 after inbound frame, got %d", got)
	}

	c.NotifyRequestTimeout()
	if got := c.ConsecutiveTimeouts(); got != 1 {
		t.Fatalf("expected consecutiveTimeouts=1 after one post-reset timeout, got %d", got)
	}
	if c.ReconnectInFlight() {
		t.Fatal("expected no reconnect in flight after reset + one timeout")
	}
}

func TestWebsocketClient_NotifyRequestTimeout_OnlyOneReconnectInFlight(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctrl := gomock.NewController(t)
	logger := mock.NewTestLogger(t, ctrl)
	transportMock := mock.NewMockITransport(ctrl)

	c := new(internal.WebsocketClient)
	transportMock.EXPECT().SetReadHandler(gomock.Not(gomock.Nil()))
	c.Init(transportMock, logger)

	// GIVEN: Reconnect() is slow — blocks until the test releases it. This lets us
	// stack additional NotifyRequestTimeout calls while the first reconnect is still
	// in flight and assert no duplicate call happens.
	release := make(chan struct{})
	started := make(chan struct{}, 1)
	// EXPECT exactly one call. If a second call to Reconnect is made, gomock fails.
	transportMock.
		EXPECT().
		Reconnect().
		DoAndReturn(func() error {
			started <- struct{}{}
			<-release
			return nil
		})

	// WHEN: threshold reached, triggering the first (slow) Reconnect.
	c.NotifyRequestTimeout()
	c.NotifyRequestTimeout()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first Reconnect never started")
	}
	if !c.ReconnectInFlight() {
		t.Fatal("expected reconnectInFlight=true while Reconnect is running")
	}

	// AND: many additional timeouts pile up while the first reconnect is still running.
	// None of these must spawn a second Reconnect call.
	for i := 0; i < 10; i++ {
		c.NotifyRequestTimeout()
	}

	// Give any (buggy) second reconnect goroutine a chance to try.
	time.Sleep(50 * time.Millisecond)

	// THEN: release the first reconnect and wait for it to finish. Only one
	// Reconnect() call was made (enforced by gomock's single EXPECT above).
	close(release)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !c.ReconnectInFlight() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("reconnectInFlight never cleared after Reconnect returned")
}
