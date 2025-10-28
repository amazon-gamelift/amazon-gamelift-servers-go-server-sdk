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

func TestNewMin(t *testing.T) {
	minDerived := NewMin()

	common.AssertEqual(t, true, minDerived.value == nil)
	common.AssertEqual(t, "min", minDerived.Key())
}

func TestMinMetric_HandleMessage(t *testing.T) {
	minDerived := NewMin()

	// Test first message.
	message1 := model.MetricMessage{
		Key:   "test_metric",
		Value: 10.5,
	}
	minDerived.HandleMessage(message1)

	common.AssertEqual(t, false, minDerived.value == nil)
	common.AssertEqual(t, message1.Value, *minDerived.value)

	// Test lower value (should update).
	message2 := model.MetricMessage{
		Key:   "test_metric",
		Value: 5.2,
	}
	minDerived.HandleMessage(message2)

	common.AssertEqual(t, false, minDerived.value == nil)
	common.AssertEqual(t, message2.Value, *minDerived.value)

	// Test higher value (should not update).
	message3 := model.MetricMessage{
		Key:   "test_metric",
		Value: 15.0,
	}
	minDerived.HandleMessage(message3)

	common.AssertEqual(t, false, minDerived.value == nil)
	common.AssertEqual(t, message2.Value, *minDerived.value)
}

func TestMinMetric_EmitMetrics_NoValues(t *testing.T) {
	minDerived := NewMin()
	source := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeGauge,
		tags:       map[string]string{"env": "test"},
	}

	messages := minDerived.EmitMetrics(source)

	common.AssertEqual(t, true, messages == nil)
}

func TestMinMetric_EmitMetrics_WithValues(t *testing.T) {
	minDerived := NewMin()
	source := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeTimer,
		tags:       map[string]string{"env": "prod"},
	}

	// Add values.
	values := []float64{15.0, 5.0, 25.0, 8.0, 10.0}
	for _, value := range values {
		message := model.MetricMessage{
			Key:   "test_metric",
			Value: value,
		}
		minDerived.HandleMessage(message)
	}

	messages := minDerived.EmitMetrics(source)

	common.AssertEqual(t, 1, len(messages))
	common.AssertEqual(t, "test_metric.min", messages[0].Key)
	common.AssertEqual(t, model.MetricTypeTimer, messages[0].Type)
	common.AssertEqual(t, 5.0, messages[0].Value)
	common.AssertEqual(t, source.tags["env"], messages[0].Tags["env"])
	common.AssertEqual(t, 1.0, messages[0].SampleRate)
}

func TestMinMetric_Reset(t *testing.T) {
	minDerived := NewMin()

	// Add a value.
	message := model.MetricMessage{
		Key:   "test_metric",
		Value: 42.0,
	}
	minDerived.HandleMessage(message)

	common.AssertEqual(t, false, minDerived.value == nil)
	common.AssertEqual(t, message.Value, *minDerived.value)

	// Reset.
	minDerived.Reset()

	common.AssertEqual(t, true, minDerived.value == nil)
}

func TestMinMetric_NegativeValues(t *testing.T) {
	minDerived := NewMin()

	// Test with negative values.
	values := []float64{-5.0, -10.0, -2.0, -15.0}
	for _, value := range values {
		message := model.MetricMessage{
			Key:   "test_metric",
			Value: value,
		}
		minDerived.HandleMessage(message)
	}

	common.AssertEqual(t, false, minDerived.value == nil)
	common.AssertEqual(t, values[len(values)-1], *minDerived.value) // Should be the lowest (most negative)
}

func TestMinMetric_SingleValue(t *testing.T) {
	minDerived := NewMin()
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
	minDerived.HandleMessage(message)

	messages := minDerived.EmitMetrics(source)

	common.AssertEqual(t, 1, len(messages))
	common.AssertEqual(t, message.Value, messages[0].Value)
}

func TestMinMetric_ZeroValue(t *testing.T) {
	minDerived := NewMin()

	// Test with zero and positive values.
	values := []float64{5.0, 0.0, 10.0, 3.0}
	for _, value := range values {
		message := model.MetricMessage{
			Key:   "test_metric",
			Value: value,
		}
		minDerived.HandleMessage(message)
	}

	common.AssertEqual(t, false, minDerived.value == nil)
	common.AssertEqual(t, 0.0, *minDerived.value)
}
