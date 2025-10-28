/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package metrics

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/model"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/samplers"

	"github.com/golang/mock/gomock"
)

var errTest = errors.New("test error")

func assertTimer(t *testing.T, key string, value float64) func(msg model.MetricMessage) {
	t.Helper()

	return func(msg model.MetricMessage) {
		common.AssertEqual(t, key, msg.Key)
		common.AssertEqual(t, model.MetricTypeTimer, msg.Type)
		common.AssertEqual(t, value, msg.Value)
		common.AssertEqual(t, 1.0, msg.SampleRate)
	}
}

func assertTimerWithTags(t *testing.T, key string, value float64, tags map[string]string) func(msg model.MetricMessage) {
	t.Helper()

	return func(msg model.MetricMessage) {
		assertTimer(t, key, value)(msg)

		for k, v := range tags {
			common.AssertEqual(t, v, msg.Tags[k])
		}
	}
}

// TestNewTimer tests the NewTimer function.
func TestNewTimer(t *testing.T) {
	builder := NewTimer("test_timer")

	if builder == nil {
		t.Fatal("Expected builder to not be nil")
	}
	common.AssertEqual(t, "test_timer", builder.key)
	if builder.baseBuilder == nil {
		t.Fatal("Expected baseBuilder to not be nil")
	}
}

// TestTimerBuilder_WithTags tests adding multiple tags.
func TestTimerBuilder_WithTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	tags := map[string]string{
		"env":     "production",
		"service": "api",
		"version": "1.0.0",
	}

	timer, err := NewTimer("response_time").
		WithTags(tags).
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	result := timer.Tags()
	common.AssertEqual(t, 3, len(result))
	common.AssertEqual(t, "production", result["env"])
	common.AssertEqual(t, "api", result["service"])
	common.AssertEqual(t, "1.0.0", result["version"])
}

// TestTimerBuilder_WithTag tests adding a single tag.
func TestTimerBuilder_WithTag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	timer, err := NewTimer("request_duration").
		WithTag("region", "us-east-1").
		WithTag("app", "game-server").
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	result := timer.Tags()
	common.AssertEqual(t, 2, len(result))
	common.AssertEqual(t, "us-east-1", result["region"])
	common.AssertEqual(t, "game-server", result["app"])
}

// TestTimerBuilder_WithSampler tests setting a sampler.
func TestTimerBuilder_WithSampler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}
	timer, err := NewTimer("sampled_timer").
		WithSampler(sampler).
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	// Verify the sampler is set by checking if it's used in the base metric.
	if timer.baseMetric == nil {
		t.Fatal("Expected baseMetric to not be nil")
	}
	// Note: sampler usage is tested indirectly through the enqueueMessage behavior.
}

// TestTimerBuilder_WithDerivedMetrics tests adding derived metrics.
func TestTimerBuilder_WithDerivedMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	derived1 := &testDerivedMetric{key: "test.latest"}
	derived2 := &testDerivedMetric{key: "test.max"}

	timer, err := NewTimer("base_metric").
		WithDerivedMetrics(derived1, derived2).
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	derivedMetrics := timer.DerivedMetrics()
	common.AssertEqual(t, 2, len(derivedMetrics))
	common.AssertEqual(t, "test.latest", derivedMetrics[0].Key())
	common.AssertEqual(t, "test.max", derivedMetrics[1].Key())
}

// TestTimerBuilder_ChainedCalls tests method chaining.
func TestTimerBuilder_ChainedCalls(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	derived := &testDerivedMetric{key: "response.latest"}
	sampler := samplers.NewAll()

	timer, err := NewTimer("http_response_time").
		WithTag("method", "POST").
		WithTag("endpoint", "/api/users").
		WithTags(map[string]string{"status": "200", "cached": "false"}).
		WithSampler(sampler).
		WithDerivedMetrics(derived).
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	// Verify all configurations are applied.
	common.AssertEqual(t, "http_response_time", timer.Key())
	common.AssertEqual(t, model.MetricTypeTimer, timer.MetricType())

	tags := timer.Tags()
	common.AssertEqual(t, 4, len(tags))
	common.AssertEqual(t, "POST", tags["method"])
	common.AssertEqual(t, "/api/users", tags["endpoint"])
	common.AssertEqual(t, "200", tags["status"])
	common.AssertEqual(t, "false", tags["cached"])

	derivedMetrics := timer.DerivedMetrics()
	common.AssertEqual(t, 1, len(derivedMetrics))
	common.AssertEqual(t, "response.latest", derivedMetrics[0].Key())
}

