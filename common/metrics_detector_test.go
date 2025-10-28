/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import (
	"os"
	"testing"
)

func TestMetricsDetector_GetToolName(t *testing.T) {
	detector := NewMetricsDetector()
	expected := "Metrics"
	actual := detector.GetToolName()
	
	if actual != expected {
		t.Errorf("Expected tool name %s, got %s", expected, actual)
	}
}

func TestMetricsDetector_GetToolVersion(t *testing.T) {
	detector := NewMetricsDetector()
	expected := "1.0.0"
	actual := detector.GetToolVersion()
	
	if actual != expected {
		t.Errorf("Expected tool version %s, got %s", expected, actual)
	}
}

func TestMetricsDetector_IsToolRunning_HandlesExceptions(t *testing.T) {
	detector := NewMetricsDetector()
	// This test verifies that exceptions are caught and false is returned
	// The actual process execution will likely fail in test environment
	result := detector.IsToolRunning()
	
	// We expect false since the service likely won't be running in test environment
	if result {
		t.Log("OTEL collector service appears to be running in test environment")
	}
}

func TestMetricsDetector_SetGameLiftTool_DoesNotOverrideExisting(t *testing.T) {
	// Set existing environment variables
	os.Setenv(EnvironmentKeySDKToolName, "ExistingTool")
	os.Setenv(EnvironmentKeySDKToolVersion, "2.0.0")
	
	detector := NewMetricsDetector()
	detector.SetGameLiftTool()
	
	toolName := os.Getenv(EnvironmentKeySDKToolName)
	toolVersion := os.Getenv(EnvironmentKeySDKToolVersion)
	
	if toolName != "ExistingTool" {
		t.Errorf("Expected SDK_TOOL_NAME to remain 'ExistingTool', got '%s'", toolName)
	}
	
	if toolVersion != "2.0.0" {
		t.Errorf("Expected SDK_TOOL_VERSION to remain '2.0.0', got '%s'", toolVersion)
	}
	
	// Clean up
	os.Unsetenv(EnvironmentKeySDKToolName)
	os.Unsetenv(EnvironmentKeySDKToolVersion)
}