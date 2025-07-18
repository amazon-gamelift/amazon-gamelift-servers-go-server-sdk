/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package request

import (
	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/model/message"
)

// StopMatchBackfillRequest
//
// Please use NewStopMatchBackfill to create a new request.
type StopMatchBackfillRequest struct {
	message.Message
	// A unique identifier for the game session. Use the game session ID.
	// When using FlexMatch as a standalone matchmaking solution, this parameter is not needed.
	// Length Constraints: Minimum length of 1. Maximum length of 256.
	GameSessionArn string `json:"GameSessionArn,omitempty"`
	// The Amazon Resource Name (ARN) associated with the Amazon GameLift Servers FlexMatch matchmaking configuration resource
	// that is used with this ticket.
	// Pattern: ^arn:.*:matchmakingconfiguration\/[a-zA-Z0-9-\.]*
	MatchmakingConfigurationArn string `json:"MatchmakingConfigurationArn,omitempty"`
	// A unique identifier for a matchmaking ticket.
	// Length Constraints: Maximum length of 128.
	TicketID string `json:"TicketId,omitempty"`
}

// NewStopMatchBackfill - creates a new StopMatchBackfillRequest
// generates a RequestID to match the request and response.
func NewStopMatchBackfill() StopMatchBackfillRequest {
	return StopMatchBackfillRequest{
		Message: message.NewMessage(message.StopMatchBackfill),
	}
}
