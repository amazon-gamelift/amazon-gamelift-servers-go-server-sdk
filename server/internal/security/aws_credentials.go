/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package security

// Holds the AWS credentials.
type AwsCredentials struct {
	AccessKey    string `json:"AccessKeyId"`
	SecretKey    string `json:"SecretAccessKey"`
	SessionToken string `json:"Token"`
}
