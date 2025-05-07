/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package server_test

import (
	"aws/amazon-gamelift-go-sdk/common"
	server "aws/amazon-gamelift-go-sdk/server"
	"testing"
)

func TestGetSDKVersion(t *testing.T) {
	version, err := server.GetSdkVersion()
	if err != nil {
		t.Fatal(err)
	}

	if version != common.SdkVersion {
		t.Errorf("expect  %v but get %v", common.SdkVersion, version)
	}
}
