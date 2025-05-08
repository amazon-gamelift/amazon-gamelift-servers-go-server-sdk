/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

//go:generate mockgen -destination ../internal/mock/logger.go -package=mock . ILogger
package log

// ILogger - interface that describes the logger used by the GameLift SDK.
//
// To inject a custom implementation of this interface to the SDK please use server.SetLoggerInterface function.
type ILogger interface {
	Debugf(string, ...any)
	Warnf(string, ...any)
	Errorf(string, ...any)
}
