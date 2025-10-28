/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package derived

import (
	"testing"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/model"
)

func TestNewPercentile(t *testing.T) {
	percentiles := []float64{50.0, 95.0, 99.0}
	p := NewPercentile(percentiles...)

	common.AssertEqual(t, "percentile", p.Key())

	common.AssertEqual(t, 3, len(p.percentiles))

	// TODO(dr): If we can move to Go 1.21+, refactor to use slices.Equal.
	common.AssertEqual(t, P50, p.percentiles[0])
	common.AssertEqual(t, P95, p.percentiles[1])
	common.AssertEqual(t, P99, p.percentiles[2])

	common.AssertEqual(t, 0, len(p.values))
}

func TestPercentileMetric_HandleMessage(t *testing.T) {
	p := NewPercentile(50.0)

	// Test adding values.
	values := []float64{10.0, 20.0, 30.0}
	for _, value := range values {
		message := model.MetricMessage{
			Key:   "test_metric",
			Value: value,
		}
		p.HandleMessage(message)
	}

	// TODO(dr): If we can move to Go 1.21+, refactor to use slices.Equal.
	common.AssertEqual(t, 3, len(p.values))
	common.AssertEqual(t, 10.0, p.values[0])
	common.AssertEqual(t, 20.0, p.values[1])
	common.AssertEqual(t, 30.0, p.values[2])
}

func TestPercentileMetric_EmitMetrics_NoValues(t *testing.T) {
	p := NewPercentile(P50, P95)
	source := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeTimer,
		tags:       map[string]string{"env": "test"},
	}

	messages := p.EmitMetrics(source)

	common.AssertEqual(t, true, messages == nil)
}

func TestPercentileMetric_EmitMetrics_SinglePercentile(t *testing.T) {
	p := NewPercentile(P50)
	source := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeTimer,
		tags:       map[string]string{"env": "prod"},
	}

	// Add values: [1, 2, 3, 4, 5] - median should be 3.
	values := []float64{3.0, 1.0, 5.0, 2.0, 4.0}
	for _, value := range values {
		message := model.MetricMessage{
			Key:   "test_metric",
			Value: value,
		}
		p.HandleMessage(message)
	}

	messages := p.EmitMetrics(source)

	common.AssertEqual(t, 1, len(messages))
	common.AssertEqual(t, "test_metric.p50", messages[0].Key)
	common.AssertEqual(t, model.MetricTypeTimer, messages[0].Type)
	common.AssertEqual(t, 3.0, messages[0].Value)
	common.AssertEqual(t, source.tags["env"], messages[0].Tags["env"])
	common.AssertEqual(t, 1.0, messages[0].SampleRate)
}

func TestPercentileMetric_EmitMetrics_MultiplePercentiles(t *testing.T) {
	p := NewPercentile(P25, P50, P75)
	source := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeGauge,
		tags:       map[string]string{},
	}

	// Add values: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10].
	for i := 1; i <= 10; i++ {
		message := model.MetricMessage{
			Key:   "test_metric",
			Value: float64(i),
		}
		p.HandleMessage(message)
	}

	messages := p.EmitMetrics(source)

	common.AssertEqual(t, 3, len(messages))

	common.AssertEqual(t, "test_metric.p25", messages[0].Key)
	common.AssertEqual(t, 3.25, messages[0].Value)

	common.AssertEqual(t, "test_metric.p50", messages[1].Key)
	common.AssertEqual(t, 5.5, messages[1].Value)

	common.AssertEqual(t, "test_metric.p75", messages[2].Key)
	common.AssertEqual(t, 7.75, messages[2].Value)
}

func TestPercentileMetric_Reset(t *testing.T) {
	p := NewPercentile(P50)

	// Add values.
	values := []float64{1.0, 2.0, 3.0}
	for _, value := range values {
		message := model.MetricMessage{
			Key:   "test_metric",
			Value: value,
		}
		p.HandleMessage(message)
	}

	common.AssertEqual(t, 3, len(p.values))

	// Reset.
	p.Reset()

	common.AssertEqual(t, 0, len(p.values))
}

func TestPercentileMetric_GetPercentileKey_IntegerPercentiles(t *testing.T) {
	p := NewPercentile(P50)

	key := p.getPercentileKey("test_metric", P50)
	common.AssertEqual(t, "test_metric.p50", key)

	key = p.getPercentileKey("test_metric", P95)
	common.AssertEqual(t, "test_metric.p95", key)
}

func TestPercentileMetric_GetPercentileKey_DecimalPercentiles(t *testing.T) {
	p := NewPercentile(99.5)

	key := p.getPercentileKey("test_metric", 99.5)
	common.AssertEqual(t, "test_metric.p99.5", key)

	key = p.getPercentileKey("test_metric", P999)
	common.AssertEqual(t, "test_metric.p99.9", key)
}

func TestPercentileMetric_CalculatePercentile_EdgeCases(t *testing.T) {
	p := NewPercentile(P50)

	// Test empty slice.
	result := p.calculatePercentile([]float64{}, 50.0)
	common.AssertEqual(t, 0.0, result)

	// Test single value.
	result = p.calculatePercentile([]float64{42.0}, 50.0)
	common.AssertEqual(t, 42.0, result)

	// Test two values.
	result = p.calculatePercentile([]float64{10.0, 20.0}, 50.0)
	common.AssertEqual(t, 15.0, result) // Linear interpolation
}

func TestPercentileMetric_CalculatePercentile_ExtremePercentiles(t *testing.T) {
	// Float can be passed in place of the Const percentile type.
	p := NewPercentile(0.0, 100.0)
	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	// P0 should be the minimum.
	result := p.calculatePercentile(values, 0.0)
	common.AssertEqual(t, 1.0, result)

	// P100 should be the maximum.
	result = p.calculatePercentile(values, 100.0)
	common.AssertEqual(t, 5.0, result)
}

func TestPercentileMetric_SingleValue(t *testing.T) {
	p := NewPercentile(P50, P95)
	source := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeCounter,
		tags:       map[string]string{},
	}

	// Add single value.
	message := model.MetricMessage{
		Key:   "test_metric",
		Value: 42.0,
	}
	p.HandleMessage(message)

	messages := p.EmitMetrics(source)

	common.AssertEqual(t, 2, len(messages))
	common.AssertEqual(t, 42.0, messages[0].Value) // P50
	common.AssertEqual(t, 42.0, messages[1].Value) // P95
}

func TestPercentileMetric_NegativeValues(t *testing.T) {
	p := NewPercentile(P50)
	source := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeGauge,
		tags:       map[string]string{},
	}

	// Add negative values: [-5, -3, -1].
	values := []float64{-1.0, -5.0, -3.0}
	for _, value := range values {
		message := model.MetricMessage{
			Key:   "test_metric",
			Value: value,
		}
		p.HandleMessage(message)
	}

	messages := p.EmitMetrics(source)

	common.AssertEqual(t, 1, len(messages))
	common.AssertEqual(t, -3.0, messages[0].Value) // Median of [-5, -3, -1]
}
