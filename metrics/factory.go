/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
package metrics

import (
	"fmt"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/derived"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/samplers"
)

// IFactory defines the interface for metrics factory operations
//
//go:generate mockgen -source=factory.go -destination=factory_mock_test.go -package=metrics
type IFactory interface {
	Gauge(key string) (*Gauge, error)
	Counter(key string) (*Counter, error)
	Timer(key string) (*Timer, error)
	OnProcessStart()
	OnStartGameSession(sessionId string)
	OnProcessTermination()
}

// Factory provides a convenient way to create metrics with common configurations.
type Factory struct {
	crashReporter CrashReporter
	processor     MetricsProcessor
	sampler       samplers.Sampler
	tags          map[string]string
}

// NewFactory returns a new factory builder.
func NewFactory() *FactoryBuilder {
	return &FactoryBuilder{
		sampler: samplers.NewAll(),
		tags:    make(map[string]string),
	}
}

// FactoryBuilder builds a Factory with fluent method chaining.
type FactoryBuilder struct {
	crashReporter CrashReporter
	processor     MetricsProcessor
	transport     Transport
	sampler       samplers.Sampler
	tags          map[string]string
}

// WithCrashReporter sets the crash reporter for the factory
func (b *FactoryBuilder) WithCrashReporter(crashReporter CrashReporter) *FactoryBuilder {
	b.crashReporter = crashReporter
	return b
}

// WithProcessor sets the metrics processor for the factory
func (b *FactoryBuilder) WithProcessor(processor MetricsProcessor) *FactoryBuilder {
	b.processor = processor
	return b
}

// WithSampler sets the default sampler for metrics created by this factory.
func (b *FactoryBuilder) WithSampler(sampler samplers.Sampler) *FactoryBuilder {
	b.sampler = sampler
	return b
}

// WithTags sets default tags for metrics created by this factory.
func (b *FactoryBuilder) WithTags(tags map[string]string) *FactoryBuilder {
	for k, v := range tags {
		b.tags[k] = v
	}

	return b
}

// WithTag adds a single tag for metrics created by this factory.
func (b *FactoryBuilder) WithTag(key, value string) *FactoryBuilder {
	b.tags[key] = value
	return b
}

// WithTransport sets the transport.
func (b *FactoryBuilder) WithTransport(transport Transport) *FactoryBuilder {
	b.transport = transport
	return b
}

// Build creates and returns the configured Factory.
func (b *FactoryBuilder) Build() (*Factory, error) {
	if b.crashReporter == nil {
		return nil, common.NewGameLiftError(
			common.MetricConfigurationException,
			"Crash Reporter required",
			"crash reporter must be set",
		)
	}
	if b.processor == nil && b.transport != nil {
		// Use the global processor if available, otherwise try to create one
		if HasGlobalProcessor() {
			b.processor = GetGlobalProcessor()
		} else {
			err := InitMetricsProcessor(WithTransport(b.transport))
			if err != nil {
				return nil, common.NewGameLiftError(
					common.MetricConfigurationException,
					"Failed to create metrics processor",
					fmt.Sprintf("failed to create metrics processor: %v", err),
				)
			}
			b.processor = GetGlobalProcessor()
		}
	} else if b.processor == nil {
		// Try to use the global processor if available
		if HasGlobalProcessor() {
			b.processor = GetGlobalProcessor()
		} else {
			return nil, common.NewGameLiftError(
				common.MetricConfigurationException,
				"Metrics processor required",
				"metrics processor must be set, transport provided, or global processor initialized with InitMetricsProcessor()",
			)
		}
	}

	// Deep copy tags to avoid shared state
	factoryTags := make(map[string]string)
	for k, v := range b.tags {
		factoryTags[k] = v
	}

	return &Factory{
		crashReporter: b.crashReporter,
		processor:     b.processor,
		sampler:       b.sampler,
		tags:          factoryTags,
	}, nil
}

// Gauge creates a new gauge metric with factory defaults.
func (f *Factory) Gauge(key string) (*Gauge, error) {
	gauge, err := NewGauge(key).
		WithMetricsProcessor(f.processor).
		WithTags(f.tags).
		WithSampler(f.sampler).
		Build()
	if err != nil {
		return nil, common.NewGameLiftError(
			common.MetricConfigurationException,
			"Failed to create gauge metric",
			fmt.Sprintf("failed to create gauge metric: %v", err),
		)
	}
	return gauge, nil
}

// Counter creates a new counter metric with factory defaults.
func (f *Factory) Counter(key string) (*Counter, error) {
	counter, err := NewCounter(key).
		WithMetricsProcessor(f.processor).
		WithTags(f.tags).
		WithSampler(f.sampler).
		Build()
	if err != nil {
		return nil, common.NewGameLiftError(
			common.MetricConfigurationException,
			"Failed to create counter metric",
			fmt.Sprintf("failed to create counter metric: %v", err),
		)
	}
	return counter, nil
}

// Timer creates a new timer metric with factory default and a derived percentile metric.
func (f *Factory) Timer(key string) (*Timer, error) {
	p := derived.NewPercentile(derived.P50, derived.P90, derived.P95)
	timer, err := NewTimer(key).
		WithMetricsProcessor(f.processor).
		WithTags(f.tags).
		WithSampler(f.sampler).
		WithDerivedMetrics(p).
		Build()
	if err != nil {
		return nil, common.NewGameLiftError(
			common.MetricConfigurationException,
			"Failed to create timer metric",
			fmt.Sprintf("failed to create timer metric: %v", err),
		)
	}
	return timer, nil
}
func (f *Factory) OnProcessStart() {
	_ = f.crashReporter.RegisterProcess()
}

func (f *Factory) OnStartGameSession(sessionId string) {
	_ = f.crashReporter.TagGameSession(sessionId)
	_ = f.processor.SetGlobalTag("session_id", sessionId)
}

func (f *Factory) OnProcessTermination() {
	_ = f.crashReporter.DeregisterProcess()
}
