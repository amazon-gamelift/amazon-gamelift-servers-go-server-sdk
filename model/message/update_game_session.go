/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package message

import "github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/model"

// UpdateGameSessionMessage - Message from Amazon GameLift Servers after the GameSession Update
type UpdateGameSessionMessage struct {
	Message
	// The UpdateGameSession object
	model.UpdateGameSession
}
