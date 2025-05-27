/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package model

// Player - represents a player in matchmaking.
// When starting a matchmaking request, a player has a player ID, attributes, and may have latency data.
type Player struct {
	// A unique identifier for a player
	// Length Constraints: Minimum length of 1. Maximum length of 1024.
	PlayerID string `json:"PlayerId"`
	// Name of the team that the player is assigned to in a match. Team names are defined in a matchmaking rule set.
	// Length Constraints: Minimum length of 1. Maximum length of 1024.
	Team string `json:"Team"`
	// A collection of key:value pairs containing player information for use in matchmaking.
	// Player attribute keys must match the playerAttributes used in a matchmaking rule set.
	// Example: "PlayerAttributes": {"skill": {"N": "23"}, "gameMode": {"S": "deathmatch"}}.
	// You can provide up to 10 PlayerAttributes.
	// Type: String to AttributeValue object map
	// Key Length Constraints: Minimum length of 1. Maximum length of 1024.
	PlayerAttributes map[string]AttributeValue `json:"PlayerAttributes"`
	// A set of values, expressed in milliseconds, that indicates the amount of latency
	// that a player experiences when connected to @aws; Regions.
	// If this property is present, FlexMatch considers placing the match only in Regions for which latency is reported.
	// Type: String to integer map
	// Key Length Constraints: Minimum length of 1.
	// Valid Range: Minimum value of 1.
	LatencyInMS map[string]int `json:"LatencyInMs"`
}

// PlayerSession - details about the connection of a player to your game server.
//
// Sessions are created either for a specific game session,
// or as part of a game session placement or matchmaking request.
type PlayerSession struct {
	// A unique identifier for a player that is associated with this player session.
	// Length Constraints: Minimum length of 1. Maximum length of 1024.
	PlayerID string `json:"PlayerId"`
	// A unique identifier for a player session.
	PlayerSessionID string `json:"PlayerSessionId"`
	// A unique identifier for the game session that the player session is connected to.
	// Length Constraints: Minimum length of 1. Maximum length of 1024.
	GameSessionID string `json:"GameSessionId"`
	// A unique identifier for the fleet that the player's game session is running on.
	FleetID string `json:"FleetId"`
	// Developer-defined information related to a player. Amazon GameLift Servers does not use this data,
	// so it can be formatted as needed for use in the game.
	// Length Constraints: Minimum length of 1. Maximum length of 2048.
	PlayerData string `json:"PlayerData"`
	// The IP address of the game session. To connect to a game server,
	// an app needs both the IP address and port number.
	// Length Constraints: Minimum length of 1. Maximum length of 128.
	IPAddress string `json:"IpAddress"`
	// Port number for the game session. To connect to a game server,
	// an app needs both the IP address and port number.
	// Valid Range: Minimum value of 1. Maximum value of 60000.
	Port int `json:"Port"`
	// A time stamp indicating when this data object was created.
	// Format is a number expressed in Unix time as milliseconds (for example "1469498468.057").
	CreationTime int64 `json:"CreationTime"`
	// A time stamp indicating when this data object was terminated.
	// Format is a number expressed in Unix time as milliseconds (for example "1469498468.057").
	TerminationTime int64 `json:"TerminationTime"`
	// The DNS identifier assigned to the instance that is running the game session.
	DNSName string `json:"DnsName"`
	// Current status of the player session.
	// Possible player session statuses include the following:
	// RESERVED -- The player session request has been received, but the player has not yet connected
	// to the server process and/or been validated.
	// ACTIVE -- The player has been validated by the server process and is currently connected.
	// COMPLETED -- The player connection has been dropped.
	// TIMEDOUT -- A player session request was received, but the player did not connect
	// and/or was not validated within the timeout limit (60 seconds).
	// Valid Values: RESERVED | ACTIVE | COMPLETED | TIMEDOUT
	Status *playerSessionStatus `json:"Status,omitempty"`
}

func (p PlayerSession) WithStatus(status playerSessionStatus) PlayerSession {
	p.Status = &status
	return p
}

func (p PlayerSession) GetStatus() playerSessionStatus {
	return *p.Status
}
