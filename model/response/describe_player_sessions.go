/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package response

import (
	"aws/amazon-gamelift-go-sdk/model/message"
	"aws/amazon-gamelift-go-sdk/model/result"
)

// DescribePlayerSessionsResponse - Represents the returned data in response to a request action.
type DescribePlayerSessionsResponse struct {
	message.Message
	result.DescribePlayerSessionsResult
}
