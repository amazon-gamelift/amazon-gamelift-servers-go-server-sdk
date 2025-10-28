/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package metrics

import (
	"sync"
	"testing"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/derived"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/model"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/samplers"

	"github.com/golang/mock/gomock"
)

func TestBaseMetric(t *testing.T) {
	derivedMetrics := []model.DerivedMetric{
		derived.NewLatest(),
		derived.NewMax(),
	}

	base := newBaseMetric("test_metric", model.MetricTypeTimer, nil, derivedMetrics, nil, nil)

	common.AssertEqual(t, "test_metric", base.Key())
	common.AssertEqual(t, model.MetricTypeTimer, base.MetricType())

	derivedList := base.DerivedMetrics()

	common.AssertEqual(t, 2, len(derivedList))
	common.AssertEqual(t, "latest", derivedList[0].Key())
	common.AssertEqual(t, "max", derivedList[1].Key())

	testMessage := model.MetricMessage{
		Key:   "test_metric",
		Value: 42.0,
	}
	derivedList[0].HandleMessage(testMessage)
	derivedList[1].HandleMessage(testMessage)

	// Test that EmitMetrics creates the full key (base.key + "." + suffix)
	latestMessages := derivedList[0].EmitMetrics(base)
	common.AssertEqual(t, 1, len(latestMessages))
	common.AssertEqual(t, "test_metric.latest", latestMessages[0].Key)

	maxMessages := derivedList[1].EmitMetrics(base)
	common.AssertEqual(t, 1, len(maxMessages))
	common.AssertEqual(t, "test_metric.max", maxMessages[0].Key)
}

func TestBaseMetric_Tags(t *testing.T) {
	tags := map[string]string{"env": "prod", "version": "1.0"}
	base := newBaseMetric("test_metric", model.MetricTypeGauge, tags, nil, nil, nil)

	result := base.Tags()

	common.AssertEqual(t, 2, len(result))
	common.AssertEqual(t, "prod", result["env"])
	common.AssertEqual(t, "1.0", result["version"])

	// Verify it returns a copy (modification doesn't affect original)
	result["new_tag"] = "new_value"
	original := base.Tags()
	common.AssertEqual(t, 2, len(original)) // Should still be 2
}

func TestBaseMetric_SetTag(t *testing.T) {
	base := newBaseMetric("test_metric", model.MetricTypeGauge, nil, nil, nil, nil)

	base.SetTag("env", "development") //nolint:errcheck
	base.SetTag("service", "auth")    //nolint:errcheck

	tags := base.Tags()
	common.AssertEqual(t, 2, len(tags))
	common.AssertEqual(t, "development", tags["env"])
	common.AssertEqual(t, "auth", tags["service"])
}

func TestBaseMetric_SetTag_Overwrite(t *testing.T) {
	tags := map[string]string{"env": "test"}
	base := newBaseMetric("test_metric", model.MetricTypeGauge, tags, nil, nil, nil)

	base.SetTag("env", "production") //nolint:errcheck

	result := base.Tags()
	common.AssertEqual(t, 1, len(result))
	common.AssertEqual(t, "production", result["env"])
}

func TestBaseMetric_SetTags(t *testing.T) {
	base := newBaseMetric("test_metric", model.MetricTypeGauge, nil, nil, nil, nil)

	base.SetTags(map[string]string{ //nolint:errcheck
		"env":     "development",
		"service": "auth",
	})

	tags := base.Tags()
	common.AssertEqual(t, 2, len(tags))
	common.AssertEqual(t, "development", tags["env"])
	common.AssertEqual(t, "auth", tags["service"])
}

func TestBaseMetric_SetTags_Overwrite(t *testing.T) {
	tags := map[string]string{"env": "test"}
	base := newBaseMetric("test_metric", model.MetricTypeGauge, tags, nil, nil, nil)

	base.SetTags(map[string]string{ //nolint:errcheck
		"env":     "production",
		"service": "api",
	})

	result := base.Tags()
	common.AssertEqual(t, 2, len(result))
	common.AssertEqual(t, "production", result["env"])
	common.AssertEqual(t, "api", result["service"])
}

func TestBaseMetric_RemoveTag(t *testing.T) {
	tags := map[string]string{"env": "test", "service": "api", "version": "1.0"}
	base := newBaseMetric("test_metric", model.MetricTypeGauge, tags, nil, nil, nil)

	base.RemoveTag("service")

	result := base.Tags()
	common.AssertEqual(t, 2, len(result))
	common.AssertEqual(t, "test", result["env"])
	common.AssertEqual(t, "1.0", result["version"])
	common.AssertEqual(t, "", result["service"]) // Should be empty
}

