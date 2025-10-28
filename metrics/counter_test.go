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

func assertCounter(t *testing.T, key string, value float64) func(msg model.MetricMessage) {
	t.Helper()

	return func(msg model.MetricMessage) {
		common.AssertEqual(t, key, msg.Key)
		common.AssertEqual(t, model.MetricTypeCounter, msg.Type)
		common.AssertEqual(t, value, msg.Value)
		common.AssertEqual(t, 1.0, msg.SampleRate)
	}
}

func assertCounterWithTags(t *testing.T, key string, value float64, tags map[string]string) func(msg model.MetricMessage) {
	t.Helper()

	return func(msg model.MetricMessage) {
		assertCounter(t, key, value)(msg)

		for k, v := range tags {
			common.AssertEqual(t, v, msg.Tags[k])
		}
	}
}

// TestFactoryCounter tests creating counters via factory.
func TestFactoryCounter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCrashReporter := NewMockCrashReporter(ctrl)
	mockTransport := NewMockTransport(ctrl)
	err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	factory, err := NewFactory().
		WithCrashReporter(mockCrashReporter).
		WithProcessor(processor).
		Build()
	common.AssertEqual(t, nil, err)

	counter, err := factory.Counter("test_counter")
	common.AssertEqual(t, nil, err)

	common.AssertEqual(t, "test_counter", counter.Key())
	common.AssertEqual(t, 0.0, counter.CurrentValue())
}

// TestFactory_CounterWithTags tests creating counters with default tags.
func TestFactory_CounterWithTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransport := NewMockTransport(ctrl)
	mockCrashReporter := NewMockCrashReporter(ctrl)
	err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	tags := map[string]string{
		"env":     "production",
		"service": "api",
		"version": "1.0.0",
	}

	factory, err := NewFactory().
		WithCrashReporter(mockCrashReporter).
		WithProcessor(processor).
		WithTags(tags).
		Build()
	common.AssertEqual(t, nil, err)

	counter, err := factory.Counter("requests_total")
	common.AssertEqual(t, nil, err)

	result := counter.Tags()
	common.AssertEqual(t, 3, len(result))
	common.AssertEqual(t, "production", result["env"])
	common.AssertEqual(t, "api", result["service"])
	common.AssertEqual(t, "1.0.0", result["version"])
}

// TestCounterBuilder_WithTag tests adding a single tag.
func TestCounterBuilder_WithTag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	counter, err := NewCounter("user_logins").
		WithTag("region", "us-east-1").
		WithTag("app", "game-server").
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	result := counter.Tags()
	common.AssertEqual(t, 2, len(result))
	common.AssertEqual(t, "us-east-1", result["region"])
	common.AssertEqual(t, "game-server", result["app"])
}

// TestCounterBuilder_WithSampler tests setting a sampler.
func TestCounterBuilder_WithSampler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}
	counter, err := NewCounter("sampled_events").
		WithSampler(sampler).
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	// Verify the sampler is set by checking if it's used in the base metric.
	if counter.baseMetric == nil {
		t.Fatal("Expected baseMetric to not be nil")
	}
	// Note: sampler usage is tested indirectly through the enqueueMessage behavior.
}

// TestCounterBuilder_WithDerivedMetrics tests adding derived metrics.
func TestCounterBuilder_WithDerivedMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	derived1 := &testDerivedMetric{key: "test.latest"}
	derived2 := &testDerivedMetric{key: "test.max"}

	counter, err := NewCounter("base_metric").
		WithDerivedMetrics(derived1, derived2).
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	derivedMetrics := counter.DerivedMetrics()
	common.AssertEqual(t, 2, len(derivedMetrics))
	common.AssertEqual(t, "test.latest", derivedMetrics[0].Key())
	common.AssertEqual(t, "test.max", derivedMetrics[1].Key())
}

// TestCounterBuilder_ChainedCalls tests method chaining.
func TestCounterBuilder_ChainedCalls(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	derived := &testDerivedMetric{key: "requests.latest"}
	sampler := samplers.NewAll()

	counter, err := NewCounter("http_requests").
		WithTag("method", "GET").
		WithTag("endpoint", "/api/users").
		WithTags(map[string]string{"status": "200", "cached": "false"}).
		WithSampler(sampler).
		WithDerivedMetrics(derived).
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	// Verify all configurations are applied.
	common.AssertEqual(t, "http_requests", counter.Key())
	common.AssertEqual(t, model.MetricTypeCounter, counter.MetricType())

	tags := counter.Tags()
	common.AssertEqual(t, 4, len(tags))
	common.AssertEqual(t, "GET", tags["method"])
	common.AssertEqual(t, "/api/users", tags["endpoint"])
	common.AssertEqual(t, "200", tags["status"])
	common.AssertEqual(t, "false", tags["cached"])

	derivedMetrics := counter.DerivedMetrics()
	common.AssertEqual(t, 1, len(derivedMetrics))
	common.AssertEqual(t, "requests.latest", derivedMetrics[0].Key())
}

