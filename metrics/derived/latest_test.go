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

func TestNewLatest(t *testing.T) {
	latest := NewLatest()
	common.AssertEqual(t, true, latest.value == nil)
	common.AssertEqual(t, "latest", latest.Key())
}
func TestLatestMetric_HandleMessage(t *testing.T) {
	latest := NewLatest()

	// Test first message.
	message1 := model.MetricMessage{
		Key:   "test_metric",
		Value: 10.5,
	}
	latest.HandleMessage(message1)

	common.AssertEqual(t, false, latest.value == nil)
	common.AssertEqual(t, message1.Value, *latest.value)

	// Test second message (should update to latest).
	message2 := model.MetricMessage{
		Key:   "test_metric",
		Value: 20.7,
	}
	latest.HandleMessage(message2)

	common.AssertEqual(t, false, latest.value == nil)
	common.AssertEqual(t, message2.Value, *latest.value)
}

func TestLatestMetric_EmitMetrics_NoValues(t *testing.T) {
	latest := NewLatest()
	source := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeGauge,
		tags:       map[string]string{"env": "test"},
	}

	messages := latest.EmitMetrics(source)

	common.AssertEqual(t, true, messages == nil)
}

func TestLatestMetric_EmitMetrics_WithValues(t *testing.T) {
	latest := NewLatest()
	source := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeGauge,
		tags:       map[string]string{"env": "test"},
	}

	// Add a value.
	message := model.MetricMessage{
		Key:   "test_metric",
		Value: 42.0,
	}
	latest.HandleMessage(message)

	messages := latest.EmitMetrics(source)

	common.AssertEqual(t, 1, len(messages))
	common.AssertEqual(t, "test_metric.latest", messages[0].Key)
	common.AssertEqual(t, model.MetricTypeGauge, messages[0].Type)
	common.AssertEqual(t, message.Value, messages[0].Value)
	common.AssertEqual(t, source.tags["env"], messages[0].Tags["env"])
	common.AssertEqual(t, 1.0, messages[0].SampleRate)
}

func TestLatestMetric_Reset(t *testing.T) {
	latest := NewLatest()

	// Add a value.
	message := model.MetricMessage{
		Key:   "test_metric",
		Value: 42.0,
	}
	latest.HandleMessage(message)

	common.AssertEqual(t, false, latest.value == nil)
	common.AssertEqual(t, message.Value, *latest.value)

	// Reset.
	latest.Reset()

	common.AssertEqual(t, true, latest.value == nil)
}

func TestLatestMetric_MultipleUpdates(t *testing.T) {
	latest := NewLatest()
	source := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeCounter,
		tags:       map[string]string{},
	}

	values := []float64{1.0, 5.0, 3.0, 8.0, 2.0}

	// Handle multiple messages.
	for _, value := range values {
		message := model.MetricMessage{
			Key:   "test_metric",
			Value: value,
		}
		latest.HandleMessage(message)
	}

	// Should only have the latest value (2.0)
	messages := latest.EmitMetrics(source)

	common.AssertEqual(t, 1, len(messages))
	common.AssertEqual(t, values[len(values)-1], messages[0].Value)
}
