/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package request

import (
	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/model/message"
)

// GetComputeCertificateRequest - This request is sent to Amazon GameLift Servers WebSocket
// during a DescribePlayerSessionsRequest call.
//
// Please use NewGetComputeCertificate to create a new request.
type GetComputeCertificateRequest struct {
	message.Message
}

// NewGetComputeCertificate - creates a new GetComputeCertificateRequest
// generates a RequestID to match the request and response.
func NewGetComputeCertificate() GetComputeCertificateRequest {
	return GetComputeCertificateRequest{
		Message: message.NewMessage(message.GetComputeCertificate),
	}
}
