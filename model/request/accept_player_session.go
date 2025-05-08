/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package request

import (
	"aws/amazon-gamelift-go-sdk/model/message"
)

// AcceptPlayerSessionRequest - This request contains details about the player's session,
// and sent to the Gamelift to accept player to the gamesession.
//
// Please use NewAcceptPlayerSession function to create this request.
type AcceptPlayerSessionRequest struct {
	message.Message
	// A unique identifier for the game session that the player session is connected to.
	// Length Constraints: Minimum length of 1. Maximum length of 1024.
	GameSessionID string `json:"GameSessionId,omitempty"`
	// A unique identifier for a player session.
	PlayerSessionID string `json:"PlayerSessionId,omitempty"`
}

// NewAcceptPlayerSession - creates a new AcceptPlayerSessionRequest
// generates a RequestID to match the request and response.
func NewAcceptPlayerSession(gameSessionID, playerSessionID string) AcceptPlayerSessionRequest {
	return AcceptPlayerSessionRequest{
		Message:         message.NewMessage(message.AcceptPlayerSession),
		GameSessionID:   gameSessionID,
		PlayerSessionID: playerSessionID,
	}
}
