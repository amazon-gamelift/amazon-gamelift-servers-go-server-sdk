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
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/derived"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/model"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server/log"

	"github.com/golang/mock/gomock"
)

func TestServerUpMetricHeartbeat(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransport := NewMockTransport(ctrl)

	// We expect up=1 to be sent at least twice (on two intervals).
	var mu sync.Mutex
	serverUpMessages := make([]model.MetricMessage, 0)
	mockTransport.EXPECT().Send(gomock.Any()).DoAndReturn(func(messages []model.MetricMessage) error {
		mu.Lock()
		defer mu.Unlock()
		for _, msg := range messages {
			if msg.Key == "up" && msg.Type == model.MetricTypeGauge {
				serverUpMessages = append(serverUpMessages, msg)
			}
		}
		return nil
	}).AnyTimes()
	mockTransport.EXPECT().Close().Return(nil).AnyTimes()

	err := InitIsolatedTestProcessor(t,
		WithTransport(mockTransport),
		WithProcessInterval(100*time.Millisecond),
	)
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = processor.Start(ctx)
	common.AssertEqual(t, nil, err)

	// Wait for at least 2 intervals to ensure heartbeat is sent.
	time.Sleep(250 * time.Millisecond)

	processor.Stop() //nolint:errcheck

	// Verify server_up=1 was sent multiple times.
	mu.Lock()
	serverUpOneCount := 0
	serverUpZeroCount := 0
	for _, msg := range serverUpMessages {
		if msg.Value == 1 {
			serverUpOneCount++
		} else if msg.Value == 0 {
			serverUpZeroCount++
		}
	}
	mu.Unlock()

	// Should have received at least 2 server_up=1 messages (heartbeats).
	if serverUpOneCount < 2 {
		t.Errorf("Expected at least 2 server_up=1 heartbeats, got %d", serverUpOneCount)
	}

	// Should have received exactly 1 server_up=0 message (on shutdown).
	common.AssertEqual(t, 1, serverUpZeroCount)
}

func TestServerUpMetricShutdown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransport := NewMockTransport(ctrl)

	serverUpZeroSent := false
	mockTransport.EXPECT().Send(gomock.Any()).DoAndReturn(func(messages []model.MetricMessage) error {
		for _, msg := range messages {
			if msg.Key == "up" && msg.Value == 0 {
				serverUpZeroSent = true
			}
		}
		return nil
	}).AnyTimes()
	mockTransport.EXPECT().Close().Return(nil).AnyTimes()

	err := InitIsolatedTestProcessor(t,
		WithTransport(mockTransport),
		WithProcessInterval(1*time.Second),
	)

	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	// Start and immediately stop.
	ctx := context.Background()
	err = processor.Start(ctx)
	common.AssertEqual(t, nil, err)

	// Stop should send server_up=0.
	err = processor.Stop()
	common.AssertEqual(t, nil, err)

	common.AssertEqual(t, true, serverUpZeroSent)
}

func TestServerUpMetricWithNoOtherMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransport := NewMockTransport(ctrl)

	var mu sync.Mutex
	serverUpCount := 0
	mockTransport.EXPECT().Send(gomock.Any()).DoAndReturn(func(messages []model.MetricMessage) error {
		mu.Lock()
		defer mu.Unlock()
		for _, msg := range messages {
			if msg.Key == "up" {
				serverUpCount++
			}
		}
		return nil
	}).AnyTimes()
	mockTransport.EXPECT().Close().Return(nil).AnyTimes()

	err := InitIsolatedTestProcessor(t,
		WithTransport(mockTransport),
		WithProcessInterval(100*time.Millisecond),
	)
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = processor.Start(ctx)
	common.AssertEqual(t, nil, err)

	// Wait for a few intervals WITHOUT sending any other metrics.
	time.Sleep(350 * time.Millisecond)

	processor.Stop() //nolint:errcheck

	mu.Lock()
	finalCount := serverUpCount
	mu.Unlock()

	if finalCount < 3 {
		t.Errorf("Expected at least 3 server_up messages (heartbeats + shutdown), got %d", finalCount)
	}
}

func TestServerUpWithStartStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransport := NewMockTransport(ctrl)

	// Track server_up messages with mutex for thread safety.
	var mu sync.Mutex
	var serverUpValues []float64
	mockTransport.EXPECT().Send(gomock.Any()).DoAndReturn(func(messages []model.MetricMessage) error {
		mu.Lock()
		defer mu.Unlock()
		for _, msg := range messages {
			if msg.Key == "up" && msg.Type == model.MetricTypeGauge {
				serverUpValues = append(serverUpValues, msg.Value)
			}
		}
		return nil
	}).AnyTimes()
	mockTransport.EXPECT().Close().Return(nil).AnyTimes()

	err := InitIsolatedTestProcessor(t,
		WithTransport(mockTransport),
		WithProcessInterval(100*time.Millisecond),
	)
	common.AssertEqual(t, nil, err)

	// Start via global start function
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = StartMetricsProcessor(ctx)
	common.AssertEqual(t, nil, err)

	time.Sleep(250 * time.Millisecond)

	// Stop via global termination function - this should send server_up=0
	err = TerminateMetricsProcessor()
	common.AssertEqual(t, nil, err)

	// Verify we got heartbeats (1) and shutdown (0).
	// Need to lock when reading the slice.
	mu.Lock()
	hasHeartbeat := false
	hasShutdown := false
	for _, value := range serverUpValues {
		if value == 1 {
			hasHeartbeat = true
		} else if value == 0 {
			hasShutdown = true
		}
	}
	mu.Unlock()

	if !hasHeartbeat {
		t.Error("Expected at least one server_up=1 heartbeat when using factory")
	}
	if !hasShutdown {
		t.Error("Expected server_up=0 on TerminateMetricsProcessor()")
	}
}

func TestProcessorWithGlobalTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tags := map[string]string{
		"env":     "production",
		"service": "api",
		"version": "1.0.0",
	}

	err := InitIsolatedTestProcessor(t,
		WithTransport(NewMockTransport(ctrl)),
		WithGlobalTags(tags),
	)
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	globalTags := processor.GlobalTags

	// Check that user-provided tags are present.
	common.AssertEqual(t, "production", globalTags["env"])
	common.AssertEqual(t, "api", globalTags["service"])
	common.AssertEqual(t, "1.0.0", globalTags["version"])
}

func TestProcessorSetGlobalTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	err := InitIsolatedTestProcessor(t,
		WithTransport(NewMockTransport(ctrl)),
	)
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()
	processor.SetGlobalTag("newTag", "newValue") //nolint:errcheck
	processor.SetGlobalTag("env", "staging")     //nolint:errcheck

	globalTags := processor.GlobalTags

	// Check that the tags we set are present.
	common.AssertEqual(t, "newValue", globalTags["newTag"])
	common.AssertEqual(t, "staging", globalTags["env"])
}

func TestProcessorGetGlobalTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tags := map[string]string{
		"env":     "production",
		"service": "api",
		"version": "1.0.0",
	}

	err := InitIsolatedTestProcessor(t,
		WithTransport(NewMockTransport(ctrl)),
	)
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	processor.GlobalTags = tags

	globalTags := processor.GetGlobalTags()
	common.AssertEqual(t, len(tags), len(globalTags))
	for k, v := range tags {
		common.AssertEqual(t, v, globalTags[k])
	}
}

func TestProcessorRemoveGlobalTag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tags := map[string]string{
		"env":     "production",
		"service": "api",
		"version": "1.0.0",
	}

	err := InitIsolatedTestProcessor(t,
		WithTransport(NewMockTransport(ctrl)),
	)
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	processor.GlobalTags = tags

	processor.RemoveGlobalTag("env")

	globalTags := processor.GlobalTags
	common.AssertEqual(t, 2, len(globalTags))
	common.AssertEqual(t, "api", globalTags["service"])
	common.AssertEqual(t, "1.0.0", globalTags["version"])
}

func TestProcessorRegisterMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock processor for the metrics.
	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).AnyTimes()

	gauge, err := NewGauge("gauge").
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	counter, err := NewCounter("counter").
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	timer, err := NewTimer("timer").
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	metrics := map[string]model.Metric{
		"gauge":   gauge,
		"counter": counter,
		"timer":   timer,
	}

	err = InitIsolatedTestProcessor(t,
		WithTransport(NewMockTransport(ctrl)),
	)
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	processor.registerMetric(metrics["gauge"])
	processor.registerMetric(metrics["counter"])
	processor.registerMetric(metrics["timer"])

	common.AssertEqual(t, len(metrics), len(processor.MetricMap))
	common.AssertEqual(t, metrics["gauge"].Key(), processor.MetricMap["gauge"].Key())
	common.AssertEqual(t, metrics["counter"].Key(), processor.MetricMap["counter"].Key())
	common.AssertEqual(t, metrics["timer"].Key(), processor.MetricMap["timer"].Key())
}

func TestProcessorUnRegisterMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock processor for the metrics.
	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).AnyTimes()

	gauge, err := NewGauge("gauge").
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	counter, err := NewCounter("counter").
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	timer, err := NewTimer("timer").
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	metrics := map[string]model.Metric{
		"gauge":   gauge,
		"counter": counter,
		"timer":   timer,
	}

	err = InitIsolatedTestProcessor(t,
		WithTransport(NewMockTransport(ctrl)),
	)
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	processor.MetricMap = metrics

	processor.UnregisterMetric("gauge")

	common.AssertEqual(t, 2, len(processor.MetricMap))
	common.AssertEqual(t, metrics["counter"].Key(), processor.MetricMap["counter"].Key())
	common.AssertEqual(t, metrics["timer"].Key(), processor.MetricMap["timer"].Key())

	processor.UnregisterMetric("counter")
	processor.UnregisterMetric("timer")

	common.AssertEqual(t, 0, len(processor.MetricMap))
}

func TestProcessorGetMetric(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock processor for the metrics.
	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).AnyTimes()

	gauge, err := NewGauge("gauge").
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	counter, err := NewCounter("counter").
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	timer, err := NewTimer("timer").
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	metrics := map[string]model.Metric{
		"gauge":   gauge,
		"counter": counter,
		"timer":   timer,
	}

	err = InitIsolatedTestProcessor(t,
		WithTransport(NewMockTransport(ctrl)),
	)
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	processor.MetricMap = metrics

	retrievedGauge, exists := processor.GetMetric("gauge")
	common.AssertEqual(t, true, exists)
	common.AssertEqual(t, "gauge", retrievedGauge.Key())

	retrievedCounter, exists := processor.GetMetric("counter")
	common.AssertEqual(t, true, exists)
	common.AssertEqual(t, "counter", retrievedCounter.Key())

	retrievedTimer, exists := processor.GetMetric("timer")
	common.AssertEqual(t, true, exists)
	common.AssertEqual(t, "timer", retrievedTimer.Key())
}

func TestProcessorListMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock processor for the metrics.
	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockProcessor.EXPECT().registerMetric(gomock.Any()).AnyTimes()

	gauge, err := NewGauge("gauge").
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	counter, err := NewCounter("counter").
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	timer, err := NewTimer("timer").
		WithMetricsProcessor(mockProcessor).
		Build()
	common.AssertEqual(t, nil, err)

	metrics := map[string]model.Metric{
		"gauge":   gauge,
		"counter": counter,
		"timer":   timer,
	}

	err = InitIsolatedTestProcessor(t,
		WithTransport(NewMockTransport(ctrl)),
	)
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	processor.MetricMap = metrics

	listedMetrics := processor.ListMetrics()
	common.AssertEqual(t, len(metrics), len(listedMetrics))

	for _, metric := range listedMetrics {
		common.AssertEqual(t, metrics[metric.Key()].Key(), metric.Key())
		common.AssertEqual(t, metrics[metric.Key()].MetricType(), metric.MetricType())
	}
}

func TestProcessorProcessLoop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockTransport := NewMockTransport(ctrl)

	// Updated expectation - we now receive multiple Send calls due to server_up heartbeat.
	totalMetricsReceived := 0
	mockTransport.EXPECT().Send(gomock.Any()).DoAndReturn(
		func(messages []model.MetricMessage) error {
			// Count total metrics received across all Send calls.
			totalMetricsReceived += len(messages)
			return nil
		}).AnyTimes() // Changed to AnyTimes since server_up is sent separately
	mockTransport.EXPECT().Close().Return(nil).Times(1)

	err := InitIsolatedTestProcessor(t,
		WithTransport(mockTransport),
		WithMaxWorkers(1),
	)
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	// Enqueue a large number of messages to simulate processing.
	for i := 0; i < 100; i++ {
		processor.enqueueMetric(model.MetricMessage{
			Key:   fmt.Sprintf("test_metric_%d", i),
			Type:  model.MetricTypeCounter,
			Value: float64(i),
			Tags: map[string]string{
				"test": fmt.Sprintf("tag_%d", i),
			},
			Timestamp:  time.Now(),
			SampleRate: 1.0,
		})
	}

	ctx := context.Background()

	err = processor.Start(ctx)
	common.AssertEqual(t, nil, err)
	time.Sleep(100 * time.Millisecond)

	processor.Stop() //nolint:errcheck
	common.AssertEqual(t, nil, err)

	// Verify we received at least 100 metrics (the test metrics).
	// We may receive more due to server_up heartbeats.
	if totalMetricsReceived < 100 {
		t.Errorf("Expected at least 100 metrics, but got %d", totalMetricsReceived)
	}
}

func TestProcessorStopsOnContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransport := NewMockTransport(ctrl)
	mockTransport.EXPECT().Close().Return(nil).Times(2)
	mockTransport.EXPECT().Send(gomock.Any()).Return(nil).AnyTimes()

	err := InitIsolatedTestProcessor(t,
		WithTransport(mockTransport),
		WithMaxWorkers(1),
	)
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	ctx, cancel := context.WithCancel(context.Background())
	err = processor.Start(ctx)
	common.AssertEqual(t, nil, err)

	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(100 * time.Millisecond)

	err = processor.Stop()
	common.AssertEqual(t, nil, err)

	err = processor.Start(context.Background())
	common.AssertEqual(t, nil, err)

	err = processor.Stop()
	common.AssertEqual(t, nil, err)
}
func TestProcessorStartStopTwice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransport := NewMockTransport(ctrl)

	mockTransport.EXPECT().Close().Return(nil).Times(3)

	mockTransport.EXPECT().Send(gomock.Any()).Return(nil).AnyTimes()

	err := InitIsolatedTestProcessor(t,
		WithTransport(mockTransport),
		WithMaxWorkers(1),
	)
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	ctx := context.Background()

	err = processor.Start(ctx)
	common.AssertEqual(t, nil, err)
	time.Sleep(100 * time.Millisecond)

	err = processor.Stop()
	common.AssertEqual(t, nil, err)

	err = processor.Start(ctx)
	common.AssertEqual(t, nil, err)
	time.Sleep(100 * time.Millisecond)
	err = processor.Stop()
	common.AssertEqual(t, nil, err)

	err = processor.Start(ctx)
	common.AssertEqual(t, nil, err)

	err = processor.Stop()
	common.AssertEqual(t, nil, err)
}

func TestProcessorStartTwice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransport := NewMockTransport(ctrl)

	mockTransport.EXPECT().Close().Return(nil).Times(1)
	mockTransport.EXPECT().Send(gomock.Any()).Return(nil).AnyTimes()

	err := InitIsolatedTestProcessor(t,
		WithTransport(mockTransport),
		WithMaxWorkers(1),
	)
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	ctx := context.Background()

	err = processor.Start(ctx)
	common.AssertEqual(t, nil, err)

	err = processor.Start(ctx)
	if err == nil {
		t.Fatal("expected error when starting processor twice, but got nil")
	}

	processor.Stop() //nolint:errcheck
}

func TestProcessorStopTwice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransport := NewMockTransport(ctrl)

	mockTransport.EXPECT().Close().Return(nil).Times(1)
	mockTransport.EXPECT().Send(gomock.Any()).Return(nil).AnyTimes()

	err := InitIsolatedTestProcessor(t,
		WithTransport(mockTransport),
		WithMaxWorkers(1),
	)
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	ctx := context.Background()

	err = processor.Start(ctx)
	common.AssertEqual(t, nil, err)

	err = processor.Stop()
	common.AssertEqual(t, nil, err)

	err = processor.Stop()
	common.AssertEqual(t, nil, err)
}
func TestProcessorStatefulMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockTransport := NewMockTransport(ctrl)

	err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	gauge, err := NewGauge("cpu_usage").WithMetricsProcessor(processor).Build()
	common.AssertEqual(t, nil, err)

	processor.registerMetric(gauge)

	gauge.Add(45.7)
	common.AssertEqual(t, 45.7, gauge.CurrentValue())

	gauge.Add(20.5)
	common.AssertEqual(t, 66.2, gauge.CurrentValue())

	gaugeTwo, err := NewGauge("memory_usage").WithMetricsProcessor(processor).Build()
	common.AssertEqual(t, nil, err)
	processor.registerMetric(gaugeTwo)

	gaugeTwo.Add(50.0)
	common.AssertEqual(t, 50.0, gaugeTwo.CurrentValue())
	common.AssertEqual(t, 66.2, gauge.CurrentValue())

	gaugeTwo.Add(25.0)
	common.AssertEqual(t, 75.0, gaugeTwo.CurrentValue())
	common.AssertEqual(t, 66.2, gauge.CurrentValue())

	gaugeTwo.Set(0.0)
	common.AssertEqual(t, 0.0, gaugeTwo.CurrentValue())
	common.AssertEqual(t, 66.2, gauge.CurrentValue())

	gaugeTwo.Subtract(10.0)
	common.AssertEqual(t, -10.0, gaugeTwo.CurrentValue())
}

func TestProcessorCalculateDerivedMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransport := NewMockTransport(ctrl)
	testPercentile := derived.NewPercentile(10.0)

	err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	gauge, err := NewGauge("test_metric").
		WithDerivedMetrics(testPercentile).
		WithTags(map[string]string{"env": "test", "status": "active"}).
		WithMetricsProcessor(processor).
		Build()
	common.AssertEqual(t, nil, err)
	processor.registerMetric(gauge)

	gauge.Set(15.0)

	derivedMessages, err := processor.emitDerivedMetrics()
	common.AssertEqual(t, nil, err)
	common.AssertEqual(t, 1, len(derivedMessages))

	derivedMsg := derivedMessages[0]
	common.AssertEqual(t, "test_metric.p10", derivedMsg.Key)
	common.AssertEqual(t, "test", derivedMsg.Tags["env"])
	common.AssertEqual(t, "active", derivedMsg.Tags["status"])

	// Test if metric is reset.
	derivedMessages2, err2 := processor.emitDerivedMetrics()
	common.AssertEqual(t, nil, err2)
	common.AssertEqual(t, 0, len(derivedMessages2))
}

func TestCreateCompositeKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransport := NewMockTransport(ctrl)

	t.Run("DimensionalMetricsEnabled_IncludesTags", func(t *testing.T) {
		// Enable dimensional metrics.
		t.Setenv(EnableDimensionalMetricsEnvVar, "true")

		err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
		processor := GetGlobalProcessor()
		common.AssertEqual(t, nil, err)
		common.AssertEqual(t, true, processor.enableDimensionalMetrics)

		result := processor.createCompositeKey("cpu_usage", nil)
		common.AssertEqual(t, "cpu_usage", result)

		result = processor.createCompositeKey("cpu_usage", map[string]string{})
		common.AssertEqual(t, "cpu_usage", result)

		result = processor.createCompositeKey("cpu_usage", map[string]string{"host": "server1"})
		common.AssertEqual(t, "cpu_usage|host=server1", result)

		result = processor.createCompositeKey("cpu_usage", map[string]string{"host": "server1", "env": "prod"})
		common.AssertEqual(t, "cpu_usage|env=prod,host=server1", result)

		result = processor.createCompositeKey("cpu_usage", map[string]string{"host": "server-1", "env": "prod=test"})
		common.AssertEqual(t, "cpu_usage|env=prod=test,host=server-1", result)

		tags := map[string]string{"b": "2", "a": "1", "c": "3"}
		key1 := processor.createCompositeKey("test", tags)
		key2 := processor.createCompositeKey("test", tags)
		common.AssertEqual(t, key1, key2)
		common.AssertEqual(t, "test|a=1,b=2,c=3", key1)
	})

	t.Run("DimensionalMetricsDisabled_IgnoresTags", func(t *testing.T) {
		// Disable dimensional metrics.
		os.Unsetenv(EnableDimensionalMetricsEnvVar)
		defer os.Unsetenv(EnableDimensionalMetricsEnvVar)

		err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
		processor := GetGlobalProcessor()
		common.AssertEqual(t, nil, err)
		common.AssertEqual(t, false, processor.enableDimensionalMetrics)

		// All should return just the metric key, ignoring tags.
		result := processor.createCompositeKey("cpu_usage", nil)
		common.AssertEqual(t, "cpu_usage", result)

		result = processor.createCompositeKey("cpu_usage", map[string]string{})
		common.AssertEqual(t, "cpu_usage", result)

		result = processor.createCompositeKey("cpu_usage", map[string]string{"host": "server1"})
		common.AssertEqual(t, "cpu_usage", result)

		result = processor.createCompositeKey("cpu_usage", map[string]string{"host": "server1", "env": "prod"})
		common.AssertEqual(t, "cpu_usage", result)

		tags := map[string]string{"b": "2", "a": "1", "c": "3"}
		key1 := processor.createCompositeKey("test", tags)
		key2 := processor.createCompositeKey("test", tags)
		common.AssertEqual(t, key1, key2)
		common.AssertEqual(t, "test", key1) // Should ignore all tags
	})
}

func TestProcessorFlushMessagesWithGlobalTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransport := NewMockTransport(ctrl)
	err := InitIsolatedTestProcessor(t,
		WithTransport(mockTransport),
		WithGlobalTags(map[string]string{
			"service": "my-service",
			"version": "1.0.0",
		}),
	)
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	testMessages := []model.MetricMessage{
		{
			Key:   "cpu_usage",
			Value: 50.0,
			Tags:  map[string]string{"host": "server1"},
		},
		{
			Key:   "memory_usage",
			Value: 75.0,
			Tags:  map[string]string{"host": "server2"},
		},
	}

	mockTransport.EXPECT().Send(gomock.Any()).DoAndReturn(
		func(messages []model.MetricMessage) error {
			common.AssertEqual(t, 2, len(messages))
			for _, msg := range messages {
				common.AssertEqual(t, "my-service", msg.Tags["service"])
				common.AssertEqual(t, "1.0.0", msg.Tags["version"])
				switch msg.Key {
				case "cpu_usage":
					common.AssertEqual(t, "server1", msg.Tags["host"])
				case "memory_usage":
					common.AssertEqual(t, "server2", msg.Tags["host"])
				}
			}
			return nil
		}).Times(1)

	err = processor.flushMessages(testMessages)
	common.AssertEqual(t, nil, err)
}

func TestProcessorFlushMessagesGlobalTagsOverridePerMetric(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransport := NewMockTransport(ctrl)
	err := InitIsolatedTestProcessor(t,
		WithTransport(mockTransport),
		WithGlobalTags(map[string]string{
			"service":     "global-service",
			"team":        "platform",
			"environment": "production",
		}),
	)
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	testMessages := []model.MetricMessage{
		{
			Key:   "cpu_usage",
			Value: 50.0,
			Tags: map[string]string{
				"service":   "game-engine",
				"team":      "gameplay",
				"map":       "dust2",
				"gamemode":  "competitive",
				"server":    "lobby-01",
				"playerid":  "player_12345",
				"weapon":    "ak47",
				"matchtype": "ranked",
			},
		},
		{
			Key:   "memory_usage",
			Value: 75.0,
			Tags: map[string]string{
				"service":  "matchmaker",
				"map":      "mirage",
				"gamemode": "deathmatch",
				"server":   "dm-server-03",
			},
		},
	}

	mockTransport.EXPECT().Send(gomock.Any()).DoAndReturn(
		func(messages []model.MetricMessage) error {
			common.AssertEqual(t, 2, len(messages))

			msg1 := messages[0]
			common.AssertEqual(t, "cpu_usage", msg1.Key)

			// Check global tags.
			common.AssertEqual(t, "global-service", msg1.Tags["service"])
			common.AssertEqual(t, "platform", msg1.Tags["team"])
			common.AssertEqual(t, "production", msg1.Tags["environment"])

			// Check per-metric tags.
			common.AssertEqual(t, "dust2", msg1.Tags["map"])
			common.AssertEqual(t, "competitive", msg1.Tags["gamemode"])
			common.AssertEqual(t, "lobby-01", msg1.Tags["server"])
			common.AssertEqual(t, "player_12345", msg1.Tags["playerid"])
			common.AssertEqual(t, "ak47", msg1.Tags["weapon"])
			common.AssertEqual(t, "ranked", msg1.Tags["matchtype"])

			msg2 := messages[1]
			common.AssertEqual(t, "memory_usage", msg2.Key)

			common.AssertEqual(t, "global-service", msg2.Tags["service"])
			common.AssertEqual(t, "platform", msg2.Tags["team"])
			common.AssertEqual(t, "production", msg2.Tags["environment"])

			common.AssertEqual(t, "mirage", msg2.Tags["map"])
			common.AssertEqual(t, "deathmatch", msg2.Tags["gamemode"])
			common.AssertEqual(t, "dm-server-03", msg2.Tags["server"])

			return nil
		}).Times(1)

	err = processor.flushMessages(testMessages)
	common.AssertEqual(t, nil, err)
}

func TestWithoutClientTelemetryOption(t *testing.T) {
	builder := NewStatsDTransport().WithoutClientTelemetry()
	common.AssertEqual(t, true, builder.config.clientTelemetryDisabled)
}

func TestMetricsProcessorBuilder_Build_TransportValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("NilTransport_ReturnsError", func(t *testing.T) {
		err := InitIsolatedTestProcessor(t) // No transport provided

		if err == nil {
			t.Fatal("Expected error when transport is nil")
		}
		var gameLiftErr *common.GameLiftError
		if !errors.As(err, &gameLiftErr) {
			t.Fatalf("Expected GameLiftError, got: %T", err)
		}

		if gameLiftErr.ErrorType != common.MetricConfigurationException {
			t.Errorf("Expected MetricConfigurationException, got: %v", gameLiftErr.ErrorType)
		}
	})

	t.Run("WithTransport_Success", func(t *testing.T) {
		mockTransport := NewMockTransport(ctrl)

		err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
		processor := GetGlobalProcessor()

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if processor == nil {
			t.Fatal("Expected processor to not be nil")
		}
		common.AssertEqual(t, mockTransport, processor.Transport)
	})
}

