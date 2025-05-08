/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package internal_test

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"testing"
	"time"

	"aws/amazon-gamelift-go-sdk/common"
	"aws/amazon-gamelift-go-sdk/model"
	"aws/amazon-gamelift-go-sdk/model/message"
	"aws/amazon-gamelift-go-sdk/model/request"
	"aws/amazon-gamelift-go-sdk/model/response"
	"aws/amazon-gamelift-go-sdk/model/result"
	"aws/amazon-gamelift-go-sdk/server/internal"
	"aws/amazon-gamelift-go-sdk/server/internal/mock"

	"github.com/golang/mock/gomock"
	"go.uber.org/goleak"
)

const (
	websocketURL = "https://example.test"
	timeDuration = time.Microsecond
	processID    = "processId"
	hostID       = "hostId"
	fleetID      = "fleetId"
	authToken    = "authToken"
	testMessage  = "test_message"
)

func TestGameliftManagerHandleRequest(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctrl := gomock.NewController(t)

	gameliftMessageHandlerMock := mock.NewMockIGameLiftMessageHandler(ctrl)
	websocketClientMock := mock.NewMockIWebSocketClient(ctrl)
	logger := mock.NewTestLogger(t, ctrl)

	gm := internal.GetGameLiftManager(gameliftMessageHandlerMock, websocketClientMock, logger)

	connectURL, err := url.Parse(websocketURL)
	if err != nil {
		t.Fatalf("parse url: %s", err)
	}

	connectURL.RawQuery = fmt.Sprintf("%s=%s&%s=%s&%s=%s&%s=%s&%s=%s&%s=%s",
		common.PidKey,
		processID,
		common.SdkVersionKey,
		common.SdkVersion,
		common.SdkLanguageKey,
		common.SdkLanguage,
		common.AuthTokenKey,
		authToken,
		common.ComputeIDKey,
		hostID,
		common.FleetIDKey,
		fleetID,
	)

	websocketClientMock.
		EXPECT().
		Connect(connectURL)

	for _, actions := range []message.MessageAction{message.CreateGameSession, message.UpdateGameSession, message.RefreshConnection, message.TerminateProcess} {
		websocketClientMock.
			EXPECT().
			AddHandler(actions, gomock.Not(gomock.Nil()))
	}

	if err := gm.Connect(websocketURL, processID, hostID, fleetID, authToken); err != nil {
		t.Fatal(err)
	}

	websocketClientMock.
		EXPECT().
		SendMessage(testMessage)

	if err := gm.SendMessage(testMessage); err != nil {
		t.Fatal(err)
	}

	req := &request.DescribePlayerSessionsRequest{
		Message: message.Message{
			Action:    message.DescribePlayerSessions,
			RequestID: "test-request-id",
		},
		PlayerID:        "test-player-id",
		PlayerSessionID: "test-player-session-id",
		NextToken:       "test-next-token",
		Limit:           1,
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

	var resp *response.DescribePlayerSessionsResponse

	websocketClientMock.
		EXPECT().
		SendRequest(req, gomock.Any()).
		Do(func(req internal.MessageGetter, resp chan<- common.Outcome) error {
			resp <- common.Outcome{Data: []byte(rawResponse)}
			return nil
		})

	respData := &response.DescribePlayerSessionsResponse{
		Message: message.Message{
			Action:    message.DescribePlayerSessions,
			RequestID: "test-request-id",
		},
		DescribePlayerSessionsResult: result.DescribePlayerSessionsResult{
			NextToken: "test-next-token",
			PlayerSessions: []model.PlayerSession{
				{
					PlayerID:        "test-player-id",
					PlayerSessionID: "test-player-session-id",
				},
			},
		},
	}

	if err := gm.HandleRequest(req, &resp, timeDuration); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(respData, resp) {
		t.Errorf("\nexpect  %v \nbut get %v", respData, resp)
	}

	websocketClientMock.
		EXPECT().
		SendRequest(req, gomock.Any()).
		Do(func(req internal.MessageGetter, resp chan<- common.Outcome) error {
			time.Sleep(time.Millisecond * 5)
			return nil
		})

	logger.
		EXPECT().
		Errorf("Response not received within time limit for request: %s", "test-request-id").
		Do(func(format string, args ...any) { t.Logf(format, args...) })

	websocketClientMock.
		EXPECT().
		CancelRequest(req.RequestID)

	err = gm.HandleRequest(req, &resp, timeDuration)
	if err == nil {
		t.Fatal(err)
	}

	websocketClientMock.
		EXPECT().
		Close()

	if err := gm.Disconnect(); err != nil {
		t.Fatal(err)
	}
}

func TestGameliftManagerHandleRequestError(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctrl := gomock.NewController(t)

	gameliftMessageHandlerMock := mock.NewMockIGameLiftMessageHandler(ctrl)
	websocketClientMock := mock.NewMockIWebSocketClient(ctrl)
	logger := mock.NewTestLogger(t, ctrl)

	gm := internal.GetGameLiftManager(gameliftMessageHandlerMock, websocketClientMock, logger)

	req := &request.DescribePlayerSessionsRequest{
		Message: message.Message{
			Action:    message.DescribePlayerSessions,
			RequestID: "test-request-id",
		},
		PlayerID:        "test-player-id",
		PlayerSessionID: "test-player-session-id",
		NextToken:       "test-next-token",
		Limit:           1,
	}

	expectedError := errors.New("test error")

	websocketClientMock.
		EXPECT().
		SendRequest(req, gomock.Any()).
		DoAndReturn(func(_ internal.MessageGetter, result chan<- common.Outcome) error {
			result <- common.Outcome{Error: expectedError}

			return nil
		})

	err := gm.HandleRequest(req, nil, time.Second)
	if !errors.Is(err, expectedError) {
		t.Fatalf("unexpected error %s, want %s", err, expectedError)
	}
}
