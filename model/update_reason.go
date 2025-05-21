/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package model

import "strconv"

type UpdateReason int

// Possible reason for updating the game session
const (
	Unknown UpdateReason = iota
	MatchmakingDataUpdated
	BackfillFailed
	BackfillTimedOut
	BackfillCancelled
)

var updateReasonStrs = []string{
	"UNKNOWN",
	"MATCHMAKING_DATA_UPDATED",
	"BACKFILL_FAILED",
	"BACKFILL_TIMED_OUT",
	"BACKFILL_CANCELLED",
}

func (u *UpdateReason) String() string {
	n := int(*u)
	if n >= len(updateReasonStrs) {
		n = 0
	}
	return updateReasonStrs[n]
}

func (u *UpdateReason) ToUpdateReason(s string) {
	for i := range updateReasonStrs {
		if updateReasonStrs[i] == s {
			*u = UpdateReason(i)
			return
		}
	}
	*u = Unknown
}

func (u *UpdateReason) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(u.String())), nil
}

func (u *UpdateReason) UnmarshalJSON(data []byte) error {
	origin, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	u.ToUpdateReason(origin)
	return nil
}
