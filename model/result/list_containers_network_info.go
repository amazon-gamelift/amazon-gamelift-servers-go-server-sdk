/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package result

// ContainerGroupType represents the type of container group definition.
type ContainerGroupType string

const (
	ContainerGroupTypeGameServer  ContainerGroupType = "GAME_SERVER"
	ContainerGroupTypePerInstance ContainerGroupType = "PER_INSTANCE"
)

// ContainerNetworkInfo contains network information for a single container on the instance.
type ContainerNetworkInfo struct {
	ContainerName      string             `json:"containerName"`
	ContainerID        string             `json:"containerId"`
	IPAddress          string             `json:"ipAddress"`
	ContainerGroupType ContainerGroupType `json:"containerGroupType"`
}

// ListContainersNetworkInfoResult contains network information for all containers on the instance.
type ListContainersNetworkInfoResult struct {
	ContainersNetworkInfo []ContainerNetworkInfo
}
