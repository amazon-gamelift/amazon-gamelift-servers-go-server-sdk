/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package common

// Outcome - represents a general Amazon GameLift Servers service response for internal use in the SDK.
type Outcome struct {
	Data  []byte
	Error error
}
