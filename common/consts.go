/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import "time"

// Default values
const (
	MaxPlayerSessions                               = 1024
	ServiceCallTimeoutDefault         time.Duration = 20 * time.Second
	MaxRetryDefault                                 = 5
	RetryFactorDefault                              = 2
	MaxReconnectBackoffDuration                     = 32 * time.Second
	RetryIntervalDefault                            = 2 * time.Second
	ConnectMaxRetries                               = 7
	ConnectRetryInterval                            = 2 * time.Second
	ServiceBufferSizeDefault                        = 2048
	HealthcheckIntervalDefault                      = 60 * time.Second
	HealthcheckRetryIntervalDefault                 = 10 * time.Second
	HealthcheckMaxJitterDefault                     = 10 * time.Second
	HealthcheckTimeoutDefault                       = HealthcheckIntervalDefault - HealthcheckRetryIntervalDefault
	DisconnectWebsocketTimeoutDefault               = 5 * time.Second
	// InstanceRoleCredentialTTL duration of expiration we retrieve new instance role credentials
	InstanceRoleCredentialTTL     = 15 * time.Minute
	RoleSessionNameMaxLength  int = 64
	// ReconnectOnReadWriteFailureNumber Number of consecutive read/write failures before reconnect is called
	ReconnectOnReadWriteFailureNumber int = 2
	// MaxReadWriteRetry The max number of retries after consecutive read/write failures, including the reconnect described above
	MaxReadWriteRetry            int = 5
	MaxStringLengthMatchmakingId     = 128
	MaxStringLengthId            int = 256
	MaxStringLengthArn           int = 256
	MaxStringLengthLong          int = 1024
	MaxStringLengthNoLimit       int = 0
	PortMin                      int = 1
	PortMax                      int = 60000
)

const (
	SdkLanguage                 = "Go"
	SdkLanguageKey              = "sdkLanguage"
	PidKey                      = "pID"
	SdkVersionKey               = "sdkVersion"
	SdkVersion                  = "5.3.0"
	AuthTokenKey                = "Authorization"
	ComputeIDKey                = "ComputeId"
	FleetIDKey                  = "FleetId"
	IdempotencyTokenKey         = "IdempotencyToken"
	ComputeTypeContainer        = "CONTAINER"
	AgentlessContainerProcessId = "ManagedResource"
)

// Environment variables
const (
	ServiceCallTimeout = "SERVICE_CALL_TIMEOUT"
	ServiceBufferSize  = "SERVICE_BUFFER_SIZE"
	RetryInterval      = "RETRY_INTERVAL"
	MaxRetry           = "MAX_RETRY"
	RetryFactor        = "RETRY_FACTOR"

	//nolint:gosec // false positive
	HealthcheckMaxJitter       = "HEALTHCHECK_MAX_JITTER"
	HealthcheckInterval        = "HEALTHCHECK_INTERVAL"
	HealthcheckTimeout         = "HEALTHCHECK_TIMEOUT"
	DisconnectWebsocketTimeout = "DISCONNECT_WEBSOCKET_TIMEOUT"
)

const (
	EnvironmentKeyWebsocketURL string = "GAMELIFT_SDK_WEBSOCKET_URL"
	EnvironmentKeyComputeType  string = "GAMELIFT_COMPUTE_TYPE"
	//nolint:gosec // false positive
	EnvironmentKeyAuthToken      string = "GAMELIFT_SDK_AUTH_TOKEN"
	EnvironmentKeyProcessID      string = "GAMELIFT_SDK_PROCESS_ID"
	EnvironmentKeyHostID         string = "GAMELIFT_SDK_HOST_ID"
	EnvironmentKeyFleetID        string = "GAMELIFT_SDK_FLEET_ID"
	EnvironmentKeyAwsRegion      string = "GAMELIFT_REGION"
	EnvironmentKeyAccessKey      string = "GAMELIFT_ACCESS_KEY"
	EnvironmentKeySecretKey      string = "GAMELIFT_SECRET_KEY"
	EnvironmentKeySessionToken   string = "GAMELIFT_SESSION_TOKEN"
	EnvironmentKeySDKToolName    string = "GAMELIFT_SDK_TOOL_NAME"
	EnvironmentKeySDKToolVersion string = "GAMELIFT_SDK_TOOL_VERSION"
)