// TestCounterBuilder_Build tests the Build method.
func TestCounterBuilder_Build(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	counter, err := NewCounter("build_test").
		WithTag("test", "true").
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	if counter == nil {
		t.Fatal("Expected counter to not be nil")
	}
	if counter.baseMetric == nil {
		t.Fatal("Expected baseMetric to not be nil")
	}
	common.AssertEqual(t, "build_test", counter.Key())
	common.AssertEqual(t, model.MetricTypeCounter, counter.MetricType())
}

// TestCounter_Add tests the Add method.
func TestCounter_Add(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a counter with a mock processor and sampler that always samples.
	mockProc := NewMockMetricsProcessor(ctrl)

	// Verify that enqueueMetric was called with delta values (not cumulative).
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertCounter(t, "add_test", 5.0)).Times(1)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertCounter(t, "add_test", 10.5)).Times(1)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertCounter(t, "add_test", 0.25)).Times(1)

	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	counter, err := NewCounter("add_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Test adding various values.
	counter.Add(5.0)
	counter.Add(10.5)
	counter.Add(0.25)
}

// TestCounter_Add_NegativeValue tests adding negative values.
func TestCounter_Add_NegativeValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	// Expect negative values to be enqueued (though conceptually unusual).
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertCounter(t, "negative_test", -1.0)).Times(1)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertCounter(t, "negative_test", -10.5)).Times(1)

	counter, err := NewCounter("negative_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Counter should handle negative values (though conceptually unusual).
	// This tests that the method doesn't panic with edge case inputs.
	counter.Add(-1.0)
	counter.Add(-10.5)
}

// TestCounter_Add_ZeroValue tests adding zero values.
func TestCounter_Add_ZeroValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	// Expect zero value to be enqueued.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertCounter(t, "zero_test", 0.0)).Times(1)

	counter, err := NewCounter("zero_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Should handle zero values without issue.
	counter.Add(0.0)
}

// TestCounter_Increment tests the Increment method.
func TestCounter_Increment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	counter, err := NewCounter("increment_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertCounter(t, "increment_test", 1.0)).Times(1)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertCounter(t, "increment_test", 1.0)).Times(1)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertCounter(t, "increment_test", 1.0)).Times(1)

	// Test multiple increments.
	counter.Increment()
	counter.Increment()
	counter.Increment()
}

// TestCounter_Count tests the Count method.
func TestCounter_Count(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	counter, err := NewCounter("count_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Verify that enqueueMetric was called with delta value of 1.0 for each Count(true)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertCounter(t, "count_test", 1.0)).Times(7)

	// Test with true condition.
	counter.Count(true) // Should increment
	counter.Count(true) // Should increment

	// Test with false condition.
	counter.Count(false) // Should not increment
	counter.Count(false) // Should not increment

	// Test with dynamic conditions.
	for i := 0; i < 10; i++ {
		counter.Count(i%2 == 0) // Should increment 5 times (even numbers)
	}
}

// TestCounter_SamplingBehavior tests that sampling controls message enqueueing.
func TestCounter_SamplingBehavior(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)

	t.Run("AlwaysSample", func(t *testing.T) {
		mockProc.EXPECT().registerMetric(gomock.Any()).Times(2)

		sampler := &testSampler{shouldSample: true}

		counter, err := NewCounter("sampling_test").
			WithSampler(sampler).
			WithMetricsProcessor(mockProc).
			Build()
		common.AssertEqual(t, nil, err)

		mockProc.EXPECT().enqueueMetric(gomock.Any()).
			Do(assertCounter(t, "sampling_test", 1.0)).Times(1)
		mockProc.EXPECT().enqueueMetric(gomock.Any()).
			Do(assertCounter(t, "sampling_test", 1.0)).Times(1)
		mockProc.EXPECT().enqueueMetric(gomock.Any()).
			Do(assertCounter(t, "sampling_test", 1.0)).Times(1)

		counter.Add(1.0)
		counter.Increment()
		counter.Count(true)
	})

	t.Run("NeverSample", func(t *testing.T) {
		sampler := &testSampler{shouldSample: false}

		counter, err := NewCounter("sampling_test").
			WithSampler(sampler).
			WithMetricsProcessor(mockProc).
			Build()
		common.AssertEqual(t, nil, err)

		mockProc.EXPECT().enqueueMetric(gomock.Any()).
			Do(assertCounter(t, "sampling_test", 1.0)).Times(0)

		counter.Add(1.0)
		counter.Increment()
		counter.Count(true)
	})
}

