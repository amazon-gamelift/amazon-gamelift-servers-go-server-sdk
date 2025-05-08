/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package result

// GetComputeCertificateResult - object containing the location of the fleet's TLS certificate file
// and certificate chain, which are stored on the instance.
type GetComputeCertificateResult struct {
	CertificatePath string `json:"CertificatePath"`
	ComputeName     string `json:"ComputeName"`
}
