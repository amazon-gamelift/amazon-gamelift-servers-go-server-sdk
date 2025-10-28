/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import (
	"os/exec"
	"runtime"
	"strings"
)

const (
	toolName    = "Metrics"
	toolVersion = "1.0.0"

	// Windows service constants
	windowsServiceCommand = "sc"
	windowsServiceName    = "GLOTelCollector"
	windowsServiceArgs    = "query"
	windowsRunningStatus  = "RUNNING"

	// Linux service constants
	linuxServiceCommand = "systemctl"
	linuxServiceName    = "gl-otel-collector.service"
	linuxServiceArgs    = "is-active"
	linuxActiveStatus   = "active"
)

// MetricsDetector detects if OTEL collector is running
type MetricsDetector struct {
	BaseGameLiftToolDetector
}

// NewMetricsDetector creates a new MetricsDetector instance
func NewMetricsDetector() *MetricsDetector {
	return &MetricsDetector{}
}

// IsToolRunning checks if the OTEL collector service is running
func (m *MetricsDetector) IsToolRunning() bool {
	defer func() {
		if r := recover(); r != nil {
			// Handle any panics and return false
		}
	}()

	if runtime.GOOS == "windows" {
		return m.checkService(windowsServiceCommand, []string{windowsServiceArgs, windowsServiceName}, func(output string) bool {
			return strings.Contains(output, windowsRunningStatus)
		})
	} else {
		return m.checkService(linuxServiceCommand, []string{linuxServiceArgs, linuxServiceName}, func(output string) bool {
			return strings.TrimSpace(output) == linuxActiveStatus
		})
	}
}

// checkService executes a system command to check service status
func (m *MetricsDetector) checkService(command string, args []string, outputValidator func(string) bool) bool {
	cmd := exec.Command(command, args...)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return cmd.ProcessState.Success() && outputValidator(string(output))
}

// GetToolName returns the tool name
func (m *MetricsDetector) GetToolName() string {
	return toolName
}

// GetToolVersion returns the tool version
func (m *MetricsDetector) GetToolVersion() string {
	return toolVersion
}

// SetGameLiftTool sets the GameLift tool environment variables if the tool is running
func (m *MetricsDetector) SetGameLiftTool() {
	m.BaseGameLiftToolDetector.SetGameLiftTool(m)
}