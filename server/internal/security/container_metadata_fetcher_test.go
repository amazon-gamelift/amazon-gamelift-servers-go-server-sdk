/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package security_test

import (
	"errors"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/server/internal/security"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/server/internal/transport"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/server/internal/mock"
	"github.com/golang/mock/gomock"
)

// GIVEN nil HttpClient WHEN NewContainerMetadataFetcher THEN an error should be returned
func TestContainerMetadataFetcher_NewContainerMetadataFetcher_NilHttpClient(t *testing.T) {

	// GIVEN
	var httpClient transport.HttpClient

	// WHEN
	_, err := security.NewContainerMetadataFetcher(httpClient)

	// THEN
	if err == nil || !strings.Contains(err.Error(), "httpClient cannot be nil") {
		t.Fatalf("expected httpClient cannot be nil, got %v", err)
	}
}

// GIVEN valid ContainerTaskMetadata response WHEN FetchContainerTaskMetadata THEN return ContainerTaskMetadata
func TestContainerMetadataFetcher_FetchContainerTaskMetadata_ValidMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHttpClient := mock.NewMockHttpClient(ctrl)

	// GIVEN
	_ = os.Setenv(security.EnvironmentVariableContainerMetadataURI, "http://dummy-url")
	validMetadata := "{\"TaskARN\": \"arn:aws:ecs:us-west-2:123456789012:task/test-task/abcdef1234567890\"}"
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(validMetadata)),
	}
	mockHttpClient.EXPECT().Get(gomock.Any()).Return(response, nil)
	fetcher, err := security.NewContainerMetadataFetcher(mockHttpClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// WHEN
	taskMetadata, err := fetcher.FetchContainerTaskMetadata()

	// THEN
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if taskMetadata.TaskId != "abcdef1234567890" {
		t.Fatalf("unexpected task metadata: %+v", taskMetadata)
	}
}

// GIVEN missing environment variable WHEN FetchContainerTaskMetadata THEN an error should be returned
func TestContainerMetadataFetcher_FetchContainerTaskMetadata_MissingEnvironmentVariable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// GIVEN
	_ = os.Unsetenv(security.EnvironmentVariableContainerMetadataURI)
	mockHttpClient := mock.NewMockHttpClient(ctrl)
	fetcher, err := security.NewContainerMetadataFetcher(mockHttpClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// WHEN
	_, err = fetcher.FetchContainerTaskMetadata()

	// THEN
	if err == nil || !strings.Contains(err.Error(), "environment variable ECS_CONTAINER_METADATA_URI_V4 is not set") {
		t.Fatalf("expected error for unset environment variable, got %v", err)
	}
}

// GIVEN failure to send HttpRequest WHEN FetchContainerTaskMetadata THEN an error should be returned
func TestContainerMetadataFetcher_FetchContainerTaskMetadata_FailureToSendRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// GIVEN
	_ = os.Setenv(security.EnvironmentVariableContainerMetadataURI, "http://invalid-url")
	mockHttpClient := mock.NewMockHttpClient(ctrl)
	mockHttpClient.EXPECT().Get(gomock.Any()).Return(nil, errors.New("failed request"))

	fetcher, err := security.NewContainerMetadataFetcher(mockHttpClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// WHEN
	_, err = fetcher.FetchContainerTaskMetadata()

	// THEN
	if err == nil || !strings.Contains(err.Error(), "failed to fetch container task metadata") {
		t.Fatalf("expected error for failed HTTP request, got %v", err)
	}
}

// GIVEN failed HttpResponse WHEN FetchContainerTaskMetadata THEN an error should be returned
func TestContainerMetadataFetcher_FetchContainerTaskMetadata_FailedHttpResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// GIVEN
	_ = os.Setenv(security.EnvironmentVariableContainerMetadataURI, "http://dummy-url")
	mockHttpClient := mock.NewMockHttpClient(ctrl)
	response := &http.Response{
		StatusCode: http.StatusServiceUnavailable,
		Body:       http.NoBody,
	}
	mockHttpClient.EXPECT().Get(gomock.Any()).Return(response, nil)

	fetcher, err := security.NewContainerMetadataFetcher(mockHttpClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// WHEN
	_, err = fetcher.FetchContainerTaskMetadata()

	// THEN
	if err == nil || !strings.Contains(err.Error(), "unsuccessful response from metadata service") {
		t.Fatalf("expected error for non-200 response, got %v", err)
	}
}

// GIVEN invalid Json response WHEN FetchContainerTaskMetadata THEN an error should be returned
func TestContainerMetadataFetcher_FetchContainerTaskMetadata_InvalidJsonResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// GIVEN
	_ = os.Setenv(security.EnvironmentVariableContainerMetadataURI, "http://dummy-url")
	mockHttpClient := mock.NewMockHttpClient(ctrl)
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("invalid json")),
	}
	mockHttpClient.EXPECT().Get(gomock.Any()).Return(response, nil)

	fetcher, err := security.NewContainerMetadataFetcher(mockHttpClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// WHEN
	_, err = fetcher.FetchContainerTaskMetadata()

	// THEN
	if err == nil || !strings.Contains(err.Error(), "failed to parse task metadata") {
		t.Fatalf("expected JSON parsing error, got %v", err)
	}
}

// GIVEN Json response with missing TaskArn WHEN FetchContainerTaskMetadata THEN an error should be returned
func TestContainerMetadataFetcher_FetchContainerTaskMetadata_MissingTaskARN(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// GIVEN
	_ = os.Setenv(security.EnvironmentVariableContainerMetadataURI, "http://dummy-url")
	mockHttpClient := mock.NewMockHttpClient(ctrl)
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("{\"A\":\"B\"}")),
	}
	mockHttpClient.EXPECT().Get(gomock.Any()).Return(response, nil)

	fetcher, err := security.NewContainerMetadataFetcher(mockHttpClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// WHEN
	_, err = fetcher.FetchContainerTaskMetadata()

	// THEN
	if err == nil || !strings.Contains(err.Error(), "TaskArn is not available in container task metadata") {
		t.Fatalf("expected error for missing TaskARN, got %v", err)
	}
}
