/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package security

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server/internal/transport"
)

const (
	EnvironmentVariableContainerMetadataURI = "ECS_CONTAINER_METADATA_URI_V4"
	taskMetadataRelativePath                = "task"
)

// ContainerMetadataFetcher handles fetching container metadata.
type ContainerMetadataFetcher struct {
	httpClient transport.HttpClient
}

// NewContainerMetadataFetcher creates a new instance of ContainerMetadataFetcher.
func NewContainerMetadataFetcher(httpClient transport.HttpClient) (*ContainerMetadataFetcher, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("httpClient cannot be nil")
	}
	return &ContainerMetadataFetcher{
		httpClient: httpClient,
	}, nil
}

// FetchContainerTaskMetadata fetches container task metadata.
func (f *ContainerMetadataFetcher) FetchContainerTaskMetadata() (*ContainerTaskMetadata, error) {
	containerMetadataURI := os.Getenv(EnvironmentVariableContainerMetadataURI)
	if containerMetadataURI == "" {
		return nil, fmt.Errorf("environment variable %s is not set", EnvironmentVariableContainerMetadataURI)
	}

	containerTaskMetadataURI := fmt.Sprintf("%s/%s", containerMetadataURI, taskMetadataRelativePath)
	response, err := f.httpClient.Get(containerTaskMetadataURI)
	defer func() {
		if response != nil {
			response.Body.Close()
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch container task metadata: %w", err)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("unsuccessful response from metadata service: %s", response.Status)
	}

	var taskMetadata map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&taskMetadata); err != nil {
		return nil, fmt.Errorf("failed to parse task metadata: %w", err)
	}

	taskArn, ok := taskMetadata["TaskARN"].(string)
	if !ok || taskArn == "" {
		return nil, fmt.Errorf("TaskArn is not available in container task metadata")
	}

	taskArnParts := strings.Split(taskArn, ":")
	if len(taskArnParts) < 4 {
		return nil, fmt.Errorf("invalid TaskArn format")
	}
	containerTaskReference := taskArnParts[len(taskArnParts)-1]
	containerTaskParts := strings.Split(containerTaskReference, "/")
	taskId := containerTaskParts[len(containerTaskParts)-1]

	return &ContainerTaskMetadata{
		TaskId: taskId,
	}, nil
}
