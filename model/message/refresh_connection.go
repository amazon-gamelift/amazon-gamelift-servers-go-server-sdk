/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package message

// RefreshConnectionMessage - Message from GameLift indicating the SDK should refresh its websocket connection.
type RefreshConnectionMessage struct {
	Message
	RefreshConnectionEndpoint string `json:"RefreshConnectionEndpoint"`
	AuthToken                 string `json:"AuthToken"`
}
