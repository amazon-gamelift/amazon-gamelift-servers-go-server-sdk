/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package request

import "github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/model/message"

// ActivateGameSessionRequest - This request is sent to Amazon GameLift Servers WebSocket during ActivateGameSessionRequest.
//
// Please use NewActivateGameSession function to create this request.
type ActivateGameSessionRequest struct {
	message.Message
	// A unique identifier for the game session that the player session is connected to.
	// Length Constraints: Minimum length of 1. Maximum length of 1024.
	GameSessionID string `json:"GameSessionId,omitempty"`
}

// NewActivateGameSession - creates a new ActivateGameSessionRequest
// generates a RequestID to match the request and response.
func NewActivateGameSession(gameSessionID string) ActivateGameSessionRequest {
	return ActivateGameSessionRequest{
		Message:       message.NewMessage(message.ActivateGameSession),
		GameSessionID: gameSessionID,
	}
}
