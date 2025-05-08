/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package server

import (
	"aws/amazon-gamelift-go-sdk/common"
	"aws/amazon-gamelift-go-sdk/model"
	"aws/amazon-gamelift-go-sdk/model/request"
	"fmt"
	"strings"
	"testing"
)

const (
	TEST_GAME_SESSION_ARN              = "arn:aws:gamelift:us-west-2::gamesession/fleet-test/location-test/gsess-test"
	TEST_MATCHMAKING_CONFIGURATION_ARN = "arn:aws:gamelift:us-west-2:000000000000:matchmakingconfiguration/test"
)

func TestValidateServerParameters_WithAuthToken(t *testing.T) {
	input := ServerParameters{
		WebSocketURL: "wss://test.url",
		ProcessID:    "test-process-id",
		HostID:       "test-host-id",
		FleetID:      "test-fleet-id",
		AuthToken:    "test-auth-token",
	}
	computeType := ""

	// Test valid parameters
	err := ValidateServerParameters(input, computeType)
	// Assert no error
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestValidateServerParameters_WithValidSigV4(t *testing.T) {
	input := ServerParameters{
		WebSocketURL: "wss://test.url",
		ProcessID:    "test-process-id",
		FleetID:      "test-fleet-id",
		AwsRegion:    "us-west-2",
	}
	computeType := common.ComputeTypeContainer

	// Test valid parameters
	err := ValidateServerParameters(input, computeType)
	// Assert no error
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	computeType = ""
	input.HostID = "test-host-id"
	input.AccessKey = "test-access-key"
	input.SecretKey = "test-secret-key"
	err = ValidateServerParameters(input, computeType)
	// Assert no error
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestValidateServerParameters_InvalidParams(t *testing.T) {
	input := ServerParameters{
		WebSocketURL: "wss://test.url",
		ProcessID:    "test-process-id",
		HostID:       "test-host-id",
		FleetID:      "test-fleet-id",
		AuthToken:    "test-auth-token",
	}
	computeType := ""

	// WHEN - websocket empty
	input.WebSocketURL = ""
	err := ValidateServerParameters(input, computeType)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), "WebSocketURL is required.")
	input.WebSocketURL = "wss://test.url"

	// WHEN - process id empty
	input.ProcessID = ""
	err = ValidateServerParameters(input, computeType)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), "ProcessID is required.")
	input.ProcessID = "test-process-id"

	// WHEN - host id invalid
	// empty w/ auth token
	input.HostID = ""
	err = ValidateServerParameters(input, computeType)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), "HostID is required.")
	// empty w/ non-containers sigV4
	input.AuthToken = ""
	input.AwsRegion = "test-region"
	input.AccessKey = "test-access-key"
	input.SecretKey = "test-secret-key"
	err = ValidateServerParameters(input, computeType)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), "HostID is required.")
	input.AuthToken = "test-auth-token"
	input.AwsRegion = ""
	input.AccessKey = ""
	input.SecretKey = ""
	// too long
	input.HostID = strings.Repeat("a", common.MaxStringLengthId+1)
	err = ValidateServerParameters(input, computeType)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("HostID is invalid. Length must be between 1 and %d characters.", common.MaxStringLengthId))
	// invalid char
	input.HostID = "test-host-id!"
	err = ValidateServerParameters(input, computeType)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("HostID is invalid. Must match the pattern: %s.", computeIdRegex.String()))
	input.HostID = "test-host-id"

	// WHEN - fleet id invalid
	// empty
	input.FleetID = ""
	err = ValidateServerParameters(input, computeType)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), "FleetID is required.")
	// too long
	input.FleetID = strings.Repeat("a", common.MaxStringLengthId+1)
	err = ValidateServerParameters(input, computeType)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("FleetID is invalid. Length must be between 1 and %d characters.", common.MaxStringLengthId))
	// invalid char
	input.FleetID = "test-fleet-id!"
	err = ValidateServerParameters(input, computeType)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("FleetID is invalid. Must match the pattern: %s.", guidRegex.String()))
	input.FleetID = "test-fleet-id"

	// WHEN - invalid credentials
	errAuthMessage := "Either AuthToken or AwsRegion and AwsCredentials are required"
	input.AuthToken = ""
	err = ValidateServerParameters(input, computeType)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), errAuthMessage)
	// empty only one of AwsRegion
	input.AwsRegion = ""
	input.AccessKey = "test-access-key"
	input.SecretKey = "test-secret-key"
	err = ValidateServerParameters(input, computeType)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), errAuthMessage)
	// too many auth options
	input.AuthToken = "test-auth-token"
	input.AwsRegion = "test-region"
	err = ValidateServerParameters(input, computeType)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), "Only one of AuthToken or AwsRegion and AwsCredentials can be provided at once")
}

