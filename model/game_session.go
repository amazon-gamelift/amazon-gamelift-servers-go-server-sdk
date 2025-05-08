/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package model

type GameSession struct {
	// A unique identifier for the game session.
	// Length Constraints: Minimum length of 1. Maximum length of 1024.
	GameSessionID string `json:"GameSessionId"`
	// A set of custom game session properties, formatted as a single string value.
	// This data is passed to a game server process in the GameSession object with a request to start a new game session.
	// Length Constraints: Minimum length of 1. Maximum length of 262144.
	GameSessionData string `json:"GameSessionData"`
	// A descriptive label that is associated with a game session. Session names do not need to be unique.
	// Length Constraints: Minimum length of 1. Maximum length of 1024.
	Name string `json:"Name"`
	// Information about the matchmaking process that was used to create the game session. It is in JSON syntax,
	// formatted as a string. In addition the matchmaking configuration used, it contains data on all players assigned
	// to the match, including player attributes and team assignments.
	// Matchmaker data is useful when requesting match backfills, and is updated whenever new players are added
	// during a successful backfill.
	// Length Constraints: Minimum length of 1. Maximum length of 390000.
	MatchmakerData string `json:"MatchmakerData"`
	// A unique identifier for the fleet that the game session is running on.
	FleetID string `json:"FleetId"`
	// The fleet location where the game session is running. This value might specify the fleet's home Region or a remote location.
	// Location is expressed as an AWS Region code such as us-west-2.
	Location string `json:"Location"`
	// The maximum number of players that can be connected simultaneously to the game session.
	// Valid Range: Minimum value of 0.
	MaximumPlayerSessionCount int `json:"MaximumPlayerSessionCount"`
	// The IP address of the game session. To connect to a GameLift game server, an app needs both the IP address and port number
	// Length Constraints: Minimum length of 1. Maximum length of 128.
	IPAddress string `json:"IpAddress"`
	// The port number for the game session. To connect to a GameLift game server, an app needs both the IP address and port number.
	// Valid Range: Minimum value of 1. Maximum value of 60000.
	Port int `json:"Port"`
	// The DNS identifier assigned to the instance that is running the game session. Values have the following format:
	DNSName string `json:"DnsName"`
	// A set of custom properties for a game session, formatted as key:value pairs. These properties are
	// passed to a game server process in the GameSession object with a request to start a new game session.
	// Array Members: Maximum number of 16 items.
	GameProperties map[string]string `json:"GameProperties"`
	// Current status of the game session. A game session must have an ACTIVE status to have player sessions.
	// Valid Values: ACTIVE | ACTIVATING | TERMINATED | TERMINATING | ERROR
	Status *gameSessionStatus `json:"Status,omitempty"`
	// Provides additional information about game session status. INTERRUPTED indicates that the game session
	// was hosted on a spot instance that was reclaimed, causing the active game session to be terminated.
	StatusReason string `json:"StatusReason"`
}

func (g GameSession) WithStatus(status gameSessionStatus) GameSession {
	g.Status = &status
	return g
}

func (g GameSession) GetStatus() gameSessionStatus {
	return *g.Status
}
