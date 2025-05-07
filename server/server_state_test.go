/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package server

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"go.uber.org/goleak"

	"aws/amazon-gamelift-go-sdk/common"
	"aws/amazon-gamelift-go-sdk/model"
	"aws/amazon-gamelift-go-sdk/model/message"
	"aws/amazon-gamelift-go-sdk/model/request"
	"aws/amazon-gamelift-go-sdk/server/internal/mock"
)

func TestGameLiftServerStateLifecycle(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctrl := gomock.NewController(t)
	manager := mock.NewMockIGameLiftManager(ctrl)

	params := ServerParameters{
		WebSocketURL: "wss://test.url",
		ProcessID:    "test-process-id",
		HostID:       "test-host-id",
		FleetID:      "test-fleet-id",
		AuthToken:    "test-auth-token",
	}

	var (
		healthCheckCalled      common.AtomicBool
		processTerminateCalled common.AtomicBool
		startGameSessionCalled common.AtomicBool
	)

	processParams := &ProcessParameters{
		OnHealthCheck: func() bool {
			healthCheckCalled.Store(true)
			return true
		},
		OnProcessTerminate: func() {
			processTerminateCalled.Store(true)
		},
		OnStartGameSession: func(session model.GameSession) {
			startGameSessionCalled.Store(true)
		},
		Port: 8080,
		LogParameters: LogParameters{
			LogPaths: []string{"/local", "game", "logs"},
		},
	}

	manager.
		EXPECT().
		Connect(params.WebSocketURL, params.ProcessID, params.HostID, params.FleetID, params.AuthToken).
		Times(1)

	manager.
		EXPECT().
		SendMessage(ignoreRequestID(request.ActivateServerProcessRequest{
			Message: message.Message{
				RequestID: "cbb9ba51-1351-415a-9c52-380347d099f7",
				Action:    message.ActivateServerProcess,
			},
			SdkVersion:  common.SdkVersion,
			SdkLanguage: common.SdkLanguage,
			Port:        processParams.Port,
			LogPaths:    processParams.LogParameters.LogPaths,
		})).
		Times(1)

	manager.
		EXPECT().
		HandleRequest(ignoreRequestID(request.NewHeartbeatServerProcess(true)), gomock.Any(), 20*time.Second).
		MinTimes(1)

	const (
		newWebSocketURL = "wss://new-test.url"
		newAuthToken    = "new-test-auth-token"
	)

	manager.
		EXPECT().
		Connect(newWebSocketURL, params.ProcessID, params.HostID, params.FleetID, newAuthToken).
		Times(1)

	manager.
		EXPECT().
		SendMessage(ignoreRequestID(request.NewTerminateServerProcess())).
		Times(1)

	manager.
		EXPECT().
		Disconnect().
		Times(1)

	var state gameLiftServerState
	err := state.init(&params, manager)
	if err != nil {
		t.Fatal(err)
	}

	err = state.processReady(processParams)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(20 * time.Microsecond)
	state.OnRefreshConnection(newWebSocketURL, newAuthToken)
	gameSession := model.GameSession{
		GameSessionID: "game_session_id_test",
	}
	state.OnStartGameSession(&gameSession)

	t.Logf("Tests are running, please wait")
	time.Sleep(state.healthCheckInterval)

	err = state.processEnding()
	if err != nil {
		t.Fatal(err)
	}

	if !healthCheckCalled.Load() {
		t.Errorf("missing call of health check callback")
	}
	if !startGameSessionCalled.Load() {
		t.Errorf("missing call OnStartGameSession handler")
	}
	if state.fleetID != gameSession.FleetID {
		t.Errorf("FleetID should be equal after OnStartGameSession call")
	}
	gameSessionID, _ := state.getGameSessionID()
	if gameSessionID != gameSession.GameSessionID {
		t.Errorf("GameSessionID should be equal after OnStartGameSession call")
	}

	nowMilliseconds, nowSeconds := time.Now().UnixMilli(), time.Now().Unix()
	state.OnTerminateProcess(nowMilliseconds)
	if !processTerminateCalled.Load() {
		t.Errorf("missing call OnTerminateProcess handler")
	}
	terminated, _ := state.getTerminationTime()
	if terminated != nowSeconds {
		t.Errorf("incorrect termination time expect: %d but get: %d", nowSeconds, terminated)
	}
	state.destroy()
}

func ignoreRequestID(expect any) gomock.Matcher {
	return &ignoreRequestIDEqual{expect: expect}
}

type ignoreRequestIDEqual struct {
	expect any
}

func (i *ignoreRequestIDEqual) Matches(x any) bool {
	return toStr(x) == toStr(i.expect)
}

func toStr(x any) string {
	return requestIDMatcher.ReplaceAllString(fmt.Sprintf("%#v", x), `RequestID:"any"`)
}

var requestIDMatcher = regexp.MustCompile(`RequestID:"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"`)

func (i *ignoreRequestIDEqual) String() string {
	return fmt.Sprintf("%v", i.expect)
}