func TestBaseMetric_RemoveTag_NonExistent(t *testing.T) {
	tags := map[string]string{"env": "test"}
	base := newBaseMetric("test_metric", model.MetricTypeGauge, tags, nil, nil, nil)

	// Should not panic
	base.RemoveTag("non_existent")

	result := base.Tags()
	common.AssertEqual(t, 1, len(result))
	common.AssertEqual(t, "test", result["env"])
}

func TestBaseMetric_ThreadSafety(t *testing.T) {
	base := newBaseMetric("test_metric", model.MetricTypeGauge, nil, nil, nil, nil)

	// Test concurrent tag operations
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)

		go func() {
			defer wg.Done()
			base.SetTag("tag1", "value1") //nolint:errcheck
		}()

		go func() {
			defer wg.Done()
			base.RemoveTag("tag2")
		}()

		go func() {
			defer wg.Done()
			base.Tags()
		}()
	}

	wg.Wait()
	// Should not panic and should be consistent
	tags := base.Tags()
	common.AssertEqual(t, "value1", tags["tag1"])
}

func TestBaseMetric_UpdateCurrentValue_RespectsSampler(t *testing.T) {
	t.Run("NeverSample", func(t *testing.T) {
		sampler := &testSampler{shouldSample: false} // Block all sampling
		base := newBaseMetric("test_metric", model.MetricTypeGauge, nil, nil, sampler, nil)

		initialValue := base.CurrentValue()

		// Multiple updateCurrentValue calls should be blocked by sampler
		base.updateCurrentValue(5.0, model.MetricOperationAdjust)
		base.updateCurrentValue(10.0, model.MetricOperationAdjust)
		base.updateCurrentValue(3.0, model.MetricOperationAdjust)

		// Value should remain unchanged since sampler blocks all updates
		common.AssertEqual(t, initialValue, base.CurrentValue())
	})

	t.Run("AlwaysSample", func(t *testing.T) {
		sampler := &testSampler{shouldSample: true} // Allow all sampling
		base := newBaseMetric("test_metric", model.MetricTypeGauge, nil, nil, sampler, nil)

		initialValue := base.CurrentValue()

		// All updateCurrentValue calls should go through
		base.updateCurrentValue(5.0, model.MetricOperationAdjust)
		base.updateCurrentValue(10.0, model.MetricOperationAdjust)
		base.updateCurrentValue(3.0, model.MetricOperationAdjust)

		// Value should be updated since sampler allows all updates
		expectedValue := initialValue + 18.0
		common.AssertEqual(t, expectedValue, base.CurrentValue())
	})

	t.Run("SetOperation", func(t *testing.T) {
		sampler := &testSampler{shouldSample: true}
		base := newBaseMetric("test_metric", model.MetricTypeGauge, nil, nil, sampler, nil)

		base.updateCurrentValue(42.0, model.MetricOperationSet)
		common.AssertEqual(t, 42.0, base.CurrentValue())

		base.updateCurrentValue(100.0, model.MetricOperationSet)
		common.AssertEqual(t, 100.0, base.CurrentValue())
	})

	t.Run("RealSamplers", func(t *testing.T) {
		allSampler := samplers.NewAll()
		base1 := newBaseMetric("test_all", model.MetricTypeGauge, nil, nil, allSampler, nil)

		base1.updateCurrentValue(10.0, model.MetricOperationAdjust)
		common.AssertEqual(t, 10.0, base1.CurrentValue())

		noneSampler := samplers.NewNone()
		base2 := newBaseMetric("test_none", model.MetricTypeGauge, nil, nil, noneSampler, nil)

		base2.updateCurrentValue(10.0, model.MetricOperationAdjust)
		common.AssertEqual(t, 0.0, base2.CurrentValue())
	})

	t.Run("FractionSampler", func(t *testing.T) {
		fractionZero := samplers.NewFraction(0.0)
		base1 := newBaseMetric("test_fraction_0", model.MetricTypeGauge, nil, nil, fractionZero, nil)

		base1.updateCurrentValue(10.0, model.MetricOperationAdjust)
		common.AssertEqual(t, 0.0, base1.CurrentValue())

		fractionOne := samplers.NewFraction(1.0)
		base2 := newBaseMetric("test_fraction_1", model.MetricTypeGauge, nil, nil, fractionOne, nil)

		base2.updateCurrentValue(10.0, model.MetricOperationAdjust)
		common.AssertEqual(t, 10.0, base2.CurrentValue())

		mockSampler := newTestPatternSampler([]bool{true, false, true, false, false})
		base3 := newBaseMetric("test_fraction_mock", model.MetricTypeGauge, nil, nil, mockSampler, nil)

		base3.updateCurrentValue(1.0, model.MetricOperationAdjust)
		base3.updateCurrentValue(1.0, model.MetricOperationAdjust)
		base3.updateCurrentValue(1.0, model.MetricOperationAdjust)
		base3.updateCurrentValue(1.0, model.MetricOperationAdjust)
		base3.updateCurrentValue(1.0, model.MetricOperationAdjust)

		common.AssertEqual(t, 2.0, base3.CurrentValue())
	})
}

