/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUpdateReason_MarshalJSON(t *testing.T) {
	cases := map[UpdateReason]string{
		Unknown:                "\"UNKNOWN\"",
		MatchmakingDataUpdated: "\"MATCHMAKING_DATA_UPDATED\"",
		BackfillFailed:         "\"BACKFILL_FAILED\"",
		BackfillTimedOut:       "\"BACKFILL_TIMED_OUT\"",
		BackfillCancelled:      "\"BACKFILL_CANCELLED\"",
	}

	for origin, expected := range cases {
		data, err := json.Marshal(&origin)
		if err != nil {
			t.Errorf("json marshal UpdateReason error: %s", err.Error())
			return
		}
		if !strings.EqualFold(expected, string(data)) {
			t.Errorf("expect %s but get %s", expected, data)
			return
		}
	}
}

func TestUpdateReason_UnmarshalJSON(t *testing.T) {
	cases := map[UpdateReason]string{
		Unknown:                "\"UNKNOWN\"",
		MatchmakingDataUpdated: "\"MATCHMAKING_DATA_UPDATED\"",
		BackfillFailed:         "\"BACKFILL_FAILED\"",
		BackfillTimedOut:       "\"BACKFILL_TIMED_OUT\"",
		BackfillCancelled:      "\"BACKFILL_CANCELLED\"",
	}

	for expected, origin := range cases {
		var reason UpdateReason
		if err := json.Unmarshal([]byte(origin), &reason); err != nil {
			t.Errorf("json unmarshal UpdateReason error: %s", err.Error())
			return
		}
		if expected != reason {
			t.Errorf("expect %v but get %v", expected, reason)
		}
	}
}
