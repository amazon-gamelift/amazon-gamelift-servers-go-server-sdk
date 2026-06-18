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

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server/internal/mock"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server/internal/security"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server/internal/transport"
	"github.com/golang/mock/gomock"
)

func clearContainerNetworkInfoEnv() {
	os.Unsetenv(common.EnvironmentKeyComputeType)
	os.Unsetenv(common.EnvironmentKeyDiscoveryEndpoint)
	os.Unsetenv(security.EnvironmentVariableContainerMetadataURI)
}

func TestContainerNetworkInfoFetcher_NewContainerNetworkInfoFetcher_NilHttpClient(t *testing.T) {
	var httpClient transport.HttpClient
	_, err := security.NewContainerNetworkInfoFetcher(httpClient)
	if err == nil || !strings.Contains(err.Error(), "httpClient cannot be nil") {
		t.Fatalf("expected httpClient cannot be nil, got %v", err)
	}
}

func TestContainerNetworkInfoFetcher_NonContainerComputeType_ReturnsUnsupportedError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer clearContainerNetworkInfoEnv()

	os.Setenv(common.EnvironmentKeyComputeType, "EC2")
	mockHttpClient := mock.NewMockHttpClient(ctrl)
	fetcher, _ := security.NewContainerNetworkInfoFetcher(mockHttpClient)

	_, err := fetcher.FetchContainersNetworkInfo()

	var glErr *common.GameLiftError
	if !errors.As(err, &glErr) || glErr.ErrorType != common.UnsupportedComputeTypeException {
		t.Fatalf("expected UnsupportedComputeTypeException, got %v", err)
	}
}

func TestContainerNetworkInfoFetcher_NoComputeType_ReturnsUnsupportedError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer clearContainerNetworkInfoEnv()

	os.Unsetenv(common.EnvironmentKeyComputeType)
	mockHttpClient := mock.NewMockHttpClient(ctrl)
	fetcher, _ := security.NewContainerNetworkInfoFetcher(mockHttpClient)

	_, err := fetcher.FetchContainersNetworkInfo()

	var glErr *common.GameLiftError
	if !errors.As(err, &glErr) || glErr.ErrorType != common.UnsupportedComputeTypeException {
		t.Fatalf("expected UnsupportedComputeTypeException, got %v", err)
	}
}

func TestContainerNetworkInfoFetcher_ContainerTypeNoEndpointNoMetadata_ReturnsServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer clearContainerNetworkInfoEnv()

	os.Setenv(common.EnvironmentKeyComputeType, common.ComputeTypeContainer)
	os.Unsetenv(common.EnvironmentKeyDiscoveryEndpoint)
	os.Unsetenv(security.EnvironmentVariableContainerMetadataURI)

	mockHttpClient := mock.NewMockHttpClient(ctrl)
	fetcher, _ := security.NewContainerNetworkInfoFetcher(mockHttpClient)

	_, err := fetcher.FetchContainersNetworkInfo()

	var glErr *common.GameLiftError
	if !errors.As(err, &glErr) || glErr.ErrorType != common.InternalServiceException {
		t.Fatalf("expected InternalServiceException, got %v", err)
	}
}

func TestContainerNetworkInfoFetcher_ContainerTypeWithUnreachableEndpoint_ReturnsServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer clearContainerNetworkInfoEnv()

	os.Setenv(common.EnvironmentKeyComputeType, common.ComputeTypeContainer)
	os.Setenv(common.EnvironmentKeyDiscoveryEndpoint, "http://192.0.2.1:9999")

	mockHttpClient := mock.NewMockHttpClient(ctrl)
	mockHttpClient.EXPECT().Get("http://192.0.2.1:9999/v1/").Return(nil, errors.New("connection refused"))
	// Fallback metadata resolution - no metadata URI set, so no additional calls

	fetcher, _ := security.NewContainerNetworkInfoFetcher(mockHttpClient)
	_, err := fetcher.FetchContainersNetworkInfo()

	var glErr *common.GameLiftError
	if !errors.As(err, &glErr) || glErr.ErrorType != common.InternalServiceException {
		t.Fatalf("expected InternalServiceException, got %v", err)
	}
}

