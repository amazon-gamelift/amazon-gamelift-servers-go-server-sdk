/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package metrics

import (
	"testing"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/model"

	"github.com/golang/mock/gomock"
)

func TestNewStatsDTransport(t *testing.T) {
	builder := NewStatsDTransport()
	if builder == nil {
		t.Fatal("NewStatsDTransport returned nil")
	}
	common.AssertEqual(t, DefaultStatsDHost, builder.config.host)
	common.AssertEqual(t, DefaultStatsDPort, builder.config.port)
}

func TestStatsDTransportBuilder(t *testing.T) {
	builder := NewStatsDTransport().
		WithHost("test-host").
		WithPort("9999").
		WithMaxPacketSize(2000).
		WithoutClientTelemetry()

	common.AssertEqual(t, "test-host", builder.config.host)
	common.AssertEqual(t, "9999", builder.config.port)
	common.AssertEqual(t, 2000, builder.config.maxPacketSize)
	common.AssertEqual(t, true, builder.config.clientTelemetryDisabled)
}

func TestStatsDTransportBuilder_WithAddress(t *testing.T) {
	builder := NewStatsDTransport().WithAddress("example.com:8888")
	common.AssertEqual(t, "example.com", builder.config.host)
	common.AssertEqual(t, "8888", builder.config.port)
}

func TestStatsDTransportBuilder_Build(t *testing.T) {
	transport, err := NewStatsDTransport().Build()
	common.AssertEqual(t, nil, err)
	if transport == nil {
		t.Fatal("Build returned nil transport")
	}
	_ = transport.Close()
}

func TestStatsDTransport_Send(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockStatsDClient(ctrl)

	// Create transport with mock client
	config := &StatsDConfig{
		host:          "localhost",
		port:          "8125",
		prefix:        "server",
		maxPacketSize: 1400,
	}
	transport := &StatsDTransport{
		client: mockClient,
		config: config,
	}

	messages := []model.MetricMessage{
		{Key: "test.gauge", Type: model.MetricTypeGauge, Value: 42.5, Tags: map[string]string{"env": "test"}, SampleRate: 1.0},
		{Key: "test.counter", Type: model.MetricTypeCounter, Value: 10, SampleRate: 1.0},
		{Key: "test.timer", Type: model.MetricTypeTimer, Value: 250, SampleRate: 1.0},
	}

	// Set up expectations
	mockClient.EXPECT().Gauge("test.gauge", 42.5, []string{"env:test"}, 1.0).Return(nil)
	mockClient.EXPECT().Count("test.counter", int64(10), []string{}, 1.0).Return(nil)
	mockClient.EXPECT().TimeInMilliseconds("test.timer", 250.0, []string{}, 1.0).Return(nil)

	// Test sending messages
	err := transport.Send(messages)
	common.AssertEqual(t, nil, err)

	// Test sending empty slice
	err = transport.Send([]model.MetricMessage{})
	common.AssertEqual(t, nil, err)

	// Test sending nil slice
	err = transport.Send(nil)
	common.AssertEqual(t, nil, err)
}

func TestParseAddress(t *testing.T) {
	tests := []struct {
		input, wantHost, wantPort string
	}{
		{"localhost:8125", "localhost", "8125"},
		{"192.168.1.1:9999", "192.168.1.1", "9999"},
		{"invalid", "invalid", DefaultStatsDPort},
	}

	for _, tt := range tests {
		host, port := parseAddress(tt.input)
		common.AssertEqual(t, tt.wantHost, host)
		common.AssertEqual(t, tt.wantPort, port)
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	t.Setenv("TEST_VAR", "test_value")
	common.AssertEqual(t, "test_value", getEnvOrDefault("TEST_VAR", "default"))
	common.AssertEqual(t, "default", getEnvOrDefault("NONEXISTENT", "default"))

	t.Setenv("EMPTY_VAR", "")
	common.AssertEqual(t, "default", getEnvOrDefault("EMPTY_VAR", "default"))
}

func TestStatsDTransport_Close(t *testing.T) {
	transport, err := NewStatsDTransport().Build()
	common.AssertEqual(t, nil, err)

	common.AssertEqual(t, nil, transport.Close())
	common.AssertEqual(t, nil, transport.Close()) // Double close should be safe
}