func TestProcessor_MetricsWithDifferentStaticTagsAreIndependent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransport := NewMockTransport(ctrl)

	t.Run("DimensionalMetricsDisabled_SameNameIgnoresTags", func(t *testing.T) {
		// Ensure environment variable is not set (dimensional disabled).
		os.Unsetenv(EnableDimensionalMetricsEnvVar)

		err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
		processor := GetGlobalProcessor()
		common.AssertEqual(t, nil, err)
		common.AssertEqual(t, false, processor.enableDimensionalMetrics)

		// Create two gauges with same name but different tags.
		gauge1, err := NewGauge("test_metric").WithTag("host", "server1").WithMetricsProcessor(processor).Build()
		common.AssertEqual(t, nil, err)

		gauge2, err := NewGauge("test_metric").WithTag("host", "server2").WithMetricsProcessor(processor).Build()
		common.AssertEqual(t, nil, err)

		// Both should resolve to the same key since dimensional metrics are disabled.
		key1 := processor.createCompositeKey(gauge1.Key(), gauge1.Tags())
		key2 := processor.createCompositeKey(gauge2.Key(), gauge2.Tags())
		common.AssertEqual(t, "test_metric", key1)
		common.AssertEqual(t, "test_metric", key2)
		common.AssertEqual(t, key1, key2)

		// Only one metric should be registered (the first one).
		common.AssertEqual(t, 1, len(processor.MetricMap))
	})

	t.Run("DimensionalMetricsEnabled_SameNameWithDifferentTags", func(t *testing.T) {
		// Set environment variable to enable dimensional metrics.
		t.Setenv(EnableDimensionalMetricsEnvVar, "true")
		defer os.Unsetenv(EnableDimensionalMetricsEnvVar)

		err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
		processor := GetGlobalProcessor()
		common.AssertEqual(t, nil, err)
		common.AssertEqual(t, true, processor.enableDimensionalMetrics)

		// Create two gauges with same name but different tags.
		gauge1, err := NewGauge("test_metric").WithTag("host", "server1").WithMetricsProcessor(processor).Build()
		common.AssertEqual(t, nil, err)

		gauge2, err := NewGauge("test_metric").WithTag("host", "server2").WithMetricsProcessor(processor).Build()
		common.AssertEqual(t, nil, err)

		// Each should resolve to a different key since dimensional metrics are enabled.
		key1 := processor.createCompositeKey(gauge1.Key(), gauge1.Tags())
		key2 := processor.createCompositeKey(gauge2.Key(), gauge2.Tags())
		common.AssertEqual(t, "test_metric|host=server1", key1)
		common.AssertEqual(t, "test_metric|host=server2", key2)
		if key1 == key2 {
			t.Fatalf("Expected keys to be different, but both were: %s", key1)
		}

		// Both metrics should be registered as separate entities.
		common.AssertEqual(t, 2, len(processor.MetricMap))
	})

	t.Run("InvalidEnvironmentValue_DefaultsToDisabled", func(t *testing.T) {
		// Set invalid environment variable value.
		t.Setenv(EnableDimensionalMetricsEnvVar, "invalid_value")
		defer os.Unsetenv(EnableDimensionalMetricsEnvVar)

		err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
		processor := GetGlobalProcessor()
		common.AssertEqual(t, nil, err)
		common.AssertEqual(t, false, processor.enableDimensionalMetrics) // Should default to false
	})
}

func TestProcessorDimensionalTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Enable dimensional metrics for this test.
	t.Setenv(EnableDimensionalMetricsEnvVar, "true")
	defer os.Unsetenv(EnableDimensionalMetricsEnvVar)

	mockTransport := NewMockTransport(ctrl)
	err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	// Create a base gauge.
	baseGauge, err := NewGauge("cpu_usage").
		WithMetricsProcessor(processor).
		Build()
	common.AssertEqual(t, nil, err)

	usEastGauge := baseGauge.WithTag("region", "us-east")
	usWestGauge := baseGauge.WithTag("region", "us-west")

	// No need to register - baseGauge already registered in Build(), others registered by WithTag.

	common.AssertEqual(t, 3, len(processor.MetricMap))

	baseMetric, exists := processor.GetMetric("cpu_usage")
	common.AssertEqual(t, true, exists)
	common.AssertEqual(t, "cpu_usage", baseMetric.Key())

	usEastMetric, exists := processor.GetMetric("cpu_usage|region=us-east")
	common.AssertEqual(t, true, exists)
	common.AssertEqual(t, "cpu_usage", usEastMetric.Key())
	common.AssertEqual(t, "us-east", usEastMetric.Tags()["region"])

	usWestMetric, exists := processor.GetMetric("cpu_usage|region=us-west")
	common.AssertEqual(t, true, exists)
	common.AssertEqual(t, "cpu_usage", usWestMetric.Key())
	common.AssertEqual(t, "us-west", usWestMetric.Tags()["region"])

	mockTransport.EXPECT().Send(gomock.Any()).DoAndReturn(
		func(messages []model.MetricMessage) error {
			common.AssertEqual(t, 3, len(messages))

			for _, msg := range messages {
				common.AssertEqual(t, "cpu_usage", msg.Key)
				common.AssertEqual(t, model.MetricTypeGauge, msg.Type)

				if region, hasRegion := msg.Tags["region"]; hasRegion {
					switch region {
					case "us-east":
						common.AssertEqual(t, 75.0, msg.Value)
					case "us-west":
						common.AssertEqual(t, 50.0, msg.Value)
					default:
						common.AssertEqual(t, 100.0, msg.Value)
					}
				}
			}
			return nil
		}).Times(1)

	baseGauge.Set(100.0)
	usEastGauge.Set(75.0)
	usWestGauge.Set(50.0)

	messages := []model.MetricMessage{
		{Key: "cpu_usage", Type: model.MetricTypeGauge, Value: 100.0, Tags: baseGauge.Tags()},
		{Key: "cpu_usage", Type: model.MetricTypeGauge, Value: 75.0, Tags: usEastGauge.Tags()},
		{Key: "cpu_usage", Type: model.MetricTypeGauge, Value: 50.0, Tags: usWestGauge.Tags()},
	}
	err = processor.flushMessages(messages)
	common.AssertEqual(t, nil, err)

	common.AssertEqual(t, 100.0, baseGauge.CurrentValue())
	common.AssertEqual(t, 75.0, usEastGauge.CurrentValue())
	common.AssertEqual(t, 50.0, usWestGauge.CurrentValue())

	// Test dimensional tags passed at call time.
	mockTransport.EXPECT().Send(gomock.Any()).DoAndReturn(
		func(messages []model.MetricMessage) error {
			common.AssertEqual(t, 2, len(messages))

			for _, msg := range messages {
				common.AssertEqual(t, "cpu_usage", msg.Key)
				common.AssertEqual(t, model.MetricTypeGauge, msg.Type)

				if operation, hasOperation := msg.Tags["operation"]; hasOperation {
					switch operation {
					case "batch-process":
						common.AssertEqual(t, "server-1", msg.Tags["server"])
						common.AssertEqual(t, 85.0, msg.Value)
					case "realtime":
						common.AssertEqual(t, "server-2", msg.Tags["server"])
						common.AssertEqual(t, 45.0, msg.Value)
					default:
						t.Errorf("Unexpected operation tag: %s", operation)
					}
				} else {
					t.Errorf("Missing operation tag in message: %v", msg.Tags)
				}
			}
			return nil
		}).Times(1)

	baseGauge.WithDimensionalTags(map[string]string{"operation": "batch-process", "server": "server-1"}).Set(85.0)
	baseGauge.WithDimensionalTags(map[string]string{"operation": "realtime", "server": "server-2"}).Set(45.0)

	dimensionalMessages := []model.MetricMessage{
		{
			Key:   "cpu_usage",
			Type:  model.MetricTypeGauge,
			Value: 85.0,
			Tags: map[string]string{
				"operation": "batch-process",
				"server":    "server-1",
			},
		},
		{
			Key:   "cpu_usage",
			Type:  model.MetricTypeGauge,
			Value: 45.0,
			Tags: map[string]string{
				"operation": "realtime",
				"server":    "server-2",
			},
		},
	}
	err = processor.flushMessages(dimensionalMessages)
	common.AssertEqual(t, nil, err)

	baseTags := baseGauge.Tags()
	common.AssertEqual(t, 0, len(baseTags))

	mockTransport.EXPECT().Send(gomock.Any()).DoAndReturn(
		func(messages []model.MetricMessage) error {
			common.AssertEqual(t, 1, len(messages))
			msg := messages[0]

			common.AssertEqual(t, "us-east", msg.Tags["region"])
			common.AssertEqual(t, "cache-refresh", msg.Tags["operation"])
			common.AssertEqual(t, "cache-server", msg.Tags["server"])
			common.AssertEqual(t, 65.0, msg.Value)

			return nil
		}).Times(1)

	// Use dimensional variant with additional dimensional tags.
	usEastGauge.WithDimensionalTags(map[string]string{"operation": "cache-refresh", "server": "cache-server"}).Set(65.0)

	combinedMessage := model.MetricMessage{
		Key:   "cpu_usage",
		Type:  model.MetricTypeGauge,
		Value: 65.0,
		Tags: map[string]string{
			"region":    "us-east",
			"operation": "cache-refresh",
			"server":    "cache-server",
		},
	}
	err = processor.flushMessages([]model.MetricMessage{combinedMessage})
	common.AssertEqual(t, nil, err)
}

