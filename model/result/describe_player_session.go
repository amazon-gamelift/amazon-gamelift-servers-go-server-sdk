/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package result

import (
	"fmt"

	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/model"
)

// DescribePlayerSessionsResult - see https://docs.aws.amazon.com/gamelift/latest/apireference/API_DescribePlayerSessions.html.
// This data type describes player session retrieved by. It can
// contain information about any of the following:
//   - A specific player sessions
//   - All players session in a specific game
//   - All player sessions for a specific player
//
// For large collections of player sessions, use the pagination parameters to retrieve results as sequential pages.
type DescribePlayerSessionsResult struct {
	// Token indicating where to resume retrieving results on the next call to this
	// action. If no token is returned, these results represent the end of the list.
	NextToken string `json:"NextToken"`
	// PlayerSessions - Collection of objects containing properties for each player session that matches the request.
	PlayerSessions []model.PlayerSession `json:"PlayerSessions"`
}

func (d *DescribePlayerSessionsResult) AddPlayerSession(value *model.PlayerSession) error {
	if value == nil {
		return fmt.Errorf("player session can not be nil")
	}
	if len(d.PlayerSessions) >= common.MaxPlayerSessions {
		return fmt.Errorf("PlayerSessions count is greater than or equal to max player sessions %d",
			common.MaxPlayerSessions,
		)
	}
	d.PlayerSessions = append(d.PlayerSessions, *value)
	return nil
}
