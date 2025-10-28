/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import "os"

// GameLiftToolDetector interface for detecting GameLift tools
type GameLiftToolDetector interface {
	IsToolRunning() bool
	GetToolName() string
	GetToolVersion() string
	SetGameLiftTool()
}

// BaseGameLiftToolDetector provides common functionality for tool detection
type BaseGameLiftToolDetector struct{}

// SetGameLiftTool sets environment variables if tool is running and not already set
func (b *BaseGameLiftToolDetector) SetGameLiftTool(detector GameLiftToolDetector) {
	if os.Getenv(EnvironmentKeySDKToolName) == "" && detector.IsToolRunning() {
		os.Setenv(EnvironmentKeySDKToolName, detector.GetToolName())
		os.Setenv(EnvironmentKeySDKToolVersion, detector.GetToolVersion())
	}
}
