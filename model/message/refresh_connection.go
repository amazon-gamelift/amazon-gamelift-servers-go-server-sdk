/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package message

// RefreshConnectionMessage - Message from Amazon GameLift Servers indicating the server SDK should refresh its WebSocket connection.
type RefreshConnectionMessage struct {
	Message
	RefreshConnectionEndpoint string `json:"RefreshConnectionEndpoint"`
	AuthToken                 string `json:"AuthToken"`
}
