/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package security_test

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/server/internal/mock"
	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/server/internal/security"
	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/server/internal/transport"

	"github.com/golang/mock/gomock"
)

// GIVEN nil HttpClient WHEN NewContainerCredentialsFetcher THEN an error should be returned
func TestContainerCredentialsFetcher_NewContainerCredentialsFetcher_NilHttpClient(t *testing.T) {

	// GIVEN
	var httpClient transport.HttpClient

	// WHEN
	_, err := security.NewContainerCredentialsFetcher(httpClient)

	// THEN
	if err == nil || !strings.Contains(err.Error(), "httpClient cannot be nil") {
		t.Fatalf("expected httpClient cannot be nil, got %v", err)
	}
}

// GIVEN valid ContainerCredentialsProvider response WHEN FetchContainerCredentials THEN return AwsCredentials
func TestContainerCredentialsFetcher_FetchContainerCredentials_ValidCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// GIVEN
	_ = os.Setenv(security.EnvironmentVariableContainerCredentials, "/v2/credentials")
	mockHttpClient := mock.NewMockHttpClient(ctrl)
	validCredentials := `{
		"AccessKeyId": "AKIAIOSFODNN7EXAMPLE",
		"SecretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"Token": "AQoDYXdzEJr...<remainder of security token>",
		"Expiration": "2024-08-08T18:44:24Z"
	}`
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(validCredentials)),
	}
	mockHttpClient.EXPECT().Get(gomock.Any()).Return(response, nil)

	fetcher, err := security.NewContainerCredentialsFetcher(mockHttpClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// WHEN
	awsCredentials, err := fetcher.FetchContainerCredentials()

	// THEN
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if awsCredentials.AccessKey != "AKIAIOSFODNN7EXAMPLE" || awsCredentials.SecretKey != "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" || awsCredentials.SessionToken != "AQoDYXdzEJr...<remainder of security token>" {
		t.Fatalf("unexpected credentials: %+v", awsCredentials)
	}
}

// GIVEN missing environment variable WHEN FetchContainerCredentials THEN an error should be returned
func TestContainerCredentialsFetcher_FetchContainerCredentials_MissingEnvironmentVariable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// GIVEN
	_ = os.Unsetenv(security.EnvironmentVariableContainerCredentials)
	mockHttpClient := mock.NewMockHttpClient(ctrl)
	fetcher, err := security.NewContainerCredentialsFetcher(mockHttpClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// WHEN
	_, err = fetcher.FetchContainerCredentials()

	// THEN
	if err == nil || !strings.Contains(err.Error(), "environment variable AWS_CONTAINER_CREDENTIALS_RELATIVE_URI is not set") {
		t.Fatalf("expected error for unset environment variable, got %v", err)
	}
}

// GIVEN failure to send HttpRequest WHEN FetchContainerCredentials THEN an error should be returned
func TestContainerCredentialsFetcher_FetchContainerCredentials_FailureToSendRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// GIVEN
	_ = os.Setenv(security.EnvironmentVariableContainerCredentials, "/v2/credentials")
	mockHttpClient := mock.NewMockHttpClient(ctrl)
	mockHttpClient.EXPECT().Get(gomock.Any()).Return(nil, errors.New("failed request"))

	fetcher, err := security.NewContainerCredentialsFetcher(mockHttpClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// WHEN
	_, err = fetcher.FetchContainerCredentials()

	// THEN
	if err == nil || !strings.Contains(err.Error(), "failed to fetch container credentials") {
		t.Fatalf("expected error for failed HTTP request, got %v", err)
	}
}

// GIVEN failed HttpResponse WHEN FetchContainerCredentials THEN an error should be returned
func TestContainerCredentialsFetcher_FetchContainerCredentials_FailedHttpResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// GIVEN
	_ = os.Setenv(security.EnvironmentVariableContainerCredentials, "/v2/credentials")
	mockHttpClient := mock.NewMockHttpClient(ctrl)
	response := &http.Response{
		StatusCode: http.StatusServiceUnavailable,
		Body:       http.NoBody,
	}
	mockHttpClient.EXPECT().Get(gomock.Any()).Return(response, nil)

	fetcher, err := security.NewContainerCredentialsFetcher(mockHttpClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// WHEN
	_, err = fetcher.FetchContainerCredentials()

	// THEN
	if err == nil || !strings.Contains(err.Error(), "unsuccessful response from credentials provider") {
		t.Fatalf("expected error for non-200 response, got %v", err)
	}
}

// GIVEN invalid Json response WHEN FetchContainerCredentials THEN an error should be returned
func TestContainerCredentialsFetcher_FetchContainerCredentials_InvalidJsonResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// GIVEN
	_ = os.Setenv(security.EnvironmentVariableContainerCredentials, "/v2/credentials")
	mockHttpClient := mock.NewMockHttpClient(ctrl)
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("invalid json")),
	}
	mockHttpClient.EXPECT().Get(gomock.Any()).Return(response, nil)

	fetcher, err := security.NewContainerCredentialsFetcher(mockHttpClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// WHEN
	_, err = fetcher.FetchContainerCredentials()

	// THEN
	if err == nil || !strings.Contains(err.Error(), "failed to decode credentials") {
		t.Fatalf("expected JSON decoding error, got %v", err)
	}
}
