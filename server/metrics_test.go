/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package server

import (
	"os"
	"testing"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
)

func TestApplyEnvironmentOverrides_NoEnvironmentVariables(t *testing.T) {
	// GIVEN: No environment variables are set
	envVars := []string{
		common.EnvironmentKeyStatsdHost,
		common.EnvironmentKeyStatsdPort,
		common.EnvironmentKeyCrashReporterHost,
		common.EnvironmentKeyCrashReporterPort,
		common.EnvironmentKeyFlushIntervalMs,
		common.EnvironmentKeyMaxPacketSize,
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}

	original := &MetricsParameters{
		StatsdHost:        "original-host",
		StatsdPort:        1234,
		CrashReporterHost: "original-crash-host",
		CrashReporterPort: 5678,
		FlushIntervalMs:   9999,
		MaxPacketSize:     512,
	}

	// WHEN: applyEnvironmentOverrides is called
	result := applyEnvironmentOverrides(original)

	// THEN: All parameters should remain unchanged
	if result.StatsdHost != "original-host" {
		t.Errorf("Expected StatsdHost 'original-host', got '%s'", result.StatsdHost)
	}
	if result.StatsdPort != 1234 {
		t.Errorf("Expected StatsdPort 1234, got %d", result.StatsdPort)
	}
	if result.CrashReporterHost != "original-crash-host" {
		t.Errorf("Expected CrashReporterHost 'original-crash-host', got '%s'", result.CrashReporterHost)
	}
	if result.CrashReporterPort != 5678 {
		t.Errorf("Expected CrashReporterPort 5678, got %d", result.CrashReporterPort)
	}
	if result.FlushIntervalMs != 9999 {
		t.Errorf("Expected FlushIntervalMs 9999, got %d", result.FlushIntervalMs)
	}
	if result.MaxPacketSize != 512 {
		t.Errorf("Expected MaxPacketSize 512, got %d", result.MaxPacketSize)
	}
}

func TestApplyEnvironmentOverrides_WithEnvironmentVariables(t *testing.T) {
	// GIVEN: All environment variables are set
	os.Setenv(common.EnvironmentKeyStatsdHost, "env-statsd-host")
	os.Setenv(common.EnvironmentKeyStatsdPort, "8125")
	os.Setenv(common.EnvironmentKeyCrashReporterHost, "env-crash-host")
	os.Setenv(common.EnvironmentKeyCrashReporterPort, "8126")
	os.Setenv(common.EnvironmentKeyFlushIntervalMs, "5000")
	os.Setenv(common.EnvironmentKeyMaxPacketSize, "1024")

	defer func() {
		os.Unsetenv(common.EnvironmentKeyStatsdHost)
		os.Unsetenv(common.EnvironmentKeyStatsdPort)
		os.Unsetenv(common.EnvironmentKeyCrashReporterHost)
		os.Unsetenv(common.EnvironmentKeyCrashReporterPort)
		os.Unsetenv(common.EnvironmentKeyFlushIntervalMs)
		os.Unsetenv(common.EnvironmentKeyMaxPacketSize)
	}()

	original := &MetricsParameters{
		StatsdHost:        "original-host",
		StatsdPort:        1234,
		CrashReporterHost: "original-crash-host",
		CrashReporterPort: 5678,
		FlushIntervalMs:   9999,
		MaxPacketSize:     512,
	}

	// WHEN: applyEnvironmentOverrides is called
	result := applyEnvironmentOverrides(original)

	// THEN: All parameters should be overridden by environment variables
	if result.StatsdHost != "env-statsd-host" {
		t.Errorf("Expected StatsdHost 'env-statsd-host', got '%s'", result.StatsdHost)
	}
	if result.StatsdPort != 8125 {
		t.Errorf("Expected StatsdPort 8125, got %d", result.StatsdPort)
	}
	if result.CrashReporterHost != "env-crash-host" {
		t.Errorf("Expected CrashReporterHost 'env-crash-host', got '%s'", result.CrashReporterHost)
	}
	if result.CrashReporterPort != 8126 {
		t.Errorf("Expected CrashReporterPort 8126, got %d", result.CrashReporterPort)
	}
	if result.FlushIntervalMs != 5000 {
		t.Errorf("Expected FlushIntervalMs 5000, got %d", result.FlushIntervalMs)
	}
	if result.MaxPacketSize != 1024 {
		t.Errorf("Expected MaxPacketSize 1024, got %d", result.MaxPacketSize)
	}
}

