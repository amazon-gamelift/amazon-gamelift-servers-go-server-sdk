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

func TestNewMean(t *testing.T) {
	mean := NewMean()

	common.AssertEqual(t, true, mean.sum == nil)
	common.AssertEqual(t, int64(0), mean.count)
	common.AssertEqual(t, "mean", mean.Key())
}

func TestMeanMetric_HandleMessage(t *testing.T) {
	mean := NewMean()

	// Test first message.
	message1 := model.MetricMessage{
		Key:   "test_metric",
		Value: 10.0,
	}
	mean.HandleMessage(message1)

	common.AssertEqual(t, false, mean.sum == nil)
	common.AssertEqual(t, message1.Value, *mean.sum)
	common.AssertEqual(t, int64(1), mean.count)

	// Test second message.
	message2 := model.MetricMessage{
		Key:   "test_metric",
		Value: 20.0,
	}
	mean.HandleMessage(message2)

	common.AssertEqual(t, false, mean.sum == nil)
	common.AssertEqual(t, message1.Value+message2.Value, *mean.sum)
	common.AssertEqual(t, int64(2), mean.count)

	// Test third message.
	message3 := model.MetricMessage{
		Key:   "test_metric",
		Value: 0.0,
	}
	mean.HandleMessage(message3)

	common.AssertEqual(t, false, mean.sum == nil)
	common.AssertEqual(t, message1.Value+message2.Value, *mean.sum)
	common.AssertEqual(t, int64(3), mean.count)
}

func TestMeanMetric_EmitMetrics_NoValues(t *testing.T) {
	mean := NewMean()
	source := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeGauge,
		tags:       map[string]string{"env": "test"},
	}

	messages := mean.EmitMetrics(source)

	common.AssertEqual(t, true, messages == nil)
}

func TestMeanMetric_EmitMetrics_WithValues(t *testing.T) {
	mean := NewMean()
	source := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeTimer,
		tags:       map[string]string{"env": "prod"},
	}

	// Add values: sum = 60, count = 4, mean = 15.
	values := []float64{10.0, 20.0, 15.0, 15.0}
	for _, value := range values {
		message := model.MetricMessage{
			Key:   "test_metric",
			Value: value,
		}
		mean.HandleMessage(message)
	}

	messages := mean.EmitMetrics(source)

	common.AssertEqual(t, 1, len(messages))
	common.AssertEqual(t, "test_metric.mean", messages[0].Key)
	common.AssertEqual(t, model.MetricTypeTimer, messages[0].Type)
	common.AssertEqual(t, 15.0, messages[0].Value)
	common.AssertEqual(t, source.tags["env"], messages[0].Tags["env"])
	common.AssertEqual(t, 1.0, messages[0].SampleRate)
}

func TestMeanMetric_Reset(t *testing.T) {
	mean := NewMean()

	// Add values.
	values := []float64{10.0, 20.0, 30.0}
	for _, value := range values {
		message := model.MetricMessage{
			Key:   "test_metric",
			Value: value,
		}
		mean.HandleMessage(message)
	}

	common.AssertEqual(t, false, mean.sum == nil)
	common.AssertEqual(t, 60.0, *mean.sum)
	common.AssertEqual(t, int64(3), mean.count)

	// Reset.
	mean.Reset()

	common.AssertEqual(t, true, mean.sum == nil)
	common.AssertEqual(t, int64(0), mean.count)
}

func TestMeanMetric_NegativeValues(t *testing.T) {
	mean := NewMean()
	source := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeGauge,
		tags:       map[string]string{},
	}

	// Test with negative values: sum = -30, count = 3, mean = -10.
	values := []float64{-5.0, -10.0, -15.0}
	for _, value := range values {
		message := model.MetricMessage{
			Key:   "test_metric",
			Value: value,
		}
		mean.HandleMessage(message)
	}

	messages := mean.EmitMetrics(source)

	common.AssertEqual(t, 1, len(messages))
	common.AssertEqual(t, -10.0, messages[0].Value)
}

func TestMeanMetric_SingleValue(t *testing.T) {
	mean := NewMean()
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
	mean.HandleMessage(message)

	messages := mean.EmitMetrics(source)

	common.AssertEqual(t, 1, len(messages))
	common.AssertEqual(t, 42.0, messages[0].Value)
}

func TestMeanMetric_MixedValues(t *testing.T) {
	mean := NewMean()
	source := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeGauge,
		tags:       map[string]string{},
	}

	// Test with mixed positive and negative values: sum = 5, count = 5, mean = 1.
	values := []float64{-10.0, 5.0, 0.0, 15.0, -5.0}
	for _, value := range values {
		message := model.MetricMessage{
			Key:   "test_metric",
			Value: value,
		}
		mean.HandleMessage(message)
	}

	messages := mean.EmitMetrics(source)

	common.AssertEqual(t, 1, len(messages))
	common.AssertEqual(t, 1.0, messages[0].Value)
}

func TestMeanMetric_DecimalPrecision(t *testing.T) {
	mean := NewMean()
	source := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeGauge,
		tags:       map[string]string{},
	}

	// Test with values that result in a decimal mean: sum = 10, count = 3, mean = 3.333...
	values := []float64{3.0, 3.0, 4.0}
	for _, value := range values {
		message := model.MetricMessage{
			Key:   "test_metric",
			Value: value,
		}
		mean.HandleMessage(message)
	}

	messages := mean.EmitMetrics(source)

	common.AssertEqual(t, 1, len(messages))
	// Check that the result is approximately 3.333...
	expectedMean := 10.0 / 3.0
	common.AssertEqual(t, expectedMean, messages[0].Value)
}