func TestContainerNetworkInfoFetcher_EndpointFromEnvVar_ValidResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer clearContainerNetworkInfoEnv()

	os.Setenv(common.EnvironmentKeyComputeType, common.ComputeTypeContainer)
	os.Setenv(common.EnvironmentKeyDiscoveryEndpoint, "http://172.17.0.1:4092")

	body := `[
		{"containerName":"otel-collector","ipAddress":"172.17.0.2","containerId":"abc123def456","containerGroupType":"PER_INSTANCE"},
		{"containerName":"game-server","ipAddress":"172.17.0.3","containerId":"def456ghi789","containerGroupType":"GAME_SERVER"}
	]`
	mockHttpClient := mock.NewMockHttpClient(ctrl)
	mockHttpClient.EXPECT().Get("http://172.17.0.1:4092/v1/").Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil)

	fetcher, _ := security.NewContainerNetworkInfoFetcher(mockHttpClient)
	res, err := fetcher.FetchContainersNetworkInfo()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.ContainersNetworkInfo) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(res.ContainersNetworkInfo))
	}
	if res.ContainersNetworkInfo[0].ContainerName != "otel-collector" {
		t.Fatalf("expected otel-collector, got %s", res.ContainersNetworkInfo[0].ContainerName)
	}
	if res.ContainersNetworkInfo[0].ContainerGroupType != "PER_INSTANCE" {
		t.Fatalf("expected PER_INSTANCE, got %s", res.ContainersNetworkInfo[0].ContainerGroupType)
	}
	if res.ContainersNetworkInfo[1].ContainerGroupType != "GAME_SERVER" {
		t.Fatalf("expected GAME_SERVER, got %s", res.ContainersNetworkInfo[1].ContainerGroupType)
	}
}

func TestContainerNetworkInfoFetcher_FallbackFromMetadata_ValidResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer clearContainerNetworkInfoEnv()

	os.Setenv(common.EnvironmentKeyComputeType, common.ComputeTypeContainer)
	os.Unsetenv(common.EnvironmentKeyDiscoveryEndpoint)
	os.Setenv(security.EnvironmentVariableContainerMetadataURI, "http://169.254.170.2/v4/metadata")

	// First call: ECS metadata to resolve endpoint
	metadataBody := `{"Networks":[{"NetworkMode":"bridge","IPv4Addresses":["172.17.0.5"]}]}`
	mockHttpClient := mock.NewMockHttpClient(ctrl)
	mockHttpClient.EXPECT().Get("http://169.254.170.2/v4/metadata").Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(metadataBody)),
	}, nil)

	// Second call: discovery server at derived gateway
	discoveryBody := `[{"containerName":"game-server","ipAddress":"172.17.0.5","containerId":"abc123","containerGroupType":"GAME_SERVER"}]`
	mockHttpClient.EXPECT().Get("http://172.17.0.1:4092/v1/").Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(discoveryBody)),
	}, nil)

	fetcher, _ := security.NewContainerNetworkInfoFetcher(mockHttpClient)
	res, err := fetcher.FetchContainersNetworkInfo()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.ContainersNetworkInfo) != 1 {
		t.Fatalf("expected 1 container, got %d", len(res.ContainersNetworkInfo))
	}
	if res.ContainersNetworkInfo[0].IPAddress != "172.17.0.5" {
		t.Fatalf("expected 172.17.0.5, got %s", res.ContainersNetworkInfo[0].IPAddress)
	}
}

