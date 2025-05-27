/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package security

type ComputeMetadata interface {
	GetHostId() string
}

// Holds container task metadata.
type ContainerTaskMetadata struct {
	TaskId string
}

func (c *ContainerTaskMetadata) GetHostId() string {
	return c.TaskId
}
