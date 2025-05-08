/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package message

import (
	"aws/amazon-gamelift-go-sdk/model"
)

// UpdateGameSessionMessage - Message from GameLift initializing GameSession Update
type UpdateGameSessionMessage struct {
	Message
	// The UpdateGameSession object
	model.UpdateGameSession
}
