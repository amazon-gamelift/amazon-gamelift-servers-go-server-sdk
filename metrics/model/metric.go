/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package model

import (
	"context"
	"time"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/samplers"
)

// MetricType represents the type of metric.
type MetricType int

const (
	// MetricTypeGauge represents point-in-time values.
	MetricTypeGauge MetricType = iota
	// MetricTypeCounter represents event counting.
	MetricTypeCounter
	// MetricTypeTimer represents duration measurements.
	MetricTypeTimer
)

// String returns the string representation of the metric type.
func (mt MetricType) String() string {
	switch mt {
	case MetricTypeGauge:
		return "gauge"
	case MetricTypeCounter:
		return "counter"
	case MetricTypeTimer:
		return "timer"
	default:
		return "unknown"
	}
}

// MetricOperation represents the type of operation called on metric.
type MetricOperation int

const (
	// MetricOperationSet represents an operation that sets a value.
	MetricOperationSet MetricOperation = iota
	// MetricOperationAdjust represents an operation that adjusts a value.
	MetricOperationAdjust
)

// Metric represents the base interface for all metrics.
type Metric interface {
	// Key returns the metric key/name.
	Key() string
	// MetricType returns the type of this metric.
	MetricType() MetricType
	// DerivedMetrics returns any derived metrics for this metric.
	DerivedMetrics() []DerivedMetric
	// CurrentValue returns the current stated value of a metric.
	CurrentValue() float64
	// Tags returns the tags associated with this metric.
	Tags() map[string]string
	// SetTag sets a tag on this metric.
	SetTag(key, value string) error
	// SetTags sets multiple tags on this metric.
	SetTags(tags map[string]string) error
	// RemoveTag removes a tag from this metric.
	RemoveTag(key string)
}

// GaugeMetric represents a point-in-time value metric.
type GaugeMetric interface {
	Metric
	// Set sets the gauge to a specific value
	Set(value float64)
	// Add adds to the current gauge value
	Add(value float64)
	// Subtract subtracts from the current gauge value
	Subtract(value float64)
	// Increment increments the gauge by 1
	Increment()
	// Decrement decrements the gauge by 1
	Decrement()
	// Reset resets the gauge to 0
	Reset()
}

// CounterMetric represents an event counting metric.
type CounterMetric interface {
	Metric
	// Add adds to the counter
	Add(value float64)
	// Increment increments the counter by 1
	Increment()
	// Count increments the counter if the condition is true
	Count(condition bool)
}

// TimerMetric represents a duration measurement metric.
type TimerMetric interface {
	Metric
	// SetDuration sets the timer to a specific duration
	SetDuration(duration time.Duration)
	// SetMilliseconds sets the timer to a specific value in milliseconds
	SetMilliseconds(ms float64)
	// SetSeconds sets the timer to a specific value in seconds
	SetSeconds(seconds float64)
	// TimeFunc times the execution of a function
	TimeFunc(ctx context.Context, fn func() error) error
	// Start returns a function that stops the timer when called
	Start() func()
}

// DerivedMetric represents a metric computed from other metrics.
type DerivedMetric interface {
	// Key returns the derived metric key
	Key() string
	// HandleMessage processes a metric message to update internal state
	HandleMessage(message MetricMessage)
	// EmitMetrics returns the computed metrics ready for emission
	EmitMetrics(source Metric) []MetricMessage
	// Reset resets the internal state
	Reset()
}

// DerivedMetricCloner defines the interface for derived metrics that support cloning.
// This allows dimensional variants to have independent copies of derived metrics.
type DerivedMetricCloner interface {
	// Clone creates a deep copy of the derived metric for use in dimensional variants
	Clone() DerivedMetric
}

// MetricMessage represents a metric data point being processed.
type MetricMessage struct {
	// Timestamp is when this metric was recorded
	Timestamp time.Time
	// Tags are the dimensions associated with this metric
	Tags map[string]string
	// Key is the metric key/name
	Key string
	// Type is the metric type
	Type MetricType
	// Value is the numeric value
	Value float64
	// SampleRate is the sampling rate (0.0 to 1.0)
	SampleRate float64
}

// MetricBuilder provides a fluent interface for creating metrics.
type MetricBuilder interface {
	WithTags(tags map[string]string) MetricBuilder
	WithTag(key, value string) MetricBuilder
	WithSampler(sampler samplers.Sampler) MetricBuilder
	WithDerivedMetrics(derived ...DerivedMetric) MetricBuilder
	Build() Metric
}

// Note: Transport interface is defined in transport.go