// TestCounter_MessageTags tests that tags are properly included in enqueued messages.
func TestCounter_MessageTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)

	// Expect enqueueMetric to be called with specific tags.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertCounterWithTags(t, "tagged_counter", 42.0, map[string]string{
			"env":     "test",
			"service": "game-server",
			"runtime": "dynamic",
		})).Times(1)

	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	counter, err := NewCounter("tagged_counter").
		WithTag("env", "test").
		WithTag("service", "game-server").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Add a tag after creation.
	counter.SetTag("runtime", "dynamic") //nolint:errcheck

	counter.Add(42.0)
}

// TestCounter_ThreadSafety tests concurrent access to counter methods.
func TestCounter_ThreadSafety(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	// For thread safety test, we expect many calls but don't need to verify exact count.
	// due to concurrent nature - just verify no panics occur.
	mockProc.EXPECT().enqueueMetric(gomock.AssignableToTypeOf(model.MetricMessage{})).AnyTimes()

	counter, err := NewCounter("thread_safety_test").
		WithTag("concurrent", "true").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	var wg sync.WaitGroup
	numGoroutines := 10
	incrementsPerGoroutine := 10

	// Test concurrent Add operations.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				counter.Add(1.0)
			}
		}()
	}

	// Test concurrent Increment operations.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				counter.Increment()
			}
		}()
	}

	// Test concurrent Count operations.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				counter.Count(id%2 == 0) // Half true, half false
			}
		}(i)
	}

	// Test concurrent tag operations.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			counter.SetTag("worker", string(rune('A'+id%26))) //nolint:errcheck
			counter.Tags()
		}(i)
	}

	wg.Wait()

	// Verify the counter is still functional after concurrent access.
	counter.Increment()
	common.AssertEqual(t, model.MetricTypeCounter, counter.MetricType())
}

// TestCounter_BuilderReuse tests that builder can be reused.
func TestCounter_BuilderReuse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(2)

	sampler := &testSampler{shouldSample: true}

	// Expect 2 Add operations.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).Times(2)

	builder := NewCounter("reuse_test").
		WithTag("shared", "value").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc)

		// Build first counter.
	counter1, err := builder.Build()
	common.AssertEqual(t, nil, err)
	counter1.SetTag("instance", "1") //nolint:errcheck

	// Build second counter from same builder.
	counter2, err := builder.Build()
	common.AssertEqual(t, nil, err)
	counter2.SetTag("instance", "2") //nolint:errcheck

	// Both counters should have the shared tag but different instance tags.
	tags1 := counter1.Tags()
	tags2 := counter2.Tags()

	common.AssertEqual(t, "value", tags1["shared"])
	common.AssertEqual(t, "value", tags2["shared"])
	common.AssertEqual(t, "1", tags1["instance"])
	common.AssertEqual(t, "2", tags2["instance"])

	// Both should be independent.
	counter1.Add(5.0)
	counter2.Add(10.0)
}

// TestCounter_DimensionalTags tests dimensional tag functionality.
func TestCounter_DimensionalTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Enable dimensional metrics for this test.
	t.Setenv(EnableDimensionalMetricsEnvVar, "true")

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)
	mockProc.EXPECT().dimensionalMetricsEnabled().Return(true).AnyTimes()

	sampler := &testSampler{shouldSample: true}

	counter, err := NewCounter("dimensional_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertCounterWithTags(t, "dimensional_test", 5.0, map[string]string{
			"region": "us-east",
			"env":    "prod",
		})).Times(1)

	counter.WithDimensionalTags(map[string]string{"region": "us-east", "env": "prod"}).Add(5.0)

	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertCounterWithTags(t, "dimensional_test", 1.0, map[string]string{
			"region": "us-west",
			"env":    "staging",
		})).Times(1)

	counter.WithDimensionalTags(map[string]string{"region": "us-west", "env": "staging"}).Increment()

	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertCounterWithTags(t, "dimensional_test", 1.0, map[string]string{
			"region": "eu-central",
			"env":    "dev",
		})).Times(1)

	counter.WithDimensionalTags(map[string]string{"region": "eu-central", "env": "dev"}).Count(true)
	counter.WithDimensionalTags(map[string]string{"region": "eu-central", "env": "dev"}).Count(false)
}

