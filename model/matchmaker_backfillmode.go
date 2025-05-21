/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package model

import "strconv"

// backfillMode
type backfillMode int

// Possible backfill modes
const (
	BackFillModeNotSet backfillMode = iota
	BackFillModeManual
	BackFillModeAutomatic
)

var backfillModeStrs = []string{"NOT_SET", "MANUAL", "AUTOMATIC"}

func toBackfillMode(s string) backfillMode {
	for i := range backfillModeStrs {
		if backfillModeStrs[i] == s {
			return backfillMode(i)
		}
	}
	return BackFillModeNotSet
}

func (bm backfillMode) String() string {
	n := int(bm)
	if n >= len(backfillModeStrs) {
		n = 0
	}
	return backfillModeStrs[n]
}

func (bm *backfillMode) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(bm.String())), nil
}

func (bm *backfillMode) UnmarshalJSON(data []byte) error {
	origin, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	*bm = toBackfillMode(origin)
	return nil
}
