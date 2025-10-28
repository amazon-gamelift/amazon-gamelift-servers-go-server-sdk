/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"
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
// That logger write all logs into both file and stdout.
func GetDefaultLogger(processId string) ILogger {
	err := os.Mkdir("logs",  0o755)
	if err != nil && !os.IsExist(err) {
		log.Fatalf("error creating log directory: %v", err)
	}
	// sanitize processId
	sanitizedProcessId := sanitizeProcessId(processId)
	f, err := os.OpenFile(filepath.Join("logs", "gamelift-server-sdk-"+sanitizedProcessId+".log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}

	// Create a MultiWriter to write to both file and stdout
	multiWriter := io.MultiWriter(f, os.Stdout)
	return &defaultLogger{log.New(multiWriter, "", log.LstdFlags)}
}

// Helper function to sanitize the processId
func sanitizeProcessId(name string) string {
	// Remove any path separators and potentially dangerous characters
	name = strings.Map(func(r rune) rune {
		// Only allow alphanumeric characters, dash and underscore
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-' || r == '_' {
			return r
		}
		return -1
	}, name)
	return name
}

