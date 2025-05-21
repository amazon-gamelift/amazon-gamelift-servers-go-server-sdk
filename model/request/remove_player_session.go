/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package request

import (
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model/message"
)

// RemovePlayerSessionRequest - Need it for action RemovePlayerSession in requirement.
//
// Please use NewRemovePlayerSession to create a new request.
type RemovePlayerSessionRequest struct {
	message.Message
	// Unique identifier for the game session to get player sessions for.
	// Maximum length: 256
	GameSessionID string `json:"GameSessionId,omitempty"`
	// A unique identifier for a player session to retrieve.
	PlayerSessionID string `json:"PlayerSessionId,omitempty"`
}

// NewRemovePlayerSession - creates a new RemovePlayerSessionRequest
// generates a RequestID to match the request and response.
func NewRemovePlayerSession(gameSessionID, playerSessionID string) RemovePlayerSessionRequest {
	return RemovePlayerSessionRequest{
		Message:         message.NewMessage(message.RemovePlayerSession),
		GameSessionID:   gameSessionID,
		PlayerSessionID: playerSessionID,
	}
}
