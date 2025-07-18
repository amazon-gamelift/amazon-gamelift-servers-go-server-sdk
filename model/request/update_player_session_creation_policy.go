/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package request

import (
	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/model"
	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/model/message"
)

// UpdatePlayerSessionCreationPolicyRequest
//
// Please use NewUpdatePlayerSessionCreationPolicy to create a new request.
type UpdatePlayerSessionCreationPolicyRequest struct {
	message.Message
	// A unique identifier for the game session to update.
	// Length Constraints: Minimum length of 1. Maximum length of 256.
	GameSessionID string `json:"GameSessionId,omitempty"`
	// A policy that determines whether the game session is accepting new players.
	// Valid Values: model.AcceptAll | model.DenyAll
	PlayerSessionPolicy *model.PlayerSessionCreationPolicy `json:"PlayerSessionPolicy,omitempty"`
}

// NewUpdatePlayerSessionCreationPolicy - creates a new UpdatePlayerSessionCreationPolicyRequest
// generates a RequestID to match the request and response.
func NewUpdatePlayerSessionCreationPolicy(
	gameSessionID string,
	policy model.PlayerSessionCreationPolicy,
) UpdatePlayerSessionCreationPolicyRequest {
	return UpdatePlayerSessionCreationPolicyRequest{
		Message:             message.NewMessage(message.UpdatePlayerSessionCreationPolicy),
		GameSessionID:       gameSessionID,
		PlayerSessionPolicy: &policy,
	}
}
