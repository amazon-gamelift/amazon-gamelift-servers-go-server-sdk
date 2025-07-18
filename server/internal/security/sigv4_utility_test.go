/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package security_test

import (
	"testing"
	"time"

	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/server/internal/security"
)

// GIVEN valid SigV4Parameters WHEN GenerateSigV4QueryParameters THEN return expected query parameters
func TestGenerateSigV4QueryParameters_ValidParameters(t *testing.T) {

	// GIVEN
	validParams := generateSigV4Parameters()

	// WHEN
	queryParams, err := security.GenerateSigV4QueryParameters(validParams)

	// THEN
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedQueryParams := map[string]string{
		"Authorization":        "SigV4",
		"X-Amz-Algorithm":      "AWS4-HMAC-SHA256",
		"X-Amz-Credential":     "testAccessKey/20240805/us-east-1/gamelift/aws4_request",
		"X-Amz-Date":           "20240805T100000Z",
		"X-Amz-Signature":      "2601fe291f4b43a63f6ffb0e1d9085a1edbaa2a866c96511e153af3408bfe771",
		"X-Amz-Security-Token": "testSessionToken",
	}

	for k, v := range expectedQueryParams {
		if queryParams[k] != v {
			t.Errorf("unexpected value for %s: got %s, want %s", k, queryParams[k], v)
		}
	}
}

// GIVEN parameters with missing AwsRegion WHEN GenerateSigV4QueryParameters THEN an error should be returned
func TestGenerateSigV4QueryParameters_MissingAwsRegion(t *testing.T) {

	// GIVEN
	parameters := generateSigV4Parameters()
	parameters.AwsRegion = ""

	// WHEN
	_, err := security.GenerateSigV4QueryParameters(parameters)

	// THEN
	if err == nil || err.Error() != "AwsRegion is required" {
		t.Errorf("expected error: AwsRegion is required, got: %v", err)
	}
}

// GIVEN parameters with missing AccessKey WHEN GenerateSigV4QueryParameters THEN an error should be returned
func TestGenerateSigV4QueryParameters_MissingAccessKey(t *testing.T) {

	// GIVEN
	parameters := generateSigV4Parameters()
	parameters.AwsCredentials.AccessKey = ""

	// WHEN
	_, err := security.GenerateSigV4QueryParameters(parameters)

	// THEN
	if err == nil || err.Error() != "AccessKey is required" {
		t.Errorf("expected error: AccessKey is required, got: %v", err)
	}
}

// GIVEN parameters with missing SecretKey WHEN GenerateSigV4QueryParameters THEN an error should be returned
func TestGenerateSigV4QueryParameters_MissingSecretKey(t *testing.T) {

	// GIVEN
	parameters := generateSigV4Parameters()
	parameters.AwsCredentials.SecretKey = ""

	// WHEN
	_, err := security.GenerateSigV4QueryParameters(parameters)

	// THEN
	if err == nil || err.Error() != "SecretKey is required" {
		t.Errorf("expected error: SecretKey is required, got: %v", err)
	}
}

// GIVEN parameters with missing SessionToken WHEN GenerateSigV4QueryParameters THEN return expected query parameters
func TestGenerateSigV4QueryParameters_MissingSessionToken(t *testing.T) {

	// GIVEN
	parameters := generateSigV4Parameters()
	parameters.AwsCredentials.SessionToken = ""

	// WHEN
	queryParams, err := security.GenerateSigV4QueryParameters(parameters)

	// THEN
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedQueryParams := map[string]string{
		"Authorization":    "SigV4",
		"X-Amz-Algorithm":  "AWS4-HMAC-SHA256",
		"X-Amz-Credential": "testAccessKey/20240805/us-east-1/gamelift/aws4_request",
		"X-Amz-Date":       "20240805T100000Z",
		"X-Amz-Signature":  "2601fe291f4b43a63f6ffb0e1d9085a1edbaa2a866c96511e153af3408bfe771",
	}

	for k, v := range expectedQueryParams {
		if queryParams[k] != v {
			t.Errorf("unexpected value for %s: got %s, want %s", k, queryParams[k], v)
		}
	}
}

// GIVEN parameters with missing QueryParameters WHEN GenerateSigV4QueryParameters THEN an error should be returned
func TestGenerateSigV4QueryParameters_MissingQueryParams(t *testing.T) {

	// GIVEN
	parameters := generateSigV4Parameters()
	parameters.QueryParams = nil

	// WHEN
	_, err := security.GenerateSigV4QueryParameters(parameters)

	// THEN
	if err == nil || err.Error() != "QueryParams is required" {
		t.Errorf("expected error: QueryParams is required, got: %v", err)
	}
}

// GIVEN parameters with missing RequestTime WHEN GenerateSigV4QueryParameters THEN an error should be returned
func TestGenerateSigV4QueryParameters_MissingRequestTime(t *testing.T) {

	// GIVEN
	parameters := generateSigV4Parameters()
	parameters.RequestTime = time.Time{}

	// WHEN
	_, err := security.GenerateSigV4QueryParameters(parameters)

	// THEN
	if err == nil || err.Error() != "RequestTime is required" {
		t.Errorf("expected error: RequestTime is required, got: %v", err)
	}
}

func generateSigV4Parameters() security.SigV4Parameters {
	return security.SigV4Parameters{
		AwsRegion: "us-east-1",
		AwsCredentials: security.AwsCredentials{
			AccessKey:    "testAccessKey",
			SecretKey:    "testSecretKey",
			SessionToken: "testSessionToken",
		},
		QueryParams: map[string]string{
			"param1": "value1",
			"param2": "value2",
		},
		RequestTime: time.Date(2024, 8, 5, 10, 0, 0, 0, time.UTC),
	}
}