// TestTimerBuilder_Build tests the Build method.
func TestTimerBuilder_Build(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	timer, err := NewTimer("build_test").
		WithTag("test", "true").
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	if timer == nil {
		t.Fatal("Expected timer to not be nil")
	}
	if timer.baseMetric == nil {
		t.Fatal("Expected baseMetric to not be nil")
	}
	common.AssertEqual(t, "build_test", timer.Key())
	common.AssertEqual(t, model.MetricTypeTimer, timer.MetricType())
}

// TestTimer_SetDuration tests the SetDuration method.
func TestTimer_SetDuration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a timer with a mock processor and sampler that always samples.
	mockProc := NewMockMetricsProcessor(ctrl)

	// Verify that enqueueMetric was called with specific MetricMessage values (in milliseconds).
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimer(t, "duration_test", 1000.0)).Times(1) // 1 second = 1000ms
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimer(t, "duration_test", 500.0)).Times(1) // 500ms
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimer(t, "duration_test", 1.0)).Times(1) // 1ms

	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	timer, err := NewTimer("duration_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Test setting various durations.
	timer.SetDuration(time.Second)
	timer.SetDuration(500 * time.Millisecond)
	timer.SetDuration(time.Millisecond)
}

// TestTimer_SetMilliseconds tests the SetMilliseconds method.
func TestTimer_SetMilliseconds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	// Verify that enqueueMetric was called with specific millisecond values.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimer(t, "milliseconds_test", 250.5)).Times(1)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimer(t, "milliseconds_test", 1000.0)).Times(1)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimer(t, "milliseconds_test", 0.1)).Times(1)

	timer, err := NewTimer("milliseconds_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Test setting various millisecond values.
	timer.SetMilliseconds(250.5)
	timer.SetMilliseconds(1000.0)
	timer.SetMilliseconds(0.1)
}

// TestTimer_SetSeconds tests the SetSeconds method.
func TestTimer_SetSeconds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	// Verify that enqueueMetric was called with values converted to milliseconds.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimer(t, "seconds_test", 2500.0)).Times(1) // 2.5 seconds = 2500ms
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimer(t, "seconds_test", 1000.0)).Times(1) // 1 second = 1000ms
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimer(t, "seconds_test", 100.0)).Times(1) // 0.1 seconds = 100ms

	timer, err := NewTimer("seconds_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Test setting various second values.
	timer.SetSeconds(2.5)
	timer.SetSeconds(1.0)
	timer.SetSeconds(0.1)
}

// TestTimer_MessageBehavior tests that all timer operations send absolute duration values.
func TestTimer_MessageBehavior(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	timer, err := NewTimer("message_behavior_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// All timer operations should send absolute values (not deltas).
	// This is correct because each timer measurement is independent.

	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimer(t, "message_behavior_test", 1000.0)).Times(1) // 1 second
	timer.SetDuration(time.Second)

	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimer(t, "message_behavior_test", 500.0)).Times(1) // 500ms
	timer.SetMilliseconds(500.0)

	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimer(t, "message_behavior_test", 2000.0)).Times(1) // 2 seconds = 2000ms
	timer.SetSeconds(2.0)

	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimer(t, "message_behavior_test", 100.0)).Times(1) // 100ms
	timer.SetMilliseconds(100.0)

	// Verify internal state reflects the last set value.
	common.AssertEqual(t, 100.0, timer.CurrentValue())
}

// TestTimer_TimeFunc tests the TimeFunc method.
func TestTimer_TimeFunc(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	timer, err := NewTimer("timefunc_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	t.Run("SuccessfulFunction", func(t *testing.T) {
		// Expect a call but don't verify exact timing due to test environment variability.
		mockProc.EXPECT().enqueueMetric(gomock.Any()).Times(1)

		err := timer.TimeFunc(context.Background(), func() error {
			time.Sleep(10 * time.Millisecond) // Small sleep to ensure measurable duration
			return nil
		})

		common.AssertEqual(t, nil, err)
	})

	t.Run("FunctionWithError", func(t *testing.T) {
		// Expect a call even if function returns error.
		mockProc.EXPECT().enqueueMetric(gomock.Any()).Times(1)

		err := timer.TimeFunc(context.Background(), func() error {
			time.Sleep(5 * time.Millisecond)
			return errTest
		})

		common.AssertEqual(t, errTest, err)
	})

	t.Run("FunctionWithContext", func(t *testing.T) {
		// Test with cancelled context.
		mockProc.EXPECT().enqueueMetric(gomock.Any()).Times(1)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := timer.TimeFunc(ctx, func() error {
			// Function should still execute even with cancelled context.
			// (TimeFunc doesn't use the context for cancellation).
			return nil
		})

		common.AssertEqual(t, nil, err)
	})
}

