/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package metrics

import (
	"errors"
	"testing"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/derived"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/model"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/samplers"

	"github.com/golang/mock/gomock"
)

func TestNewFactory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCrashReporter := NewMockCrashReporter(ctrl)
	mockProcessor := NewMockMetricsProcessor(ctrl)
	factory, err := NewFactory().WithCrashReporter(mockCrashReporter).WithProcessor(mockProcessor).Build()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	// Verify factory is created with correct defaults
	if factory == nil {
		t.Fatal("Expected factory to not be nil")
	}
	common.AssertEqual(t, mockProcessor, factory.processor)
	common.AssertEqual(t, mockCrashReporter, factory.crashReporter)
	if factory.sampler == nil {
		t.Fatal("Expected factory.sampler to not be nil")
	}
	if factory.tags == nil {
		t.Fatal("Expected factory.tags to not be nil")
	}
	common.AssertEqual(t, 0, len(factory.tags))

	// Verify default sampler is AllSampler
	if !factory.sampler.ShouldSample() {
		t.Error("Expected default sampler to sample all metrics")
	}
}

func TestFactory_WithSampler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCrashReporter := NewMockCrashReporter(ctrl)
	mockProcessor := NewMockMetricsProcessor(ctrl)
	noneSampler := samplers.NewNone()

	factory, err := NewFactory().
		WithCrashReporter(mockCrashReporter).
		WithProcessor(mockProcessor).
		WithSampler(noneSampler).
		Build()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	// Verify sampler was set correctly
	common.AssertEqual(t, noneSampler, factory.sampler)
	if factory.sampler.ShouldSample() {
		t.Error("Expected NoneSampler to not sample metrics")
	}
}

func TestFactory_WithTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCrashReporter := NewMockCrashReporter(ctrl)
	mockProcessor := NewMockMetricsProcessor(ctrl)
	tags := map[string]string{
		"env":     "test",
		"service": "metrics",
		"version": "1.0",
	}

	factory, err := NewFactory().
		WithCrashReporter(mockCrashReporter).
		WithProcessor(mockProcessor).
		WithTags(tags).
		Build()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	// Verify tags were set correctly
	if len(factory.tags) != len(tags) {
		t.Errorf("Expected %d tags, got %d", len(tags), len(factory.tags))
	}
	common.AssertEqual(t, "test", factory.tags["env"])
	common.AssertEqual(t, "metrics", factory.tags["service"])
	common.AssertEqual(t, "1.0", factory.tags["version"])
}

func TestFactory_WithTags_EmptyTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCrashReporter := NewMockCrashReporter(ctrl)
	mockProcessor := NewMockMetricsProcessor(ctrl)
	emptyTags := map[string]string{}

	factory, err := NewFactory().
		WithCrashReporter(mockCrashReporter).
		WithProcessor(mockProcessor).
		WithTags(emptyTags).
		Build()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	// Verify empty tags were set correctly
	if len(factory.tags) != 0 {
		t.Errorf("Expected 0 tags, got %d", len(factory.tags))
	}
}

func TestFactory_WithTags_NilTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCrashReporter := NewMockCrashReporter(ctrl)
	mockProcessor := NewMockMetricsProcessor(ctrl)

	factory, err := NewFactory().
		WithCrashReporter(mockCrashReporter).
		WithProcessor(mockProcessor).
		WithTags(nil).
		Build()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	// Verify nil tags were handled correctly
	if factory.tags == nil {
		t.Error("Expected factory.tags to not be nil")
	}
}

