/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package response

import (
	"aws/amazon-gamelift-go-sdk/model/message"
	"aws/amazon-gamelift-go-sdk/model/result"
)

// StartMatchBackfillResponse - is successful response of StartMatchBackfill action.
type StartMatchBackfillResponse struct {
	message.Message
	result.StartMatchBackfillResult
}