func TestContainerNetworkInfoFetcher_EnvVarFails_FallbackSucceeds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer clearContainerNetworkInfoEnv()

	os.Setenv(common.EnvironmentKeyComputeType, common.ComputeTypeContainer)
	os.Setenv(common.EnvironmentKeyDiscoveryEndpoint, "http://10.0.0.1:4092")
	os.Setenv(security.EnvironmentVariableContainerMetadataURI, "http://169.254.170.2/v4/metadata")

	mockHttpClient := mock.NewMockHttpClient(ctrl)

	// Primary endpoint fails
	mockHttpClient.EXPECT().Get("http://10.0.0.1:4092/v1/").Return(nil, errors.New("connection refused"))

	// Metadata resolution for fallback - different subnet so fallback != endpoint
	metadataBody := `{"Networks":[{"NetworkMode":"bridge","IPv4Addresses":["172.17.0.5"]}]}`
	mockHttpClient.EXPECT().Get("http://169.254.170.2/v4/metadata").Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(metadataBody)),
	}, nil)

	// Fallback endpoint succeeds
	discoveryBody := `[{"containerName":"game-server","ipAddress":"172.17.0.5","containerId":"xyz789","containerGroupType":"GAME_SERVER"}]`
	mockHttpClient.EXPECT().Get("http://172.17.0.1:4092/v1/").Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(discoveryBody)),
	}, nil)

	fetcher, _ := security.NewContainerNetworkInfoFetcher(mockHttpClient)
	res, err := fetcher.FetchContainersNetworkInfo()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.ContainersNetworkInfo) != 1 {
		t.Fatalf("expected 1 container, got %d", len(res.ContainersNetworkInfo))
	}
	if res.ContainersNetworkInfo[0].ContainerName != "game-server" {
		t.Fatalf("expected game-server, got %s", res.ContainersNetworkInfo[0].ContainerName)
	}
}

func TestContainerNetworkInfoFetcher_EmptyList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer clearContainerNetworkInfoEnv()

	os.Setenv(common.EnvironmentKeyComputeType, common.ComputeTypeContainer)
	os.Setenv(common.EnvironmentKeyDiscoveryEndpoint, "http://172.17.0.1:4092")

	mockHttpClient := mock.NewMockHttpClient(ctrl)
	mockHttpClient.EXPECT().Get("http://172.17.0.1:4092/v1/").Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("[]")),
	}, nil)

	fetcher, _ := security.NewContainerNetworkInfoFetcher(mockHttpClient)
	res, err := fetcher.FetchContainersNetworkInfo()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.ContainersNetworkInfo) != 0 {
		t.Fatalf("expected 0 containers, got %d", len(res.ContainersNetworkInfo))
	}
}

func TestContainerNetworkInfoFetcher_InvalidJsonResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer clearContainerNetworkInfoEnv()

	os.Setenv(common.EnvironmentKeyComputeType, common.ComputeTypeContainer)
	os.Setenv(common.EnvironmentKeyDiscoveryEndpoint, "http://172.17.0.1:4092")

	mockHttpClient := mock.NewMockHttpClient(ctrl)
	mockHttpClient.EXPECT().Get("http://172.17.0.1:4092/v1/").Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("not json")),
	}, nil)

	fetcher, _ := security.NewContainerNetworkInfoFetcher(mockHttpClient)
	_, err := fetcher.FetchContainersNetworkInfo()

	if err == nil || !strings.Contains(err.Error(), "Invalid response from discovery server") {
		t.Fatalf("expected invalid response error, got %v", err)
	}
}

func TestContainerNetworkInfoFetcher_HttpNon200Response(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer clearContainerNetworkInfoEnv()

	os.Setenv(common.EnvironmentKeyComputeType, common.ComputeTypeContainer)
	os.Setenv(common.EnvironmentKeyDiscoveryEndpoint, "http://172.17.0.1:4092")

	mockHttpClient := mock.NewMockHttpClient(ctrl)
	mockHttpClient.EXPECT().Get("http://172.17.0.1:4092/v1/").Return(&http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil)

	fetcher, _ := security.NewContainerNetworkInfoFetcher(mockHttpClient)
	_, err := fetcher.FetchContainersNetworkInfo()

	if err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Fatalf("expected HTTP 500 error, got %v", err)
	}
}
