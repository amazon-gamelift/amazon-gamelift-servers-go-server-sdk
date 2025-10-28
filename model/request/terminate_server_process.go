/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package request

import (
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/model/message"
)

// TerminateServerProcessRequest -
//
// Please use NewTerminateServerProcess to create a new request.
type TerminateServerProcessRequest struct {
	message.Message
}

// NewTerminateServerProcess - creates a new TerminateServerProcessRequest
// generates a RequestID to match the request and response.
func NewTerminateServerProcess() TerminateServerProcessRequest {
	return TerminateServerProcessRequest{
		Message: message.NewMessage(message.TerminateServerProcess),
	}
}
