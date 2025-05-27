/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package model

// UpdateGameSession - updates the mutable properties of a game session.
type UpdateGameSession struct {
	// Unique identifier for a matchmaking or match backfill request ticket.
	BackfillTicketID string `json:"BackfillTicketId"`
	// The  updated game session object.
	GameSession GameSession `json:"GameSession"`
	// The reason this update is being supplied.
	UpdateReason *UpdateReason `json:"UpdateReason,omitempty"`
}

func (u UpdateGameSession) WithReason(reason UpdateReason) UpdateGameSession {
	u.UpdateReason = &reason
	return u
}

func (u UpdateGameSession) GetReason() UpdateReason {
	return *u.UpdateReason
}
