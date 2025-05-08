/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package request

import (
	"aws/amazon-gamelift-go-sdk/model/message"
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