func TestValidateProcessParameters(t *testing.T) {
	input := ProcessParameters{
		Port: common.PortMin + 1000,
	}
	// Test valid parameters
	err := ValidateProcessParameters(input)
	// Assert no error
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestValidateProcessParameters_InvalidParams(t *testing.T) {
	input := ProcessParameters{}
	// WHEN - port invalid
	// too low
	input.Port = common.PortMin - 1
	err := ValidateProcessParameters(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	errMessage := fmt.Sprintf("Port must be between %d and %d", common.PortMin, common.PortMax)
	common.AssertContains(t, err.Error(), errMessage)
	// too high
	input.Port = common.PortMax + 1
	err = ValidateProcessParameters(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), errMessage)
}

func TestValidatePlayerSessionCreationPolicy(t *testing.T) {
	// Test valid parameters
	err := ValidatePlayerSessionCreationPolicy(model.AcceptAll)
	// Assert no error
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	err = ValidatePlayerSessionCreationPolicy(model.DenyAll)
	// Assert no error
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestValidatePlayerSessionCreationPolicy_InvalidParams(t *testing.T) {
	err := ValidatePlayerSessionCreationPolicy(model.NotSet)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), "Player session creation policy must be one of [ACCEPT_ALL, DENY_ALL]")
}

func TestValidateDescribePlayerSessionsRequest(t *testing.T) {
	input := request.DescribePlayerSessionsRequest{
		PlayerSessionID:           "test-player-session-id",
		PlayerSessionStatusFilter: "ACTIVE",
	}
	err := ValidateDescribePlayerSessionsRequest(input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	// WHEN - game session id
	input.PlayerSessionID = ""
	input.GameSessionID = TEST_GAME_SESSION_ARN

	err = ValidateDescribePlayerSessionsRequest(input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	input.GameSessionID = ""
	// WHEN - player id
	input.PlayerID = "test-player-id"
	err = ValidateDescribePlayerSessionsRequest(input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestValidateDescribePlayerSessionsRequest_InvalidParams(t *testing.T) {
	input := request.DescribePlayerSessionsRequest{
		PlayerSessionID: "test-player-session-id",
	}

	// WHEN - too many params
	input.PlayerID = "test-player-id"
	err := ValidateDescribePlayerSessionsRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), "Exactly one of GameSessionId, PlayerSessionId, or PlayerId is required")
	// WHEN - no params
	input.PlayerID = ""
	input.PlayerSessionID = ""
	err = ValidateDescribePlayerSessionsRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), "Exactly one of GameSessionId, PlayerSessionId, or PlayerId is required")
	input.PlayerSessionID = "test-player-session-id"
	// WHEN - invalid filter
	input.PlayerSessionStatusFilter = "INVALID"
	err = ValidateDescribePlayerSessionsRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), "PlayerSessionStatusFilter must be one of [RESERVED, ACTIVE, COMPLETED, TIMEDOUT]")
	input.PlayerSessionStatusFilter = ""
	// WHEN - invalid player id
	// too long
	input.PlayerSessionID = ""
	input.PlayerID = strings.Repeat("a", common.MaxStringLengthLong+1)
	err = ValidateDescribePlayerSessionsRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("PlayerID is invalid. Length must be between 1 and %d characters.", common.MaxStringLengthLong))
	// WHEN - invalid player session id
	// too long
	input.PlayerID = ""
	input.PlayerSessionID = strings.Repeat("a", common.MaxStringLengthLong+1)
	err = ValidateDescribePlayerSessionsRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("PlayerSessionID is invalid. Length must be between 1 and %d characters.", common.MaxStringLengthLong))
	// WHEN - invalid game session id
	// too long
	input.PlayerSessionID = ""
	input.GameSessionID = strings.Repeat("a", common.MaxStringLengthArn+1)
	err = ValidateDescribePlayerSessionsRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("GameSessionID is invalid. Length must be between 1 and %d characters.", common.MaxStringLengthArn))
	// invalid format
	input.GameSessionID = "arn!"
	err = ValidateDescribePlayerSessionsRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("GameSessionID is invalid. Invalid ARN format."))
}

