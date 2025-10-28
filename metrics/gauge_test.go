/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package metrics

import (
	"os"
	"sync"
	"testing"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/model"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/samplers"

	"github.com/golang/mock/gomock"
)

func assertGauge(t *testing.T, key string, value float64) func(msg model.MetricMessage) {
	t.Helper()

	return func(msg model.MetricMessage) {
		common.AssertEqual(t, key, msg.Key)
		common.AssertEqual(t, model.MetricTypeGauge, msg.Type)
		common.AssertEqual(t, value, msg.Value)
		common.AssertEqual(t, 1.0, msg.SampleRate)
	}
}

func assertGaugeWithTags(t *testing.T, key string, value float64, tags map[string]string) func(msg model.MetricMessage) {
	t.Helper()

	return func(msg model.MetricMessage) {
		assertGauge(t, key, value)(msg)

		for k, v := range tags {
			common.AssertEqual(t, v, msg.Tags[k])
		}
	}
}

// TestNewGauge tests the NewGauge function.
func TestNewGauge(t *testing.T) {
	builder := NewGauge("test_gauge")

	if builder == nil {
		t.Fatal("Expected builder to not be nil")
	}
	common.AssertEqual(t, "test_gauge", builder.key)
	if builder.baseBuilder == nil {
		t.Fatal("Expected baseBuilder to not be nil")
	}
}

// TestGaugeBuilder_WithTags tests adding multiple tags.
func TestGaugeBuilder_WithTags(t *testing.T) {
	tags := map[string]string{
		"env":     "production",
		"service": "api",
		"version": "1.0.0",
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	gauge, err := NewGauge("memory_usage").
		WithTags(tags).
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	result := gauge.Tags()
	common.AssertEqual(t, 3, len(result))
	common.AssertEqual(t, "production", result["env"])
	common.AssertEqual(t, "api", result["service"])
	common.AssertEqual(t, "1.0.0", result["version"])
}

// TestGaugeBuilder_WithTag tests adding a single tag.
func TestGaugeBuilder_WithTag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	gauge, err := NewGauge("cpu_usage").
		WithTag("region", "us-east-1").
		WithTag("app", "game-server").
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	result := gauge.Tags()
	common.AssertEqual(t, 2, len(result))
	common.AssertEqual(t, "us-east-1", result["region"])
	common.AssertEqual(t, "game-server", result["app"])
}

// TestGaugeBuilder_WithSampler tests setting a sampler.
func TestGaugeBuilder_WithSampler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}
	gauge, err := NewGauge("sampled_gauge").
		WithSampler(sampler).
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	// Verify the sampler is set by checking if it's used in the base metric.
	if gauge.baseMetric == nil {
		t.Fatal("Expected baseMetric to not be nil")
	}
	// Note: sampler usage is tested indirectly through the enqueueMessage behavior.
}

// TestGaugeBuilder_WithDerivedMetrics tests adding derived metrics.
func TestGaugeBuilder_WithDerivedMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	derived1 := &testDerivedMetric{key: "test.latest"}
	derived2 := &testDerivedMetric{key: "test.max"}

	gauge, err := NewGauge("base_metric").
		WithDerivedMetrics(derived1, derived2).
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	derivedMetrics := gauge.DerivedMetrics()
	common.AssertEqual(t, 2, len(derivedMetrics))
	common.AssertEqual(t, "test.latest", derivedMetrics[0].Key())
	common.AssertEqual(t, "test.max", derivedMetrics[1].Key())
}

