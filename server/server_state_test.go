/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
package server

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"go.uber.org/goleak"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model/message"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model/request"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model/result"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/server/internal/mock"
)

const TestHealthCheckInterval = "200ms"
const TestHealthCheckTimeout = "50ms"
const TestRequestId = "00000000-1111-2222-3333-444444444444"
const ActivateSeverProcessRequestTimeoutInSeconds = time.Duration(6) * time.Second

// Speeds up the health check code to make our tests run at a reasonable speed. Default would be 60 seconds per test
func setHealthCheckEnvironmentVariables(t *testing.T) {
	envErr := os.Setenv(common.HealthcheckInterval, TestHealthCheckInterval)
	if envErr != nil {
		t.Fatalf("Failed to set HealthcheckInterval environment variable: %s", envErr)
	}
	envErr = os.Setenv(common.HealthcheckTimeout, TestHealthCheckTimeout)
	if envErr != nil {
		t.Fatalf("Failed to set HealthcheckTimeout environment variable: %s", envErr)
	}
}

func TestInit(t *testing.T) {
	// GIVEN
	defer goleak.VerifyNone(t)

	manager := setupNewMockIGameLiftManager(t)

	params := ServerParameters{
		WebSocketURL: "wss://test.url",
		ProcessID:    "test-process-id",
		HostID:       "test-host-id",
		FleetID:      "test-fleet-id",
		AuthToken:    "test-auth-token",
	}

	manager.
		EXPECT().
		Connect(params.WebSocketURL, params.ProcessID, params.HostID, params.FleetID, params.AuthToken, nil).
		Times(1)

	manager.
		EXPECT().
		Disconnect().
		Times(1)

	var state gameLiftServerState

	// WHEN
	err := state.init(params, manager)

	// THEN
	if err != nil {
		t.Fatal(err)
	}

	if state.fleetRoleResultCache == nil {
		t.Fatalf("fleetRoleResultCache is uninitialized")
	}

	if state.fleetID != params.FleetID {
		t.Fatalf("fleetID is %s, expected %s", state.fleetID, params.FleetID)
	}

	if state.hostID != params.HostID {
		t.Fatalf("hostID is %s, expected %s", state.fleetID, params.FleetID)
	}

	if state.processID != params.ProcessID {
		t.Fatalf("processID is %s, expected %s", state.fleetID, params.FleetID)
	}

	state.destroy()
}

func TestGameLiftServerStateLifecycle_AuthTokenPassed(t *testing.T) {
	defer goleak.VerifyNone(t)

	manager := setupNewMockIGameLiftManager(t)

	setHealthCheckEnvironmentVariables(t)

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
		Connect(params.WebSocketURL, params.ProcessID, params.HostID, params.FleetID, params.AuthToken, nil).
		Times(1)

	manager.
		EXPECT().
		HandleRequest(ignoreRequestID(request.ActivateServerProcessRequest{
			Message: message.Message{
				RequestID: "cbb9ba51-1351-415a-9c52-380347d099f7",
				Action:    message.ActivateServerProcess,
			},
			SdkVersion:  common.SdkVersion,
			SdkLanguage: common.SdkLanguage,
			Port:        processParams.Port,
			LogPaths:    processParams.LogParameters.LogPaths,
		}), gomock.Any(), ActivateSeverProcessRequestTimeoutInSeconds).
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
		Connect(newWebSocketURL, params.ProcessID, params.HostID, params.FleetID, newAuthToken, nil).
		Times(1)

	manager.
		EXPECT().
		HandleRequest(ignoreRequestID(request.NewTerminateServerProcess()), nil, 20*time.Second).
		Times(1)

	manager.
		EXPECT().
		Disconnect().
		Times(1)

	var state gameLiftServerState
	err := state.init(params, manager)
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
		t.Fatalf("missing call of health check callback")
	}
	if !startGameSessionCalled.Load() {
		t.Fatalf("missing call OnStartGameSession handler")
	}
	if state.fleetID != gameSession.FleetID {
		t.Fatalf("FleetID should be equal after OnStartGameSession call")
	}
	gameSessionID, _ := state.getGameSessionID()
	if gameSessionID != gameSession.GameSessionID {
		t.Fatalf("GameSessionID should be equal after OnStartGameSession call")
	}

	nowMilliseconds, nowSeconds := time.Now().UnixMilli(), time.Now().Unix()
	state.OnTerminateProcess(nowMilliseconds)
	if !processTerminateCalled.Load() {
		t.Fatalf("missing call OnTerminateProcess handler")
	}
	terminated, _ := state.getTerminationTime()
	if terminated != nowSeconds {
		t.Fatalf("incorrect termination time expect: %d but get: %d", nowSeconds, terminated)
	}
	state.destroy()
}

