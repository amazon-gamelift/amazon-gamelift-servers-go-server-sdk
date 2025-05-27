/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package response

import (
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model/message"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/model/result"
)

// StartMatchBackfillResponse - is successful response of StartMatchBackfill action.
type StartMatchBackfillResponse struct {
	message.Message
	result.StartMatchBackfillResult
}