func TestApplyEnvironmentOverrides_PartialOverrides(t *testing.T) {
	// GIVEN: Only some environment variables are set
	os.Setenv(common.EnvironmentKeyStatsdHost, "partial-host")
	os.Setenv(common.EnvironmentKeyFlushIntervalMs, "3000")

	defer func() {
		os.Unsetenv(common.EnvironmentKeyStatsdHost)
		os.Unsetenv(common.EnvironmentKeyFlushIntervalMs)
	}()

	original := &MetricsParameters{
		StatsdHost:        "original-host",
		StatsdPort:        1234,
		CrashReporterHost: "original-crash-host",
		CrashReporterPort: 5678,
		FlushIntervalMs:   9999,
		MaxPacketSize:     512,
	}

	// WHEN: applyEnvironmentOverrides is called
	result := applyEnvironmentOverrides(original)

	// THEN: Only specified env vars should be overridden, others remain original
	if result.StatsdHost != "partial-host" {
		t.Errorf("Expected StatsdHost 'partial-host', got '%s'", result.StatsdHost)
	}
	if result.FlushIntervalMs != 3000 {
		t.Errorf("Expected FlushIntervalMs 3000, got %d", result.FlushIntervalMs)
	}

	if result.StatsdPort != 1234 {
		t.Errorf("Expected StatsdPort 1234, got %d", result.StatsdPort)
	}
	if result.CrashReporterHost != "original-crash-host" {
		t.Errorf("Expected CrashReporterHost 'original-crash-host', got '%s'", result.CrashReporterHost)
	}
	if result.CrashReporterPort != 5678 {
		t.Errorf("Expected CrashReporterPort 5678, got %d", result.CrashReporterPort)
	}
	if result.MaxPacketSize != 512 {
		t.Errorf("Expected MaxPacketSize 512, got %d", result.MaxPacketSize)
	}
}

func TestApplyEnvironmentOverrides_EmptyStringValues(t *testing.T) {
	// GIVEN: Environment variables are set to empty strings
	os.Setenv(common.EnvironmentKeyStatsdHost, "")
	os.Setenv(common.EnvironmentKeyCrashReporterHost, "")

	defer func() {
		os.Unsetenv(common.EnvironmentKeyStatsdHost)
		os.Unsetenv(common.EnvironmentKeyCrashReporterHost)
	}()

	original := &MetricsParameters{
		StatsdHost:        "original-host",
		CrashReporterHost: "original-crash-host",
	}

	// WHEN: applyEnvironmentOverrides is called
	result := applyEnvironmentOverrides(original)

	// THEN: Empty strings should not override original values
	if result.StatsdHost != "original-host" {
		t.Errorf("Expected StatsdHost 'original-host', got '%s'", result.StatsdHost)
	}
	if result.CrashReporterHost != "original-crash-host" {
		t.Errorf("Expected CrashReporterHost 'original-crash-host', got '%s'", result.CrashReporterHost)
	}
}

func TestApplyEnvironmentOverrides_ZeroIntValues(t *testing.T) {
	// GIVEN: Environment variables are set to zero values
	os.Setenv(common.EnvironmentKeyStatsdPort, "0")
	os.Setenv(common.EnvironmentKeyFlushIntervalMs, "0")

	defer func() {
		os.Unsetenv(common.EnvironmentKeyStatsdPort)
		os.Unsetenv(common.EnvironmentKeyFlushIntervalMs)
	}()

	original := &MetricsParameters{
		StatsdPort:      1234,
		FlushIntervalMs: 9999,
	}

	// WHEN: applyEnvironmentOverrides is called
	result := applyEnvironmentOverrides(original)

	// THEN: Zero values should override original values (env vars take precedence)
	if result.StatsdPort != 0 {
		t.Errorf("Expected StatsdPort 0, got %d", result.StatsdPort)
	}
	if result.FlushIntervalMs != 0 {
		t.Errorf("Expected FlushIntervalMs 0, got %d", result.FlushIntervalMs)
	}
}
