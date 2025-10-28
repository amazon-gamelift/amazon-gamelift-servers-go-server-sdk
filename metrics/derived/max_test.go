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

func TestNewMax(t *testing.T) {
	maxDerived := NewMax()

	common.AssertEqual(t, true, maxDerived.value == nil)
	common.AssertEqual(t, "max", maxDerived.Key())
}

func TestMaxMetric_HandleMessage(t *testing.T) {
	maxDerived := NewMax()

	// Test first message.
	message1 := model.MetricMessage{
		Key:   "test_metric",
		Value: 10.5,
	}
	maxDerived.HandleMessage(message1)

	common.AssertEqual(t, false, maxDerived.value == nil)
	common.AssertEqual(t, message1.Value, *maxDerived.value)

	// Test higher value (should update).
	message2 := model.MetricMessage{
		Key:   "test_metric",
		Value: 20.7,
	}
	maxDerived.HandleMessage(message2)

	common.AssertEqual(t, false, maxDerived.value == nil)
	common.AssertEqual(t, message2.Value, *maxDerived.value)

	// Test lower value (should not update).
	message3 := model.MetricMessage{
		Key:   "test_metric",
		Value: 15.0,
	}
	maxDerived.HandleMessage(message3)

	common.AssertEqual(t, false, maxDerived.value == nil)
	common.AssertEqual(t, message2.Value, *maxDerived.value)
}

func TestMaxMetric_EmitMetrics_NoValues(t *testing.T) {
	maxDerived := NewMax()
	source := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeGauge,
		tags:       map[string]string{"env": "test"},
	}

	messages := maxDerived.EmitMetrics(source)

	common.AssertEqual(t, true, messages == nil)
}

func TestMaxMetric_EmitMetrics_WithValues(t *testing.T) {
	maxDerived := NewMax()
	source := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeTimer,
		tags:       map[string]string{"env": "prod"},
	}

	// Add values.
	values := []float64{5.0, 15.0, 10.0, 25.0, 8.0}
	for _, value := range values {
		message := model.MetricMessage{
			Key:   "test_metric",
			Value: value,
		}
		maxDerived.HandleMessage(message)
	}

	messages := maxDerived.EmitMetrics(source)

	common.AssertEqual(t, 1, len(messages))
	common.AssertEqual(t, "test_metric.max", messages[0].Key)
	common.AssertEqual(t, model.MetricTypeTimer, messages[0].Type)
	common.AssertEqual(t, 25.0, messages[0].Value)
	common.AssertEqual(t, source.tags["env"], messages[0].Tags["env"])
	common.AssertEqual(t, 1.0, messages[0].SampleRate)
}

func TestMaxMetric_Reset(t *testing.T) {
	maxDerived := NewMax()

	// Add a value.
	message := model.MetricMessage{
		Key:   "test_metric",
		Value: 42.0,
	}
	maxDerived.HandleMessage(message)

	common.AssertEqual(t, false, maxDerived.value == nil)
	common.AssertEqual(t, message.Value, *maxDerived.value)

	// Reset.
	maxDerived.Reset()

	common.AssertEqual(t, true, maxDerived.value == nil)
}

func TestMaxMetric_NegativeValues(t *testing.T) {
	maxDerived := NewMax()

	// Test with negative values.
	values := []float64{-10.0, -5.0, -15.0, -2.0}
	for _, value := range values {
		message := model.MetricMessage{
			Key:   "test_metric",
			Value: value,
		}
		maxDerived.HandleMessage(message)
	}

	common.AssertEqual(t, false, maxDerived.value == nil)
	common.AssertEqual(t, values[len(values)-1], *maxDerived.value) // Should be the highest (least negative)
}

func TestMaxMetric_SingleValue(t *testing.T) {
	maxDerived := NewMax()
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
	maxDerived.HandleMessage(message)

	messages := maxDerived.EmitMetrics(source)

	common.AssertEqual(t, 1, len(messages))
	common.AssertEqual(t, message.Value, messages[0].Value)
}
