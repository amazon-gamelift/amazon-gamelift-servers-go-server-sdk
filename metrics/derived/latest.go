/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package derived

import "github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/model"

var _ model.DerivedMetric = (*LatestMetric)(nil)
var _ model.DerivedMetricCloner = (*LatestMetric)(nil)

// LatestMetric keeps track of the most recent value.
type LatestMetric struct {
	value *float64
}

// NewLatest creates a new latest value derived metric.
func NewLatest() *LatestMetric {
	return &LatestMetric{}
}

// Key returns the key suffix for this derived metric.
func (l *LatestMetric) Key() string {
	return "latest"
}

// HandleMessage processes a new metric message and updates the latest value.
func (l *LatestMetric) HandleMessage(message model.MetricMessage) {
	l.value = &message.Value
}

// EmitMetrics returns the latest metric value if one exists.
func (l *LatestMetric) EmitMetrics(source model.Metric) []model.MetricMessage {
	if l.value == nil {
		return nil
	}

	return []model.MetricMessage{
		{
			Key:        source.Key() + "." + l.Key(),
			Type:       source.MetricType(),
			Value:      *l.value,
			Tags:       source.Tags(),
			SampleRate: 1.0,
		},
	}
}

// Reset clears the latest value and resets the metric state.
func (l *LatestMetric) Reset() {
	l.value = nil
}

// Clone creates a new LatestMetric instance.
func (l *LatestMetric) Clone() model.DerivedMetric { //nolint:ireturn
	return NewLatest()
}
