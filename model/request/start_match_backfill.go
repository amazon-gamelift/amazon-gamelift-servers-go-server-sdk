/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package request

import (
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/model"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/model/message"
)

// StartMatchBackfillRequest - This request is sent to Amazon GameLift Servers WebSocket during a DescribePlayerSessionsRequest call.
//
// Please use NewStartMatchBackfill to create a new request.
type StartMatchBackfillRequest struct {
	message.Message
	// A unique identifier for the game session. Use the game session ID.
	// When using FlexMatch as a standalone matchmaking solution, this parameter is not needed.
	// Length Constraints: Minimum length of 1. Maximum length of 256.
	GameSessionArn string `json:"GameSessionArn,omitempty"`
	// The Amazon Resource Name (ARN) associated with the Amazon GameLift Servers FlexMatch matchmaking configuration resource
	// that is used with this ticket.
	// Pattern: ^arn:.*:matchmakingconfiguration\/[a-zA-Z0-9-\.]*
	MatchmakingConfigurationArn string `json:"MatchmakingConfigurationArn,omitempty"`
	// A unique identifier for a matchmaking ticket. If no ticket ID is specified here,
	// Amazon GameLift Servers will generate one in the form of a UUID.
	// Use this identifier to track the match backfill ticket status and retrieve match results.
	// Length Constraints: Maximum length of 128.
	TicketID string `json:"TicketId"`
	// Match information on all players that are currently assigned to the game session.
	// This information is used by the matchmaker to find new players and add them to the existing game.
	// You can include up to 10 Players in a StartMatchBackfill request.
	Players []model.Player `json:"Players"`
}

// NewStartMatchBackfill - creates a new StartMatchBackfillRequest
// generates a RequestID to match the request and response.
func NewStartMatchBackfill(gameSessionArn, matchmakingConfigurationArn string, players []model.Player) StartMatchBackfillRequest {
	return StartMatchBackfillRequest{
		Message:                     message.NewMessage(message.StartMatchBackfill),
		GameSessionArn:              gameSessionArn,
		MatchmakingConfigurationArn: matchmakingConfigurationArn,
		Players:                     players,
	}
}
