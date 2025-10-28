/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package metrics

import (
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/model"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/samplers"
)

var _ model.CounterMetric = (*Counter)(nil)

// Counter is a monotonically increasing metric that counts occurrences of events.
type Counter struct {
	*baseMetric
}

// NewCounter returns a new counter metric builder.
func NewCounter(key string) *CounterBuilder {
	return &CounterBuilder{
		baseBuilder: newBaseBuilder(key),
	}
}

// CounterBuilder builds a Counter metric with optional tags, samplers, and.
// derived metrics.
type CounterBuilder struct {
	*baseBuilder
}

// WithTags adds multiple tags to the counter metric.
func (b *CounterBuilder) WithTags(tags map[string]string) *CounterBuilder {
	b.withTags(tags)
	return b
}

// WithTag adds a single tag key-value pair to the counter metric.
func (b *CounterBuilder) WithTag(key, value string) *CounterBuilder {
	b.withTag(key, value)
	return b
}

// WithSampler sets the sampler for the counter metric to control sampling behavior.
func (b *CounterBuilder) WithSampler(sampler samplers.Sampler) *CounterBuilder {
	b.withSampler(sampler)
	return b
}

// WithDerivedMetrics adds derived metrics that will be calculated from this counter.
func (b *CounterBuilder) WithDerivedMetrics(derived ...model.DerivedMetric) *CounterBuilder {
	b.withDerivedMetrics(derived...)
	return b
}

// WithMetricsProcessor sets the MetricsProcessor for this counter metric.
func (b *CounterBuilder) WithMetricsProcessor(processor MetricsProcessor) *CounterBuilder {
	b.withMetricsProcessor(processor)
	return b
}

// Build creates and returns the configured Counter metric.
func (b *CounterBuilder) Build() (*Counter, error) {
	if b.processor == nil {
		return nil, common.NewGameLiftError(
			common.MetricConfigurationException,
			"Counter processor required",
			"Counter requires a processor - use WithMetricsProcessor() or factory creation",
		)
	}
	base := newBaseMetric(b.key, model.MetricTypeCounter, b.getTags(), b.derivedMetrics, b.sampler, b.processor)
	b.processor.registerMetric(base)

	return &Counter{
		baseMetric: base,
	}, nil
}

// Add adds to the counter.
func (c *Counter) Add(value float64) {
	c.updateCurrentValue(value, model.MetricOperationAdjust)
	c.enqueueMessage(value)
}

// Increment increments the counter by 1.
func (c *Counter) Increment() {
	c.Add(1)
}

// Count increments the counter if the condition is true.
func (c *Counter) Count(condition bool) {
	if condition {
		c.Increment()
	}
}

// WithTags returns a new counter instance with additional dimensional tags.
// If dimensional metrics are disabled, this returns the same instance with updated tags.
func (c *Counter) WithTags(tags map[string]string) *Counter {
	newBase := c.baseMetric.withTags(tags)
	if newBase != nil {
		return &Counter{baseMetric: newBase} // Wrap new base instance
	}
	return c // Return self if no new instance was created
}

// WithTag returns a counter with an additional tag.
func (c *Counter) WithTag(key, value string) *Counter {
	return c.WithTags(map[string]string{key: value})
}

// WithDimensionalTag adds a single tag and returns a dimensional counter for chaining.
func (c *Counter) WithDimensionalTag(key, value string) *Counter {
	tags := map[string]string{key: value}
	return c.WithDimensionalTags(tags)
}

// WithDimensionalTags adds multiple tags and returns a dimensional counter for chaining.
func (c *Counter) WithDimensionalTags(tags map[string]string) *Counter {
	if len(tags) == 0 {
		return c
	}
	// Check if dimensional metrics are enabled
	if c.processor != nil && !c.processor.dimensionalMetricsEnabled() {
		// When dimensional metrics are disabled, return self without creating new instances
		return c
	}
	// Create dimensional metric with combined tags
	dimensionalBase := c.baseMetric.createDimensionalMetric(tags)
	return &Counter{baseMetric: dimensionalBase}
}
