/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package mock

import (
	"testing"

	"github.com/golang/mock/gomock"
)

// NewTestLogger creates a new *MockILogger and configures it so that Debugf and Warnf
// can be called with any arguments any number of times.
// For further configuration, you can call the EXPECT method as usual
func NewTestLogger(t *testing.T, ctrl *gomock.Controller) *MockILogger {
	t.Helper()

	logger := NewMockILogger(ctrl)
	logger.
		EXPECT().
		Debugf(gomock.Any(), gomock.Any()).
		Do(func(format string, args ...any) { t.Logf(format, args...) }).
		AnyTimes()

	logger.
		EXPECT().
		Warnf(gomock.Any(), gomock.Any()).
		Do(func(format string, args ...any) { t.Logf(format, args...) }).
		AnyTimes()

	return logger
}