// TestDimensionalMetricsCompletelyDisabled verifies that when dimensional metrics are disabled,.
// NO dimensional behavior works - everything behaves like a single metric instance.
func TestDimensionalMetricsCompletelyDisabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransport := NewMockTransport(ctrl)

	// Ensure dimensional metrics are disabled.
	os.Unsetenv(EnableDimensionalMetricsEnvVar)
	defer os.Unsetenv(EnableDimensionalMetricsEnvVar)

	err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()
	common.AssertEqual(t, false, processor.enableDimensionalMetrics)

	t.Run("WithTags_CreatesOnlyOneMetric", func(t *testing.T) {
		// Create base metric.
		baseGauge, err := NewGauge("cpu_usage").WithMetricsProcessor(processor).Build()
		common.AssertEqual(t, nil, err)

		// Create dimensional variants - should NOT create separate metrics.
		usEastGauge := baseGauge.WithTag("region", "us-east")
		usWestGauge := baseGauge.WithTag("region", "us-west")
		euGauge := baseGauge.WithTag("region", "eu-central")

		// No need to manually register - baseGauge is registered by Build(), and.
		// the dimensional variants don't create new metrics when disabled.

		// Should only have ONE metric registered (first one wins).
		common.AssertEqual(t, 1, len(processor.MetricMap))

		// All should resolve to the same composite key.
		baseKey := processor.createCompositeKey(baseGauge.Key(), baseGauge.Tags())
		usEastKey := processor.createCompositeKey(usEastGauge.Key(), usEastGauge.Tags())
		usWestKey := processor.createCompositeKey(usWestGauge.Key(), usWestGauge.Tags())
		euKey := processor.createCompositeKey(euGauge.Key(), euGauge.Tags())

		common.AssertEqual(t, "cpu_usage", baseKey)
		common.AssertEqual(t, "cpu_usage", usEastKey)
		common.AssertEqual(t, "cpu_usage", usWestKey)
		common.AssertEqual(t, "cpu_usage", euKey)
	})

	t.Run("DimensionalTagsAtCallTime_IgnoredForIdentity", func(t *testing.T) {
		// Initialize global processor for this test.
		err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
		common.AssertEqual(t, nil, err)
		processor = GetGlobalProcessor()

		gauge, err := NewGauge("memory_usage").WithMetricsProcessor(processor).Build()
		common.AssertEqual(t, nil, err)

		// Set values with different dimensional tags - but since dimensional is disabled,.
		// WithDimensionalTags returns the same gauge instance, so these Set() calls.
		// all update the same metric instance.
		gauge.WithDimensionalTags(map[string]string{"server": "web-1", "tier": "frontend"}).Set(100.0)
		gauge.WithDimensionalTags(map[string]string{"server": "web-2", "tier": "frontend"}).Set(200.0)
		gauge.WithDimensionalTags(map[string]string{"server": "db-1", "tier": "backend"}).Set(300.0)

		// Should still only have ONE metric registered.
		common.AssertEqual(t, 1, len(processor.MetricMap))

		// The gauge should have the LAST set value (300.0) since it's the same metric instance
		common.AssertEqual(t, 300.0, gauge.CurrentValue())
	})

	t.Run("DifferentMetricNames_StillCreatesSeparateMetrics", func(t *testing.T) {
		// Create a fresh processor for this test.
		err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
		common.AssertEqual(t, nil, err)
		processor3 := GetGlobalProcessor()

		// Create metrics with different names - these SHOULD be separate.
		cpuGauge, err := NewGauge("cpu_usage").WithTag("region", "us-east").WithMetricsProcessor(processor3).Build()
		common.AssertEqual(t, nil, err)
		memoryGauge, err := NewGauge("memory_usage").WithTag("region", "us-east").WithMetricsProcessor(processor3).Build()
		common.AssertEqual(t, nil, err)
		diskGauge, err := NewGauge("disk_usage").WithTag("region", "us-east").WithMetricsProcessor(processor3).Build()
		common.AssertEqual(t, nil, err)

		// Should have 3 separate metrics (different names).
		common.AssertEqual(t, 3, len(processor3.MetricMap))

		// But all should ignore their tags in the composite key.
		common.AssertEqual(t, "cpu_usage", processor3.createCompositeKey(cpuGauge.Key(), cpuGauge.Tags()))
		common.AssertEqual(t, "memory_usage", processor3.createCompositeKey(memoryGauge.Key(), memoryGauge.Tags()))
		common.AssertEqual(t, "disk_usage", processor3.createCompositeKey(diskGauge.Key(), diskGauge.Tags()))
	})

	t.Run("AllMetricTypes_SameBehavior", func(t *testing.T) {
		// Create a fresh processor for this test.
		err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
		common.AssertEqual(t, nil, err)
		processor4 := GetGlobalProcessor()

		// Test all metric types behave the same way.
		_, err = NewGauge("test_gauge").WithTag("type", "gauge1").WithMetricsProcessor(processor4).Build()
		common.AssertEqual(t, nil, err)
		_, err = NewGauge("test_gauge").WithTag("type", "gauge2").WithMetricsProcessor(processor4).Build()
		common.AssertEqual(t, nil, err)

		_, err = NewCounter("test_counter").WithTag("type", "counter1").WithMetricsProcessor(processor4).Build()
		common.AssertEqual(t, nil, err)
		_, err = NewCounter("test_counter").WithTag("type", "counter2").WithMetricsProcessor(processor4).Build()
		common.AssertEqual(t, nil, err)

		_, err = NewTimer("test_timer").WithTag("type", "timer1").WithMetricsProcessor(processor4).Build()
		common.AssertEqual(t, nil, err)
		_, err = NewTimer("test_timer").WithTag("type", "timer2").WithMetricsProcessor(processor4).Build()
		common.AssertEqual(t, nil, err)

		// Should only have 3 metrics total (one per metric name, tags ignored).
		common.AssertEqual(t, 3, len(processor4.MetricMap))
	})
}