func TestValidateStartMatchBackfillRequest(t *testing.T) {
	input := request.StartMatchBackfillRequest{
		TicketID:                    "test-ticket-id",
		GameSessionArn:              TEST_GAME_SESSION_ARN,
		MatchmakingConfigurationArn: TEST_MATCHMAKING_CONFIGURATION_ARN,
		Players: []model.Player{
			{
				PlayerID: "test-player-id",
			},
		},
	}
	// Test valid parameters
	err := ValidateStartMatchBackfillRequest(input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestValidateStartMatchBackfillRequest_InvalidParams(t *testing.T) {
	input := request.StartMatchBackfillRequest{
		TicketID:                    "test-ticket-id",
		GameSessionArn:              TEST_GAME_SESSION_ARN,
		MatchmakingConfigurationArn: TEST_MATCHMAKING_CONFIGURATION_ARN,
		Players: []model.Player{
			{
				PlayerID: "test-player-id",
			},
		},
	}
	// WHEN - invalid GameSessionArn
	// empty
	input.GameSessionArn = ""
	err := ValidateStartMatchBackfillRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), "GameSessionArn is required.")
	// invalid format
	input.GameSessionArn = "arn!"
	err = ValidateStartMatchBackfillRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("GameSessionArn is invalid. Invalid ARN format."))
	// invalid length
	input.GameSessionArn = strings.Repeat("a", common.MaxStringLengthArn+1)
	err = ValidateStartMatchBackfillRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("GameSessionArn is invalid. Length must be between 1 and %d characters.", common.MaxStringLengthArn))
	input.GameSessionArn = TEST_GAME_SESSION_ARN
	// WHEN - invalid MatchmakingConfigurationArn
	// empty
	input.MatchmakingConfigurationArn = ""
	err = ValidateStartMatchBackfillRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), "MatchmakingConfigurationArn is required.")
	// invalid format
	input.MatchmakingConfigurationArn = "arn!"
	err = ValidateStartMatchBackfillRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("MatchmakingConfigurationArn is invalid. Invalid GameLift ARN format."))
	// invalid length
	input.MatchmakingConfigurationArn = strings.Repeat("a", common.MaxStringLengthArn+1)
	err = ValidateStartMatchBackfillRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("MatchmakingConfigurationArn is invalid. Length must be between 1 and %d characters.", common.MaxStringLengthArn))
	input.MatchmakingConfigurationArn = TEST_MATCHMAKING_CONFIGURATION_ARN
	// WHEN - invalid TicketID
	// invalid length
	input.TicketID = strings.Repeat("a", common.MaxStringLengthMatchmakingId+1)
	err = ValidateStartMatchBackfillRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("TicketID is invalid. Length must be between 1 and %d characters.", common.MaxStringLengthMatchmakingId))
	// invalid char
	input.TicketID = "test-ticket-id!"
	err = ValidateStartMatchBackfillRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("TicketID is invalid. Must match the pattern: %s.", matchmakingIdRegex.String()))
	input.TicketID = "test-ticket-id"
	// WHEN - invalid Players
	// empty
	input.Players = []model.Player{}
	err = ValidateStartMatchBackfillRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), "Players cannot be empty.")
}

