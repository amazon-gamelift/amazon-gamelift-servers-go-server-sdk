/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package response

import (
	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/model/message"
	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/model/result"
)

// DescribePlayerSessionsResponse - Represents the returned data in response to a request action.
type DescribePlayerSessionsResponse struct {
	message.Message
	result.DescribePlayerSessionsResult
}