// GIVEN websocket returns error response for ActivateServerProcess call WHEN ProcessReady() is called THEN ProcessReady() should return error
func TestGameLiftServerStateProcessReady_WithActivateServerProcessErrorResponse_ReturnError(t *testing.T) {
	// Set up the test case
	defer goleak.VerifyNone(t)
	manager := setupNewMockIGameLiftManager(t)

	params := ServerParameters{
		WebSocketURL: "wss://test.url",
		ProcessID:    "test-process-id",
		HostID:       "test-host-id",
		FleetID:      "test-fleet-id",
		AuthToken:    "test-auth-token",
	}

	processParams := &ProcessParameters{
		OnHealthCheck: func() bool {
			return false
		},
		OnProcessTerminate: func() {},
		OnStartGameSession: func(session model.GameSession) {},
		Port:               8080,
	}

	// GIVEN
	activateServerProcessWithErrorResponse := errors.New("test error")
	expectedError := common.NewGameLiftError(common.ProcessNotReady, "", activateServerProcessWithErrorResponse.Error())

	manager.
		EXPECT().
		Connect(params.WebSocketURL, params.ProcessID, params.HostID, params.FleetID, params.AuthToken, nil).
		Times(1)

	// mocking to return an error in response when ActivateServerProcess() request is sent via websocket
	manager.
		EXPECT().
		HandleRequest(ignoreRequestID(request.ActivateServerProcessRequest{
			Message: message.Message{
				RequestID: "cbb9ba51-1351-415a-9c52-380347d099f7",
				Action:    message.ActivateServerProcess,
			},
			SdkVersion:  common.SdkVersion,
			SdkLanguage: common.SdkLanguage,
			Port:        processParams.Port,
			LogPaths:    processParams.LogParameters.LogPaths,
		}), gomock.Any(), ActivateSeverProcessRequestTimeoutInSeconds).
		Times(1).
		Return(activateServerProcessWithErrorResponse)

	manager.
		EXPECT().
		Disconnect().
		Times(1)

	var state gameLiftServerState
	err := state.init(params, manager)
	if err != nil {
		t.Fatal(err)
	}

	// WHEN
	err = state.processReady(processParams)

	// THEN
	// err should NOT be nil as ProcessReady() should fail
	if err == nil {
		t.Fatal("ProcessReady() did not return an error when ActivateServerProcess() request failed.")
	}

	// check to receive the expected error from ProcessReady()
	if err.Error() != expectedError.Error() {
		t.Fatalf("Unexpected error, got %s, want %s", err, expectedError)
	}

	state.destroy()
}

