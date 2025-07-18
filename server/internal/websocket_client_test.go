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

	"github.com/golang/mock/gomock"
	"go.uber.org/goleak"

	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/model/message"
	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/model/request"
	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/server/internal"
	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/server/internal/mock"
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