// TestTimer_Start tests the Start method.
func TestTimer_Start(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	timer, err := NewTimer("start_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	t.Run("BasicStartStop", func(_ *testing.T) {
		// Expect a call but don't verify exact timing due to test environment variability.
		mockProc.EXPECT().enqueueMetric(gomock.Any()).Times(1)

		stop := timer.Start()
		time.Sleep(10 * time.Millisecond) // Small sleep to ensure measurable duration
		stop()
	})

	t.Run("MultipleStartStop", func(_ *testing.T) {
		// Expect multiple calls for multiple start/stop cycles.
		mockProc.EXPECT().enqueueMetric(gomock.Any()).Times(3)

		// First timing.
		stop1 := timer.Start()
		time.Sleep(5 * time.Millisecond)
		stop1()

		// Second timing.
		stop2 := timer.Start()
		time.Sleep(5 * time.Millisecond)
		stop2()

		// Third timing.
		stop3 := timer.Start()
		time.Sleep(5 * time.Millisecond)
		stop3()
	})

	t.Run("StopFunctionCanBeCalledMultipleTimes", func(_ *testing.T) {
		// Each call to stop() will trigger a metric (based on current implementation).
		mockProc.EXPECT().enqueueMetric(gomock.Any()).Times(2)

		stop := timer.Start()
		time.Sleep(5 * time.Millisecond)
		stop()
		stop() // Second call will trigger another metric with same duration
	})
}

// TestTimer_SamplingBehavior tests that sampling controls message enqueueing.
func TestTimer_SamplingBehavior(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)

	t.Run("AlwaysSample", func(t *testing.T) {
		mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

		sampler := &testSampler{shouldSample: true}

		timer, err := NewTimer("sampling_test").
			WithSampler(sampler).
			WithMetricsProcessor(mockProc).
			Build()
		common.AssertEqual(t, nil, err)

		mockProc.EXPECT().enqueueMetric(gomock.Any()).
			Do(assertTimer(t, "sampling_test", 1000.0)).Times(1)
		mockProc.EXPECT().enqueueMetric(gomock.Any()).
			Do(assertTimer(t, "sampling_test", 500.0)).Times(1)
		mockProc.EXPECT().enqueueMetric(gomock.Any()).
			Do(assertTimer(t, "sampling_test", 2000.0)).Times(1)

		timer.SetDuration(time.Second)
		timer.SetMilliseconds(500.0)
		timer.SetSeconds(2.0)
	})

	t.Run("NeverSample", func(t *testing.T) {
		mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

		sampler := &testSampler{shouldSample: false}

		timer, err := NewTimer("sampling_test").
			WithSampler(sampler).
			WithMetricsProcessor(mockProc).
			Build()
		common.AssertEqual(t, nil, err)

		// Expect no calls when sampling is disabled.
		mockProc.EXPECT().enqueueMetric(gomock.Any()).Times(0)

		timer.SetDuration(time.Second)
		timer.SetMilliseconds(500.0)
		timer.SetSeconds(2.0)
	})
}

// TestTimer_MessageTags tests that tags are properly included in enqueued messages.
func TestTimer_MessageTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)

	// Expect enqueueMetric to be called with specific tags.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimerWithTags(t, "tagged_timer", 750.0, map[string]string{
			"env":     "test",
			"service": "game-server",
			"runtime": "dynamic",
		})).Times(1)

	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	timer, err := NewTimer("tagged_timer").
		WithTag("env", "test").
		WithTag("service", "game-server").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Add a tag after creation.
	timer.SetTag("runtime", "dynamic") //nolint:errcheck

	timer.SetMilliseconds(750.0)
}