func TestFactory_Gauge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCrashReporter := NewMockCrashReporter(ctrl)
	mockProcessor := NewMockMetricsProcessor(ctrl)
	customSampler := samplers.NewAll()
	tags := map[string]string{"env": "test"}

	factory, err := NewFactory().
		WithCrashReporter(mockCrashReporter).
		WithProcessor(mockProcessor).
		WithSampler(customSampler).
		WithTags(tags).
		Build()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	// Expect registerMetric to be called
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	gauge, err := factory.Gauge("test_gauge")
	common.AssertEqual(t, nil, err)

	// Verify gauge was created and configured correctly
	if gauge == nil {
		t.Fatal("Expected gauge to not be nil")
	}
	common.AssertEqual(t, "test_gauge", gauge.Key())
	common.AssertEqual(t, model.MetricTypeGauge, gauge.MetricType())

	// Verify tags were applied
	gaugeTags := gauge.Tags()
	common.AssertEqual(t, "test", gaugeTags["env"])
}
func TestFactory_Counter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Set up factory with custom sampler and tags
	customSampler := samplers.NewAll()
	tags := map[string]string{"service": "counter-test"}

	mockProcessor := NewMockMetricsProcessor(ctrl)
	mockCrashReporter := NewMockCrashReporter(ctrl)
	factory, err := NewFactory().WithProcessor(mockProcessor).WithCrashReporter(mockCrashReporter).WithSampler(customSampler).WithTags(tags).Build()

	if err != nil {
		t.Fatalf("Failed to create test factory: %v", err)
	}

	// Expect registerMetric to be called
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	counter, err := factory.Counter("test_counter")
	common.AssertEqual(t, nil, err)

	// Verify counter was created and configured correctly
	if counter == nil {
		t.Fatal("Expected counter to not be nil")
	}
	common.AssertEqual(t, "test_counter", counter.Key())
	common.AssertEqual(t, model.MetricTypeCounter, counter.MetricType())

	// Verify tags were applied
	counterTags := counter.Tags()
	common.AssertEqual(t, "counter-test", counterTags["service"])
}

func TestFactory_Timer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	customSampler := samplers.NewAll()
	tags := map[string]string{"type": "timer"}

	mockCrashReporter := NewMockCrashReporter(ctrl)
	mockProcessor := NewMockMetricsProcessor(ctrl)

	factory, err := NewFactory().
		WithCrashReporter(mockCrashReporter).
		WithProcessor(mockProcessor).
		WithSampler(customSampler).
		WithTags(tags).
		Build()

	if err != nil {
		t.Fatalf("Failed to create test factory: %v", err)
	}

	// Expect registerMetric to be called
	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	timer, err := factory.Timer("test_timer")
	common.AssertEqual(t, nil, err)

	// Verify timer was created and configured correctly
	if timer == nil {
		t.Fatal("Expected timer to not be nil")
	}
	common.AssertEqual(t, "test_timer", timer.Key())
	common.AssertEqual(t, model.MetricTypeTimer, timer.MetricType())

	// Verify tags were applied
	timerTags := timer.Tags()
	common.AssertEqual(t, "timer", timerTags["type"])
}

func TestFactory_FluentInterface(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCrashReporter := NewMockCrashReporter(ctrl)
	mockProcessor := NewMockMetricsProcessor(ctrl)

	// Test fluent interface chaining
	tags := map[string]string{"env": "test", "version": "1.0"}
	sampler := samplers.NewNone()

	// Expect registerMetric to be called three times
	expectedKeys := map[string]bool{
		"chained_gauge":   false,
		"chained_counter": false,
		"chained_timer":   false,
	}

	mockProcessor.EXPECT().registerMetric(gomock.Any()).Do(func(metric model.Metric) {
		key := metric.Key()
		if _, exists := expectedKeys[key]; exists {
			expectedKeys[key] = true
		} else {
			t.Errorf("Unexpected metric key: %s", key)
		}
	}).Times(3)

	factory, err := NewFactory().
		WithCrashReporter(mockCrashReporter).
		WithProcessor(mockProcessor).
		WithSampler(sampler).
		WithTags(tags).
		Build()
	if err != nil {
		t.Fatalf("Failed to create test transport: %v", err)
	}
	// Create metrics using the configured factory
	gauge, err := factory.Gauge("chained_gauge")
	common.AssertEqual(t, nil, err)
	counter, err := factory.Counter("chained_counter")
	common.AssertEqual(t, nil, err)
	timer, err := factory.Timer("chained_timer")
	common.AssertEqual(t, nil, err)

	// Verify all metrics were created with the same configuration
	if gauge == nil || counter == nil || timer == nil {
		t.Fatal("Expected all metrics to not be nil")
	}

	// Verify tags were applied to all metrics
	for _, metric := range []model.Metric{gauge, counter, timer} {
		metricTags := metric.Tags()
		common.AssertEqual(t, "test", metricTags["env"])
		common.AssertEqual(t, "1.0", metricTags["version"])
	}
}