// GIVEN no handler provided for onProcessTerminate WHEN OnTerminateProcess called THEN exit with success
func TestGameLiftServerStateOnTerminateProcess_WithNilOnProcessTerminateHandler_ExitSuccessfully(t *testing.T) {

	defer goleak.VerifyNone(t)

	manager := setupNewMockIGameLiftManager(t)

	processParams := &ProcessParameters{
		OnHealthCheck:      func() bool { return true },
		OnProcessTerminate: nil,
		OnStartGameSession: nil,
		Port:               8080,
		LogParameters: LogParameters{
			LogPaths: []string{"/local", "game", "logs"},
		},
	}

	params := ServerParameters{
		WebSocketURL: "wss://test.url",
		ProcessID:    "test-process-id",
		HostID:       "test-host-id",
		FleetID:      "test-fleet-id",
		AuthToken:    "test-auth-token",
	}

	var exitCalled common.AtomicBool
	var exitFailureCalled common.AtomicBool

	exitFunc = func(code int) {
		if code != 0 {
			exitFailureCalled.Store(true)
			exitCalled.Store(false)
		} else {
			exitCalled.Store(true)
			exitFailureCalled.Store(false)
		}
	}

	manager.
		EXPECT().
		Connect(params.WebSocketURL, params.ProcessID, params.HostID, params.FleetID, params.AuthToken, nil).
		Times(1)

	manager.
		EXPECT().
		HandleRequest(ignoreRequestID(request.NewTerminateServerProcess()), nil, 20*time.Second).
		Times(1)

	manager.
		EXPECT().
		Disconnect().
		Times(1)

	var state gameLiftServerState
	state.parameters = processParams
	state.wsGameLift = manager
	state.isReadyProcess.Store(true)
	err := state.init(params, manager)
	if err != nil {
		t.Fatal(err)
	}

	nowMilliseconds := time.Now().UnixMilli()
	state.OnTerminateProcess(nowMilliseconds)
	if !exitCalled.Load() {
		t.Fatalf("missing call of exit function")
	}
	if exitFailureCalled.Load() {
		t.Fatalf("unexpected exit failure")
	}
}

// GIVEN no handler provided for onProcessTerminate WHEN OnTerminateProcess AND ProcessEnding failed THEN ProcessEnded() and Destroy() are called
func TestGameLiftServerStateOnTerminateProcess_WithNilOnProcessTerminateHandler_AndProcessEndingFailed_ExitWithFailure(t *testing.T) {

	defer goleak.VerifyNone(t)

	manager := setupNewMockIGameLiftManager(t)

	processParams := &ProcessParameters{
		OnHealthCheck:      func() bool { return true },
		OnProcessTerminate: nil,
		OnStartGameSession: nil,
		Port:               8080,
		LogParameters: LogParameters{
			LogPaths: []string{"/local", "game", "logs"},
		},
	}

	params := ServerParameters{
		WebSocketURL: "wss://test.url",
		ProcessID:    "test-process-id",
		HostID:       "test-host-id",
		FleetID:      "test-fleet-id",
		AuthToken:    "test-auth-token",
	}

	var exitCalled common.AtomicBool
	var exitFailureCalled common.AtomicBool

	exitFunc = func(code int) {
		if code != 0 {
			exitFailureCalled.Store(true)
			exitCalled.Store(false)
		} else {
			exitCalled.Store(true)
			exitFailureCalled.Store(false)
		}
	}

	// GIVEN
	processEndingWithErrorResponse := errors.New("test error")
	expectedError := common.NewGameLiftError(common.ProcessEndingFailed, "", processEndingWithErrorResponse.Error())

	manager.
		EXPECT().
		Connect(params.WebSocketURL, params.ProcessID, params.HostID, params.FleetID, params.AuthToken, nil).
		Times(1)

	// mocking to return an error in response when TerminateServerProcess() request is sent via websocket
	manager.
		EXPECT().
		HandleRequest(ignoreRequestID(request.NewTerminateServerProcess()), nil, 20*time.Second).
		Times(1).
		Return(expectedError)

	manager.
		EXPECT().
		Disconnect().
		Times(1)

	var state gameLiftServerState
	state.parameters = processParams
	state.wsGameLift = manager
	state.isReadyProcess.Store(true)
	err := state.init(params, manager)
	if err != nil {
		t.Fatal(err)
	}

	nowMilliseconds := time.Now().UnixMilli()
	state.OnTerminateProcess(nowMilliseconds)
	if exitCalled.Load() {
		t.Fatalf("unexpected successful exit")
	}
	if !exitFailureCalled.Load() {
		t.Fatalf("expected exit failure")
	}
}