// TestTimer_ThreadSafety tests concurrent access to timer methods.
func TestTimer_ThreadSafety(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	// For thread safety test, we expect many calls but don't need to verify exact count.
	// due to concurrent nature - just verify no panics occur.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).AnyTimes()

	timer, err := NewTimer("thread_safety_test").
		WithTag("concurrent", "true").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	var wg sync.WaitGroup
	numGoroutines := 100
	operationsPerGoroutine := 10

	// Test concurrent SetDuration operations.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				timer.SetDuration(time.Duration(id*10+j) * time.Millisecond)
			}
		}(i)
	}

	// Test concurrent SetMilliseconds operations.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				timer.SetMilliseconds(float64(id*10 + j))
			}
		}(i)
	}

	// Test concurrent SetSeconds operations.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				timer.SetSeconds(float64(id+j) * 0.001)
			}
		}(i)
	}

	// Test concurrent Start/Stop operations.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				stop := timer.Start()
				time.Sleep(time.Microsecond) // Tiny sleep
				stop()
			}
		}()
	}

	// Test concurrent TimeFunc operations.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				timer.TimeFunc(context.Background(), func() error { //nolint:errcheck
					time.Sleep(time.Microsecond) // Tiny sleep
					return nil
				})
			}
		}()
	}

	// Test concurrent tag operations.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			timer.SetTag("worker", string(rune('A'+id%26))) //nolint:errcheck
			timer.Tags()
		}(i)
	}

	wg.Wait()

	// Verify the timer is still functional after concurrent access.
	timer.SetMilliseconds(100.0)
	common.AssertEqual(t, model.MetricTypeTimer, timer.MetricType())
}

// TestTimer_BuilderReuse tests that builder can be reused.
func TestTimer_BuilderReuse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(2)

	sampler := &testSampler{shouldSample: true}

	// Expect 2 SetMilliseconds operations.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).Times(2)

	builder := NewTimer("reuse_test").
		WithTag("shared", "value").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc)

		// Build first timer.
	timer1, err := builder.Build()
	common.AssertEqual(t, nil, err)

	timer1.SetTag("instance", "1") //nolint:errcheck

	// Build second timer from same builder.
	timer2, err := builder.Build()
	common.AssertEqual(t, nil, err)

	timer2.SetTag("instance", "2") //nolint:errcheck

	// Both timers should have the shared tag but different instance tags.
	tags1 := timer1.Tags()
	tags2 := timer2.Tags()

	common.AssertEqual(t, "value", tags1["shared"])
	common.AssertEqual(t, "value", tags2["shared"])
	common.AssertEqual(t, "1", tags1["instance"])
	common.AssertEqual(t, "2", tags2["instance"])

	// Both should be independent.
	timer1.SetMilliseconds(250.0)
	timer2.SetMilliseconds(500.0)
}

// TestTimer_DurationConversions tests that duration conversions are accurate.
func TestTimer_DurationConversions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	timer, err := NewTimer("conversion_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Test various duration conversions.
	testCases := []struct {
		operation   func()
		description string
		expectedMs  float64
	}{
		{
			description: "1 nanosecond",
			operation:   func() { timer.SetDuration(time.Nanosecond) },
			expectedMs:  0.0, // 1ns rounds to 0ms
		},
		{
			description: "1 microsecond",
			operation:   func() { timer.SetDuration(time.Microsecond) },
			expectedMs:  0.0, // 1μs rounds to 0ms
		},
		{
			description: "1 millisecond",
			operation:   func() { timer.SetDuration(time.Millisecond) },
			expectedMs:  1.0,
		},
		{
			description: "1 second",
			operation:   func() { timer.SetDuration(time.Second) },
			expectedMs:  1000.0,
		},
		{
			description: "1 minute",
			operation:   func() { timer.SetDuration(time.Minute) },
			expectedMs:  60000.0,
		},
		{
			description: "2.5 seconds via SetSeconds",
			operation:   func() { timer.SetSeconds(2.5) },
			expectedMs:  2500.0,
		},
		{
			description: "0.5 seconds via SetSeconds",
			operation:   func() { timer.SetSeconds(0.5) },
			expectedMs:  500.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			mockProc.EXPECT().enqueueMetric(gomock.Any()).
				Do(assertTimer(t, "conversion_test", tc.expectedMs)).Times(1)

			tc.operation()
		})
	}
}