func TestBaseMetric_DerivedMetricsReceiveMessages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)
	mockProcessor.EXPECT().enqueueMetric(gomock.Any()).AnyTimes()

	gauge, err := NewGauge("test_metric").
		WithDerivedMetrics(derived.NewLatest(), derived.NewMean(), derived.NewMin()).
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	latestMetric := gauge.derivedMetrics[0]
	meanMetric := gauge.derivedMetrics[1]
	minMetric := gauge.derivedMetrics[2]

	// Initially, no derived metrics should have values
	latestMessages := latestMetric.EmitMetrics(gauge)
	common.AssertEqual(t, 0, len(latestMessages))

	meanMessages := meanMetric.EmitMetrics(gauge)
	common.AssertEqual(t, 0, len(meanMessages))

	minMessages := minMetric.EmitMetrics(gauge)
	common.AssertEqual(t, 0, len(minMessages))

	// Use gauge.Set() which will trigger enqueueMessage and HandleMessage on all derived metrics
	gauge.Set(25.0)

	latestMessages = latestMetric.EmitMetrics(gauge)
	common.AssertEqual(t, 1, len(latestMessages))
	common.AssertEqual(t, 25.0, latestMessages[0].Value)

	meanMessages = meanMetric.EmitMetrics(gauge)
	common.AssertEqual(t, 1, len(meanMessages))
	common.AssertEqual(t, 25.0, meanMessages[0].Value)

	minMessages = minMetric.EmitMetrics(gauge)
	common.AssertEqual(t, 1, len(minMessages))
	common.AssertEqual(t, 25.0, minMessages[0].Value)

	gauge.Set(15.0)

	latestMessages = latestMetric.EmitMetrics(gauge)
	common.AssertEqual(t, 1, len(latestMessages))
	common.AssertEqual(t, 15.0, latestMessages[0].Value)

	meanMessages = meanMetric.EmitMetrics(gauge)
	common.AssertEqual(t, 1, len(meanMessages))
	common.AssertEqual(t, 20.0, meanMessages[0].Value) // (25.0 + 15.0) / 2

	minMessages = minMetric.EmitMetrics(gauge)
	common.AssertEqual(t, 1, len(minMessages))
	common.AssertEqual(t, 15.0, minMessages[0].Value)

	gauge.Set(35.0)

	meanMessages = meanMetric.EmitMetrics(gauge)
	common.AssertEqual(t, 1, len(meanMessages))
	common.AssertEqual(t, 25.0, meanMessages[0].Value) // (25.0 + 15.0 + 35.0) / 3

	minMessages = minMetric.EmitMetrics(gauge)
	common.AssertEqual(t, 1, len(minMessages))
	common.AssertEqual(t, 15.0, minMessages[0].Value)
}

