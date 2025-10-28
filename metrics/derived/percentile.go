/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package derived

import (
	"fmt"
	"sort"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/model"
)

// Common percentile constants to avoid ambiguity (0-100 scale).
const (
	P25  float64 = 25.0
	P50  float64 = 50.0
	P75  float64 = 75.0
	P90  float64 = 90.0
	P95  float64 = 95.0
	P99  float64 = 99.0
	P999 float64 = 99.9

	percentileBase = 100.0
)

var _ model.DerivedMetric = (*PercentileMetric)(nil)
var _ model.DerivedMetricCloner = (*PercentileMetric)(nil)

// PercentileMetric calculates percentiles from a set of values.
type PercentileMetric struct {
	percentiles []float64
	values      []float64
}

// NewPercentile creates a new percentile derived metric.
func NewPercentile(percentiles ...float64) *PercentileMetric {
	return &PercentileMetric{
		percentiles: percentiles,
	}
}

// Key returns the key suffix for this derived metric.
func (p *PercentileMetric) Key() string {
	return "percentile"
}

// HandleMessage processes a new metric message and stores the value for percentile calculation.
func (p *PercentileMetric) HandleMessage(message model.MetricMessage) {
	p.values = append(p.values, message.Value)
}

// EmitMetrics returns percentile metric values if values exist.
func (p *PercentileMetric) EmitMetrics(source model.Metric) []model.MetricMessage {
	if len(p.values) == 0 {
		return nil
	}

	// Sort values for percentile calculation.
	sortedValues := make([]float64, len(p.values))
	copy(sortedValues, p.values)
	sort.Float64s(sortedValues)

	messages := make([]model.MetricMessage, 0, len(p.percentiles))
	for _, percentile := range p.percentiles {
		value := p.calculatePercentile(sortedValues, percentile)
		messages = append(messages, model.MetricMessage{
			Key:        p.getPercentileKey(source.Key(), percentile),
			Type:       source.MetricType(),
			Value:      value,
			Tags:       source.Tags(),
			SampleRate: 1.0,
		})
	}

	return messages
}

// Reset clears all stored values and resets the metric state.
func (p *PercentileMetric) Reset() {
	p.values = p.values[:0] // Clear slice but keep capacity
}

// Clone creates a new PercentileMetric instance.
func (p *PercentileMetric) Clone() model.DerivedMetric { //nolint:ireturn
	return NewPercentile(p.percentiles...)
}

// calculatePercentile calculates the value at the given percentile.
func (p *PercentileMetric) calculatePercentile(sortedValues []float64, percentile float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}
	if len(sortedValues) == 1 {
		return sortedValues[0]
	}

	// Initially, use the nearest-rank method for percentile calculation.

	index := (percentile / percentileBase) * float64(len(sortedValues)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sortedValues) {
		return sortedValues[len(sortedValues)-1]
	}

	// Linear interpolation between the two nearest values.
	weight := index - float64(lower)
	return sortedValues[lower]*(1-weight) + sortedValues[upper]*weight
}

// getPercentileKey generates the key for a specific percentile.
func (p *PercentileMetric) getPercentileKey(baseKey string, percentile float64) string {
	if percentile == float64(int(percentile)) {
		return fmt.Sprintf("%s.p%.0f", baseKey, percentile)
	}
	return fmt.Sprintf("%s.p%.1f", baseKey, percentile)
}