// GIVEN no handler provided for onProcessTerminate WHEN OnTerminateProcess AND Destroy failed THEN ProcessEnded() and Destroy() are called
func TestGameLiftServerStateOnTerminateProcess_WithNilOnProcessTerminateHandler_AndDestroyFailed_ExitWithFailure(t *testing.T) {

	defer goleak.VerifyNone(t)

	manager := setupNewMockIGameLiftManager(t)

	processParams := &ProcessParameters{
		OnHealthCheck:      func() bool { return true },
		OnProcessTerminate: nil,
		OnStartGameSession: nil,
		Port:               8080,
		LogParameters: LogParameters{
			LogPaths: []string{"/local", "game", "logs"},
		},
	}

	params := ServerParameters{
		WebSocketURL: "wss://test.url",
		ProcessID:    "test-process-id",
		HostID:       "test-host-id",
		FleetID:      "test-fleet-id",
		AuthToken:    "test-auth-token",
	}

	var exitCalled common.AtomicBool
	var exitFailureCalled common.AtomicBool

	exitFunc = func(code int) {
		if code != 0 {
			exitFailureCalled.Store(true)
			exitCalled.Store(false)
		} else {
			exitCalled.Store(true)
			exitFailureCalled.Store(false)
		}
	}

	// GIVEN
	destroyWithErrorResponse := errors.New("test error")
	expectedError := common.NewGameLiftError(common.WebsocketClosingError, "", destroyWithErrorResponse.Error())

	manager.
		EXPECT().
		Connect(params.WebSocketURL, params.ProcessID, params.HostID, params.FleetID, params.AuthToken, nil).
		Times(1)

	// mocking to return an error in response when TerminateServerProcess() request is sent via websocket
	manager.
		EXPECT().
		HandleRequest(ignoreRequestID(request.NewTerminateServerProcess()), nil, 20*time.Second).
		Times(1)

	manager.
		EXPECT().
		Disconnect().
		Times(1).
		Return(expectedError)

	var state gameLiftServerState
	state.parameters = processParams
	state.wsGameLift = manager
	state.isReadyProcess.Store(true)

	err := state.init(params, manager)
	if err != nil {
		t.Fatal(err)
	}

	nowMilliseconds := time.Now().UnixMilli()
	state.OnTerminateProcess(nowMilliseconds)
	if exitCalled.Load() {
		t.Fatalf("unexpected successful exit")
	}
	if !exitFailureCalled.Load() {
		t.Fatalf("expected exit failure")
	}
}

func TestGameLiftServerStateLifecycle_AwsCredentialsAndRegionPassed(t *testing.T) {
	defer goleak.VerifyNone(t)

	manager := setupNewMockIGameLiftManager(t)

	setHealthCheckEnvironmentVariables(t)

	params := ServerParameters{
		WebSocketURL: "wss://test.url",
		ProcessID:    "test-process-id",
		HostID:       "test-host-id",
		FleetID:      "test-fleet-id",
		AwsRegion:    "us-west-2",
		AccessKey:    "test_access_key",
		SecretKey:    "test_secret_key",
		SessionToken: "test_session_token"}

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
		Connect(params.WebSocketURL, params.ProcessID, params.HostID, params.FleetID, "", gomock.Any()).
		Times(1)

	manager.
		EXPECT().
		HandleRequest(ignoreRequestID(request.ActivateServerProcessRequest{
			Message: message.Message{
				RequestID: "cbb9ba51-1351-415a-9c52-380347d099f7",
				Action:    message.ActivateServerProcess,
			},
			SdkVersion:  common.SdkVersion,
			SdkLanguage: common.SdkLanguage,
			Port:        processParams.Port,
			LogPaths:    processParams.LogParameters.LogPaths,
		}), gomock.Any(), ActivateSeverProcessRequestTimeoutInSeconds).
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
		Connect(newWebSocketURL, params.ProcessID, params.HostID, params.FleetID, newAuthToken, nil).
		Times(1)

	manager.
		EXPECT().
		HandleRequest(ignoreRequestID(request.NewTerminateServerProcess()), nil, 20*time.Second).
		Times(1)

	manager.
		EXPECT().
		Disconnect().
		Times(1)

	var state gameLiftServerState
	err := state.init(params, manager)
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
		t.Fatalf("missing call of health check callback")
	}
	if !startGameSessionCalled.Load() {
		t.Fatalf("missing call OnStartGameSession handler")
	}
	if state.fleetID != gameSession.FleetID {
		t.Fatalf("FleetID should be equal after OnStartGameSession call")
	}
	gameSessionID, _ := state.getGameSessionID()
	if gameSessionID != gameSession.GameSessionID {
		t.Fatalf("GameSessionID should be equal after OnStartGameSession call")
	}

	nowMilliseconds, nowSeconds := time.Now().UnixMilli(), time.Now().Unix()
	state.OnTerminateProcess(nowMilliseconds)
	if !processTerminateCalled.Load() {
		t.Fatalf("missing call OnTerminateProcess handler")
	}
	terminated, _ := state.getTerminationTime()
	if terminated != nowSeconds {
		t.Fatalf("incorrect termination time expect: %d but get: %d", nowSeconds, terminated)
	}
	state.destroy()
}

