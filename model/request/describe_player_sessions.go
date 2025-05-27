/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package request

import (
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model/message"
)

// DescribePlayerSessionsRequest - This request is sent to Amazon GameLift Servers WebSocket
// during a DescribePlayerSessionsRequest call.
//
// Please use NewDescribePlayerSessions function to create this request.
type DescribePlayerSessionsRequest struct {
	message.Message
	// Unique identifier for the game session to get player sessions for.
	// Maximum length: 256
	GameSessionID string `json:"GameSessionId,omitempty"`
	// A unique identifier for a player to retrieve player sessions for.
	// Maximum length: 1024
	PlayerID string `json:"PlayerId,omitempty"`
	// A unique identifier for a player session to retrieve.
	PlayerSessionID string `json:"PlayerSessionId,omitempty"`
	// Player session status to filter results on
	// Maximum length: 1024
	// Possible player session statuses include the following:
	//  - RESERVED - The player session request has been received, but the player has not yet connected to the server process and/or been validated.
	//  - ACTIVE - The player has been validated by the server process and is currently connected.
	//  - COMPLETED - The player connection has been dropped.
	//  - TIMEDOUT - A player session request was received,
	// 		but the player did not connect and/or was not validated within the time-out limit (60 seconds).
	PlayerSessionStatusFilter string `json:"PlayerSessionStatusFilter,omitempty"`
	// Indicating the start of the next sequential page of results.
	// Use the token that is returned with a previous call to this action.
	// To specify the start of the result set, do not specify a value.
	// If a player session ID is specified, this parameter is ignored.
	// Maximum length: 1024
	NextToken string `json:"NextToken,omitempty"`
	// Maximum number of results to return.
	// Use this parameter with NextToken to get results as a set of sequential pages.
	// If a player session ID is specified, this parameter is ignored.
	// Valid Range: Minimum value of 1
	Limit int `json:"Limit,omitempty"`
}

// NewDescribePlayerSessions - creates a new DescribePlayerSessionsRequest
// generates a RequestID to match the request and response.
func NewDescribePlayerSessions() DescribePlayerSessionsRequest {
	return DescribePlayerSessionsRequest{
		Message: message.NewMessage(message.DescribePlayerSessions),
	}
}
