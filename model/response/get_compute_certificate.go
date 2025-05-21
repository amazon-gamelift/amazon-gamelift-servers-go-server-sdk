/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package response

import (
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model/message"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model/result"
)

// GetComputeCertificateResponse - containing the path to your compute's TLS certificate and it's host name.
type GetComputeCertificateResponse struct {
	message.Message
	result.GetComputeCertificateResult
}
