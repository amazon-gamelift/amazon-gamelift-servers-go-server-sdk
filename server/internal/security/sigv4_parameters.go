/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package security

import "time"

// SigV4Parameters holds the parameters for the SigV4 signature generation.
type SigV4Parameters struct {
	AwsRegion      string
	AwsCredentials AwsCredentials
	QueryParams    map[string]string
	RequestTime    time.Time
}