func TestGameLiftServerStateLifecycle_AuthTokenAndAwsCredentialsAndRegionAndToolPassed(t *testing.T) {
	defer goleak.VerifyNone(t)

	sdkToolName := "test-sdk-tool"
	err := os.Setenv(common.EnvironmentKeySDKToolName, sdkToolName)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.Unsetenv(common.EnvironmentKeySDKToolName)
		if err != nil {
			t.Fatal(err)
		}
	}()

	sdkToolVersion := "1.2.3"
	err = os.Setenv(common.EnvironmentKeySDKToolVersion, sdkToolVersion)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.Unsetenv(common.EnvironmentKeySDKToolVersion)
		if err != nil {
			t.Fatal(err)
		}
	}()

	manager := setupNewMockIGameLiftManager(t)

	setHealthCheckEnvironmentVariables(t)

	params := ServerParameters{
		WebSocketURL: "wss://test.url",
		ProcessID:    "test-process-id",
		HostID:       "test-host-id",
		FleetID:      "test-fleet-id",
		AuthToken:    "test-auth-token",
		AwsRegion:    "us-west-2",
		AccessKey:    "test_access_key",
		SecretKey:    "test_secret_key",
		SessionToken: "test_session_token"}

	var (
		healthCheckCalled       common.AtomicBool
		processTerminateCalled  common.AtomicBool
		startGameSessionCalled  common.AtomicBool
		updateGameSessionCalled common.AtomicBool
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
		OnUpdateGameSession: func(session model.UpdateGameSession) {
			updateGameSessionCalled.Store(true)
		},
		Port: 8080,
		LogParameters: LogParameters{
			LogPaths: []string{"/local", "game", "logs"},
		},
	}

	manager.
		EXPECT().
		Connect(params.WebSocketURL, params.ProcessID, params.HostID, params.FleetID, params.AuthToken, nil).
		Times(1)

	manager.
		EXPECT().
		HandleRequest(ignoreRequestID(request.ActivateServerProcessRequest{
			Message: message.Message{
				RequestID: "cbb9ba51-1351-415a-9c52-380347d099f7",
				Action:    message.ActivateServerProcess,
			},
			SdkVersion:     common.SdkVersion,
			SdkLanguage:    common.SdkLanguage,
			SdkToolName:    sdkToolName,
			SdkToolVersion: sdkToolVersion,
			Port:           processParams.Port,
			LogPaths:       processParams.LogParameters.LogPaths,
		}), gomock.Any(), ActivateSeverProcessRequestTimeoutInSeconds).
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
		Connect(newWebSocketURL, params.ProcessID, params.HostID, params.FleetID, newAuthToken, nil).
		Times(1)

	manager.
		EXPECT().
		HandleRequest(ignoreRequestID(request.NewTerminateServerProcess()), nil, 20*time.Second).
		Times(1)

	manager.
		EXPECT().
		Disconnect().
		Times(1)

	var state gameLiftServerState
	err = state.init(params, manager)
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

	state.OnUpdateGameSession(&gameSession, nil, "backfillTicketId")

	err = state.processEnding()
	if err != nil {
		t.Fatal(err)
	}

	if !healthCheckCalled.Load() {
		t.Fatalf("missing call of health check callback")
	}
	if !startGameSessionCalled.Load() {
		t.Fatalf("missing call OnStartGameSession handler")
	}
	if !updateGameSessionCalled.Load() {
		t.Fatalf("missing call OnUpdateGameSession handler")
	}
	if state.fleetID != gameSession.FleetID {
		t.Fatalf("FleetID should be equal after OnStartGameSession call")
	}
	gameSessionID, _ := state.getGameSessionID()
	if gameSessionID != gameSession.GameSessionID {
		t.Fatalf("GameSessionID should be equal after OnStartGameSession call")
	}

	nowMilliseconds, nowSeconds := time.Now().UnixMilli(), time.Now().Unix()
	state.OnTerminateProcess(nowMilliseconds)
	if !processTerminateCalled.Load() {
		t.Fatalf("missing call OnTerminateProcess handler")
	}
	terminated, _ := state.getTerminationTime()
	if terminated != nowSeconds {
		t.Fatalf("incorrect termination time expect: %d but get: %d", nowSeconds, terminated)
	}
	state.destroy()
}

