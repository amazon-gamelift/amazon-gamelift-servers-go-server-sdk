/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package message

import (
	"github.com/google/uuid"
)

//nolint:revive // Name specified by reference implementation
type MessageAction string

// Possible messages types
const (
	AcceptPlayerSession               MessageAction = "AcceptPlayerSession"
	ActivateGameSession               MessageAction = "ActivateGameSession"
	TerminateServerProcess            MessageAction = "TerminateServerProcess"
	ActivateServerProcess             MessageAction = "ActivateServerProcess"
	UpdatePlayerSessionCreationPolicy MessageAction = "UpdatePlayerSessionCreationPolicy"
	CreateGameSession                 MessageAction = "CreateGameSession"
	UpdateGameSession                 MessageAction = "UpdateGameSession"
	StartMatchBackfill                MessageAction = "StartMatchBackfill"
	TerminateProcess                  MessageAction = "TerminateProcess"
	DescribePlayerSessions            MessageAction = "DescribePlayerSessions"
	StopMatchBackfill                 MessageAction = "StopMatchBackfill"
	HeartbeatServerProcess            MessageAction = "HeartbeatServerProcess"
	GetComputeCertificate             MessageAction = "GetComputeCertificate"
	GetFleetRoleCredentials           MessageAction = "GetFleetRoleCredentials"
	RefreshConnection                 MessageAction = "RefreshConnection"
	RemovePlayerSession               MessageAction = "RemovePlayerSession"
)

type Message struct {
	Action MessageAction `json:"Action"`
	// The ID of the request
	RequestID string `json:"RequestId"`
}

// GetMessage allows types that embed Message to implement the MessageGetter interface
func (m Message) GetMessage() Message {
	return m
}

type ResponseMessage struct {
	Message

	StatusCode   int    `json:"StatusCode"`
	ErrorMessage string `json:"ErrorMessage"`
}

// NewMessage retrieves a new generated Message with RequestID filled with UUID random string.
func NewMessage(action MessageAction) Message {
	return Message{
		Action:    action,
		RequestID: uuid.New().String(),
	}
}
