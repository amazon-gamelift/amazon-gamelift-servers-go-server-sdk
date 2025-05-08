/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package security

import (
	"aws/amazon-gamelift-go-sdk/server/internal/transport"
	"encoding/json"
	"fmt"
	"os"
)

// ContainerCredentialsFetcher fetches AWS credentials for the container.
type ContainerCredentialsFetcher struct {
	httpClient transport.HttpClient
}

const (
	containerCredentialProviderURL          = "http://169.254.170.2"
	EnvironmentVariableContainerCredentials = "AWS_CONTAINER_CREDENTIALS_RELATIVE_URI"
)

// NewContainerCredentialsFetcher creates a new instance of ContainerCredentialsFetcher.
func NewContainerCredentialsFetcher(httpClient transport.HttpClient) (*ContainerCredentialsFetcher, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("httpClient cannot be nil")
	}
	return &ContainerCredentialsFetcher{
		httpClient: httpClient,
	}, nil
}

// FetchContainerCredentials fetches container credentials from Container Credential Provider service.
func (f *ContainerCredentialsFetcher) FetchContainerCredentials() (*AwsCredentials, error) {
	relativeURI := os.Getenv(EnvironmentVariableContainerCredentials)
	if relativeURI == "" {
		return nil, fmt.Errorf("environment variable %s is not set", EnvironmentVariableContainerCredentials)
	}

	credentialsProviderURI := fmt.Sprintf("%s%s", containerCredentialProviderURL, relativeURI)
	response, err := f.httpClient.Get(credentialsProviderURI)
	defer func() {
		if response != nil {
			response.Body.Close()
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch container credentials: %w", err)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("unsuccessful response from credentials provider: %s", response.Status)
	}

	var awsCredentials AwsCredentials
	if err := json.NewDecoder(response.Body).Decode(&awsCredentials); err != nil {
		return nil, fmt.Errorf("failed to decode credentials: %w", err)
	}

	return &awsCredentials, nil
}