// TestGaugeBuilder_ChainedCalls tests method chaining.
func TestGaugeBuilder_ChainedCalls(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	derived := &testDerivedMetric{key: "memory.latest"}
	sampler := samplers.NewAll()

	gauge, err := NewGauge("memory_utilization").
		WithTag("type", "heap").
		WithTag("process", "server").
		WithTags(map[string]string{"status": "active", "pool": "main"}).
		WithSampler(sampler).
		WithDerivedMetrics(derived).
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	// Verify all configurations are applied.
	common.AssertEqual(t, "memory_utilization", gauge.Key())
	common.AssertEqual(t, model.MetricTypeGauge, gauge.MetricType())

	tags := gauge.Tags()
	common.AssertEqual(t, 4, len(tags))
	common.AssertEqual(t, "heap", tags["type"])
	common.AssertEqual(t, "server", tags["process"])
	common.AssertEqual(t, "active", tags["status"])
	common.AssertEqual(t, "main", tags["pool"])

	derivedMetrics := gauge.DerivedMetrics()
	common.AssertEqual(t, 1, len(derivedMetrics))
	common.AssertEqual(t, "memory.latest", derivedMetrics[0].Key())
}

// TestGaugeBuilder_Build tests the Build method.
func TestGaugeBuilder_Build(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	gauge, err := NewGauge("build_test").
		WithTag("test", "true").
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	if gauge == nil {
		t.Fatal("Expected gauge to not be nil")
	}
	if gauge.baseMetric == nil {
		t.Fatal("Expected baseMetric to not be nil")
	}
	common.AssertEqual(t, "build_test", gauge.Key())
	common.AssertEqual(t, model.MetricTypeGauge, gauge.MetricType())
}

// TestGauge_Set tests the Set method.
func TestGauge_Set(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a gauge with a mock processor and sampler that always samples.
	mockProc := NewMockMetricsProcessor(ctrl)

	// Verify that enqueueMetric was called with specific MetricMessage values.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGauge(t, "set_test", 100.0)).Times(1)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGauge(t, "set_test", 50.5)).Times(1)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGauge(t, "set_test", 0.0)).Times(1)

	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	gauge, err := NewGauge("set_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Test setting various values.
	gauge.Set(100.0)
	gauge.Set(50.5)
	gauge.Set(0.0)
}

// TestGauge_Set_NegativeValue tests setting negative values.
func TestGauge_Set_NegativeValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	// Expect negative values to be enqueued.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGauge(t, "negative_test", -50.0)).Times(1)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGauge(t, "negative_test", -100.5)).Times(1)

	gauge, err := NewGauge("negative_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Gauge should handle negative values.
	gauge.Set(-50.0)
	gauge.Set(-100.5)
}

// TestGauge_Add tests the Add method.
func TestGauge_Add(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	// Verify that enqueueMetric was called with delta values (not cumulative).
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGauge(t, "add_test", 10.0)).Times(1)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGauge(t, "add_test", 25.5)).Times(1)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGauge(t, "add_test", 0.75)).Times(1)

	gauge, err := NewGauge("add_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Test adding various values.
	gauge.Add(10.0)
	gauge.Add(25.5)
	gauge.Add(0.75)
}

// TestGauge_Subtract tests the Subtract method.
func TestGauge_Subtract(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	// Verify that enqueueMetric was called with negative delta values.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGauge(t, "subtract_test", -5.0)).Times(1)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGauge(t, "subtract_test", -10.5)).Times(1)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGauge(t, "subtract_test", -0.25)).Times(1)

	gauge, err := NewGauge("subtract_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Test subtracting various values.
	gauge.Subtract(5.0)
	gauge.Subtract(10.5)
	gauge.Subtract(0.25)
}

// TestGauge_MessageBehavior tests that Set sends absolute values and Add/Subtract send deltas.
func TestGauge_MessageBehavior(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	gauge, err := NewGauge("message_behavior_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Set operations should send absolute values.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGauge(t, "message_behavior_test", 100.0)).Times(1)
	gauge.Set(100.0)

	// Add operations should send delta values (not cumulative).
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGauge(t, "message_behavior_test", 20.0)).Times(1) // Delta: +20
	gauge.Add(20.0)

	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGauge(t, "message_behavior_test", 5.0)).Times(1) // Delta: +5
	gauge.Add(5.0)

	// Subtract operations should send negative delta values.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGauge(t, "message_behavior_test", -10.0)).Times(1) // Delta: -10
	gauge.Subtract(10.0)

	// Another set should send absolute value.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGauge(t, "message_behavior_test", 50.0)).Times(1)
	gauge.Set(50.0)

	// Verify internal state is maintained correctly.
	common.AssertEqual(t, 50.0, gauge.CurrentValue())
}

