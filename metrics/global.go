/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package metrics

import (
	"context"
	"sync"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/model"
)

var (
	//nolint:gochecknoglobals
	globalProcessor *Processor
	//nolint:gochecknoglobals
	globalMutex sync.RWMutex
	//nolint:gochecknoglobals
	once sync.Once
)

// SetGlobalProcessor sets the global metrics processor instance.
// This is called automatically when the first processor is built.
// Only the first call will actually set the processor.
func SetGlobalProcessor(processor *Processor) {
	globalMutex.Lock()
	defer globalMutex.Unlock()
	if globalProcessor == nil {
		globalProcessor = processor
	}
}

// GetGlobalProcessor returns the global metrics processor instance.
// Returns nil if no processor has been initialized yet.
func GetGlobalProcessor() *Processor {
	globalMutex.RLock()
	defer globalMutex.RUnlock()

	return globalProcessor
}

// HasGlobalProcessor checks if a global processor exists.
func HasGlobalProcessor() bool {
	globalMutex.RLock()
	defer globalMutex.RUnlock()

	return globalProcessor != nil
}

// OnGameSessionStarted is a global function that forwards to the global processor.
func OnGameSessionStarted(session model.GameSession) {
	if processor := GetGlobalProcessor(); processor != nil {
		processor.OnGameSessionStarted(session.GameSessionID)
	}
}

// StartMetricsProcessor starts the global metrics processor.
// This is a convenience function that calls Start(ctx) on the global processor.
func StartMetricsProcessor(ctx context.Context) error {
	if processor := GetGlobalProcessor(); processor != nil {
		return processor.Start(ctx)
	}
	return common.NewGameLiftError(
		common.MetricConfigurationException,
		"Failed to start metrics processor",
		"metrics processor not initialized - call InitMetricsProcessor() first")
}

// TerminateMetricsProcessor stops the global metrics processor.
// This is a convenience function that calls Stop() on the global processor.
func TerminateMetricsProcessor() error {
	if processor := GetGlobalProcessor(); processor != nil {
		return processor.Stop()
	}
	return nil
}
