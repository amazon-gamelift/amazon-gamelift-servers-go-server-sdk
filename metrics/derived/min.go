/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package derived //nolint:dupl

import "github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/model"

var _ model.DerivedMetric = (*MinMetric)(nil)
var _ model.DerivedMetricCloner = (*MinMetric)(nil)

// MinMetric tracks the minimum value during capture period.
type MinMetric struct {
	value *float64
}

// NewMin creates a new minimum value derived metric.
func NewMin() *MinMetric {
	return &MinMetric{}
}

// Key returns the key suffix for this derived metric.
func (m *MinMetric) Key() string {
	return "min"
}

// HandleMessage processes a new metric message and updates the minimum value.
func (m *MinMetric) HandleMessage(message model.MetricMessage) {
	if m.value == nil || message.Value < *m.value {
		m.value = &message.Value
	}
}

// EmitMetrics returns the minimum metric value if one exists.
func (m *MinMetric) EmitMetrics(source model.Metric) []model.MetricMessage {
	if m.value == nil {
		return nil
	}

	return []model.MetricMessage{
		{
			Key:        source.Key() + "." + m.Key(),
			Type:       source.MetricType(),
			Value:      *m.value,
			Tags:       source.Tags(),
			SampleRate: 1.0,
		},
	}
}

// Reset clears the minimum value and resets the metric state.
func (m *MinMetric) Reset() {
	m.value = nil
}

// Clone creates a new MinMetric instance.
func (m *MinMetric) Clone() model.DerivedMetric { //nolint:ireturn
	return NewMin()
}