// TestTimer_StartStopBehavior tests specific behaviors of Start/Stop functionality.
func TestTimer_StartStopBehavior(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)

	sampler := &testSampler{shouldSample: true}

	timer, err := NewTimer("startstop_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	t.Run("StopWithoutMeasurableTime", func(_ *testing.T) {
		// Even with no measurable time, should still record a metric.
		mockProc.EXPECT().enqueueMetric(gomock.Any()).Times(1)

		stop := timer.Start()
		stop() // Stop immediately
	})

	t.Run("NestedStartStop", func(_ *testing.T) {
		// Each start creates an independent timer.
		mockProc.EXPECT().enqueueMetric(gomock.Any()).Times(2)

		stop1 := timer.Start()
		stop2 := timer.Start() // Independent timer

		time.Sleep(5 * time.Millisecond)

		stop1() // Stops first timer
		stop2() // Stops second timer
	})
}

// TestTimer_DimensionalTags tests dimensional tag functionality.
func TestTimer_DimensionalTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)
	// Expect dimensionalMetricsEnabled to be called when using WithTags.
	mockProc.EXPECT().dimensionalMetricsEnabled().Return(true).AnyTimes()

	sampler := &testSampler{shouldSample: true}

	timer, err := NewTimer("dimensional_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimerWithTags(t, "dimensional_test", 1000.0, map[string]string{
			"region": "us-east",
			"env":    "prod",
		})).Times(1)

	timer.WithDimensionalTags(map[string]string{"region": "us-east", "env": "prod"}).SetDuration(time.Second)

	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimerWithTags(t, "dimensional_test", 500.0, map[string]string{
			"region": "us-west",
			"env":    "staging",
		})).Times(1)

	timer.WithDimensionalTags(map[string]string{"region": "us-west", "env": "staging"}).SetMilliseconds(500.0)

	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimerWithTags(t, "dimensional_test", 2000.0, map[string]string{
			"region": "eu-central",
			"env":    "dev",
		})).Times(1)

	timer.WithDimensionalTags(map[string]string{"region": "eu-central", "env": "dev"}).SetSeconds(2.0)

	mockProc.EXPECT().enqueueMetric(gomock.Any()).Times(1)
	err = timer.WithDimensionalTags(map[string]string{"operation": "test"}).TimeFunc(context.Background(), func() error {
		time.Sleep(5 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Errorf("TimeFunc should not return error: %v", err)
	}

	mockProc.EXPECT().enqueueMetric(gomock.Any()).Times(1)
	stop := timer.WithDimensionalTags(map[string]string{"method": "start"}).Start()
	time.Sleep(5 * time.Millisecond)
	stop()
}

func TestTimer_WithTagsMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Enable dimensional metrics for this test.
	t.Setenv(EnableDimensionalMetricsEnvVar, "true")

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).AnyTimes()
	mockProc.EXPECT().dimensionalMetricsEnabled().Return(true).AnyTimes()

	baseTimer, err := NewTimer("base_test").
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	regionalTimer := baseTimer.WithDimensionalTags(map[string]string{"region": "us-west"})

	baseTags := baseTimer.Tags()
	regionalTags := regionalTimer.Tags()

	if len(baseTags) != 0 {
		t.Errorf("Base timer should have no tags, got %v", baseTags)
	}

	if regionalTags["region"] != "us-west" {
		t.Errorf("Regional timer should have region=us-west, got %s", regionalTags["region"])
	}
}

// TestTimer_WithDimensionalTagMethod tests the WithDimensionalTag method for single tag addition.
func TestTimer_WithDimensionalTagMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Enable dimensional metrics for this test.
	t.Setenv(EnableDimensionalMetricsEnvVar, "true")

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)
	mockProc.EXPECT().dimensionalMetricsEnabled().Return(true).AnyTimes()

	sampler := &testSampler{shouldSample: true}

	baseTimer, err := NewTimer("withtag_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Test single tag addition and verify tags.
	taggedTimer := baseTimer.WithDimensionalTag("endpoint", "/api/users")

	baseTags := baseTimer.Tags()
	taggedTags := taggedTimer.Tags()

	common.AssertEqual(t, 0, len(baseTags))
	common.AssertEqual(t, 1, len(taggedTags))
	common.AssertEqual(t, "/api/users", taggedTags["endpoint"])

	// Test functional operation on tagged timer.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimerWithTags(t, "withtag_test", 250.0, map[string]string{
			"endpoint": "/api/users",
		})).Times(1)

	taggedTimer.SetMilliseconds(250.0)
}