// TestCreateDimensionalMetric tests the core dimensional metric creation functionality..
func TestCreateDimensionalMetric(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a base metric with initial tags and derived metrics
	baseTags := map[string]string{"service": "api", "version": "1.0"}
	derivedMetrics := []model.DerivedMetric{
		derived.NewLatest(),
		derived.NewMax(),
	}

	mockProcessor := NewMockMetricsProcessor(ctrl)
	base := newBaseMetric("request_count", model.MetricTypeCounter, baseTags, derivedMetrics, nil, mockProcessor)

	t.Run("EmptyDimensionalTags_ReturnsSelf", func(t *testing.T) {
		// Empty tags should return the same instance
		dimensional := base.createDimensionalMetric(map[string]string{})
		if dimensional != base {
			t.Error("Expected same instance for empty dimensional tags")
		}

		dimensional = base.createDimensionalMetric(nil)
		if dimensional != base {
			t.Error("Expected same instance for nil dimensional tags")
		}
	})

	t.Run("DimensionalTags_CreatesNewInstance", func(t *testing.T) {
		dimensionalTags := map[string]string{"region": "us-west", "env": "prod"}
		dimensional := base.createDimensionalMetric(dimensionalTags)

		// Should be a different instance
		if dimensional == base {
			t.Error("Expected different instance for dimensional metric")
		}

		// Should have same key and type
		common.AssertEqual(t, base.Key(), dimensional.Key())
		common.AssertEqual(t, base.MetricType(), dimensional.MetricType())

		// Should have combined tags
		combinedTags := dimensional.Tags()
		common.AssertEqual(t, 4, len(combinedTags)) // 2 base + 2 dimensional

		// Base tags should be preserved
		common.AssertEqual(t, "api", combinedTags["service"])
		common.AssertEqual(t, "1.0", combinedTags["version"])

		// Dimensional tags should be added
		common.AssertEqual(t, "us-west", combinedTags["region"])
		common.AssertEqual(t, "prod", combinedTags["env"])

		// Base metric should remain unchanged
		baseTags := base.Tags()
		common.AssertEqual(t, 2, len(baseTags))
		common.AssertEqual(t, "", baseTags["region"]) // Should not have dimensional tags
	})

	t.Run("DimensionalTags_OverwriteBaseTags", func(t *testing.T) {
		// Dimensional tags should overwrite base tags with same key
		dimensionalTags := map[string]string{"service": "auth", "region": "eu-central"}
		dimensional := base.createDimensionalMetric(dimensionalTags)

		combinedTags := dimensional.Tags()
		common.AssertEqual(t, 3, len(combinedTags)) // service overwritten, version kept, region added

		common.AssertEqual(t, "auth", combinedTags["service"])      // Overwritten
		common.AssertEqual(t, "1.0", combinedTags["version"])       // Preserved
		common.AssertEqual(t, "eu-central", combinedTags["region"]) // Added
	})

	t.Run("DerivedMetrics_AreCloned", func(t *testing.T) {
		dimensionalTags := map[string]string{"instance": "web-1"}
		dimensional := base.createDimensionalMetric(dimensionalTags)

		// Should have same number of derived metrics
		baseDerived := base.DerivedMetrics()
		dimensionalDerived := dimensional.DerivedMetrics()
		common.AssertEqual(t, len(baseDerived), len(dimensionalDerived))

		// Should have same keys but different instances
		common.AssertEqual(t, baseDerived[0].Key(), dimensionalDerived[0].Key()) // "latest"
		common.AssertEqual(t, baseDerived[1].Key(), dimensionalDerived[1].Key()) // "max"

		// Should be different instances (cloned)
		if baseDerived[0] == dimensionalDerived[0] {
			t.Error("Expected cloned derived metrics, got same instance")
		}
		if baseDerived[1] == dimensionalDerived[1] {
			t.Error("Expected cloned derived metrics, got same instance")
		}
	})

	t.Run("DerivedMetrics_IndependentState", func(t *testing.T) {
		dimensionalTags := map[string]string{"worker": "background"}
		dimensional := base.createDimensionalMetric(dimensionalTags)

		// Send different messages to base and dimensional derived metrics
		baseMessage := model.MetricMessage{Key: "request_count", Value: 100}
		dimensionalMessage := model.MetricMessage{Key: "request_count", Value: 200}

		base.DerivedMetrics()[0].HandleMessage(baseMessage)               // Latest on base = 100
		dimensional.DerivedMetrics()[0].HandleMessage(dimensionalMessage) // Latest on dimensional = 200

		// Verify independent state
		baseEmitted := base.DerivedMetrics()[0].EmitMetrics(base)
		dimensionalEmitted := dimensional.DerivedMetrics()[0].EmitMetrics(dimensional)

		if len(baseEmitted) != 1 || baseEmitted[0].Value != 100 {
			t.Errorf("Base derived metric should have value 100, got %v", baseEmitted)
		}
		if len(dimensionalEmitted) != 1 || dimensionalEmitted[0].Value != 200 {
			t.Errorf("Dimensional derived metric should have value 200, got %v", dimensionalEmitted)
		}
	})

	t.Run("CurrentValue_IndependentFromBase", func(t *testing.T) {
		dimensional := base.createDimensionalMetric(map[string]string{"test": "isolation"})

		// Both should start at 0
		common.AssertEqual(t, 0.0, base.CurrentValue())
		common.AssertEqual(t, 0.0, dimensional.CurrentValue())

		// Update base value
		base.updateCurrentValue(50.0, model.MetricOperationSet)
		common.AssertEqual(t, 50.0, base.CurrentValue())
		common.AssertEqual(t, 0.0, dimensional.CurrentValue()) // Should remain 0

		// Update dimensional value
		dimensional.updateCurrentValue(25.0, model.MetricOperationSet)
		common.AssertEqual(t, 50.0, base.CurrentValue())        // Should remain 50
		common.AssertEqual(t, 25.0, dimensional.CurrentValue()) // Should be 25
	})
}
