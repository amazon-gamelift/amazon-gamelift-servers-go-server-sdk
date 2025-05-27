/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package server

import (
	"fmt"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model/request"
	"regexp"
)

// Regex Patterns
const (
	guidStringPattern                = `^[a-zA-Z0-9.-]+$`
	computeIdStringPattern           = `^[a-zA-Z0-9\-]+(\/[a-zA-Z0-9\-]+)?$`
	arnStringPattern                 = "^[a-zA-Z0-9:/-]+$"
	playerSessionStatusFilterPattern = "^(RESERVED|ACTIVE|COMPLETED|TIMEDOUT)$"
	gameLiftArnPattern               = `^arn:(aws|aws-cn):gamelift:([a-z]{2}-[a-z]+-\d{1}):(\d{12})?:([a-z]+)\/(.+)$`
	matchmakingIdPattern             = `^[a-zA-Z0-9-\.]*$`
	roleSessionNamePattern           = `^[\w+=,.@-]*$`
)

var guidRegex = regexp.MustCompile(guidStringPattern)
var computeIdRegex = regexp.MustCompile(computeIdStringPattern)
var arnRegex = regexp.MustCompile(arnStringPattern)
var playerSessionStatusFilterRegex = regexp.MustCompile(playerSessionStatusFilterPattern)
var gameLiftArnRegex = regexp.MustCompile(gameLiftArnPattern)
var matchmakingIdRegex = regexp.MustCompile(matchmakingIdPattern)
var roleSessionNameRegex = regexp.MustCompile(roleSessionNamePattern)

func ValidateServerParameters(input ServerParameters, computeType string) error {
	isContainerComputeType := computeType == common.ComputeTypeContainer
	isUsingAuthToken := input.AuthToken != ""
	isUsingSigV4Auth := input.AwsRegion != ""

	var authOptions string
	if isContainerComputeType {
		authOptions = "AuthToken or AwsRegion"
	} else {
		authOptions = "AuthToken or AwsRegion and AwsCredentials"
	}
	if isUsingAuthToken && isUsingSigV4Auth {
		return common.NewGameLiftError(common.ValidationException, "", fmt.Sprintf("Failed to provide a valid authorization strategy: Only one of %s can be provided at once", authOptions))
	}
	if !isUsingAuthToken && !isUsingSigV4Auth {
		return common.NewGameLiftError(common.ValidationException, "", fmt.Sprintf("Failed to provide a valid authorization strategy: Either %s are required", authOptions))
	}

	if isUsingAuthToken {
		return validateSpecificServerParameters(input, []property{WebSocketUrl, ProcessId, FleetId, HostId})
	} else if computeType == common.ComputeTypeContainer {
		return validateSpecificServerParameters(input, []property{WebSocketUrl, ProcessId, FleetId})
	} else {
		return validateSpecificServerParameters(input, []property{WebSocketUrl, ProcessId, FleetId, HostId, AwsCredentials})
	}
}

type property string

const (
	WebSocketUrl   property = "WebSocketURL"
	ProcessId      property = "ProcessID"
	FleetId        property = "FleetID"
	HostId         property = "HostID"
	AwsCredentials property = "AwsCredentials"
)

func validateSpecificServerParameters(input ServerParameters, propertiesToValidate []property) error {
	for _, property := range propertiesToValidate {
		switch property {
		case WebSocketUrl:
			err := common.ValidateString(string(property), input.WebSocketURL, nil, 1, common.MaxStringLengthNoLimit, true, "")
			if err != nil {
				return err
			}
		case ProcessId:
			err := common.ValidateString(string(property), input.ProcessID, nil, 1, common.MaxStringLengthNoLimit, true, "")
			if err != nil {
				return err
			}
		case FleetId:
			err := common.ValidateString(string(property), input.FleetID, guidRegex, 1, common.MaxStringLengthId, true, "")
			if err != nil {
				return err
			}
		case HostId:
			err := common.ValidateString(string(property), input.HostID, computeIdRegex, 1, common.MaxStringLengthId, true, "")
			if err != nil {
				return err
			}
		case AwsCredentials:
			if input.AccessKey == "" || input.SecretKey == "" {
				return common.NewGameLiftError(common.ValidationException, "", "Failed to provide a valid authorization strategy: AccessKey and SecretKey are required")
			}
		default:
			return common.NewGameLiftError(common.ValidationException, "", fmt.Sprintf("Unknown property %s", property))
		}
	}
	return nil
}

