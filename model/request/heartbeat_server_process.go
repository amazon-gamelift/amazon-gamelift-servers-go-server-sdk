/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package request

import "aws/amazon-gamelift-go-sdk/model/message"

// HeartbeatServerProcessRequest - Message from GameLift or Customer reporting process health.
//
// Please use NewHeartbeatServerProcess to create a new request.
type HeartbeatServerProcessRequest struct {
	message.Message
	// The boolean value to identify the health check to HEALTHY.
	HealthStatus bool `json:"HealthStatus,omitempty"`
}

// NewHeartbeatServerProcess - creates a new HeartbeatServerProcessRequest
// generates a RequestID to match the request and response.
func NewHeartbeatServerProcess(status bool) HeartbeatServerProcessRequest {
	return HeartbeatServerProcessRequest{
		Message:      message.NewMessage(message.HeartbeatServerProcess),
		HealthStatus: status,
	}
}
