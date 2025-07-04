/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package response

import (
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model/message"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model/result"
)

// DescribePlayerSessionsResponse - Represents the returned data in response to a request action.
type DescribePlayerSessionsResponse struct {
	message.Message
	result.DescribePlayerSessionsResult
}
