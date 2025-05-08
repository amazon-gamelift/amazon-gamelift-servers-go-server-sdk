/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package model

import "strconv"

// playerSessionStatus - unexported (private) data type, has a several predefined values see below
type playerSessionStatus int

// Possible player session statuses
const (
	PlayerNotSet playerSessionStatus = iota
	PlayerReserved
	PlayerActive
	PlayerCompleted
	PlayerTimedout
)

var playerSessionStatusStrs = []string{"NOT_SET", "RESERVED", "ACTIVE", "COMPLETED", "TIMEDOUT"}

func (p *playerSessionStatus) String() string {
	n := int(*p)
	if n >= len(playerSessionStatusStrs) {
		n = 0
	}
	return playerSessionStatusStrs[n]
}

func (p *playerSessionStatus) toPlayerSessionStatus(s string) {
	for i := range playerSessionStatusStrs {
		if playerSessionStatusStrs[i] == s {
			*p = playerSessionStatus(i)
			return
		}
	}
	*p = PlayerNotSet
}

func (p *playerSessionStatus) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(p.String())), nil
}

func (p *playerSessionStatus) UnmarshalJSON(data []byte) error {
	origin, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	p.toPlayerSessionStatus(origin)
	return nil
}