// TestDimensionalMetricsCompletelyEnabled verifies that when dimensional metrics are enabled,.
// ALL dimensional behavior works - every unique tag combination creates separate metrics.
func TestDimensionalMetricsCompletelyEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransport := NewMockTransport(ctrl)

	// Enable dimensional metrics.
	t.Setenv(EnableDimensionalMetricsEnvVar, "true")
	defer os.Unsetenv(EnableDimensionalMetricsEnvVar)

	err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()
	common.AssertEqual(t, true, processor.enableDimensionalMetrics)

	t.Run("WithTags_CreatesMultipleMetrics", func(t *testing.T) {
		// Create base metric.
		baseGauge, err := NewGauge("cpu_usage").WithMetricsProcessor(processor).Build()
		common.AssertEqual(t, nil, err)

		// Create dimensional variants - should create separate metrics.
		usEastGauge := baseGauge.WithTag("region", "us-east")
		usWestGauge := baseGauge.WithTag("region", "us-west")
		euGauge := baseGauge.WithTag("region", "eu-central")

		// Create more complex dimensional variants.
		usEastProdGauge := baseGauge.WithTag("region", "us-east").WithTag("env", "prod")
		usEastStagingGauge := baseGauge.WithTag("region", "us-east").WithTag("env", "staging")

		// WithTag automatically registers new metrics when dimensional metrics are enabled.

		// Should have 6 separate metrics registered.
		common.AssertEqual(t, 6, len(processor.MetricMap))

		// Each should resolve to a unique composite key.
		baseKey := processor.createCompositeKey(baseGauge.Key(), baseGauge.Tags())
		usEastKey := processor.createCompositeKey(usEastGauge.Key(), usEastGauge.Tags())
		usWestKey := processor.createCompositeKey(usWestGauge.Key(), usWestGauge.Tags())
		euKey := processor.createCompositeKey(euGauge.Key(), euGauge.Tags())
		usEastProdKey := processor.createCompositeKey(usEastProdGauge.Key(), usEastProdGauge.Tags())
		usEastStagingKey := processor.createCompositeKey(usEastStagingGauge.Key(), usEastStagingGauge.Tags())

		common.AssertEqual(t, "cpu_usage", baseKey)
		common.AssertEqual(t, "cpu_usage|region=us-east", usEastKey)
		common.AssertEqual(t, "cpu_usage|region=us-west", usWestKey)
		common.AssertEqual(t, "cpu_usage|region=eu-central", euKey)
		common.AssertEqual(t, "cpu_usage|env=prod,region=us-east", usEastProdKey)
		common.AssertEqual(t, "cpu_usage|env=staging,region=us-east", usEastStagingKey)

		// Verify each metric maintains separate state.
		baseGauge.Set(10.0)
		usEastGauge.Set(20.0)
		usWestGauge.Set(30.0)
		euGauge.Set(40.0)
		usEastProdGauge.Set(50.0)
		usEastStagingGauge.Set(60.0)

		common.AssertEqual(t, 10.0, baseGauge.CurrentValue())
		common.AssertEqual(t, 20.0, usEastGauge.CurrentValue())
		common.AssertEqual(t, 30.0, usWestGauge.CurrentValue())
		common.AssertEqual(t, 40.0, euGauge.CurrentValue())
		common.AssertEqual(t, 50.0, usEastProdGauge.CurrentValue())
		common.AssertEqual(t, 60.0, usEastStagingGauge.CurrentValue())
	})

	t.Run("DimensionalTagsAtCallTime_CreateSeparateEmissions", func(t *testing.T) {
		// Create a fresh processor for this test.
		err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
		common.AssertEqual(t, nil, err)
		processor = GetGlobalProcessor()

		gauge, err := NewGauge("memory_usage").WithMetricsProcessor(processor).Build()
		common.AssertEqual(t, nil, err)

		// Mock transport should receive 3 separate messages with different tag combinations.
		mockTransport.EXPECT().Send(gomock.Any()).DoAndReturn(
			func(messages []model.MetricMessage) error {
				common.AssertEqual(t, 3, len(messages))

				// Verify each message has different tags.
				for _, msg := range messages {
					common.AssertEqual(t, "memory_usage", msg.Key)
					common.AssertEqual(t, model.MetricTypeGauge, msg.Type)
					switch msg.Tags["server"] {
					case "web-1":
						common.AssertEqual(t, "frontend", msg.Tags["tier"])
						common.AssertEqual(t, 100.0, msg.Value)
					case "web-2":
						common.AssertEqual(t, "frontend", msg.Tags["tier"])
						common.AssertEqual(t, 200.0, msg.Value)
					case "db-1":
						common.AssertEqual(t, "backend", msg.Tags["tier"])
						common.AssertEqual(t, 300.0, msg.Value)
					default:
						t.Errorf("Unexpected server tag: %s", msg.Tags["server"])
					}
				}
				return nil
			}).Times(1)

		// Set values with different dimensional tags - should create separate emissions.
		gauge.WithDimensionalTags(map[string]string{"server": "web-1", "tier": "frontend"}).Set(100.0)
		gauge.WithDimensionalTags(map[string]string{"server": "web-2", "tier": "frontend"}).Set(200.0)
		gauge.WithDimensionalTags(map[string]string{"server": "db-1", "tier": "backend"}).Set(300.0)

		// Manually flush to trigger the mock expectation.
		messages := []model.MetricMessage{
			{Key: "memory_usage", Type: model.MetricTypeGauge, Value: 100.0, Tags: map[string]string{"server": "web-1", "tier": "frontend"}},
			{Key: "memory_usage", Type: model.MetricTypeGauge, Value: 200.0, Tags: map[string]string{"server": "web-2", "tier": "frontend"}},
			{Key: "memory_usage", Type: model.MetricTypeGauge, Value: 300.0, Tags: map[string]string{"server": "db-1", "tier": "backend"}},
		}
		err = processor.flushMessages(messages)
		common.AssertEqual(t, nil, err)

		// Base gauge should only have one registration but maintain its own state.
		common.AssertEqual(t, 1, len(processor.MetricMap))
		// The gauge itself maintains whatever was the last call without dimensional tags.
		// Since we only called with dimensional tags, it should still be at its initial value.
		common.AssertEqual(t, 0.0, gauge.CurrentValue())
	})

	t.Run("AllMetricTypes_CreateSeparateDimensionalInstances", func(t *testing.T) {
		// Create a fresh processor for this test.
		err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
		common.AssertEqual(t, nil, err)
		processor3 := GetGlobalProcessor()

		// Test all metric types create dimensional instances.
		_, err = NewGauge("test_metric").WithTag("type", "gauge").WithTag("instance", "1").WithMetricsProcessor(processor3).Build()
		common.AssertEqual(t, nil, err)
		_, err = NewGauge("test_metric").WithTag("type", "gauge").WithTag("instance", "2").WithMetricsProcessor(processor3).Build()
		common.AssertEqual(t, nil, err)

		_, err = NewCounter("test_metric").WithTag("type", "counter").WithTag("instance", "1").WithMetricsProcessor(processor3).Build()
		common.AssertEqual(t, nil, err)
		_, err = NewCounter("test_metric").WithTag("type", "counter").WithTag("instance", "2").WithMetricsProcessor(processor3).Build()
		common.AssertEqual(t, nil, err)

		_, err = NewTimer("test_metric").WithTag("type", "timer").WithTag("instance", "1").WithMetricsProcessor(processor3).Build()
		common.AssertEqual(t, nil, err)
		_, err = NewTimer("test_metric").WithTag("type", "timer").WithTag("instance", "2").WithMetricsProcessor(processor3).Build()
		common.AssertEqual(t, nil, err)

		// Should have 6 separate metrics (each unique tag combination creates a separate metric).
		common.AssertEqual(t, 6, len(processor3.MetricMap))

		// Verify each has a unique composite key.
		keys := make(map[string]bool)
		for key := range processor3.MetricMap {
			keys[key] = true
		}
		common.AssertEqual(t, 6, len(keys)) // All keys should be unique
	})

	t.Run("ComplexTagCombinations_AllCreateUniqueMetrics", func(t *testing.T) {
		// Create a fresh processor for this test.
		err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
		common.AssertEqual(t, nil, err)
		processor4 := GetGlobalProcessor()

		// Create metrics with complex tag combinations.
		baseMetric, err := NewGauge("api_requests").WithMetricsProcessor(processor4).Build()
		common.AssertEqual(t, nil, err)

		// Different combinations should create different metrics.
		m1 := baseMetric.WithTags(map[string]string{"method": "GET", "status": "200"})
		m2 := baseMetric.WithTags(map[string]string{"method": "GET", "status": "404"})
		m3 := baseMetric.WithTags(map[string]string{"method": "POST", "status": "200"})
		m4 := baseMetric.WithTags(map[string]string{"method": "POST", "status": "500"})
		m5 := baseMetric.WithTags(map[string]string{"method": "GET", "status": "200", "region": "us-east"})

		// Should have 6 unique metrics (baseMetric + 5 variants).
		common.AssertEqual(t, 6, len(processor4.MetricMap))

		// Each should have independent state.
		baseMetric.Set(1.0)
		m1.Set(2.0)
		m2.Set(3.0)
		m3.Set(4.0)
		m4.Set(5.0)
		m5.Set(6.0)

		common.AssertEqual(t, 1.0, baseMetric.CurrentValue())
		common.AssertEqual(t, 2.0, m1.CurrentValue())
		common.AssertEqual(t, 3.0, m2.CurrentValue())
		common.AssertEqual(t, 4.0, m3.CurrentValue())
		common.AssertEqual(t, 5.0, m4.CurrentValue())
		common.AssertEqual(t, 6.0, m5.CurrentValue())
	})
}

