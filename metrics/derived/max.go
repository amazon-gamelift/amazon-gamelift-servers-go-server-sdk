/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package derived //nolint:dupl

import "github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/model"

var _ model.DerivedMetric = (*MaxMetric)(nil)
var _ model.DerivedMetricCloner = (*MaxMetric)(nil)

// MaxMetric tracks the maximum value during capture period.
type MaxMetric struct {
	value *float64
}

// NewMax creates a new maximum value derived metric.
func NewMax() *MaxMetric {
	return &MaxMetric{}
}

// Key returns the key suffix for this derived metric.
func (m *MaxMetric) Key() string {
	return "max"
}

// HandleMessage processes a new metric message and updates the maximum value.
func (m *MaxMetric) HandleMessage(message model.MetricMessage) {
	if m.value == nil || message.Value > *m.value {
		m.value = &message.Value
	}
}

// EmitMetrics returns the maximum metric value if one exists.
func (m *MaxMetric) EmitMetrics(source model.Metric) []model.MetricMessage {
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

// Reset clears the maximum value and resets the metric state.
func (m *MaxMetric) Reset() {
	m.value = nil
}

// Clone creates a new MaxMetric instance.
func (m *MaxMetric) Clone() model.DerivedMetric { //nolint:ireturn
	return NewMax()
}
