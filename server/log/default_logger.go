/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package log

import (
	"fmt"
	"log"
)

type defaultLogger struct {
	*log.Logger
}

func (d *defaultLogger) Debugf(pattern string, arg ...any) {
	_ = d.Output(2, fmt.Sprintf("[DEBUG]:"+pattern, arg...))
}

func (d *defaultLogger) Errorf(pattern string, arg ...any) {
	_ = d.Output(2, fmt.Sprintf("[ERROR]:"+pattern, arg...))
}

func (d *defaultLogger) Warnf(pattern string, arg ...any) {
	_ = d.Output(2, fmt.Sprintf("[WARN]:"+pattern, arg...))
}

// GetDefaultLogger - returns a default logger implementation.
// That logger write all logs into stderr and based on standard golang logger implementation.
func GetDefaultLogger() ILogger {
	return &defaultLogger{log.Default()}
}