// TestGauge_Increment tests the Increment method.
func TestGauge_Increment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	gauge, err := NewGauge("increment_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Verify that enqueueMetric was called with delta value 1.0 for each increment
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGauge(t, "increment_test", 1.0)).Times(3)
	// Test multiple increments.
	gauge.Increment()
	gauge.Increment()
	gauge.Increment()
}

// TestGauge_Decrement tests the Decrement method.
func TestGauge_Decrement(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	gauge, err := NewGauge("decrement_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Verify that enqueueMetric was called with delta value -1.0 for each decrement
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGauge(t, "decrement_test", -1.0)).Times(3)

	// Test multiple decrements.
	gauge.Decrement()
	gauge.Decrement()
	gauge.Decrement()
}

// TestGauge_Reset tests the Reset method.
func TestGauge_Reset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	gauge, err := NewGauge("reset_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Verify that enqueueMetric was called with value 0.0 (reset implementation)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGauge(t, "reset_test", 0.0)).Times(2)

	// Test multiple resets.
	gauge.Reset()
	gauge.Reset()
}

// TestGauge_SamplingBehavior tests that sampling controls message enqueueing.
func TestGauge_SamplingBehavior(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)

	t.Run("AlwaysSample", func(t *testing.T) {
		mockProc.EXPECT().registerMetric(gomock.Any()).Times(2)

		sampler := &testSampler{shouldSample: true}

		gauge, err := NewGauge("sampling_test").
			WithSampler(sampler).
			WithMetricsProcessor(mockProc).
			Build()
		common.AssertEqual(t, nil, err)

		mockProc.EXPECT().enqueueMetric(gomock.Any()).
			Do(assertGauge(t, "sampling_test", 42.0)).Times(1) // Set(42.0) -> absolute
		mockProc.EXPECT().enqueueMetric(gomock.Any()).
			Do(assertGauge(t, "sampling_test", 5.0)).Times(1) // Add(5.0) -> delta
		mockProc.EXPECT().enqueueMetric(gomock.Any()).
			Do(assertGauge(t, "sampling_test", -3.0)).Times(1) // Subtract(3.0) -> delta
		mockProc.EXPECT().enqueueMetric(gomock.Any()).
			Do(assertGauge(t, "sampling_test", 1.0)).Times(1) // Increment() -> delta
		mockProc.EXPECT().enqueueMetric(gomock.Any()).
			Do(assertGauge(t, "sampling_test", -1.0)).Times(1) // Decrement() -> delta
		mockProc.EXPECT().enqueueMetric(gomock.Any()).
			Do(assertGauge(t, "sampling_test", 0.0)).Times(1) // Reset() -> absolute

		gauge.Set(42.0)
		gauge.Add(5.0)
		gauge.Subtract(3.0)
		gauge.Increment()
		gauge.Decrement()
		gauge.Reset()
	})

	t.Run("NeverSample", func(t *testing.T) {
		sampler := &testSampler{shouldSample: false}

		gauge, err := NewGauge("sampling_test").
			WithSampler(sampler).
			WithMetricsProcessor(mockProc).
			Build()
		common.AssertEqual(t, nil, err)

		// Expect no calls when sampling is disabled.
		mockProc.EXPECT().enqueueMetric(gomock.Any()).Times(0)

		gauge.Set(42.0)
		gauge.Add(5.0)
		gauge.Subtract(3.0)
		gauge.Increment()
		gauge.Decrement()
		gauge.Reset()
	})
}

