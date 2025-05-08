/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package response

import (
	"aws/amazon-gamelift-go-sdk/model/message"
	"aws/amazon-gamelift-go-sdk/model/result"
)

// GetComputeCertificateResponse - containing the path to your compute's TLS certificate and it's host name.
type GetComputeCertificateResponse struct {
	message.Message
	result.GetComputeCertificateResult
}