func TestFactory_WithDifferentSamplers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCrashReporter := NewMockCrashReporter(ctrl)
	mockProcessor := NewMockMetricsProcessor(ctrl)

	testCases := []struct {
		sampler  samplers.Sampler
		name     string
		expected bool
	}{
		{
			name:     "AllSampler",
			sampler:  samplers.NewAll(),
			expected: true,
		},
		{
			name:     "NoneSampler",
			sampler:  samplers.NewNone(),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factory := NewFactory().WithCrashReporter(mockCrashReporter).WithSampler(tc.sampler).WithProcessor(mockProcessor)

			// Verify sampler behavior
			common.AssertEqual(t, tc.expected, factory.sampler.ShouldSample())
		})
	}
}

func TestFactory_DefaultTimerPercentile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCrashReporter := NewMockCrashReporter(ctrl)
	mockProcessor := NewMockMetricsProcessor(ctrl)

	factory, err := NewFactory().
		WithCrashReporter(mockCrashReporter).
		WithProcessor(mockProcessor).
		Build()

	if err != nil {
		t.Fatalf("Failed to build factory: %v", err)
	}

	mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(1)

	timer, err := factory.Timer("test_timer")
	common.AssertEqual(t, nil, err)
	if timer == nil {
		t.Fatal("Expected timer to not be nil")
	}

	derivedMetrics := timer.DerivedMetrics()
	common.AssertEqual(t, 1, len(derivedMetrics))
	_, ok := derivedMetrics[0].(*derived.PercentileMetric)
	if !ok {
		t.Fatal("Expected first derived metric to be default derived metric")
	}
}

func TestFactory_MetricsCreateSeparateInstancesWithAdditionalTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Enable dimensional metrics for this test
	t.Setenv(EnableDimensionalMetricsEnvVar, "true")

	mockCrashReporter := NewMockCrashReporter(ctrl)
	mockProcessor := NewMockMetricsProcessor(ctrl)
	factoryDefaultTags := map[string]string{"env": "test"}

	factory, err := NewFactory().
		WithCrashReporter(mockCrashReporter).
		WithProcessor(mockProcessor).
		WithTags(factoryDefaultTags).
		Build()

	common.AssertEqual(t, nil, err)

	t.Run("GaugeWithTagCreatesNewMetricInstance", func(t *testing.T) {
		mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(2) // Original gauge + new tagged gauge
		mockProcessor.EXPECT().dimensionalMetricsEnabled().Return(true).AnyTimes()

		originalGauge, err := factory.Gauge("test_gauge")
		common.AssertEqual(t, nil, err)
		gaugeWithRouteTag := originalGauge.WithTag("route", "/api/v1")

		common.AssertEqual(t, "test", originalGauge.Tags()["env"])
		common.AssertEqual(t, "", originalGauge.Tags()["route"])

		common.AssertEqual(t, "test", gaugeWithRouteTag.Tags()["env"])
		common.AssertEqual(t, "/api/v1", gaugeWithRouteTag.Tags()["route"])
	})

	t.Run("CounterWithTagsCreatesNewMetricInstance", func(t *testing.T) {
		mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(2)
		mockProcessor.EXPECT().dimensionalMetricsEnabled().Return(true).AnyTimes()

		originalCounter, err := factory.Counter("test_counter")
		common.AssertEqual(t, nil, err)
		additionalTags := map[string]string{
			"status": "200",
			"method": "GET",
		}
		counterWithHTTPTags := originalCounter.WithTags(additionalTags)

		common.AssertEqual(t, "test", originalCounter.Tags()["env"])
		common.AssertEqual(t, "", originalCounter.Tags()["status"])
		common.AssertEqual(t, "", originalCounter.Tags()["method"])

		common.AssertEqual(t, "test", counterWithHTTPTags.Tags()["env"])
		common.AssertEqual(t, "200", counterWithHTTPTags.Tags()["status"])
		common.AssertEqual(t, "GET", counterWithHTTPTags.Tags()["method"])
	})

	t.Run("TimerWithTagsCreatesChainOfNewInstances", func(t *testing.T) {
		mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(3)
		mockProcessor.EXPECT().dimensionalMetricsEnabled().Return(true).AnyTimes()

		baseTimer, err := factory.Timer("test_timer")
		common.AssertEqual(t, nil, err)
		timerWithEndpoint := baseTimer.WithTag("endpoint", "/users")
		timerWithAllTags := timerWithEndpoint.WithTags(map[string]string{
			"region":  "us-west-2",
			"version": "v2",
		})

		common.AssertEqual(t, "test", baseTimer.Tags()["env"])
		common.AssertEqual(t, 1, len(baseTimer.Tags()))

		common.AssertEqual(t, "test", timerWithEndpoint.Tags()["env"])
		common.AssertEqual(t, "/users", timerWithEndpoint.Tags()["endpoint"])
		common.AssertEqual(t, 2, len(timerWithEndpoint.Tags()))

		common.AssertEqual(t, "test", timerWithAllTags.Tags()["env"])
		common.AssertEqual(t, "/users", timerWithAllTags.Tags()["endpoint"])
		common.AssertEqual(t, "us-west-2", timerWithAllTags.Tags()["region"])
		common.AssertEqual(t, "v2", timerWithAllTags.Tags()["version"])
		common.AssertEqual(t, 4, len(timerWithAllTags.Tags()))
	})

	t.Run("WithTags idempotency", func(t *testing.T) {
		// Calling WithTags twice with same tags should result in same metric being returned from processor
		mockProcessor.EXPECT().registerMetric(gomock.Any()).Times(3) // original + 2 calls to WithTags
		mockProcessor.EXPECT().dimensionalMetricsEnabled().Return(true).AnyTimes()

		gauge, err := factory.Gauge("idempotent_gauge")
		common.AssertEqual(t, nil, err)
		tags := map[string]string{"key": "value"}

		gauge1 := gauge.WithTags(tags)
		gauge2 := gauge.WithTags(tags)

		// Both should have the same tags
		common.AssertEqual(t, gauge1.Tags()["key"], gauge2.Tags()["key"])
		common.AssertEqual(t, gauge1.Tags()["env"], gauge2.Tags()["env"])
	})
}

