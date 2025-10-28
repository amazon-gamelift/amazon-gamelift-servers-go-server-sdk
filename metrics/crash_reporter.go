/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
package metrics

import (
	"fmt"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"strconv"
)

const (
	// CrashReporterHostEnvironmentVariable is the environment variable name for configuring the Crash Reporter host address
	CrashReporterHostEnvironmentVariable = "GAMELIFT_CRASH_REPORTER_HOST"

	// CrashReporterPortEnvironmentVariable is the environment variable name for configuring the Crash Reporter port
	CrashReporterPortEnvironmentVariable = "GAMELIFT_CRASH_REPORTER_PORT"

	// DefaultCrashReporterHost is the default Crash Reporter server host when no environment variable is set
	DefaultCrashReporterHost = "localhost"

	// DefaultCrashReporterPort is the default Crash Reporter server port when no environment variable is set
	DefaultCrashReporterPort = "8126"
)

type CrashReporter interface {
	RegisterProcess() error
	TagGameSession(sessionID string) error
	DeregisterProcess() error
}

// CrashReporterConfig contains configuration for the Crash Reporter transport
type CrashReporterConfig struct {
	// host is the Crash Reporter server host
	host string
	// port is the Crash Reporter server port
	port string
}

// NewCrashReporter returns a new CrashReporter builder
func NewCrashReporter() *CrashReporterBuilder {
	return &CrashReporterBuilder{
		config: &CrashReporterConfig{
			host: getEnvOrDefault(CrashReporterHostEnvironmentVariable, DefaultCrashReporterHost),
			port: getEnvOrDefault(CrashReporterPortEnvironmentVariable, DefaultCrashReporterPort),
		},
	}
}

// CrashReporterBuilder builds a CrashReporter with fluent method chaining
type CrashReporterBuilder struct {
	config *CrashReporterConfig
}

// WithHost sets the Crash Reporter host (overrides environment variable)
func (b *CrashReporterBuilder) WithHost(host string) *CrashReporterBuilder {
	b.config.host = host
	return b
}

// WithPort sets the Crash Reporter port (overrides environment variable)
func (b *CrashReporterBuilder) WithPort(port string) *CrashReporterBuilder {
	b.config.port = port
	return b
}

// Build creates and returns the configured CrashReporter
func (b *CrashReporterBuilder) Build() (*HttpCrashReporter, error) {

	crashReporterPortNumber, err := strconv.Atoi(b.config.port)
	if err != nil {
		return nil, common.NewGameLiftError(common.MetricConfigurationException, "", fmt.Sprintf("failed to create Crash Reporter client due to invaid port number %d, error: %v", crashReporterPortNumber, err))
	}

	crashReporterClient, err := NewCrashReporterClient(b.config.host, crashReporterPortNumber)
	if err != nil {
		return nil, common.NewGameLiftError(common.MetricConfigurationException, "", fmt.Sprintf("failed to create Crash Reporter client, error: %v", err))
	}

	return &HttpCrashReporter{
		client: crashReporterClient,
		config: b.config,
	}, nil
}

type HttpCrashReporter struct {
	client *CrashReporterClient
	config *CrashReporterConfig
}

var _ CrashReporter = (*HttpCrashReporter)(nil)

func (t *HttpCrashReporter) RegisterProcess() error {
	return t.client.RegisterProcess()
}

func (t *HttpCrashReporter) TagGameSession(sessionID string) error {
	return t.client.TagGameSession(sessionID)
}

func (t *HttpCrashReporter) DeregisterProcess() error {
	return t.client.DeregisterProcess()
}
