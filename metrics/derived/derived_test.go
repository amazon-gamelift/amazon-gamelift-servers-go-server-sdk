/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package derived

import (
	"testing"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/model"
)

// testMetric implements the model.Metric interface for testing.
type testMetric struct {
	tags         map[string]string
	key          string
	metricType   model.MetricType
	currentValue float64
}

func (m *testMetric) Key() string                           { return m.key }
func (m *testMetric) MetricType() model.MetricType          { return m.metricType }
func (m *testMetric) DerivedMetrics() []model.DerivedMetric { return nil }
func (m *testMetric) CurrentValue() float64                 { return m.currentValue }
func (m *testMetric) Tags() map[string]string               { return m.tags }
func (m *testMetric) RemoveTag(key string)                  { delete(m.tags, key) }
func (m *testMetric) SetTag(key, value string) error {
	if err := model.ValidateTagKey(key); err != nil {
		return err
	}
	if err := model.ValidateTagValue(value); err != nil {
		return err
	}
	m.tags[key] = value
	return nil
}
func (m *testMetric) SetTags(tags map[string]string) error {
	for key, value := range tags {
		if err := model.ValidateTagKey(key); err != nil {
			return err
		}
		if err := model.ValidateTagValue(value); err != nil {
			return err
		}
		m.tags[key] = value
	}
	return nil
}

// TestMaxMetricClone tests that cloning creates independent instances.
func TestMaxMetricClone(t *testing.T) {
	original := NewMax()

	// Add a value to the original.
	msg1 := model.MetricMessage{
		Key:   "test_metric",
		Type:  model.MetricTypeGauge,
		Value: 100,
		Tags:  map[string]string{"region": "us-east"},
	}
	original.HandleMessage(msg1)

	// Clone the metric.
	cloned := original.Clone()

	// Add a different value to the clone.
	msg2 := model.MetricMessage{
		Key:   "test_metric",
		Type:  model.MetricTypeGauge,
		Value: 200,
		Tags:  map[string]string{"region": "us-west"},
	}
	cloned.HandleMessage(msg2)
	// Create test metrics for emission.
	metric1 := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeGauge,
		tags:       map[string]string{"region": "us-east"},
	}
	metric2 := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeGauge,
		tags:       map[string]string{"region": "us-west"},
	}
	// Verify they have independent values.
	emitted1 := original.EmitMetrics(metric1)
	if len(emitted1) != 1 || emitted1[0].Value != 100 {
		t.Errorf("Expected original to emit 100, got %v", emitted1)
	}
	emitted2 := cloned.EmitMetrics(metric2)
	if len(emitted2) != 1 || emitted2[0].Value != 200 {
		t.Errorf("Expected cloned to emit 200, got %v", emitted2)
	}
}

// TestSimpleDerivedMetricCloneIsolation tests that cloned simple derived metrics maintain independent state.
func TestSimpleDerivedMetricCloneIsolation(t *testing.T) {
	original := NewLatest()
	cloned, ok := original.Clone().(*LatestMetric)
	if !ok {
		t.Fatal("Clone should return LatestMetric")
	}

	// Send different values to original and cloned.
	original.HandleMessage(model.MetricMessage{Value: 100})
	cloned.HandleMessage(model.MetricMessage{Value: 200})

	// Create test metrics for emission.
	testMetric := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeGauge,
		tags:       map[string]string{"region": "us-east"},
	}

	// Verify independent state.
	emittedOriginal := original.EmitMetrics(testMetric)
	emittedCloned := cloned.EmitMetrics(testMetric)

	if len(emittedOriginal) != 1 || emittedOriginal[0].Value != 100 {
		t.Errorf("Original should emit 100, got %v", emittedOriginal)
	}
	if len(emittedCloned) != 1 || emittedCloned[0].Value != 200 {
		t.Errorf("Cloned should emit 200, got %v", emittedCloned)
	}
}

// TestAccumulatingDerivedMetricCloneIsolation tests that cloned accumulating derived metrics maintain independent state.
func TestAccumulatingDerivedMetricCloneIsolation(t *testing.T) {
	original := NewMean()
	cloned, ok := original.Clone().(*MeanMetric)
	if !ok {
		t.Fatal("Clone should return MeanMetric")
	}

	// Send different value sequences to build up independent state.
	for i := 1; i <= 3; i++ {
		original.HandleMessage(model.MetricMessage{Value: float64(i)})    // 1,2,3 -> mean=2
		cloned.HandleMessage(model.MetricMessage{Value: float64(i * 10)}) // 10,20,30 -> mean=20
	}

	testMetric := &testMetric{
		key:        "test_metric",
		metricType: model.MetricTypeGauge,
	}

	emittedOriginal := original.EmitMetrics(testMetric)
	emittedCloned := cloned.EmitMetrics(testMetric)

	if len(emittedOriginal) != 1 || emittedOriginal[0].Value != 2.0 {
		t.Errorf("Original mean should be 2.0, got %v", emittedOriginal)
	}
	if len(emittedCloned) != 1 || emittedCloned[0].Value != 20.0 {
		t.Errorf("Cloned mean should be 20.0, got %v", emittedCloned)
	}
}
