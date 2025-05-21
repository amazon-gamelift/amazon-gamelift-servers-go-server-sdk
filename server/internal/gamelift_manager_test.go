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
	"regexp"
	"testing"
	"time"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model/message"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model/request"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model/response"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model/result"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/server/internal"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/server/internal/mock"

	"github.com/golang/mock/gomock"
	"go.uber.org/goleak"
)

const (
	websocketURL         = "https://example.test"
	timeDuration         = 100 * time.Millisecond
	processID            = "processId"
	hostID               = "hostId"
	fleetID              = "fleetId"
	authToken            = "authToken"
	testIdempotencyToken = "00000000-1111-2222-3333-444444444444"
)

func TestGameliftManagerHandleRequest_AuthTokenPassed(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctrl := gomock.NewController(t)

	gameliftMessageHandlerMock := mock.NewMockIGameLiftMessageHandler(ctrl)
	websocketClientMock := mock.NewMockIWebSocketClient(ctrl)
	logger := mock.NewTestLogger(t, ctrl)
	httpClientMock := mock.NewMockHttpClient(ctrl)

	gm := internal.GetGameLiftManager(gameliftMessageHandlerMock, websocketClientMock, logger, httpClientMock)

	connectURL, err := url.Parse(websocketURL)
	if err != nil {
		t.Fatalf("parse url: %s", err)
	}

	params := url.Values{}
	params.Add(common.AuthTokenKey, authToken)
	params.Add(common.ComputeIDKey, hostID)
	params.Add(common.FleetIDKey, fleetID)
	params.Add(common.PidKey, processID)
	params.Add(common.SdkLanguageKey, common.SdkLanguage)
	params.Add(common.SdkVersionKey, common.SdkVersion)
	params.Add(common.IdempotencyTokenKey, testIdempotencyToken)
	connectURL.RawQuery = params.Encode()

	websocketClientMock.
		EXPECT().
		Connect(ignoreIdempotencyToken(connectURL))

	for _, actions := range []message.MessageAction{message.CreateGameSession, message.UpdateGameSession, message.RefreshConnection, message.TerminateProcess} {
		websocketClientMock.
			EXPECT().
			AddHandler(actions, gomock.Not(gomock.Nil()))
	}

	if err := gm.Connect(websocketURL, processID, hostID, fleetID, authToken, nil); err != nil {
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

func TestGameliftManagerHandleRequest_SigV4QueryParametersPassed(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctrl := gomock.NewController(t)

	gameliftMessageHandlerMock := mock.NewMockIGameLiftMessageHandler(ctrl)
	websocketClientMock := mock.NewMockIWebSocketClient(ctrl)
	logger := mock.NewTestLogger(t, ctrl)
	httpClientMock := mock.NewMockHttpClient(ctrl)

	gm := internal.GetGameLiftManager(gameliftMessageHandlerMock, websocketClientMock, logger, httpClientMock)

	connectURL, err := url.Parse(websocketURL)
	if err != nil {
		t.Fatalf("parse url: %s", err)
	}

	sigV4QueryParameters := map[string]string{
		"Authorization":        "SigV4",
		"X-Amz-Algorithm":      "AWS4-HMAC-SHA256",
		"X-Amz-Credential":     "testAccessKey/20240805/us-east-1/gamelift/aws4_request",
		"X-Amz-Date":           "20240805T100000Z",
		"X-Amz-Signature":      "2601fe291f4b43a63f6ffb0e1d9085a1edbaa2a866c96511e153af3408bfe771",
		"X-Amz-Security-Token": "testSessionToken",
	}

	params := url.Values{}
	params.Add(common.ComputeIDKey, hostID)
	params.Add(common.FleetIDKey, fleetID)
	params.Add(common.PidKey, processID)
	params.Add(common.SdkLanguageKey, common.SdkLanguage)
	params.Add(common.SdkVersionKey, common.SdkVersion)
	params.Add(common.IdempotencyTokenKey, testIdempotencyToken)

	for key, value := range sigV4QueryParameters {
		params.Add(key, value)
	}
	connectURL.RawQuery = params.Encode()

	websocketClientMock.
		EXPECT().
		Connect(ignoreIdempotencyToken(connectURL))

	for _, actions := range []message.MessageAction{message.CreateGameSession, message.UpdateGameSession, message.RefreshConnection, message.TerminateProcess} {
		websocketClientMock.
			EXPECT().
			AddHandler(actions, gomock.Not(gomock.Nil()))
	}

	if err := gm.Connect(websocketURL, processID, hostID, fleetID, "", sigV4QueryParameters); err != nil {
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

func TestGameliftManagerHandleRequest_AuthTokenAndSigV4QueryParametersPassed(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctrl := gomock.NewController(t)

	gameliftMessageHandlerMock := mock.NewMockIGameLiftMessageHandler(ctrl)
	websocketClientMock := mock.NewMockIWebSocketClient(ctrl)
	logger := mock.NewTestLogger(t, ctrl)
	httpClientMock := mock.NewMockHttpClient(ctrl)

	gm := internal.GetGameLiftManager(gameliftMessageHandlerMock, websocketClientMock, logger, httpClientMock)

	connectURL, err := url.Parse(websocketURL)
	if err != nil {
		t.Fatalf("parse url: %s", err)
	}

	sigV4QueryParameters := map[string]string{
		"Authorization":        "SigV4",
		"X-Amz-Algorithm":      "AWS4-HMAC-SHA256",
		"X-Amz-Credential":     "testAccessKey/20240805/us-east-1/gamelift/aws4_request",
		"X-Amz-Date":           "20240805T100000Z",
		"X-Amz-Signature":      "2601fe291f4b43a63f6ffb0e1d9085a1edbaa2a866c96511e153af3408bfe771",
		"X-Amz-Security-Token": "testSessionToken",
	}

	params := url.Values{}
	params.Add(common.AuthTokenKey, authToken)
	params.Add(common.ComputeIDKey, hostID)
	params.Add(common.FleetIDKey, fleetID)
	params.Add(common.PidKey, processID)
	params.Add(common.SdkLanguageKey, common.SdkLanguage)
	params.Add(common.SdkVersionKey, common.SdkVersion)
	params.Add(common.IdempotencyTokenKey, testIdempotencyToken)

	connectURL.RawQuery = params.Encode()

	websocketClientMock.
		EXPECT().
		Connect(ignoreIdempotencyToken(connectURL))

	for _, actions := range []message.MessageAction{message.CreateGameSession, message.UpdateGameSession, message.RefreshConnection, message.TerminateProcess} {
		websocketClientMock.
			EXPECT().
			AddHandler(actions, gomock.Not(gomock.Nil()))
	}

	if err := gm.Connect(websocketURL, processID, hostID, fleetID, authToken, sigV4QueryParameters); err != nil {
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

/*
Implementation of a gomock Matcher for a URL type that ignores the exact uuid
of the IdempotencyToken field of the url query.
Matcher interface requires a Matches() and String() function
*/
func ignoreIdempotencyToken(expect *url.URL) gomock.Matcher {
	return &ignoreIdempotencyTokenEqual{expect: expect}
}

type ignoreIdempotencyTokenEqual struct {
	expect *url.URL
}

func (i *ignoreIdempotencyTokenEqual) Matches(u interface{}) bool {
	return toStr(u) == toStr(i.expect)
}

func toStr(u interface{}) string {
	return idempotencyTokenMatcher.ReplaceAllString(fmt.Sprintf("%#v", u), `IdempotencyToken=any`)
}

var idempotencyTokenMatcher = regexp.MustCompile(`IdempotencyToken=[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

func (i *ignoreIdempotencyTokenEqual) String() string {
	return fmt.Sprintf("%v", i.expect)
}

func TestGameliftManagerHandleRequestError(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctrl := gomock.NewController(t)

	gameliftMessageHandlerMock := mock.NewMockIGameLiftMessageHandler(ctrl)
	websocketClientMock := mock.NewMockIWebSocketClient(ctrl)
	logger := mock.NewTestLogger(t, ctrl)
	httpClientMock := mock.NewMockHttpClient(ctrl)

	gm := internal.GetGameLiftManager(gameliftMessageHandlerMock, websocketClientMock, logger, httpClientMock)

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

// GIVEN delayed response from sever WHEN HandleRequest is called with timeout THEN return time out error
func TestGameliftManagerHandleRequest_Timeout_ReturnError(t *testing.T) {
	// Set up the test case
	defer goleak.VerifyNone(t)
	ctrl := gomock.NewController(t)

	gameliftMessageHandlerMock := mock.NewMockIGameLiftMessageHandler(ctrl)
	websocketClientMock := mock.NewMockIWebSocketClient(ctrl)
	logger := mock.NewTestLogger(t, ctrl)
	httpClientMock := mock.NewMockHttpClient(ctrl)

	gm := internal.GetGameLiftManager(gameliftMessageHandlerMock, websocketClientMock, logger, httpClientMock)

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

	// GIVEN
	expectedError := common.NewGameLiftError(common.ServiceCallFailed, "", "")
	const DesiredRequestTimeout = time.Duration(1) * time.Millisecond
	const MockDelayInResponse = time.Duration(5) * time.Millisecond

	websocketClientMock.
		EXPECT().
		SendRequest(req, gomock.Any()).
		Do(func(req internal.MessageGetter, resp chan<- common.Outcome) error {
			time.Sleep(MockDelayInResponse)
			return nil
		})

	websocketClientMock.
		EXPECT().
		CancelRequest(req.RequestID)

	logger.
		EXPECT().
		Errorf("Response not received within time limit for request: %s", "test-request-id").
		Do(func(format string, args ...any) { t.Logf(format, args...) })

	// WHEN
	err := gm.HandleRequest(req, nil, DesiredRequestTimeout)

	// THEN
	if err.Error() != expectedError.Error() {
		t.Fatalf("unexpected error %s, want %s", err, expectedError)
	}
}
