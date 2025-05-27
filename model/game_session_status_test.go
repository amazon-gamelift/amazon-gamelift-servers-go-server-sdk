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

func TestGameSessionStatus_MarshalJSON(t *testing.T) {
	cases := map[gameSessionStatus]string{
		GameActive:      "\"ACTIVE\"",
		GameActivating:  "\"ACTIVATING\"",
		GameTerminated:  "\"TERMINATED\"",
		GameTerminating: "\"TERMINATING\"",
		GameNotSet:      "\"NOT_SET\"",
	}

	for key := range cases {
		val, err := json.Marshal(&key)
		if err != nil {
			t.Errorf("json marshal gameSessionStatus error: %s", err.Error())
			return
		}
		if !strings.EqualFold(cases[key], string(val)) {
			t.Errorf("expect %s but get %s", cases[key], val)
			return
		}
	}
}

func TestGameSessionStatus_UnmarshalJSON(t *testing.T) {
	cases := map[gameSessionStatus]string{
		GameActive:      "\"ACTIVE\"",
		GameActivating:  "\"ACTIVATING\"",
		GameTerminated:  "\"TERMINATED\"",
		GameTerminating: "\"TERMINATING\"",
		GameNotSet:      "\"NOT_SET\"",
	}

	for key, v := range cases {
		var val gameSessionStatus
		if err := json.Unmarshal([]byte(v), &val); err != nil {
			t.Errorf("json unmarshal gameSessionStatus error: %s", err.Error())
		}
		if val != key {
			t.Errorf("failed ")
		}
	}
}