func TestValidateStopMatchBackfillRequest(t *testing.T) {
	input := request.StopMatchBackfillRequest{
		TicketID:                    "test-ticket-id",
		GameSessionArn:              TEST_GAME_SESSION_ARN,
		MatchmakingConfigurationArn: TEST_MATCHMAKING_CONFIGURATION_ARN,
	}
	// Test valid parameters
	err := ValidateStopMatchBackfillRequest(input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestValidateStopMatchBackfillRequest_InvalidParams(t *testing.T) {
	input := request.StopMatchBackfillRequest{
		TicketID:                    "test-ticket-id",
		GameSessionArn:              TEST_GAME_SESSION_ARN,
		MatchmakingConfigurationArn: TEST_MATCHMAKING_CONFIGURATION_ARN,
	}
	// WHEN - invalid GameSessionArn
	// empty
	input.GameSessionArn = ""
	err := ValidateStopMatchBackfillRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), "GameSessionArn is required.")
	// invalid format
	input.GameSessionArn = "arn!"
	err = ValidateStopMatchBackfillRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("GameSessionArn is invalid. Invalid GameLift ARN format."))
	// invalid length
	input.GameSessionArn = strings.Repeat("a", common.MaxStringLengthArn+1)
	err = ValidateStopMatchBackfillRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("GameSessionArn is invalid. Length must be between 1 and %d characters.", common.MaxStringLengthArn))
	input.GameSessionArn = TEST_GAME_SESSION_ARN
	// WHEN - invalid MatchmakingConfigurationArn
	// empty
	input.MatchmakingConfigurationArn = ""
	err = ValidateStopMatchBackfillRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), "MatchmakingConfigurationArn is required.")
	// invalid format
	input.MatchmakingConfigurationArn = "arn!"
	err = ValidateStopMatchBackfillRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("MatchmakingConfigurationArn is invalid. Invalid GameLift ARN format."))
	input.MatchmakingConfigurationArn = TEST_MATCHMAKING_CONFIGURATION_ARN
	// WHEN - invalid TicketID
	// empty
	input.TicketID = ""
	err = ValidateStopMatchBackfillRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), "TicketID is required.")
	// invalid char
	input.TicketID = "test-ticket-id!"
	err = ValidateStopMatchBackfillRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("TicketID is invalid. Must match the pattern: %s.", matchmakingIdRegex.String()))
	// invalid length
	input.TicketID = strings.Repeat("a", common.MaxStringLengthMatchmakingId+1)
	err = ValidateStopMatchBackfillRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("TicketID is invalid. Length must be between 1 and %d characters.", common.MaxStringLengthMatchmakingId))
}

func TestValidateGetFleetRoleCredentialsRequest(t *testing.T) {
	input := request.GetFleetRoleCredentialsRequest{
		RoleArn:         "test-role-arn",
		RoleSessionName: "test-role-session-name",
	}
	// Test valid parameters
	err := ValidateGetFleetRoleCredentialsRequest(input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestValidateGetFleetRoleCredentialsRequest_InvalidParams(t *testing.T) {
	input := request.GetFleetRoleCredentialsRequest{
		RoleArn:         "test-role-arn",
		RoleSessionName: "test-role-session-name",
	}
	// WHEN - invalid RoleArn
	// empty
	input.RoleArn = ""
	err := ValidateGetFleetRoleCredentialsRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), "RoleArn is required.")
	// invalid format
	input.RoleArn = "arn!"
	err = ValidateGetFleetRoleCredentialsRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("RoleArn is invalid. Invalid ARN format."))
	// invalid length
	input.RoleArn = strings.Repeat("a", common.MaxStringLengthArn+1)
	err = ValidateGetFleetRoleCredentialsRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("RoleArn is invalid. Length must be between 1 and %d characters.", common.MaxStringLengthArn))
	input.RoleArn = "test-role-arn"
	// WHEN - invalid RoleSessionName
	// empty
	input.RoleSessionName = ""
	err = ValidateGetFleetRoleCredentialsRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), "RoleSessionName is required.")
	// too long
	input.RoleSessionName = strings.Repeat("a", common.RoleSessionNameMaxLength+1)
	err = ValidateGetFleetRoleCredentialsRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("RoleSessionName is invalid. Length must be between 2 and %d characters.", common.RoleSessionNameMaxLength))
	// too short
	input.RoleSessionName = "a"
	err = ValidateGetFleetRoleCredentialsRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("RoleSessionName is invalid. Length must be between 2 and %d characters.", common.RoleSessionNameMaxLength))
	// invalid char
	input.RoleSessionName = "test-role-session-name!"
	err = ValidateGetFleetRoleCredentialsRequest(input)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	common.AssertContains(t, err.Error(), fmt.Sprintf("RoleSessionName is invalid. Must match the pattern: %s.", roleSessionNameRegex.String()))
}
