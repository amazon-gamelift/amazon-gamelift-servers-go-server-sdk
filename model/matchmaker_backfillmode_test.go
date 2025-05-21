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

// GIVEN backfill mode values WHEN converting to json/string THEN no error
func TestMatchmaker_BackfillMode_MarshalJSON(t *testing.T) {
	cases := map[backfillMode]string{
		BackFillModeAutomatic: "\"AUTOMATIC\"",
		BackFillModeNotSet:    "\"NOT_SET\"",
		BackFillModeManual:    "\"MANUAL\"",
	}

	for key := range cases {
		val, err := json.Marshal(&key)
		if err != nil {
			t.Errorf("json marshal matchmaker backfill mode error: %s", err.Error())
			return
		}
		if !strings.EqualFold(cases[key], string(val)) {
			t.Errorf("expect %s but get %s", cases[key], val)
			return
		}
	}
}

// GIVEN backfill mode strings WHEN converting to enum/int type THEN no error
func TestMatchmaker_BackfillMode_UnmarshalJSON(t *testing.T) {
	cases := map[backfillMode]string{
		BackFillModeAutomatic: "\"AUTOMATIC\"",
		BackFillModeNotSet:    "\"NOT_SET\"",
		BackFillModeManual:    "\"MANUAL\"",
	}

	for key, v := range cases {
		var val backfillMode
		if err := json.Unmarshal([]byte(v), &val); err != nil {
			t.Errorf("json unmarshal matchmaker backfill mode error: %s", err.Error())
		}
		if val != key {
			t.Errorf("failed ")
		}
	}
}