func TestFactoryBuilder_Build_ProcessorTransportLogic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCrashReporter := NewMockCrashReporter(ctrl)
	t.Run("NilProcessor_WithTransport_CreatesProcessor", func(t *testing.T) {
		resetGlobalProcessor()
		mockTransport := NewMockTransport(ctrl)

		factory, err := NewFactory().WithCrashReporter(mockCrashReporter).WithTransport(mockTransport).Build()

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if factory.processor == nil {
			t.Fatal("Expected processor to be created")
		}
	})

	t.Run("NilProcessor_NilTransport_ReturnsError", func(t *testing.T) {
		resetGlobalProcessor()
		_, err := NewFactory().Build()

		if err == nil {
			t.Fatal("Expected error when both processor and transport are nil")
		}

		var gameLiftErr *common.GameLiftError
		if !errors.As(err, &gameLiftErr) {
			t.Fatalf("Expected GameLiftError, got: %T", err)
		}
		if gameLiftErr.ErrorType != common.MetricConfigurationException {
			t.Errorf("Expected MetricConfigurationException, got: %v", gameLiftErr.ErrorType)
		}
	})

	t.Run("ExistingProcessor_IgnoresTransport", func(t *testing.T) {
		mockCrashReporter := NewMockCrashReporter(ctrl)
		mockProcessor := NewMockMetricsProcessor(ctrl)
		mockTransport := NewMockTransport(ctrl)

		factory, err := NewFactory().WithCrashReporter(mockCrashReporter).WithProcessor(mockProcessor).WithTransport(mockTransport).Build()

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		common.AssertEqual(t, mockProcessor, factory.processor)
	})
}

func TestFactory_OnStartGameSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCrashReporter := NewMockCrashReporter(ctrl)
	mockProcessor := NewMockMetricsProcessor(ctrl)
	sessionId := "test-session-123"

	factory, err := NewFactory().
		WithCrashReporter(mockCrashReporter).
		WithProcessor(mockProcessor).
		Build()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	// Expect the crash reporter and processor to be called with the session ID
	mockCrashReporter.EXPECT().TagGameSession(sessionId).Return(nil)
	mockProcessor.EXPECT().SetGlobalTag("session_id", sessionId)

	factory.OnStartGameSession(sessionId)
}
