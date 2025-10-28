/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package server

import (
	"context"
	"strconv"
	"time"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics"
)

// Metrics provides a convenient way to create metrics with common configurations.
type Metrics struct {
	factory metrics.IFactory
}

// Gauge creates a new gauge metric with factory defaults.
func (m *Metrics) Gauge(key string) (*metrics.Gauge, error) {
	return m.factory.Gauge(key)
}

// Counter creates a new counter metric with factory defaults.
func (m *Metrics) Counter(key string) (*metrics.Counter, error) {
	return m.factory.Counter(key)
}

// Timer creates a new timer metric with factory default and a derived percentile metric.
func (m *Metrics) Timer(key string) (*metrics.Timer, error) {
	return m.factory.Timer(key)
}

// SetGlobalTag sets a global tag that will be applied to all metrics.
func (m *Metrics) SetGlobalTag(key, value string) error {
	return metrics.SetGlobalTag(key, value)
}

// RemoveGlobalTag removes a global tag.
func (m *Metrics) RemoveGlobalTag(key string) {
	metrics.RemoveGlobalTag(key)
}

// newMetrics creates a new Metrics instance (internal use only)
func newMetrics(factory metrics.IFactory) *Metrics {
	return &Metrics{factory: factory}
}

func applyEnvironmentOverrides(params *MetricsParameters) *MetricsParameters {
	params.StatsdHost = common.GetEnvStringOrDefault(common.EnvironmentKeyStatsdHost, params.StatsdHost)
	params.StatsdPort = common.GetEnvIntOrDefault(common.EnvironmentKeyStatsdPort, params.StatsdPort, lg)
	params.CrashReporterHost = common.GetEnvStringOrDefault(common.EnvironmentKeyCrashReporterHost, params.CrashReporterHost)
	params.CrashReporterPort = common.GetEnvIntOrDefault(common.EnvironmentKeyCrashReporterPort, params.CrashReporterPort, lg)
	params.FlushIntervalMs = common.GetEnvIntOrDefault(common.EnvironmentKeyFlushIntervalMs, params.FlushIntervalMs, lg)
	params.MaxPacketSize = common.GetEnvIntOrDefault(common.EnvironmentKeyMaxPacketSize, params.MaxPacketSize, lg)
	return params
}

func initMetricsFactory(metricsParameters *MetricsParameters, serverState iGameLiftServerState) (metrics.IFactory, error) {
	if metricsParameters == nil {
		return nil, common.NewGameLiftError(common.ValidationException, "MetricsParameters cannot be nil", "")
	}

	err := ValidateMetricsParameters(metricsParameters)
	if err != nil {
		return nil, err
	}

	crashReporter, err := metrics.NewCrashReporter().
		WithHost(metricsParameters.CrashReporterHost).
		WithPort(strconv.Itoa(metricsParameters.CrashReporterPort)).
		Build()
	if err != nil {
		return nil, common.NewGameLiftError(common.MetricConfigurationException, "Failed to create crash reporter", err.Error())
	}

	transport, err := metrics.NewStatsDTransport().
		WithHost(metricsParameters.StatsdHost).
		WithPort(strconv.Itoa(metricsParameters.StatsdPort)).
		WithMaxPacketSize(metricsParameters.MaxPacketSize).
		Build()
	if err != nil {
		return nil, common.NewGameLiftError(common.MetricConfigurationException, "Failed to create transport", err.Error())
	}

	err = metrics.InitMetricsProcessor(
		metrics.WithTransport(transport),
		metrics.WithProcessInterval(time.Duration(metricsParameters.FlushIntervalMs)*time.Millisecond),
	)
	if err != nil {
		return nil, common.NewGameLiftError(common.MetricConfigurationException, "Failed to initialize metrics processor", err.Error())
	}

	localMetricsFactory, err := metrics.NewFactory().
		WithCrashReporter(crashReporter).
		Build()
	if err != nil {
		return nil, common.NewGameLiftError(common.MetricConfigurationException, "Failed to create metrics factory", err.Error())
	}

	// Start metrics processor
	ctx := context.Background()
	if err = metrics.StartMetricsProcessor(ctx); err != nil {
		return nil, err
	}

	localMetricsFactory.OnProcessStart()

	// Set the MetricsFactory of the server state (if available)
	if serverState != nil {
		serverState.setMetricsFactory(localMetricsFactory)
		var gameSessionID string
		if gameSessionID, err = serverState.getGameSessionID(); err == nil && gameSessionID != "" {
			localMetricsFactory.OnStartGameSession(gameSessionID)
		}
	}

	return localMetricsFactory, nil
}

func createMetrics(params *MetricsParameters, serverState iGameLiftServerState) (*Metrics, metrics.IFactory, error) {
	localMetricsFactory, err := initMetricsFactory(params, serverState)
	if err != nil {
		return nil, nil, err
	}
	return newMetrics(localMetricsFactory), localMetricsFactory, nil
}

func terminateMetricsFactory(factory metrics.IFactory) {
	if factory != nil {
		if stopErr := metrics.TerminateMetricsProcessor(); stopErr != nil {
			lg.Warnf("Failed to stop metrics factory: %v", stopErr)
		}
	}
}
