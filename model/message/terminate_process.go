/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package message

// TerminateProcessMessage - Message from Amazon GameLift Servers that the current server process has been terminated.
type TerminateProcessMessage struct {
	Message
	// TerminationTime is milliseconds that have elapsed since Unix epoch time begins (00:00:00 UTC Jan 1 1970).
	TerminationTime int64 `json:"TerminationTime"`
}
