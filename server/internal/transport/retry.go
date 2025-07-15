/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package transport

import (
	"time"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server/log"
)

type retryTransport struct {
	ITransport
	log      log.ILogger
	attempt  int
	factor   int
	interval time.Duration
}

// WithRetry wraps the specified transport by adding a retry mechanism to the Write method.
func WithRetry(next ITransport, l log.ILogger) ITransport {
	return &retryTransport{
		ITransport: next,
		log:        l,
		factor:     common.GetEnvIntOrDefault(common.RetryFactor, common.RetryFactorDefault, l),
		attempt:    common.GetEnvIntOrDefault(common.MaxRetry, common.MaxRetryDefault, l),
		interval:   common.GetEnvDurationOrDefault(common.RetryInterval, common.RetryIntervalDefault, l),
	}
}

func (r *retryTransport) Write(data []byte) error {
	for i := 0; i < r.attempt; i++ {
		err := r.ITransport.Write(data)
		if err == nil {
			return nil
		}
		r.log.Debugf("Call Failed: %s. Retrying attempt: %d of %d", err.Error(), i+1, r.attempt)
		time.Sleep(time.Duration((i+1)*r.factor) * r.interval)
	}

	return common.NewGameLiftError(
		common.WebsocketRetriableSendMessageFailure,
		"Failed write retry",
		"write attempt overflow",
	)
}
