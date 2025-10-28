/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
package metrics

import (
	"fmt"
	"net"
	"os"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/model"

	"github.com/DataDog/datadog-go/v5/statsd"
)

const (
	// StatsDHostEnvironmentVariable is the environment variable name for configuring the StatsD host address.
	StatsDHostEnvironmentVariable = "GAMELIFT_STATSD_HOST"

	// StatsDPortEnvironmentVariable is the environment variable name for configuring the StatsD port.
	StatsDPortEnvironmentVariable = "GAMELIFT_STATSD_PORT"

	// DefaultStatsDHost is the default StatsD server host when no environment variable is set.
	DefaultStatsDHost = "localhost"

	// DefaultStatsDPort is the default StatsD server port when no environment variable is set.
	DefaultStatsDPort = "8125"

	// DefaultStatsDPrefix is the default metric prefix assigned to all server level metrics.
	DefaultStatsDPrefix = "server"

	// DefaultAddress defines the default address for metrics transport.
	DefaultAddress = "localhost:8125"

	// DefaultMaxPacketSize defines the default maximum packet size for UDP.
	DefaultMaxPacketSize = 1400
)

// Transport interface for sending metrics without contexts.
//
//go:generate mockgen -destination ./transport_mock_test.go -package=metrics . Transport
type Transport interface {
	// Send sends a batch of metrics
	Send(messages []model.MetricMessage) error
	// Close closes the transport and releases resources
	Close() error
}

// StatsDConfig contains configuration for the StatsD transport.
type StatsDConfig struct {
	// host is the StatsD server host
	host string
	// port is the StatsD server port
	port string
	// prefix is the StatsD metric prefix
	prefix string
	// maxPacketSize is the maximum size of packets to send
	maxPacketSize int
	// clientTelemetryDisabled indicates whether telemetry should be enabled
	clientTelemetryDisabled bool
}

// NewStatsDTransport returns a new StatsD transport builder.
func NewStatsDTransport() *StatsDTransportBuilder {
	return &StatsDTransportBuilder{
		config: &StatsDConfig{
			host:          getEnvOrDefault(StatsDHostEnvironmentVariable, DefaultStatsDHost),
			port:          getEnvOrDefault(StatsDPortEnvironmentVariable, DefaultStatsDPort),
			prefix:        DefaultStatsDPrefix,
			maxPacketSize: DefaultMaxPacketSize,
		},
	}
}

// StatsDTransportBuilder builds a StatsDTransport with fluent method chaining.
type StatsDTransportBuilder struct {
	config *StatsDConfig
}

// WithHost sets the StatsD host (overrides environment variable).
func (b *StatsDTransportBuilder) WithHost(host string) *StatsDTransportBuilder {
	b.config.host = host
	return b
}

// WithPort sets the StatsD port (overrides environment variable).
func (b *StatsDTransportBuilder) WithPort(port string) *StatsDTransportBuilder {
	b.config.port = port
	return b
}

// WithAddress sets both host and port from address string (overrides environment variables).
func (b *StatsDTransportBuilder) WithAddress(addr string) *StatsDTransportBuilder {
	// Parse host:port format
	host, port := parseAddress(addr)
	b.config.host = host
	b.config.port = port

	return b
}

// WithMaxPacketSize sets the maximum packet size.
func (b *StatsDTransportBuilder) WithMaxPacketSize(size int) *StatsDTransportBuilder {
	b.config.maxPacketSize = size
	return b
}

// WithoutClientTelemetry disables client telemetry.
func (b *StatsDTransportBuilder) WithoutClientTelemetry() *StatsDTransportBuilder {
	b.config.clientTelemetryDisabled = true
	return b
}

// Build creates and returns the configured StatsDTransport.
func (b *StatsDTransportBuilder) Build() (*StatsDTransport, error) {
	// Build StatsD client options
	statsdOptions := []statsd.Option{
		statsd.WithMaxBytesPerPayload(b.config.maxPacketSize),
		statsd.WithoutClientSideAggregation(),
	}

	address := fmt.Sprintf("%s:%s", b.config.host, b.config.port)

	if b.config.prefix != "" {
		statsdOptions = append(statsdOptions, statsd.WithNamespace(b.config.prefix))
	}

	if b.config.clientTelemetryDisabled {
		statsdOptions = append(statsdOptions, statsd.WithoutTelemetry())
	}

	statsdClient, err := statsd.New(address, statsdOptions...)
	if err != nil {
		return nil, common.NewGameLiftError(common.MetricConfigurationException, "", fmt.Sprintf("failed to create StatsD client: %v", err))
	}

	return &StatsDTransport{
		client: statsdClient,
		config: b.config,
	}, nil
}

// StatsDClient is an interface representing the methods used from the DataDog StatsD client.
//
//go:generate mockgen -destination ./statsd_client_mock_test.go -package=metrics . StatsDClient
type StatsDClient interface {
	Gauge(name string, value float64, tags []string, rate float64) error
	Count(name string, value int64, tags []string, rate float64) error
	TimeInMilliseconds(name string, value float64, tags []string, rate float64) error
	Close() error
}

// StatsDTransport implements Transport using DataDog StatsD client.
type StatsDTransport struct {
	client StatsDClient
	config *StatsDConfig
}

var _ Transport = (*StatsDTransport)(nil)

// Send sends a batch of metrics to the StatsD server.
func (t *StatsDTransport) Send(messages []model.MetricMessage) error {
	for _, msg := range messages {
		tags := make([]string, 0, len(msg.Tags))
		for k, v := range msg.Tags {
			tags = append(tags, fmt.Sprintf("%s:%s", k, v))
		}

		switch msg.Type {
		case model.MetricTypeGauge:
			if err := t.client.Gauge(msg.Key, msg.Value, tags, msg.SampleRate); err != nil {
				return common.NewGameLiftError(common.MetricTransportException, "", fmt.Sprintf("failed to send gauge metric: %v", err))
			}
		case model.MetricTypeCounter:
			if err := t.client.Count(msg.Key, int64(msg.Value), tags, msg.SampleRate); err != nil {
				return common.NewGameLiftError(common.MetricTransportException, "", fmt.Sprintf("failed to send counter metric: %v", err))
			}
		case model.MetricTypeTimer:
			// Timer messages are always put onto the queue as milliseconds.
			if err := t.client.TimeInMilliseconds(msg.Key, msg.Value, tags, msg.SampleRate); err != nil {
				return common.NewGameLiftError(common.MetricTransportException, "", fmt.Sprintf("failed to send timer metric: %v", err))
			}
		default:
			return common.NewGameLiftError(common.MetricUnsupportedTypeException, "", "unsupported metric type: "+msg.Type.String())
		}
	}
	return nil
}

// Close closes the StatsD client and releases resources.
func (t *StatsDTransport) Close() error {
	return t.client.Close() //nolint:wrapcheck // Caller provides context
}

func getEnvOrDefault(envVar, defaultValue string) string {
	if value := os.Getenv(envVar); value != "" {
		return value
	}
	return defaultValue
}

func parseAddress(addr string) (string, string) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, DefaultStatsDPort // Use defaults if parsing fails
	}
	return host, port
}
