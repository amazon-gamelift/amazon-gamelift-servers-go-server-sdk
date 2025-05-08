/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package server

import (
	"aws/amazon-gamelift-go-sdk/common"
	"aws/amazon-gamelift-go-sdk/server/internal/mock"
	"github.com/golang/mock/gomock"
	"os"
	"testing"
)

var testServerParams = ServerParameters{
	WebSocketURL: "wss://test.url",
	ProcessID:    "test-process-id",
	HostID:       "test-host-id",
	FleetID:      "test-fleet-id",
	AuthToken:    "test-auth-token",
}

func assertEqual(t *testing.T, expected interface{}, actual interface{}) {
	if expected != actual {
		t.Fatalf("Expected %v but got %v", expected, actual)
	}
}

func verifyServerStateParams(t *testing.T, expectedParams *ServerParameters) {
	assertEqual(t, state.processID, expectedParams.ProcessID)
	assertEqual(t, state.processID, expectedParams.ProcessID)
	assertEqual(t, state.hostID, expectedParams.HostID)
	assertEqual(t, state.fleetID, expectedParams.FleetID)
}

func setEnvironmentVariables() error {
	if err := os.Setenv(common.EnvironmentKeyAuthToken, testServerParams.AuthToken); err != nil {
		return err
	}
	if err := os.Setenv(common.EnvironmentKeyFleetID, testServerParams.FleetID); err != nil {
		return err
	}
	if err := os.Setenv(common.EnvironmentKeyProcessID, testServerParams.ProcessID); err != nil {
		return err
	}
	if err := os.Setenv(common.EnvironmentKeyHostID, testServerParams.HostID); err != nil {
		return err
	}
	if err := os.Setenv(common.EnvironmentKeyWebsocketURL, testServerParams.WebSocketURL); err != nil {
		return err
	}
	return nil
}

func clearEnvironmentVariables() error {
	if err := os.Unsetenv(common.EnvironmentKeyAuthToken); err != nil {
		return err
	}
	if err := os.Unsetenv(common.EnvironmentKeyFleetID); err != nil {
		return err
	}
	if err := os.Unsetenv(common.EnvironmentKeyProcessID); err != nil {
		return err
	}
	if err := os.Unsetenv(common.EnvironmentKeyHostID); err != nil {
		return err
	}
	if err := os.Unsetenv(common.EnvironmentKeyWebsocketURL); err != nil {
		return err
	}
	return nil
}

func newMockManager(t *testing.T) *mock.MockIGameLiftManager {
	ctrl := gomock.NewController(t)
	mockManager := mock.NewMockIGameLiftManager(ctrl)
	manager = mockManager
	return mockManager
}

func mockSuccessfulConnect(mockManager *mock.MockIGameLiftManager, times int) {
	mockManager.
		EXPECT().
		Connect(testServerParams.WebSocketURL, testServerParams.ProcessID, testServerParams.HostID, testServerParams.FleetID, testServerParams.AuthToken).
		Return(nil).
		Times(times)
}

func TestGetSDKVersion(t *testing.T) {
	version, err := GetSdkVersion()
	if err != nil {
		t.Fatal(err)
	}

	if version != common.SdkVersion {
		t.Errorf("expect  %v but get %v", common.SdkVersion, version)
	}
}

func TestInitSDK(t *testing.T) {
	// GIVEN
	mockManager := newMockManager(t)
	mockSuccessfulConnect(mockManager, 1)
	mockManager.EXPECT().Disconnect().Times(1)

	// WHEN
	err := InitSDK(testServerParams)

	// THEN
	verifyServerStateParams(t, &testServerParams)
	if err != nil {
		t.Fatal(err)
	}
	Destroy()
}

func TestInitSDKFromEnvironment(t *testing.T) {
	// GIVEN
	if err := setEnvironmentVariables(); err != nil {
		t.Fatalf("Could not set environment variable: %v", err)
	}
	mockManager := newMockManager(t)
	mockSuccessfulConnect(mockManager, 1)
	mockManager.EXPECT().Disconnect().Times(1)

	// WHEN
	err := InitSDKFromEnvironment()

	// THEN
	verifyServerStateParams(t, &testServerParams)
	if err != nil {
		t.Fatal(err)
	}
	clearEnvironmentVariables()
	Destroy()
}

func TestInitSDKFromEnvironment_UndefinedEnv(t *testing.T) {
	// GIVEN
	if err := clearEnvironmentVariables(); err != nil {
		t.Fatalf("Could not clear environment variables: %v", err)
	}

	// WHEN
	err := InitSDKFromEnvironment()

	// THEN
	verifyServerStateParams(t, &testServerParams)
	if err == nil {
		t.Fatal("Expected exception")
	}
}

func TestDestroy(t *testing.T) {
	// GIVEN
	mockManager := newMockManager(t)
	mockSuccessfulConnect(mockManager, 1)
	mockManager.EXPECT().Disconnect().Times(1)

	// WHEN
	if err := InitSDK(testServerParams); err != nil {
		t.Fatal(err)
	}
	if err := Destroy(); err != nil {
		t.Fatal(err)
	}

	// THEN
	if manager != nil {
		t.Fatal("Manager should be uninitialized")
	}
	if srv != nil {
		t.Fatal("Server should be uninitialized")
	}
}
