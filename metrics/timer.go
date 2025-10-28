/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package metrics

import (
	"context"
	"time"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/model"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/samplers"
)

const (
	millisecondsPerSecond = 1000.0
)

var _ model.TimerMetric = (*Timer)(nil)

// Timer is a metric that measures the duration of events.
type Timer struct {
	*baseMetric
}

// NewTimer returns a new timer metric builder.
func NewTimer(key string) *TimerBuilder {
	return &TimerBuilder{
		baseBuilder: newBaseBuilder(key),
	}
}

// TimerBuilder builds a Timer metric with optional tags, samplers, and derived metrics.
type TimerBuilder struct {
	*baseBuilder
}

// WithTags adds multiple tags to the timer metric.
func (b *TimerBuilder) WithTags(tags map[string]string) *TimerBuilder {
	b.withTags(tags)
	return b
}

// WithTag adds a single tag key-value pair to the timer metric.
func (b *TimerBuilder) WithTag(key, value string) *TimerBuilder {
	b.withTag(key, value)
	return b
}

// WithSampler sets the sampler for the timer metric to control sampling behavior.
func (b *TimerBuilder) WithSampler(sampler samplers.Sampler) *TimerBuilder {
	b.withSampler(sampler)
	return b
}

// WithDerivedMetrics adds derived metrics that will be calculated from this timer.
func (b *TimerBuilder) WithDerivedMetrics(derived ...model.DerivedMetric) *TimerBuilder {
	b.withDerivedMetrics(derived...)
	return b
}

// WithMetricsProcessor sets the MetricsProcessor for this timer metric.
func (b *TimerBuilder) WithMetricsProcessor(processor MetricsProcessor) *TimerBuilder {
	b.withMetricsProcessor(processor)
	return b
}

// Build creates and returns the configured Timer metric.
func (b *TimerBuilder) Build() (*Timer, error) {
	if b.processor == nil {
		return nil, common.NewGameLiftError(
			common.MetricConfigurationException,
			"Timer processor required",
			"Timer requires a processor - use WithMetricsProcessor() or factory creation",
		)
	}
	base := newBaseMetric(b.key, model.MetricTypeTimer, b.getTags(), b.derivedMetrics, b.sampler, b.processor)
	b.processor.registerMetric(base)

	return &Timer{
		baseMetric: base,
	}, nil
}

// SetDuration sets the timer to a specific duration.
func (t *Timer) SetDuration(duration time.Duration) {
	t.updateCurrentValue(float64(duration.Milliseconds()), model.MetricOperationSet)
	t.enqueueMessage(t.CurrentValue())
}

// SetMilliseconds sets the timer to a specific value in milliseconds.
func (t *Timer) SetMilliseconds(ms float64) {
	t.updateCurrentValue(ms, model.MetricOperationSet)
	t.enqueueMessage(t.CurrentValue())
}

// SetSeconds sets the timer to a specific value in seconds.
func (t *Timer) SetSeconds(seconds float64) {
	milliseconds := seconds * millisecondsPerSecond
	t.updateCurrentValue(milliseconds, model.MetricOperationSet)
	t.enqueueMessage(t.CurrentValue())
}

// TimeFunc times the execution of a function.
func (t *Timer) TimeFunc(ctx context.Context, fn func() error) error { //nolint:revive // Context provided for user cancellation/timeout handling
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	t.SetDuration(duration)
	return err
}

// Start returns a function that stops the timer when called.
func (t *Timer) Start() func() {
	start := time.Now()
	return func() {
		duration := time.Since(start)
		t.SetDuration(duration)
	}
}

// WithTag returns a timer with an additional tag.
func (t *Timer) WithTag(key, value string) *Timer {
	return t.WithTags(map[string]string{key: value})
}

// WithTags returns a new timer instance with additional dimensional tags.
// If dimensional metrics are disabled, this returns the same instance with updated tags.
func (t *Timer) WithTags(tags map[string]string) *Timer {
	newBase := t.baseMetric.withTags(tags)
	if newBase != nil {
		return &Timer{baseMetric: newBase}
	}
	return t // Return self if no new instance was created
}

// WithDimensionalTag adds a single tag and returns a dimensional timer for chaining.
func (t *Timer) WithDimensionalTag(key, value string) *Timer {
	tags := map[string]string{key: value}
	return t.WithDimensionalTags(tags)
}

// WithDimensionalTags adds multiple tags and returns a dimensional timer for chaining.
func (t *Timer) WithDimensionalTags(tags map[string]string) *Timer {
	if len(tags) == 0 {
		return t
	}
	// Check if dimensional metrics are enabled
	if t.processor != nil && !t.processor.dimensionalMetricsEnabled() {
		// When dimensional metrics are disabled, return self without creating new instances
		return t
	}
	// Create dimensional metric with combined tags (includes cloning derived metrics)
	dimensionalBase := t.baseMetric.createDimensionalMetric(tags)
	return &Timer{baseMetric: dimensionalBase}
}