func TestProcessorLoggerCreation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockTransport := NewMockTransport(ctrl)

	t.Run("UsesDefaultProcessIDWhenEnvNotSet", func(t *testing.T) {
		// Clear environment variable for this test
		t.Setenv(common.EnvironmentKeyProcessID, "")

		err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
		common.AssertEqual(t, nil, err)

		processor := GetGlobalProcessor()
		if processor.logger == nil {
			t.Fatal("processor.logger should not be nil")
		}
	})

	t.Run("UsesEnvironmentProcessID", func(t *testing.T) {
		testProcessID := "test_process_123"
		t.Setenv(common.EnvironmentKeyProcessID, testProcessID)

		err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
		common.AssertEqual(t, nil, err)

		processor := GetGlobalProcessor()
		if processor.logger == nil {
			t.Fatal("processor.logger should not be nil")
		}
	})

	t.Run("LoggerNotNilAfterInit", func(t *testing.T) {
		err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
		common.AssertEqual(t, nil, err)

		processor := GetGlobalProcessor()
		if processor == nil {
			t.Fatal("processor should not be nil")
		}
		if processor.logger == nil {
			t.Fatal("processor.logger should not be nil")
		}
	})
}

// TestProcessor_GetLogger tests that getLogger returns the processor's logger.
func TestProcessor_GetLogger(t *testing.T) {
	processor := &Processor{
		logger: log.GetDefaultLogger("test"),
	}
	common.AssertEqual(t, processor.logger, processor.getLogger())
}

// TestWithTags_InvalidTagNotSet tests that invalid tags are not set.
func TestWithTags_InvalidTagNotSet(t *testing.T) {
	processor := &Processor{
		logger:                   log.GetDefaultLogger("test"),
		enableDimensionalMetrics: false,
		GlobalTags:               make(map[string]string),
		MetricMap:                make(map[string]model.Metric),
	}

	counter := &Counter{
		baseMetric: &baseMetric{
			key:       "test",
			processor: processor,
			tags:      make(map[string]string),
		},
	}

	// Invalid tag (colon in key) should not be set
	counter.WithTags(map[string]string{"invalid:key": "value"})

	_, exists := counter.Tags()["invalid:key"]
	common.AssertEqual(t, false, exists)
}

// TestWithTags_ValidTagIsSet tests that valid tags are properly set.
func TestWithTags_ValidTagIsSet(t *testing.T) {
	processor := &Processor{
		logger:                   log.GetDefaultLogger("test"),
		enableDimensionalMetrics: false,
		GlobalTags:               make(map[string]string),
		MetricMap:                make(map[string]model.Metric),
	}

	gauge := &Gauge{
		baseMetric: &baseMetric{
			key:       "test",
			processor: processor,
			tags:      make(map[string]string),
		},
	}

	gauge.WithTags(map[string]string{"environment": "prod"})

	common.AssertEqual(t, "prod", gauge.Tags()["environment"])
}

// TestWithTags_NilLoggerDoesNotPanic tests nil logger safety.
func TestWithTags_NilLoggerDoesNotPanic(t *testing.T) {
	processor := &Processor{
		logger:                   nil, // Nil logger
		enableDimensionalMetrics: false,
		GlobalTags:               make(map[string]string),
		MetricMap:                make(map[string]model.Metric),
	}

	timer := &Timer{
		baseMetric: &baseMetric{
			key:       "test",
			processor: processor,
			tags:      make(map[string]string),
		},
	}

	// Should not panic with invalid tag and nil logger
	timer.WithTags(map[string]string{"invalid:key": "value"})

	// Invalid tag still shouldn't be set
	_, exists := timer.Tags()["invalid:key"]
	common.AssertEqual(t, false, exists)
}
