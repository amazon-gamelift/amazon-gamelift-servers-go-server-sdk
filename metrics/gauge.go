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

var _ model.GaugeMetric = (*Gauge)(nil)

// Gauge is a metric that represents a single numerical value that can go up or down.
type Gauge struct {
	*baseMetric
}

// NewGauge returns a new gauge metric builder.
func NewGauge(key string) *GaugeBuilder {
	return &GaugeBuilder{
		baseBuilder: newBaseBuilder(key),
	}
}

// GaugeBuilder builds a Gauge metric with optional tags, samplers, and.
// derived metrics.
type GaugeBuilder struct {
	*baseBuilder
}

// WithTags adds multiple tags to the gauge metric.
func (b *GaugeBuilder) WithTags(tags map[string]string) *GaugeBuilder {
	b.withTags(tags)
	return b
}

// WithTag adds a single tag key-value pair to the gauge metric.
func (b *GaugeBuilder) WithTag(key, value string) *GaugeBuilder {
	b.withTag(key, value)
	return b
}

// WithSampler sets the sampler for the gauge metric to control sampling behavior.
func (b *GaugeBuilder) WithSampler(sampler samplers.Sampler) *GaugeBuilder {
	b.withSampler(sampler)
	return b
}

// WithDerivedMetrics adds derived metrics that will be calculated from this gauge.
func (b *GaugeBuilder) WithDerivedMetrics(derived ...model.DerivedMetric) *GaugeBuilder {
	b.withDerivedMetrics(derived...)
	return b
}

// WithMetricsProcessor sets the MetricsProcessor for this gauge metric.
func (b *GaugeBuilder) WithMetricsProcessor(processor MetricsProcessor) *GaugeBuilder {
	b.withMetricsProcessor(processor)
	return b
}

// Build creates and returns the configured Gauge metric.
func (b *GaugeBuilder) Build() (*Gauge, error) {
	if b.processor == nil {
		return nil, common.NewGameLiftError(
			common.MetricConfigurationException,
			"Gauge processor required",
			"Gauge requires a processor - use WithMetricsProcessor() or factory creation",
		)
	}
	base := newBaseMetric(b.key, model.MetricTypeGauge, b.getTags(), b.derivedMetrics, b.sampler, b.processor)
	b.processor.registerMetric(base)

	return &Gauge{
		baseMetric: base,
	}, nil
}

// Set sets the gauge to a specific value.
func (g *Gauge) Set(value float64) {
	g.updateCurrentValue(value, model.MetricOperationSet)
	g.enqueueMessage(value)
}

// Add adds to the current gauge value.
func (g *Gauge) Add(value float64) {
	g.updateCurrentValue(value, model.MetricOperationAdjust)
	g.enqueueMessage(value)
}

// Subtract subtracts from the current gauge value.
func (g *Gauge) Subtract(value float64) {
	g.updateCurrentValue(-value, model.MetricOperationAdjust)
	g.enqueueMessage(-value)
}

// Increment increments the gauge by 1.
func (g *Gauge) Increment() {
	g.Add(1)
}

// Decrement decrements the gauge by 1.
func (g *Gauge) Decrement() {
	g.Add(-1)
}

// Reset resets the gauge to 0.
func (g *Gauge) Reset() {
	g.Set(0)
}

// WithTag returns a gauge with an additional tag.
func (g *Gauge) WithTag(key, value string) *Gauge {
	return g.WithTags(map[string]string{key: value})
}

// WithTags returns a new gauge instance with additional dimensional tags.
// If dimensional metrics are disabled, this returns the same instance with updated tags.
func (g *Gauge) WithTags(tags map[string]string) *Gauge {
	newBase := g.baseMetric.withTags(tags)
	if newBase != nil {
		return &Gauge{baseMetric: newBase} // Wrap new base instance
	}
	return g // Return self if no new instance was created
}

// WithDimensionalTag adds a single tag and returns a dimensional gauge for chaining.
func (g *Gauge) WithDimensionalTag(key, value string) *Gauge {
	tags := map[string]string{key: value}
	return g.WithDimensionalTags(tags)
}

// WithDimensionalTags adds multiple tags and returns a dimensional gauge for chaining.
func (g *Gauge) WithDimensionalTags(tags map[string]string) *Gauge {
	if len(tags) == 0 {
		return g
	}
	// Check if dimensional metrics are enabled
	if g.processor != nil && !g.processor.dimensionalMetricsEnabled() {
		// When dimensional metrics are disabled, return self without creating new instances
		return g
	}
	// Create dimensional metric with combined tags (includes cloning derived metrics)
	dimensionalBase := g.baseMetric.createDimensionalMetric(tags)
	return &Gauge{baseMetric: dimensionalBase}
}
