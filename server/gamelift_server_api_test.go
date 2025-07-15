/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package server

import (
	"os"
	"testing"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server/internal/mock"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server/internal/security"
	"github.com/golang/mock/gomock"
)

type TestEnvironment struct {
	ServerParameters
	ComputeType string
}

func cleanUpInitSdkTest(t *testing.T, ctrl *gomock.Controller) {
	clearEnvironmentVariables(t)
	err := Destroy()
	if err != nil {
		t.Fatalf("Failed to call Destroy during clean up: %s", err)
	}
	ctrl.Finish()
}

func setEnvironmentVariables(t *testing.T, environment TestEnvironment) {
	// Create map of environment keys to their values
	envMap := map[string]string{
		common.EnvironmentKeyWebsocketURL: environment.WebSocketURL,
		common.EnvironmentKeyComputeType:  environment.ComputeType,
		common.EnvironmentKeyAuthToken:    environment.AuthToken,
		common.EnvironmentKeyProcessID:    environment.ProcessID,
		common.EnvironmentKeyHostID:       environment.HostID,
		common.EnvironmentKeyFleetID:      environment.FleetID,
		common.EnvironmentKeyAwsRegion:    environment.AwsRegion,
		common.EnvironmentKeyAccessKey:    environment.AccessKey,
		common.EnvironmentKeySecretKey:    environment.SecretKey,
		common.EnvironmentKeySessionToken: environment.SessionToken,
	}

	// Loop through and set each environment variable
	for key, value := range envMap {
		if err := os.Setenv(key, value); err != nil {
			t.Fatalf("Failed to set environment variable %s: %s", key, err)
		}
	}
}

func clearEnvironmentVariables(t *testing.T) {
	envKeys := []string{
		common.EnvironmentKeyWebsocketURL,
		common.EnvironmentKeyComputeType,
		common.EnvironmentKeyAuthToken,
		common.EnvironmentKeyProcessID,
		common.EnvironmentKeyHostID,
		common.EnvironmentKeyFleetID,
		common.EnvironmentKeyAwsRegion,
		common.EnvironmentKeyAccessKey,
		common.EnvironmentKeySecretKey,
		common.EnvironmentKeySessionToken,
	}

	for _, key := range envKeys {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("Failed to clear environment variable %s: %s", key, err)
		}
	}
}

func mockGameLiftManager(ctrl *gomock.Controller) *mock.MockIGameLiftManager {
	mockManager := mock.NewMockIGameLiftManager(ctrl)
	manager = mockManager
	return mockManager
}

func mockSuccessfulConnect(mockManager *mock.MockIGameLiftManager, times int, testServerParams ServerParameters) {
	var sigV4QueryParameters map[string]string
	mockManager.
		EXPECT().
		Connect(
			common.MockEquals(testServerParams.WebSocketURL),
			common.MockEquals(testServerParams.ProcessID),
			common.MockEquals(testServerParams.HostID),
			common.MockEquals(testServerParams.FleetID),
			common.MockEquals(testServerParams.AuthToken),
			sigV4QueryParameters).
		Return(nil).
		Times(times)
}

func mockAgentlessConnect(mockManager *mock.MockIGameLiftManager, times int, testServerParams ServerParameters, unexpectedProcessIds ...string) {
	mockManager.
		EXPECT().
		Connect(
			common.MockEquals(testServerParams.WebSocketURL),
			common.MockNoneOf(unexpectedProcessIds...),
			common.MockEquals(testServerParams.HostID),
			common.MockEquals(testServerParams.FleetID),
			"", // AuthToken
			common.MockStringMapContainsExpectedValue(testServerParams.AccessKey)).
		Return(nil).
		Times(times)
}

func mockSigV4Connect(mockManager *mock.MockIGameLiftManager, times int, testServerParams ServerParameters) {
	mockManager.
		EXPECT().
		Connect(
			common.MockEquals(testServerParams.WebSocketURL),
			common.MockEquals(testServerParams.ProcessID),
			common.MockEquals(testServerParams.HostID),
			common.MockEquals(testServerParams.FleetID),
			"", // AuthToken
			common.MockStringMapContainsExpectedValue(testServerParams.AccessKey)).
		Return(nil).
		Times(times)
}

