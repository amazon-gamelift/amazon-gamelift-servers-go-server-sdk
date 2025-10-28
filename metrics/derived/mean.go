/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package derived

import "github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/model"

var _ model.DerivedMetric = (*MeanMetric)(nil)
var _ model.DerivedMetricCloner = (*MeanMetric)(nil)

// MeanMetric calculates the running mean of values.
type MeanMetric struct {
	sum   *float64
	count int64
}

// NewMean creates a new mean derived metric.
func NewMean() *MeanMetric {
	return &MeanMetric{}
}

// Key returns the key suffix for this derived metric.
func (m *MeanMetric) Key() string {
	return "mean"
}

// HandleMessage processes a new metric message and updates the running mean calculation.
func (m *MeanMetric) HandleMessage(message model.MetricMessage) {
	if m.sum == nil {
		m.sum = &message.Value
	} else {
		*m.sum += message.Value
	}
	m.count++
}

// EmitMetrics returns the mean metric value if one exists.
func (m *MeanMetric) EmitMetrics(source model.Metric) []model.MetricMessage {
	if m.sum == nil || m.count == 0 {
		return nil
	}

	meanValue := *m.sum / float64(m.count)

	return []model.MetricMessage{
		{
			Key:        source.Key() + "." + m.Key(),
			Type:       source.MetricType(),
			Value:      meanValue,
			Tags:       source.Tags(),
			SampleRate: 1.0,
		},
	}
}

// Reset clears the mean calculation and resets the metric state.
func (m *MeanMetric) Reset() {
	m.sum = nil
	m.count = 0
}

// Clone creates a new MeanMetric instance.
func (m *MeanMetric) Clone() model.DerivedMetric { //nolint:ireturn
	return NewMean()
}
