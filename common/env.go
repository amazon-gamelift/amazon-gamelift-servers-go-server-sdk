/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import (
	"os"
	"strconv"
	"time"

	"aws/amazon-gamelift-go-sdk/server/log"
)

// GetEnvStringOrDefault - returns environment variable by key or the default string value otherwise
func GetEnvStringOrDefault(key, defValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defValue
	}
	return value
}

// GetEnvIntOrDefault - returns environment variable by key or the default int value otherwise
//
// In case the function can't parse an int value from env variable it logs a warning.
func GetEnvIntOrDefault(key string, defValue int, l log.ILogger) int {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defValue
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		if l != nil {
			l.Warnf("Error %s when try parse int in %s", err.Error(), value)
		}
		return defValue
	}
	return int(n)
}

// GetEnvDurationOrDefault - returns environment variable by key or the default duration value otherwise
// decimal numbers, each with optional fraction and a unit suffix,
// such as "300ms", "-1.5h" or "2h45m".
//
// In case the function can't parse a duration value from env variable it logs a warning.
func GetEnvDurationOrDefault(key string, defValue time.Duration, l log.ILogger) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		if l != nil {
			l.Warnf("Error %s when try parse duration in %s", err.Error(), value)
		}
		return defValue
	}
	return duration
}
