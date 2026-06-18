/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package security

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/model/result"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server/internal/transport"
)

const (
	discoveryServerPath = "/v1/"
	discoveryServerPort = 4092
)

// ContainerNetworkInfoFetcher handles fetching container network info from the discovery server.
type ContainerNetworkInfoFetcher struct {
	httpClient transport.HttpClient
}

// NewContainerNetworkInfoFetcher creates a new instance of ContainerNetworkInfoFetcher.
func NewContainerNetworkInfoFetcher(httpClient transport.HttpClient) (*ContainerNetworkInfoFetcher, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("httpClient cannot be nil")
	}
	return &ContainerNetworkInfoFetcher{
		httpClient: httpClient,
	}, nil
}

// FetchContainersNetworkInfo fetches network info for all containers on the instance.
func (f *ContainerNetworkInfoFetcher) FetchContainersNetworkInfo() (result.ListContainersNetworkInfoResult, error) {
	var res result.ListContainersNetworkInfoResult

	computeType := os.Getenv(common.EnvironmentKeyComputeType)
	if computeType != common.ComputeTypeContainer {
		return res, common.NewGameLiftError(
			common.UnsupportedComputeTypeException,
			"Unsupported compute type.",
			"ListContainersNetworkInfo is only supported on container fleets.",
		)
	}

	response, err := f.fetchDiscoveryServerResponse()
	if err != nil {
		return res, err
	}
	if response == nil {
		return res, common.NewGameLiftError(
			common.InternalServiceException,
			"Discovery server error.",
			"Received nil response from discovery server",
		)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return res, common.NewGameLiftError(
			common.InternalServiceException,
			"Discovery server error.",
			fmt.Sprintf("Discovery server returned HTTP %d", response.StatusCode),
		)
	}

	var containers []result.ContainerNetworkInfo
	if jsonErr := json.NewDecoder(response.Body).Decode(&containers); jsonErr != nil {
		return res, common.NewGameLiftError(
			common.InternalServiceException,
			"Invalid response from discovery server.",
			"Invalid response from discovery server",
		)
	}

	res.ContainersNetworkInfo = containers
	return res, nil
}

// fetchDiscoveryServerResponse resolves the discovery server endpoint and fetches the /v1/ response.
// Strategy:
//  1. Use GAMELIFT_CONTAINER_DISCOVERY_SERVER_ENDPOINT env var
//  2. Fallback: derive endpoint from ECS container metadata
//  3. If env var endpoint fails, retry with metadata-derived endpoint
func (f *ContainerNetworkInfoFetcher) fetchDiscoveryServerResponse() (*http.Response, error) {
	envEndpoint := os.Getenv(common.EnvironmentKeyDiscoveryEndpoint)
	endpoint := envEndpoint

	if endpoint == "" {
		endpoint = f.resolveDiscoveryEndpointFromMetadata()
		if endpoint == "" {
			return nil, common.NewGameLiftError(
				common.InternalServiceException,
				"Discovery endpoint not available.",
				"Could not resolve discovery server endpoint.",
			)
		}
	}

	response, err := f.httpClient.Get(endpoint + discoveryServerPath)
	if err != nil {
		// If we used the env var, try fallback from metadata
		if envEndpoint != "" {
			fallback := f.resolveDiscoveryEndpointFromMetadata()
			if fallback != "" && fallback != endpoint {
				fallbackResp, fallbackErr := f.httpClient.Get(fallback + discoveryServerPath)
				if fallbackErr == nil {
					return fallbackResp, nil
				}
			}
		}
		return nil, common.NewGameLiftError(
			common.InternalServiceException,
			"Failed to connect to discovery server.",
			fmt.Sprintf("Failed to connect to discovery server: %v", err),
		)
	}

	return response, nil
}

// resolveDiscoveryEndpointFromMetadata queries ECS container metadata to derive the bridge gateway IP.
// ECS metadata returns: {"Networks":[{"NetworkMode":"bridge","IPv4Addresses":["172.17.0.5"]}]}
// The bridge gateway is always .1 on the container's subnet (e.g., 172.17.0.5 → 172.17.0.1)
func (f *ContainerNetworkInfoFetcher) resolveDiscoveryEndpointFromMetadata() string {
	metadataURI := os.Getenv(EnvironmentVariableContainerMetadataURI)
	if metadataURI == "" {
		return ""
	}

	response, err := f.httpClient.Get(metadataURI)
	if err != nil {
		return ""
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ""
	}

	var metadata struct {
		Networks []struct {
			IPv4Addresses []string `json:"IPv4Addresses"`
		} `json:"Networks"`
	}
	if err := json.NewDecoder(response.Body).Decode(&metadata); err != nil {
		return ""
	}

	if len(metadata.Networks) == 0 || len(metadata.Networks[0].IPv4Addresses) == 0 {
		return ""
	}

	containerIP := metadata.Networks[0].IPv4Addresses[0]
	lastDot := strings.LastIndex(containerIP, ".")
	if lastDot < 0 {
		return ""
	}
	gatewayIP := containerIP[:lastDot] + ".1"
	return fmt.Sprintf("http://%s:%d", gatewayIP, discoveryServerPort)
}
