/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package mock

import (
	"testing"

	"github.com/golang/mock/gomock"
)

type TestLoggerOption func(*testLoggerOptions)

type testLoggerOptions struct {
	expectAnyDebug bool
	expectAnyWarn  bool
	expectAnyError bool
}

// WithExpectAnyDebug expects any debug logs when true
func WithExpectAnyDebug(enable bool) TestLoggerOption {
	return func(o *testLoggerOptions) {
		o.expectAnyDebug = enable
	}
}

// WithExpectAnyWarn expects any warn logs when true
func WithExpectAnyWarn(enable bool) TestLoggerOption {
	return func(o *testLoggerOptions) {
		o.expectAnyWarn = enable
	}
}

// WithExpectAnyError expects any error logs when true
func WithExpectAnyError(enable bool) TestLoggerOption {
	return func(o *testLoggerOptions) {
		o.expectAnyError = enable
	}
}

// NewTestLogger creates a new *MockILogger with any number of test options. By default, the logger
// is configured to allow Debugf and Warnf calls with any arguments any number of times.
// For further configuration, you can call the EXPECT method as usual
func NewTestLogger(t *testing.T, ctrl *gomock.Controller, opts ...TestLoggerOption) *MockILogger {
	t.Helper()

	options := &testLoggerOptions{
		expectAnyDebug: true,  // default to expect any debug
		expectAnyWarn:  true,  // default to expect any warn
		expectAnyError: false, // default to not expecting errors
	}

	for _, opt := range opts {
		opt(options)
	}

	logger := NewMockILogger(ctrl)

	if options.expectAnyDebug {
		logger.
			EXPECT().
			Debugf(gomock.Any(), gomock.Any()).
			Do(func(format string, args ...any) { t.Logf(format, args...) }).
			AnyTimes()
	}

	if options.expectAnyWarn {
		logger.
			EXPECT().
			Warnf(gomock.Any(), gomock.Any()).
			Do(func(format string, args ...any) { t.Logf(format, args...) }).
			AnyTimes()
	}

	if options.expectAnyError {
		logger.
			EXPECT().
			Errorf(gomock.Any(), gomock.Any()).
			Do(func(format string, args ...any) { t.Logf(format, args...) }).
			AnyTimes()
	}

	return logger
}
