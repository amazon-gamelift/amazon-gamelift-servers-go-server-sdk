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

func TestPlayerSessionStatus_MarshalJSON(t *testing.T) {
	cases := map[playerSessionStatus]string{
		PlayerActive:    "\"ACTIVE\"",
		PlayerCompleted: "\"COMPLETED\"",
		PlayerNotSet:    "\"NOT_SET\"",
		PlayerReserved:  "\"RESERVED\"",
		PlayerTimedout:  "\"TIMEDOUT\"",
	}

	for key := range cases {
		val, err := json.Marshal(&key)
		if err != nil {
			t.Errorf("json marshal playerSessionStatus error: %s", err.Error())
			return
		}
		if !strings.EqualFold(cases[key], string(val)) {
			t.Errorf("expect %s but get %s", cases[key], val)
			return
		}
	}
}

func TestPlayerSessionStatus_UnmarshalJSON(t *testing.T) {
	cases := map[playerSessionStatus]string{
		PlayerActive:    "\"ACTIVE\"",
		PlayerCompleted: "\"COMPLETED\"",
		PlayerNotSet:    "\"NOT_SET\"",
		PlayerReserved:  "\"RESERVED\"",
		PlayerTimedout:  "\"TIMEDOUT\"",
	}

	for key, v := range cases {
		var val playerSessionStatus
		if err := json.Unmarshal([]byte(v), &val); err != nil {
			t.Errorf("json unmarshal playerSessionStatus error: %s", err.Error())
		}
		if val != key {
			t.Errorf("failed ")
		}
	}
}