func ValidateProcessParameters(input ProcessParameters) error {
	if input.Port < common.PortMin || input.Port > common.PortMax {
		return common.NewGameLiftError(common.ValidationException, "", fmt.Sprintf("Port must be between %d and %d", common.PortMin, common.PortMax))
	}
	return nil
}

func ValidatePlayerSessionCreationPolicy(input model.PlayerSessionCreationPolicy) error {
	if input != model.AcceptAll && input != model.DenyAll {
		return common.NewGameLiftError(common.ValidationException, "", "Player session creation policy must be one of [ACCEPT_ALL, DENY_ALL]")
	}
	return nil
}

func ValidatePlayerSessionId(input string) error {
	return common.ValidateString("PlayerSessionID", input, guidRegex, 1, common.MaxStringLengthId, true, "")
}

func ValidateDescribePlayerSessionsRequest(input request.DescribePlayerSessionsRequest) error {
	numDefined := 0
	if input.GameSessionID != "" {
		numDefined++
	}
	if input.PlayerSessionID != "" {
		numDefined++
	}
	if input.PlayerID != "" {
		numDefined++
	}
	if numDefined != 1 {
		return common.NewGameLiftError(common.ValidationException, "", "Exactly one of GameSessionId, PlayerSessionId, or PlayerId is required")
	}
	err := common.ValidateString("GameSessionID", input.GameSessionID, arnRegex, 1, common.MaxStringLengthArn, false, "GameSessionID is invalid. Invalid ARN format.")
	if err != nil {
		return err
	}
	err = common.ValidateString("PlayerSessionID", input.PlayerSessionID, nil, 1, common.MaxStringLengthLong, false, "")
	if err != nil {
		return err
	}
	err = common.ValidateString("PlayerID", input.PlayerID, nil, 1, common.MaxStringLengthLong, false, "")
	if err != nil {
		return err
	}
	err = common.ValidateString("PlayerSessionStatusFilter", input.PlayerSessionStatusFilter, playerSessionStatusFilterRegex, 1, common.MaxStringLengthLong, false, "PlayerSessionStatusFilter must be one of [RESERVED, ACTIVE, COMPLETED, TIMEDOUT]")
	if err != nil {
		return err
	}
	return nil
}

func ValidateStartMatchBackfillRequest(input request.StartMatchBackfillRequest) error {
	err := common.ValidateString("GameSessionArn", input.GameSessionArn, arnRegex, 1, common.MaxStringLengthArn, true, "GameSessionArn is invalid. Invalid ARN format.")
	if err != nil {
		return err
	}
	err = common.ValidateString("MatchmakingConfigurationArn", input.MatchmakingConfigurationArn, gameLiftArnRegex, 1, common.MaxStringLengthArn, true, "MatchmakingConfigurationArn is invalid. Invalid GameLift ARN format.")
	if err != nil {
		return err
	}
	err = common.ValidateString("TicketID", input.TicketID, matchmakingIdRegex, 1, common.MaxStringLengthMatchmakingId, false, "")
	if err != nil {
		return err
	}
	if input.Players == nil || len(input.Players) == 0 {
		return common.NewGameLiftError(common.ValidationException, "", "Players cannot be empty.")
	}
	return nil
}

func ValidateStopMatchBackfillRequest(input request.StopMatchBackfillRequest) error {
	err := common.ValidateString("GameSessionArn", input.GameSessionArn, gameLiftArnRegex, 1, common.MaxStringLengthArn, true, "GameSessionArn is invalid. Invalid GameLift ARN format.")
	if err != nil {
		return err
	}
	err = common.ValidateString("MatchmakingConfigurationArn", input.MatchmakingConfigurationArn, gameLiftArnRegex, 1, common.MaxStringLengthNoLimit, true, "MatchmakingConfigurationArn is invalid. Invalid GameLift ARN format.")
	if err != nil {
		return err
	}
	err = common.ValidateString("TicketID", input.TicketID, matchmakingIdRegex, 1, common.MaxStringLengthMatchmakingId, true, "")
	if err != nil {
		return err
	}
	return nil
}

func ValidateGetFleetRoleCredentialsRequest(input request.GetFleetRoleCredentialsRequest) error {
	err := common.ValidateString("RoleArn", input.RoleArn, arnRegex, 1, common.MaxStringLengthArn, true, "RoleArn is invalid. Invalid ARN format.")
	if err != nil {
		return err
	}
	err = common.ValidateString("RoleSessionName", input.RoleSessionName, roleSessionNameRegex, 2, common.RoleSessionNameMaxLength, true, "")
	if err != nil {
		return err
	}
	return nil
}
