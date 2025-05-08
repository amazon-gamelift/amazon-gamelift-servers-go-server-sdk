/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package model

import "strconv"

// gameSessionStatus - unexported (private) data type, has a several predefined values see below
type gameSessionStatus int

// Possible game session statuses
const (
	GameNotSet gameSessionStatus = iota
	GameActive
	GameActivating
	GameTerminated
	GameTerminating
)

var gameSessionStatusStrs = []string{"NOT_SET", "ACTIVE", "ACTIVATING", "TERMINATED", "TERMINATING"}

func (g *gameSessionStatus) toGameSessionStatus(s string) {
	for i := range gameSessionStatusStrs {
		if gameSessionStatusStrs[i] == s {
			*g = gameSessionStatus(i)
			return
		}
	}
	*g = GameNotSet
}

func (g *gameSessionStatus) String() string {
	n := int(*g)
	if n >= len(gameSessionStatusStrs) {
		n = 0
	}
	return gameSessionStatusStrs[n]
}

func (g *gameSessionStatus) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(g.String())), nil
}

func (g *gameSessionStatus) UnmarshalJSON(data []byte) error {
	origin, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	g.toGameSessionStatus(origin)
	return nil
}