// TestGauge_MessageTags tests that tags are properly included in enqueued messages.
func TestGauge_MessageTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)

	// Expect enqueueMetric to be called with specific tags.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGaugeWithTags(t, "tagged_gauge", 75.5, map[string]string{
			"env":     "test",
			"service": "game-server",
			"runtime": "dynamic",
		})).Times(1)

	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	gauge, err := NewGauge("tagged_gauge").
		WithTag("env", "test").
		WithTag("service", "game-server").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Add a tag after creation.
	gauge.SetTag("runtime", "dynamic") //nolint:errcheck

	gauge.Set(75.5)
}

// TestGauge_ThreadSafety tests concurrent access to gauge methods.
func TestGauge_ThreadSafety(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	// For thread safety test, we expect many calls but don't need to verify exact count.
	// due to concurrent nature - just verify no panics occur.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).AnyTimes()

	gauge, err := NewGauge("thread_safety_test").
		WithTag("concurrent", "true").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	var wg sync.WaitGroup
	numGoroutines := 10
	operationsPerGoroutine := 10

	// Test concurrent Set operations.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				gauge.Set(float64(id*10 + j))
			}
		}(i)
	}

	// Test concurrent Add operations.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				gauge.Add(1.0)
			}
		}()
	}

	// Test concurrent Subtract operations.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				gauge.Subtract(0.5)
			}
		}()
	}

	// Test concurrent Increment/Decrement operations.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				if id%2 == 0 {
					gauge.Increment()
				} else {
					gauge.Decrement()
				}
			}
		}(i)
	}

	// Test concurrent Reset operations.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				gauge.Reset()
			}
		}()
	}

	// Test concurrent tag operations.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			gauge.SetTag("worker", string(rune('A'+id%26))) //nolint:errcheck
			gauge.Tags()
		}(i)
	}

	wg.Wait()

	// Verify the gauge is still functional after concurrent access.
	gauge.Set(100.0)
	common.AssertEqual(t, model.MetricTypeGauge, gauge.MetricType())
}

// TestGauge_BuilderReuse tests that builder can be reused.
func TestGauge_BuilderReuse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(2)

	sampler := &testSampler{shouldSample: true}

	// Expect 2 Set operations.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).Times(2)

	builder := NewGauge("reuse_test").
		WithTag("shared", "value").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc)

		// Build first gauge.
	gauge1, err := builder.Build()
	common.AssertEqual(t, nil, err)

	gauge1.SetTag("instance", "1") //nolint:errcheck

	// Build second gauge from same builder.
	gauge2, err := builder.Build()
	common.AssertEqual(t, nil, err)
	gauge2.SetTag("instance", "2") //nolint:errcheck

	// Both gauges should have the shared tag but different instance tags.
	tags1 := gauge1.Tags()
	tags2 := gauge2.Tags()

	common.AssertEqual(t, "value", tags1["shared"])
	common.AssertEqual(t, "value", tags2["shared"])
	common.AssertEqual(t, "1", tags1["instance"])
	common.AssertEqual(t, "2", tags2["instance"])

	// Both should be independent.
	gauge1.Set(25.0)
	gauge2.Set(75.0)
}

