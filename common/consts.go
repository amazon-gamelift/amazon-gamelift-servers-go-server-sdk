/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import "time"

// Default values
const (
	MaxPlayerSessions                             = 1024
	ServiceCallTimeoutDefaut        time.Duration = 20 * time.Second
	MaxRetryDefault                               = 5
	RetryFactorDefault                            = 2
	RetryIntervalDefault                          = 2 * time.Second
	ServiceBufferSizeDefault                      = 1024
	HealthcheckIntervalDefault                    = 60 * time.Second
	HealthcheckRetryIntervalDefault               = 10 * time.Second
	HealthcheckMaxJitterDefault                   = 10 * time.Second
	HealthcheckTimeoutDefault                     = HealthcheckIntervalDefault - HealthcheckRetryIntervalDefault
	// InstanceRoleCredentialTTL duration of expiration we retrieve new instance role credentials
	InstanceRoleCredentialTTL     = 15 * time.Minute
	RoleSessionNameMaxLength  int = 64
)

const (
	SdkLanguage    = "Go"
	SdkLanguageKey = "sdkLanguage"
	PidKey         = "pID"
	SdkVersionKey  = "sdkVersion"
	SdkVersion     = "5.0.0"
	AuthTokenKey   = "Authorization"
	ComputeIDKey   = "ComputeId"
	FleetIDKey     = "FleetId"
)

// Environment variables
const (
	ServiceCallTimeout = "SERVICE_CALL_TIMEOUT"
	ServiceBufferSize  = "SERVICE_BUFFER_SIZE"
	RetryInterval      = "RETRY_INTERVAL"
	MaxRetry           = "MAX_RETRY"
	RetryFactor        = "RETRY_FACTOR"

	GameliftSdkWebsocketURL = "GAMELIFT_SDK_WEBSOCKET_URL"
	GameliftSdkProcessID    = "GAMELIFT_SDK_PROCESS_ID"
	GameliftSdkHostID       = "GAMELIFT_SDK_HOST_ID"
	GameliftSdkFleetID      = "GAMELIFT_SDK_FLEET_ID"
	// GameliftSdkAuthToken is an environment variable name where potential user can store token.
	//nolint:gosec // false positive
	GameliftSdkAuthToken = "GAMELIFT_SDK_AUTH_TOKEN"

	HealthcheckMaxJitter = "HEALTHCHECK_MAX_JITTER"
	HealthcheckInterval  = "HEALTHCHECK_INTERVAL"
	HealthcheckTimeout   = "HEALTHCHECK_TIMEOUT"
)