// TestTimer_WithDimensionalTag_Chaining tests chaining multiple WithDimensionalTag calls.
func TestTimer_WithDimensionalTag_Chaining(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Enable dimensional metrics for this test.
	t.Setenv(EnableDimensionalMetricsEnvVar, "true")

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)
	mockProc.EXPECT().dimensionalMetricsEnabled().Return(true).AnyTimes()

	sampler := &testSampler{shouldSample: true}

	baseTimer, err := NewTimer("withtag_chain_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	// Test chaining multiple WithTag calls.
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimerWithTags(t, "withtag_chain_test", 1500.0, map[string]string{
			"method": "POST",
			"status": "200",
		})).Times(1)

	baseTimer.
		WithDimensionalTag("method", "POST").
		WithDimensionalTag("status", "200").
		SetSeconds(1.5)

		// Verify that the base timer remains unchanged after dimensional operations.
	baseTags := baseTimer.Tags()
	if len(baseTags) != 0 {
		t.Errorf("Base timer should remain unchanged with no tags after dimensional operations, got %v", baseTags)
	}
}

// TestTimer_WithDimensionalTags_Functional tests WithDimensionalTags with multiple operations.
func TestTimer_WithDimensionalTags_Functional(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Enable dimensional metrics for this test.
	t.Setenv(EnableDimensionalMetricsEnvVar, "true")

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1)
	mockProc.EXPECT().dimensionalMetricsEnabled().Return(true).AnyTimes()

	sampler := &testSampler{shouldSample: true}

	baseTimer, err := NewTimer("withtags_func_test").
		WithSampler(sampler).
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	tags := map[string]string{
		"operation": "db_query",
		"table":     "users",
	}

	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimerWithTags(t, "withtags_func_test", 1000.0, tags)).Times(1)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimerWithTags(t, "withtags_func_test", 500.0, tags)).Times(1)
	mockProc.EXPECT().enqueueMetric(gomock.Any()).
		Do(assertTimerWithTags(t, "withtags_func_test", 2000.0, tags)).Times(1)

	taggedTimer := baseTimer.WithDimensionalTags(tags)
	taggedTimer.SetDuration(time.Second)
	taggedTimer.SetMilliseconds(500.0)
	taggedTimer.SetSeconds(2.0)

	baseTags := baseTimer.Tags()
	common.AssertEqual(t, 0, len(baseTags))
}

func TestTimer_DimensionalBehaviorWhenDisabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	os.Unsetenv(EnableDimensionalMetricsEnvVar)
	defer os.Unsetenv(EnableDimensionalMetricsEnvVar)

	mockProc := NewMockMetricsProcessor(ctrl)
	mockProc.EXPECT().dimensionalMetricsEnabled().Return(false).AnyTimes()
	mockProc.EXPECT().registerMetric(gomock.Any()).Times(1) // Single registration only

	baseTimer, err := NewTimer("response_time").
		WithMetricsProcessor(mockProc).
		Build()
	common.AssertEqual(t, nil, err)

	t.Run("WithTag_ReturnsIdenticalInstance", func(t *testing.T) {
		// When disabled, WithTag() returns the same timer and modifies it directly.
		endpointTimer := baseTimer.WithTag("endpoint", "/api/health")

		// Verify it's the same timer instance.
		if baseTimer != endpointTimer {
			t.Error("Expected WithTag to return identical timer instance when dimensional metrics disabled")
		}

		// Tag applied directly to base timer.
		tags := baseTimer.Tags()
		common.AssertEqual(t, "/api/health", tags["endpoint"])

		// Both references have identical tags.
		endpointTags := endpointTimer.Tags()
		common.AssertEqual(t, "/api/health", endpointTags["endpoint"])
	})

	t.Run("WithTags_AccumulatesOnSameInstance", func(t *testing.T) {
		// WithTags() accumulates tags on the same timer instance.
		serviceMetadata := map[string]string{
			"service":  "user-api",
			"version":  "v2.1",
			"region":   "us-west",
			"protocol": "https",
		}

		serviceTimer := baseTimer.WithTags(serviceMetadata)

		// Same timer instance returned.
		if baseTimer != serviceTimer {
			t.Error("Expected WithTags to return identical timer instance when dimensional metrics disabled")
		}

		// All tags accumulated on the base timer.
		allTags := baseTimer.Tags()
		common.AssertEqual(t, "/api/health", allTags["endpoint"]) // From previous test
		common.AssertEqual(t, "user-api", allTags["service"])
		common.AssertEqual(t, "v2.1", allTags["version"])
		common.AssertEqual(t, "us-west", allTags["region"])
		common.AssertEqual(t, "https", allTags["protocol"])
	})

	t.Run("WithDimensionalTags_NoSpecialBehavior", func(t *testing.T) {
		// WithDimensionalTags() returns the same instance when disabled.
		volatileRequestTags := map[string]string{
			"trace_id":   "trace_xyz789",
			"span_id":    "span_abc123",
			"request_id": "req_def456",
		}

		tracedTimer := baseTimer.WithDimensionalTags(volatileRequestTags)

		// Same instance returned.
		if baseTimer != tracedTimer {
			t.Error("Expected WithDimensionalTags to return same timer when dimensional metrics disabled")
		}
	})

	t.Run("UnifiedTiming_SharedStateBehavior", func(t *testing.T) {
		// All timer references operate on the same underlying timing data.
		mockProc.EXPECT().enqueueMetric(gomock.Any()).Times(4) // 4 timing operations

		// Create "different" timer references - all the same instance.
		apiTimer := baseTimer.WithTag("operation", "api_call")
		dbTimer := baseTimer.WithTag("operation", "db_query") // Overwrites operation tag
		cacheTimer := baseTimer.WithDimensionalTags(map[string]string{"layer": "cache"})

		// All references are identical.
		common.AssertEqual(t, baseTimer, apiTimer)
		common.AssertEqual(t, baseTimer, dbTimer)
		common.AssertEqual(t, baseTimer, cacheTimer)

		// Any timing operation affects the shared timer state.
		apiTimer.SetMilliseconds(100.0)     // baseTimer = 100ms
		dbTimer.SetSeconds(0.5)             // baseTimer = 500ms (overwrites)
		cacheTimer.SetDuration(time.Second) // baseTimer = 1000ms (overwrites)

		// Test the Start/Stop pattern.
		stopFunc := baseTimer.Start()
		time.Sleep(1 * time.Millisecond) // Small delay
		stopFunc()                       // This sets the timer to the elapsed duration

		// All references show the same final timing value.
		finalValue := baseTimer.CurrentValue()
		common.AssertEqual(t, finalValue, apiTimer.CurrentValue())
		common.AssertEqual(t, finalValue, dbTimer.CurrentValue())
		common.AssertEqual(t, finalValue, cacheTimer.CurrentValue())

		// Value should be > 0 from the elapsed time.
		if finalValue <= 0 {
			t.Error("Expected timer to have positive elapsed time value")
		}
	})

	t.Run("TimingMethods_AllAffectSameTimer", func(t *testing.T) {
		// Different timing methods all operate on the shared instance.
		mockProc.EXPECT().enqueueMetric(gomock.Any()).Times(3) // TimeFunc + 2 direct sets

		// Different ways to set timing values - all affect same timer.
		baseTimer.SetDuration(200 * time.Millisecond) // 200ms
		common.AssertEqual(t, 200.0, baseTimer.CurrentValue())

		baseTimer.SetSeconds(1.5) // 1500ms (overwrites)
		common.AssertEqual(t, 1500.0, baseTimer.CurrentValue())

		// TimeFunc times function execution.
		testErr := baseTimer.TimeFunc(context.Background(), func() error {
			time.Sleep(2 * time.Millisecond) // Small delay
			return nil
		})
		common.AssertEqual(t, nil, testErr)

		// Timer now contains the function execution time.
		timedValue := baseTimer.CurrentValue()
		if timedValue < 1.0 { // Should be at least 1ms
			t.Errorf("Expected function timing to be >= 1ms, got %f", timedValue)
		}
	})

	t.Run("NoMetricExplosion_SingleRegistrationOnly", func(t *testing.T) {
		// When disabled, only one timer gets registered regardless of tagging calls.

		// These calls would create separate timers when dimensional metrics are enabled.
		for i := 0; i < 10; i++ {
			baseTimer.WithTag("iteration", fmt.Sprintf("iter_%d", i))
			baseTimer.WithDimensionalTags(map[string]string{
				"request_id": fmt.Sprintf("req_%d", i),
				"user_id":    fmt.Sprintf("user_%d", i),
			})
		}

		// But when disabled, they all return the same timer instance.
		// The registerMetric expectation (Times(1)) at the top proves this.

		// Final state: only the last "iteration" tag value remains.
		finalTags := baseTimer.Tags()
		common.AssertEqual(t, "iter_9", finalTags["iteration"]) // Last WithTag call wins
	})
}