// TestCounter_WithTagsMethod tests the WithTags method for creating dimensional variants.
func TestCounter_WithTagsMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Enable dimensional metrics for this test.
	t.Setenv(EnableDimensionalMetricsEnvVar, "true")

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).AnyTimes()
	mockProc.EXPECT().dimensionalMetricsEnabled().Return(true).AnyTimes()

	baseCounter, err := NewCounter("base_test").
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	regionalCounter := baseCounter.WithDimensionalTags(map[string]string{"region": "us-west"})

	baseTags := baseCounter.Tags()
	regionalTags := regionalCounter.Tags()

	if len(baseTags) != 0 {
		t.Errorf("Base counter should have no tags, got %v", baseTags)
	}

	if regionalTags["region"] != "us-west" {
		t.Errorf("Regional counter should have region=us-west, got %s", regionalTags["region"])
	}
}

// TestCounter_WithDimensionalTagMethod tests the WithDimensionalTag method for single tag addition.
func TestCounter_WithDimensionalTagMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Enable dimensional metrics for this test.
	t.Setenv(EnableDimensionalMetricsEnvVar, "true")

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)
	mockProc.EXPECT().dimensionalMetricsEnabled().Return(true).AnyTimes()

	sampler := &testSampler{shouldSample: true}

	baseCounter, err := NewCounter("withtag_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	taggedCounter := baseCounter.WithDimensionalTag("region", "us-east")

	baseTags := baseCounter.Tags()
	taggedTags := taggedCounter.Tags()

	common.AssertEqual(t, 0, len(baseTags))
	common.AssertEqual(t, 1, len(taggedTags))
	common.AssertEqual(t, "us-east", taggedTags["region"])

	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertCounterWithTags(t, "withtag_test", 5.0, map[string]string{
			"region": "us-east",
		})).Times(1)

	taggedCounter.Add(5.0)
}

// TestCounter_WithDimensionalTag_Chaining tests chaining multiple WithDimensionalTag calls.
func TestCounter_WithDimensionalTag_Chaining(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Enable dimensional metrics for this test.
	t.Setenv(EnableDimensionalMetricsEnvVar, "true")

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)
	mockProc.EXPECT().dimensionalMetricsEnabled().Return(true).AnyTimes()

	sampler := &testSampler{shouldSample: true}

	baseCounter, err := NewCounter("withtag_chain_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Test chaining multiple WithTag calls.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertCounterWithTags(t, "withtag_chain_test", 3.0, map[string]string{
			"region": "us-west",
			"env":    "prod",
		})).Times(1)

	baseCounter.
		WithDimensionalTag("region", "us-west").
		WithDimensionalTag("env", "prod").
		Add(3.0)

		// Verify that the base counter remains unchanged after dimensional operations.
	baseTags := baseCounter.Tags()
	if len(baseTags) != 0 {
		t.Errorf("Base counter should remain unchanged with no tags after dimensional operations, got %v", baseTags)
	}
}

// TestCounter_WithDimensionalTags_Functional tests WithDimensionalTags with multiple operations.
func TestCounter_WithDimensionalTags_Functional(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Enable dimensional metrics for this test.
	t.Setenv(EnableDimensionalMetricsEnvVar, "true")

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)
	mockProc.EXPECT().dimensionalMetricsEnabled().Return(true).AnyTimes()

	sampler := &testSampler{shouldSample: true}

	baseCounter, err := NewCounter("withtags_func_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	tags := map[string]string{
		"region":  "eu-central",
		"service": "api",
	}

	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertCounterWithTags(t, "withtags_func_test", 10.0, tags)).Times(1)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertCounterWithTags(t, "withtags_func_test", 1.0, tags)).Times(1)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertCounterWithTags(t, "withtags_func_test", 1.0, tags)).Times(1)

	taggedCounter := baseCounter.WithDimensionalTags(tags)
	taggedCounter.Add(10.0)
	taggedCounter.Increment()
	taggedCounter.Count(true)

	// Verify base counter remains unchanged.
	baseTags := baseCounter.Tags()
	common.AssertEqual(t, 0, len(baseTags))
}

func TestCounter_DimensionalBehaviorWhenDisabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	os.Unsetenv(EnableDimensionalMetricsEnvVar)
	defer os.Unsetenv(EnableDimensionalMetricsEnvVar)

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().dimensionalMetricsEnabled().Return(false).AnyTimes()
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1) // Only the base counter gets registered

	baseCounter, err := NewCounter("http_requests").
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	t.Run("WithTag_ReturnsSameInstance", func(t *testing.T) {
		// When dimensional metrics are disabled, WithTag() returns the same counter instance.
		taggedCounter := baseCounter.WithTag("endpoint", "/api/users")

		// Verify it's the same instance.
		if baseCounter != taggedCounter {
			t.Error("Expected WithTag to return the same counter instance when dimensional metrics are disabled")
		}

		// Tag was applied directly to the base counter.
		tags := baseCounter.Tags()
		common.AssertEqual(t, "/api/users", tags["endpoint"])

		// Both references show the same tags.
		taggedTags := taggedCounter.Tags()
		common.AssertEqual(t, "/api/users", taggedTags["endpoint"])
	})

	t.Run("WithTags_ReturnsSameInstance", func(t *testing.T) {
		// WithTags() returns the same instance and applies tags directly.
		moreTags := map[string]string{
			"method": "GET",
			"status": "200",
			"region": "us-east",
		}

		taggedCounter := baseCounter.WithTags(moreTags)

		// Same instance returned.
		if baseCounter != taggedCounter {
			t.Error("Expected WithTags to return the same counter instance when dimensional metrics are disabled")
		}

		// All tags accumulated on the base counter.
		finalTags := baseCounter.Tags()
		common.AssertEqual(t, "/api/users", finalTags["endpoint"]) // Previous tag still there
		common.AssertEqual(t, "GET", finalTags["method"])
		common.AssertEqual(t, "200", finalTags["status"])
		common.AssertEqual(t, "us-east", finalTags["region"])
	})

	t.Run("WithDimensionalTags_ReturnsSameInstance", func(t *testing.T) {
		// WithDimensionalTags() returns the same instance when disabled.
		volatileTags := map[string]string{
			"user_id":    "user12345",
			"session_id": "sess98765",
		}

		dimensionalCounter := baseCounter.WithDimensionalTags(volatileTags)

		// Same instance returned.
		if baseCounter != dimensionalCounter {
			t.Error("Expected WithDimensionalTags to return the same counter instance when dimensional metrics are disabled")
		}
	})

	t.Run("SharedState_AllReferencesUpdateSameCounter", func(t *testing.T) {
		// All operations affect the same underlying counter.
		mockProc.EXPECT().enqueueMetric(gomock.Any()).Times(3) // 3 operations below

		// Create different "tagged" references - all the same instance.
		getCounter := baseCounter.WithTag("method", "GET")
		postCounter := baseCounter.WithTag("method", "POST") // Overwrites the method tag
		userCounter := baseCounter.WithDimensionalTags(map[string]string{"user": "alice"})

		// All references point to the same counter.
		common.AssertEqual(t, baseCounter, getCounter)
		common.AssertEqual(t, baseCounter, postCounter)
		common.AssertEqual(t, baseCounter, userCounter)

		// Operations on any reference affect the same underlying counter.
		getCounter.Add(10) // baseCounter.currentValue = 10
		postCounter.Add(5) // baseCounter.currentValue = 15
		userCounter.Add(3) // baseCounter.currentValue = 18

		// All references show the same accumulated value.
		common.AssertEqual(t, 18.0, baseCounter.CurrentValue())
		common.AssertEqual(t, 18.0, getCounter.CurrentValue())
		common.AssertEqual(t, 18.0, postCounter.CurrentValue())
		common.AssertEqual(t, 18.0, userCounter.CurrentValue())

		// Final tag state reflects the last values set.
		finalTags := baseCounter.Tags()
		common.AssertEqual(t, "POST", finalTags["method"]) // Last WithTag call wins
	})

	t.Run("NoNewRegistrations_OnlyBaseCounterExists", func(_ *testing.T) {
		// When disabled, only the original base counter gets registered with the processor.

		// These calls would create separate counters when dimensional metrics are enabled.
		baseCounter.WithTag("version", "v1.0")
		baseCounter.WithTags(map[string]string{"datacenter": "us-west"})
		baseCounter.WithDimensionalTags(map[string]string{"request_id": "req123"})

		// But when disabled, they all operate on the same counter instance.
		// The registerMetric expectation at the top (Times(1)) proves only one registration occurred.
	})
}
