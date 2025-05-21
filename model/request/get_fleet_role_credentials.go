/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package request

import (
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model/message"
)

// GetFleetRoleCredentialsRequest - Request to Amazon GameLift Servers to get credentials for the fleet.
//
// Please use NewGetFleetRoleCredentials to create a new request.
type GetFleetRoleCredentialsRequest struct {
	message.Message
	// The Amazon Resource Name (ARN) of the role to assume.
	// Length Constraints: Minimum length of 20. Maximum length of 2048.
	RoleArn string `json:"RoleArn,omitempty"`
	// An identifier for the assumed role session.
	// Length Constraints: Minimum length of 2. Maximum length of 64.
	RoleSessionName string `json:"RoleSessionName,omitempty"`
}

// NewGetFleetRoleCredentials - creates a new GetFleetRoleCredentialsRequest
// generates a RequestID to match the request and response.
func NewGetFleetRoleCredentials() GetFleetRoleCredentialsRequest {
	return GetFleetRoleCredentialsRequest{
		Message: message.NewMessage(message.GetFleetRoleCredentials),
	}
}