// TestGauge_DimensionalTags tests dimensional tag functionality.
func TestGauge_DimensionalTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)
	// Expect DimensionalMetricsEnabled to be called when using WithTags.
	mockProc.EXPECT().dimensionalMetricsEnabled().Return(true).AnyTimes()

	sampler := &testSampler{shouldSample: true}

	gauge, err := NewGauge("dimensional_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Test Set with dimensional tags.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGaugeWithTags(t, "dimensional_test", 100.0, map[string]string{
			"region": "us-east",
			"env":    "prod",
		})).Times(1)

	gauge.WithDimensionalTags(map[string]string{"region": "us-east", "env": "prod"}).Set(100.0)

	// Test Add with dimensional tags.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGaugeWithTags(t, "dimensional_test", 50.0, map[string]string{
			"region": "us-west",
			"env":    "staging",
		})).Times(1)

	gauge.WithDimensionalTags(map[string]string{"region": "us-west", "env": "staging"}).Add(50.0)

	// Test Subtract with dimensional tags.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGaugeWithTags(t, "dimensional_test", -25.0, map[string]string{
			"region": "eu-central",
			"env":    "dev",
		})).Times(1)

	gauge.WithDimensionalTags(map[string]string{"region": "eu-central", "env": "dev"}).Subtract(25.0)

	// Test Increment with dimensional tags.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGaugeWithTags(t, "dimensional_test", 1.0, map[string]string{
			"region": "ap-south",
			"env":    "test",
		})).Times(1)

	gauge.WithDimensionalTags(map[string]string{"region": "ap-south", "env": "test"}).Increment()

	// Test Decrement with dimensional tags.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGaugeWithTags(t, "dimensional_test", -1.0, map[string]string{
			"region": "ap-northeast",
			"env":    "prod",
		})).Times(1)

	gauge.WithDimensionalTags(map[string]string{"region": "ap-northeast", "env": "prod"}).Decrement()

	// Test Reset with dimensional tags.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGaugeWithTags(t, "dimensional_test", 0.0, map[string]string{
			"region": "ca-central",
			"env":    "staging",
		})).Times(1)

	gauge.WithDimensionalTags(map[string]string{"region": "ca-central", "env": "staging"}).Reset()
}

// TestGauge_WithTagsMethod tests the WithTags method for creating dimensional variants.
func TestGauge_WithTagsMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).AnyTimes()
	mockProc.EXPECT().dimensionalMetricsEnabled().Return(true).AnyTimes()

	baseGauge, err := NewGauge("base_test").
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Create dimensional variant.
	regionalGauge := baseGauge.WithDimensionalTags(map[string]string{"region": "us-west"})

	// Verify tags are different.
	baseTags := baseGauge.Tags()
	regionalTags := regionalGauge.Tags()

	if len(baseTags) != 0 {
		t.Errorf("Base gauge should have no tags, got %v", baseTags)
	}

	if regionalTags["region"] != "us-west" {
		t.Errorf("Regional gauge should have region=us-west, got %s", regionalTags["region"])
	}
}

// TestGauge_WithDimensionalTagMethod tests the WithDimensionalTag method for single tag addition.
func TestGauge_WithDimensionalTagMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Enable dimensional metrics for this test.
	t.Setenv(EnableDimensionalMetricsEnvVar, "true")

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)
	mockProc.EXPECT().dimensionalMetricsEnabled().Return(true).AnyTimes()

	sampler := &testSampler{shouldSample: true}

	baseGauge, err := NewGauge("withtag_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Test single tag addition and verify tags.
	taggedGauge := baseGauge.WithDimensionalTag("env", "staging")

	baseTags := baseGauge.Tags()
	taggedTags := taggedGauge.Tags()

	common.AssertEqual(t, 0, len(baseTags))
	common.AssertEqual(t, 1, len(taggedTags))
	common.AssertEqual(t, "staging", taggedTags["env"])

	// Test functional operation on tagged gauge.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGaugeWithTags(t, "withtag_test", 42.5, map[string]string{
			"env": "staging",
		})).Times(1)

	taggedGauge.Set(42.5)
}