func TestFleetRoleCredentialsCache(t *testing.T) {
	defer goleak.VerifyNone(t)

	manager := setupNewMockIGameLiftManager(t)

	params := ServerParameters{
		WebSocketURL: "wss://test.url",
		ProcessID:    "test-process-id",
		HostID:       "test-host-id",
		FleetID:      "test-fleet-id",
		AuthToken:    "test-auth-token",
	}

	manager.
		EXPECT().
		Connect(params.WebSocketURL, params.ProcessID, params.HostID, params.FleetID, params.AuthToken, nil).
		Times(1)

	manager.
		EXPECT().
		Disconnect().
		Times(1)

	roleArn := "TEST_ROLE_ARN"

	var state gameLiftServerState
	err := state.init(params, manager)
	if err != nil {
		t.Fatal(err)
	}

	// When there's nothing in the cache, return nothing
	credentials, returnedPrevious := state.getRoleCredentialsFromCache(roleArn)
	if returnedPrevious {
		t.Error("First get call on cache unexpectedly returned a value", credentials, returnedPrevious)
	}

	// When the cache has credentials that aren't yet close to expiration, return the credentials
	state.fleetRoleResultCache[roleArn] = result.GetFleetRoleCredentialsResult{
		Expiration: time.Now().Add(60 * time.Minute).UnixMilli(), // Expiration time is in milliseconds
	}
	credentials, returnedPrevious = state.getRoleCredentialsFromCache(roleArn)
	if !returnedPrevious {
		t.Error("Second get call failed to return the credentials even though they should be fresh", state.fleetRoleResultCache[roleArn], returnedPrevious)
	}

	// When the cache has credentials that are old, return nothing so system can refresh them
	state.fleetRoleResultCache[roleArn] = result.GetFleetRoleCredentialsResult{
		Expiration: time.Now().Add(5 * time.Minute).UnixMilli(), // Expiration time is in milliseconds
	}
	credentials, returnedPrevious = state.getRoleCredentialsFromCache(roleArn)
	if returnedPrevious {
		t.Error("Third get call incorrectly returned the credentials when they're close to expiring", credentials, returnedPrevious)
	}

	// The rest of the life cycle is already unit tested

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

func setupNewMockIGameLiftManager(t *testing.T) *mock.MockIGameLiftManager {
	ctrl := gomock.NewController(t)
	manager := mock.NewMockIGameLiftManager(ctrl)

	// Create mock logger and ignore Error message expectations
	logger := mock.NewTestLogger(t, ctrl,
		mock.WithExpectAnyDebug(true),
		mock.WithExpectAnyWarn(true),
		mock.WithExpectAnyError(true),
	)
	SetLoggerInterface(logger)

	return manager
}
