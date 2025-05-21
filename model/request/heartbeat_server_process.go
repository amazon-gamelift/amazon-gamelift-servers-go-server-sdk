/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package request

import "github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model/message"

// HeartbeatServerProcessRequest - Message from Amazon GameLift Servers or Customer reporting process health.
//
// Please use NewHeartbeatServerProcess to create a new request.
type HeartbeatServerProcessRequest struct {
	message.Message
	// The boolean value to identify the health check to HEALTHY.
	HealthStatus bool `json:"HealthStatus"`
}

// NewHeartbeatServerProcess - creates a new HeartbeatServerProcessRequest
// generates a RequestID to match the request and response.
func NewHeartbeatServerProcess(status bool) HeartbeatServerProcessRequest {
	return HeartbeatServerProcessRequest{
		Message:      message.NewMessage(message.HeartbeatServerProcess),
		HealthStatus: status,
	}
}