// TestGauge_WithDimensionalTag_Chaining tests chaining multiple WithDimensionalTag calls.
func TestGauge_WithDimensionalTag_Chaining(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Enable dimensional metrics for this test.
	t.Setenv(EnableDimensionalMetricsEnvVar, "true")

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)
	mockProc.EXPECT().dimensionalMetricsEnabled().Return(true).AnyTimes()

	sampler := &testSampler{shouldSample: true}

	baseGauge, err := NewGauge("withtag_chain_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Test chaining multiple WithTag calls.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGaugeWithTags(t, "withtag_chain_test", 75.0, map[string]string{
			"datacenter": "us-east-1",
			"instance":   "web-01",
		})).Times(1)

	baseGauge.
		WithDimensionalTag("datacenter", "us-east-1").
		WithDimensionalTag("instance", "web-01").
		Set(75.0)

		// Verify that the base gauge remains unchanged after dimensional operations.
	baseTags := baseGauge.Tags()
	if len(baseTags) != 0 {
		t.Errorf("Base gauge should remain unchanged with no tags after dimensional operations, got %v", baseTags)
	}
}

// TestGauge_WithDimensionalTags_Functional tests WithDimensionalTags with multiple operations.
func TestGauge_WithDimensionalTags_Functional(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Enable dimensional metrics for this test.
	t.Setenv(EnableDimensionalMetricsEnvVar, "true")

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)
	mockProc.EXPECT().dimensionalMetricsEnabled().Return(true).AnyTimes()

	sampler := &testSampler{shouldSample: true}

	baseGauge, err := NewGauge("withtags_func_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	tags := map[string]string{
		"region": "ap-southeast",
		"type":   "memory",
	}

	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGaugeWithTags(t, "withtags_func_test", 100.0, tags)).Times(1) // Set(100.0) -> absolute
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGaugeWithTags(t, "withtags_func_test", 25.0, tags)).Times(1) // Add(25.0) -> delta
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGaugeWithTags(t, "withtags_func_test", -5.0, tags)).Times(1) // Subtract(5.0) -> delta
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertGaugeWithTags(t, "withtags_func_test", 0.0, tags)).Times(1) // Reset() -> absolute

	taggedGauge := baseGauge.WithDimensionalTags(tags)
	taggedGauge.Set(100.0)
	taggedGauge.Add(25.0)
	taggedGauge.Subtract(5.0)
	taggedGauge.Reset()

	// Verify base gauge remains unchanged.
	baseTags := baseGauge.Tags()
	common.AssertEqual(t, 0, len(baseTags))
}

func TestGauge_DimensionalBehaviorWhenDisabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	os.Unsetenv(EnableDimensionalMetricsEnvVar)
	defer os.Unsetenv(EnableDimensionalMetricsEnvVar)

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().dimensionalMetricsEnabled().Return(false).AnyTimes()
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1) // Only base gauge registered

	// Create the base gauge - this is the only metric that gets registered.
	baseGauge, err := NewGauge("memory_usage").
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	t.Run("WithTag_ModifiesSameInstance", func(t *testing.T) {
		// In disabled mode, WithTag() returns the same gauge and applies the tag directly.
		taggedGauge := baseGauge.WithTag("component", "cache")

		// Verify same instance returned.
		if baseGauge != taggedGauge {
			t.Error("Expected WithTag to return same gauge instance when dimensional metrics disabled")
		}

		// Tag was applied directly to the base gauge.
		tags := baseGauge.Tags()
		common.AssertEqual(t, "cache", tags["component"])

		// Both references show the same tags.
		taggedTags := taggedGauge.Tags()
		common.AssertEqual(t, "cache", taggedTags["component"])
	})

	t.Run("WithTags_AccumulatesOnSameInstance", func(t *testing.T) {
		// WithTags() also modifies the same instance, accumulating all tags.
		additionalTags := map[string]string{
			"server":      "redis-001",
			"environment": "production",
			"team":        "platform",
		}

		taggedGauge := baseGauge.WithTags(additionalTags)

		// Same instance returned.
		if baseGauge != taggedGauge {
			t.Error("Expected WithTags to return same gauge instance when dimensional metrics disabled")
		}

		// All tags accumulated on the base gauge.
		allTags := baseGauge.Tags()
		common.AssertEqual(t, "cache", allTags["component"]) // From previous test
		common.AssertEqual(t, "redis-001", allTags["server"])
		common.AssertEqual(t, "production", allTags["environment"])
		common.AssertEqual(t, "platform", allTags["team"])
	})

	t.Run("WithDimensionalTags_BehavesLikeRegularTags", func(t *testing.T) {
		// WithDimensionalTags() is intended for volatile high-cardinality tags,.
		// but when disabled, it behaves just like regular tagging.
		volatileTags := map[string]string{
			"request_id": "req_abc123",
			"user_id":    "user_456",
		}

		dimensionalGauge := baseGauge.WithDimensionalTags(volatileTags)

		// Same instance returned - no dimensional behavior.
		if baseGauge != dimensionalGauge {
			t.Error("Expected WithDimensionalTags to return same gauge when dimensional metrics disabled")
		}
	})

	t.Run("SharedState_UnifiedBehavior", func(t *testing.T) {
		// All operations affect the same underlying gauge state.
		mockProc.EXPECT().enqueueMetric(gomock.Any()).Times(6) // 6 operations below

		// Create various "tagged" references - all point to same gauge.
		cacheGauge := baseGauge.WithTag("type", "cache")
		dbGauge := baseGauge.WithTag("type", "database") // Overwrites "type" tag
		userGauge := baseGauge.WithDimensionalTags(map[string]string{"scope": "user"})

		// All are the same instance.
		common.AssertEqual(t, baseGauge, cacheGauge)
		common.AssertEqual(t, baseGauge, dbGauge)
		common.AssertEqual(t, baseGauge, userGauge)

		// Operations on any reference affect the shared state.
		cacheGauge.Set(100.0)    // baseGauge = 100.0
		dbGauge.Add(50.0)        // baseGauge = 150.0
		userGauge.Subtract(25.0) // baseGauge = 125.0
		cacheGauge.Increment()   // baseGauge = 126.0
		dbGauge.Decrement()      // baseGauge = 125.0
		userGauge.Reset()        // baseGauge = 0.0

		// All references show the same final value.
		common.AssertEqual(t, 0.0, baseGauge.CurrentValue())
		common.AssertEqual(t, 0.0, cacheGauge.CurrentValue())
		common.AssertEqual(t, 0.0, dbGauge.CurrentValue())
		common.AssertEqual(t, 0.0, userGauge.CurrentValue())
	})

	t.Run("TagOverwriting_LastValueWins", func(t *testing.T) {
		// When disabled, tags are applied directly to the base gauge.
		// Multiple calls with the same key will overwrite previous values.

		baseGauge.WithTag("environment", "staging")
		baseGauge.WithTag("environment", "production") // Overwrites previous
		baseGauge.WithTags(map[string]string{
			"environment": "development", // Overwrites again
			"new_tag":     "new_value",
		})

		finalTags := baseGauge.Tags()
		common.AssertEqual(t, "development", finalTags["environment"]) // Last value wins
		common.AssertEqual(t, "new_value", finalTags["new_tag"])
	})

	t.Run("MemoryEfficient_SingleMetricOnly", func(t *testing.T) {
		// These calls would create 4 separate gauges when dimensional metrics enabled.
		perfGauge := baseGauge.WithTag("metric_type", "performance")
		errorGauge := baseGauge.WithTag("metric_type", "errors")
		userGauge := baseGauge.WithDimensionalTags(map[string]string{"user_segment": "premium"})
		requestGauge := baseGauge.WithDimensionalTags(map[string]string{"request_type": "api"})

		// But when disabled, they're all the same single gauge instance.
		common.AssertEqual(t, baseGauge, perfGauge)
		common.AssertEqual(t, baseGauge, errorGauge)
		common.AssertEqual(t, baseGauge, userGauge)
		common.AssertEqual(t, baseGauge, requestGauge)
	})
}