func mockContainerPreSigV4Fetching(mockManager *mock.MockIGameLiftManager, computeType string, credentials *security.AwsCredentials, taskId string) {
	mockManager.
		EXPECT().
		FetchCredentials(computeType).
		Return(credentials, nil).
		Times(1)
	mockManager.
		EXPECT().
		FetchMetadata(computeType).
		Return(&security.ContainerTaskMetadata{TaskId: taskId}, nil).
		Times(1)
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

// Goal - Customers can deploy exactly what they have verified locally in Anywhere
func TestInitSDK_ManagedEc2AfterAnywhere(t *testing.T) {
	// GIVEN
	ctrl := gomock.NewController(t)
	defer cleanUpInitSdkTest(t, ctrl)

	inputParameters := ServerParameters{
		WebSocketURL: "wss://test.input.url",
		ProcessID:    "input-process-id",
		HostID:       "input-host-id",
		FleetID:      "input-fleet-id",
		AuthToken:    "input-auth-token",
	}
	managedEc2Environment := TestEnvironment{
		ServerParameters: ServerParameters{
			WebSocketURL: "wss://test.env.url",
			ProcessID:    "env-process-id",
			HostID:       "env-host-id",
			FleetID:      "env-fleet-id",
			AuthToken:    "env-auth-token",
		},
	}
	setEnvironmentVariables(t, managedEc2Environment)
	mockManager := mockGameLiftManager(ctrl)
	mockSuccessfulConnect(mockManager, 1, managedEc2Environment.ServerParameters)
	mockManager.EXPECT().Disconnect().Times(1)

	// WHEN
	err := InitSDK(inputParameters)

	// THEN - mock expectation successful and no errors
	if err != nil {
		t.Fatal(err)
	}
}

// Goal - Customers can deploy exactly what they have verified locally in Anywhere
// If a customer uses SigV4 with local Anywhere testing, this should not break ManagedEc2
func TestInitSDK_ManagedEc2AfterSigV4Anywhere(t *testing.T) {
	// GIVEN
	ctrl := gomock.NewController(t)
	defer cleanUpInitSdkTest(t, ctrl)

	inputParameters := ServerParameters{
		WebSocketURL: "wss://test.input.url",
		ProcessID:    "input-process-id",
		HostID:       "input-host-id",
		FleetID:      "input-fleet-id",
		AwsRegion:    "us-west-2",
		AccessKey:    "input-access-key",
		SecretKey:    "input-secret-key",
		SessionToken: "input-session-token",
	}
	managedEc2Environment := TestEnvironment{
		ServerParameters: ServerParameters{
			WebSocketURL: "wss://test.env.url",
			ProcessID:    "env-process-id",
			HostID:       "env-host-id",
			FleetID:      "env-fleet-id",
			AuthToken:    "env-auth-token",
		},
	}
	setEnvironmentVariables(t, managedEc2Environment)
	mockManager := mockGameLiftManager(ctrl)
	mockSuccessfulConnect(mockManager, 1, managedEc2Environment.ServerParameters)
	mockManager.EXPECT().Disconnect().Times(1)

	// WHEN
	err := InitSDK(inputParameters)

	// THEN - mock expectation successful and no errors
	if err != nil {
		t.Fatal(err)
	}
}

func TestInitSDK_AnywhereWithAuthToken(t *testing.T) {
	// GIVEN
	ctrl := gomock.NewController(t)
	defer cleanUpInitSdkTest(t, ctrl)

	anywhereParameters := ServerParameters{
		WebSocketURL: "wss://test.input.url",
		ProcessID:    "input-process-id",
		HostID:       "input-compute-id",
		FleetID:      "input-fleet-id",
		AuthToken:    "input-auth-token",
	}
	clearEnvironmentVariables(t)
	mockManager := mockGameLiftManager(ctrl)
	mockSuccessfulConnect(mockManager, 1, anywhereParameters)
	mockManager.EXPECT().Disconnect().Times(1)

	// WHEN
	err := InitSDK(anywhereParameters)

	// THEN
	if err != nil {
		t.Fatal(err)
	}
}

func TestInitSDK_AnywhereWithSigV4(t *testing.T) {
	// GIVEN
	ctrl := gomock.NewController(t)
	defer cleanUpInitSdkTest(t, ctrl)

	anywhereParameters := ServerParameters{
		WebSocketURL: "wss://test.input.url",
		ProcessID:    "test-process-id",
		HostID:       "test-compute-id",
		FleetID:      "test-fleet-id",
		AwsRegion:    "us-west-2",
		AccessKey:    "test-access-key",
		SecretKey:    "test-secret-key",
		SessionToken: "test-session-token",
	}
	clearEnvironmentVariables(t)
	mockManager := mockGameLiftManager(ctrl)
	mockSigV4Connect(mockManager, 1, anywhereParameters)
	mockManager.EXPECT().Disconnect().Times(1)

	// WHEN
	err := InitSDK(anywhereParameters)

	// THEN
	if err != nil {
		t.Fatal(err)
	}
}

// Customers may play around with using environment variables for Anywhere. If they clean up the environment with
// "" instead of un-setting the environment. This would previously block input arguments, so this test is explicit
// documentation of the change to allow this edge-case behavior.
func TestInitSDK_AnywhereWithBadEnvironment(t *testing.T) {
	// GIVEN
	ctrl := gomock.NewController(t)
	defer cleanUpInitSdkTest(t, ctrl)

	// Setup and assert the environment is as expected before the test
	setEnvironmentVariables(t, TestEnvironment{ServerParameters: ServerParameters{}})
	val, ok := os.LookupEnv(common.EnvironmentKeyAuthToken)
	if !ok {
		t.Fatal("Environment variables should be explicitly set to empty string ''")
	}
	if val != "" {
		t.Fatalf("Environment variables should be empty but AuthToken was '%s'", val)
	}

	anywhereParameters := ServerParameters{
		WebSocketURL: "wss://test.input.url",
		ProcessID:    "input-process-id",
		HostID:       "input-compute-id",
		FleetID:      "input-fleet-id",
		AuthToken:    "input-auth-token",
	}
	mockManager := mockGameLiftManager(ctrl)
	mockSuccessfulConnect(mockManager, 1, anywhereParameters)
	mockManager.EXPECT().Disconnect().Times(1)

	// WHEN
	err := InitSDK(anywhereParameters)

	// THEN
	if err != nil {
		t.Fatal(err)
	}
}

// Agentless containers do not receive the HostId or AuthToken environment variables and must use SigV4 to authenticate.
// Additionally, agentless containers generate process ids at runtime.
func TestInitSDK_ContainersAgentless(t *testing.T) {
	// GIVEN
	ctrl := gomock.NewController(t)
	defer cleanUpInitSdkTest(t, ctrl)

	// Extra: verify parameters are ignored if left in the code from customer's Anywhere testing
	// Note: Containers agentless does NOT work if a non-empty AuthToken is leftover from customer's test environment
	inputParameters := ServerParameters{
		WebSocketURL: "wss://test.input.url",
		ProcessID:    "input-process-id",
		FleetID:      "input-fleet-id",
		AccessKey:    "input-access-key",
		AuthToken:    "",
	}
	containersEnvironment := TestEnvironment{
		ServerParameters: ServerParameters{
			WebSocketURL: "wss://test.env.url",
			ProcessID:    common.AgentlessContainerProcessId,
			FleetID:      "env-fleet-id",
			AwsRegion:    "us-west-2",
		},
		ComputeType: common.ComputeTypeContainer,
	}
	fetchedCredentials := security.AwsCredentials{
		AccessKey:    "fetched-access-key",
		SecretKey:    "fetched-secret-key",
		SessionToken: "fetched-session-token",
	}
	taskId := "metadata-task-id"

	expectedConnectParameters := containersEnvironment.ServerParameters
	// Container's host id comes from the metadata
	expectedConnectParameters.HostID = taskId
	// The SigV4 parameters in containers come from the fetched credentials
	expectedConnectParameters.AccessKey = fetchedCredentials.AccessKey

	// Ensure we are not populating the host id in the environment
	setEnvironmentVariables(t, containersEnvironment)
	err := os.Unsetenv(common.EnvironmentKeyHostID)
	if err != nil {
		t.Fatalf("Failed to clear HostId environment variable. It must not be set for Agentless containers. %s", err)
	}

	mockManager := mockGameLiftManager(ctrl)
	mockContainerPreSigV4Fetching(mockManager, containersEnvironment.ComputeType, &fetchedCredentials, taskId)
	mockAgentlessConnect(mockManager, 1, expectedConnectParameters,
		// ensure a unique value is generated and that it overwrote the input server parameter
		common.AgentlessContainerProcessId, inputParameters.ProcessID)
	mockManager.EXPECT().Disconnect().Times(1)

	// WHEN
	err = InitSDK(inputParameters)

	// THEN
	if err != nil {
		t.Fatal(err)
	}
}

// Game servers run from the containers agent will generally have their environment populated with a process id,
// AuthToken, and host id, but it is not restricted and can be run similar to Agentless with SigV4 and runtime
// process ids.
// This test covers the cases not verified by Agentless
func TestInitSDK_ContainersWithAgent(t *testing.T) {
	// GIVEN
	ctrl := gomock.NewController(t)
	defer cleanUpInitSdkTest(t, ctrl)

	containersEnvironment := TestEnvironment{
		ServerParameters: ServerParameters{
			WebSocketURL: "wss://test.env.url",
			ProcessID:    "env-process-id",
			HostID:       "env-compute-id",
			FleetID:      "env-fleet-id",
			AuthToken:    "env-auth-token",
		},
		ComputeType: common.ComputeTypeContainer,
	}
	setEnvironmentVariables(t, containersEnvironment)
	mockManager := mockGameLiftManager(ctrl)
	mockSuccessfulConnect(mockManager, 1, containersEnvironment.ServerParameters)
	mockManager.EXPECT().Disconnect().Times(1)

	// WHEN
	err := InitSDK(containersEnvironment.ServerParameters)

	// THEN
	if err != nil {
		t.Fatal(err)
	}
}

func TestInitSDKFromEnvironment_ManagedEc2(t *testing.T) {
	// GIVEN
	ctrl := gomock.NewController(t)
	defer cleanUpInitSdkTest(t, ctrl)

	managedEc2Environment := TestEnvironment{
		ServerParameters: ServerParameters{
			WebSocketURL: "wss://test.env.url",
			ProcessID:    "env-process-id",
			HostID:       "env-host-id",
			FleetID:      "env-fleet-id",
			AuthToken:    "env-auth-token",
		},
	}
	setEnvironmentVariables(t, managedEc2Environment)
	mockManager := mockGameLiftManager(ctrl)
	mockSuccessfulConnect(mockManager, 1, managedEc2Environment.ServerParameters)
	mockManager.EXPECT().Disconnect().Times(1)

	// WHEN
	err := InitSDKFromEnvironment()

	// THEN
	if err != nil {
		t.Fatal(err)
	}
}

func TestInitSDKFromEnvironment_Anywhere(t *testing.T) {
	// GIVEN
	ctrl := gomock.NewController(t)
	defer cleanUpInitSdkTest(t, ctrl)

	clearEnvironmentVariables(t)
	mockManager := mockGameLiftManager(ctrl)
	mockManager.EXPECT().Disconnect().Times(0)

	// WHEN
	err := InitSDKFromEnvironment()

	// THEN
	if err == nil {
		t.Fatal("Expected error due to missing environment variables")
	}
}

// Agentless containers do not receive the HostId or AuthToken environment variables and must use SigV4 to authenticate.
// Additionally, agentless containers generate process ids at runtime.
func TestInitSDKFromEnvironment_ContainersAgentless(t *testing.T) {
	// GIVEN
	ctrl := gomock.NewController(t)
	defer cleanUpInitSdkTest(t, ctrl)

	containersEnvironment := TestEnvironment{
		ServerParameters: ServerParameters{
			WebSocketURL: "wss://test.env.url",
			ProcessID:    common.AgentlessContainerProcessId,
			FleetID:      "env-fleet-id",
			AwsRegion:    "us-west-2",
			// Extra: verify environment credentials are always ignored by containers
			AccessKey:    "env-access-key",
			SecretKey:    "env-secret-key",
			SessionToken: "env-session-token",
		},
		ComputeType: common.ComputeTypeContainer,
	}
	fetchedCredentials := security.AwsCredentials{
		AccessKey:    "fetched-access-key",
		SecretKey:    "fetched-secret-key",
		SessionToken: "fetched-session-token",
	}
	taskId := "metadata-task-id"

	expectedConnectParameters := containersEnvironment.ServerParameters
	// Container's host id comes from the metadata
	expectedConnectParameters.HostID = taskId
	// The SigV4 parameters in containers come from the fetched credentials
	expectedConnectParameters.AccessKey = fetchedCredentials.AccessKey

	// Ensure we are not populating the host id in the environment
	setEnvironmentVariables(t, containersEnvironment)
	err := os.Unsetenv(common.EnvironmentKeyHostID)
	if err != nil {
		t.Fatalf("Failed to clear HostId environment variable. It must not be set for Agentless containers. %s", err)
	}

	mockManager := mockGameLiftManager(ctrl)
	mockContainerPreSigV4Fetching(mockManager, containersEnvironment.ComputeType, &fetchedCredentials, taskId)
	mockAgentlessConnect(mockManager, 1, expectedConnectParameters, common.AgentlessContainerProcessId)
	mockManager.EXPECT().Disconnect().Times(1)

	// WHEN
	err = InitSDKFromEnvironment()

	// THEN
	if err != nil {
		t.Fatal(err)
	}
}

// Game servers run from the containers agent will generally have their environment populated with a process id,
// AuthToken, and host id, but it is not restricted and can be run similar to Agentless with SigV4 and runtime
// process ids.
// This test covers the cases not verified by Agentless
func TestInitSDKFromEnvironment_ContainersWithAgent(t *testing.T) {
	// GIVEN
	ctrl := gomock.NewController(t)
	defer cleanUpInitSdkTest(t, ctrl)

	containersEnvironment := TestEnvironment{
		ServerParameters: ServerParameters{
			WebSocketURL: "wss://test.env.url",
			ProcessID:    "env-process-id",
			HostID:       "env-compute-id",
			FleetID:      "env-fleet-id",
			AuthToken:    "env-auth-token",
		},
		ComputeType: common.ComputeTypeContainer,
	}
	setEnvironmentVariables(t, containersEnvironment)
	mockManager := mockGameLiftManager(ctrl)
	mockSuccessfulConnect(mockManager, 1, containersEnvironment.ServerParameters)
	mockManager.EXPECT().Disconnect().Times(1)

	// WHEN
	err := InitSDKFromEnvironment()

	// THEN
	if err != nil {
		t.Fatal(err)
	}
}

// Now that we initialize the Logger during InitSDK so it utilizes the process id, we must explicitly verify
// that we still use the customer's custom Logger implementation when provided.
func TestInitSDKFromEnvironment_WithLogger(t *testing.T) {
	// GIVEN
	ctrl := gomock.NewController(t)
	defer cleanUpInitSdkTest(t, ctrl)
	logger := mock.NewTestLogger(t, ctrl)

	// WHEN
	SetLoggerInterface(logger)
	err := InitSDKFromEnvironment()

	// THEN
	if err == nil {
		t.Fatal("Expected a validation error because there is no environment state")
	}
	common.AssertEqual(t, logger, manager.GetLogger())
}

func TestDestroy(t *testing.T) {
	// GIVEN
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var inputParameters = ServerParameters{
		WebSocketURL: "wss://test.url",
		ProcessID:    "test-process-id",
		HostID:       "test-host-id",
		FleetID:      "test-fleet-id",
		AuthToken:    "test-auth-token",
	}
	mockManager := mockGameLiftManager(ctrl)
	mockSuccessfulConnect(mockManager, 1, inputParameters)
	mockManager.EXPECT().Disconnect().Times(1)

	// WHEN
	if err := InitSDK(inputParameters); err != nil {
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
