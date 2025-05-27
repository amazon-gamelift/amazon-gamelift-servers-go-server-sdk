/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package result

// StartMatchBackfillResult - is successful result of StartMatchBackfill action.
type StartMatchBackfillResult struct {
	// A unique identifier for a matchmaking ticket.
	// Length Constraints: Maximum length of 128.
	TicketID string `json:"TicketId"`
}
