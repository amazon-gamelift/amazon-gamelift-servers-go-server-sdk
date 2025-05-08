/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

const (
	DateFormat                 = "20060102"
	DateTimeFormat             = "20060102T150405Z"
	ServiceName                = "gamelift"
	TerminationString          = "aws4_request"
	SignatureSecretKeyPrefix   = "AWS4"
	Algorithm                  = "AWS4-HMAC-SHA256"
	AuthorizationKey           = "Authorization"
	AuthorizationValue         = "SigV4"
	AmzAlgorithmKey            = "X-Amz-Algorithm"
	AmzCredentialKey           = "X-Amz-Credential"
	AmzDateKey                 = "X-Amz-Date"
	AmzSecurityTokenHeadersKey = "X-Amz-Security-Token"
	AmzSignatureKey            = "X-Amz-Signature"
)

// GenerateSigV4QueryParameters generates a SigV4 query string based on the passed parameters.
func GenerateSigV4QueryParameters(params SigV4Parameters) (map[string]string, error) {
	if err := validateParameters(params); err != nil {
		return nil, err
	}

	formattedRequestDate := params.RequestTime.UTC().Format(DateFormat)
	formattedRequestDateTime := params.RequestTime.UTC().Format(DateTimeFormat)

	canonicalRequest := toSortedEncodedQueryString(params.QueryParams)
	hashedCanonicalRequest := computeSha256Hash(canonicalRequest)

	scope := fmt.Sprintf("%s/%s/%s/%s", formattedRequestDate, params.AwsRegion, ServiceName, TerminationString)
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s", Algorithm, formattedRequestDateTime, scope, hashedCanonicalRequest)

	signature := generateSignature(params.AwsRegion, params.AwsCredentials.SecretKey, formattedRequestDate, ServiceName, stringToSign)

	credential := fmt.Sprintf("%s/%s", params.AwsCredentials.AccessKey, scope)
	queryParameters := generateSigV4QueryParameters(credential, formattedRequestDateTime, params.AwsCredentials.SessionToken, signature)

	return queryParameters, nil
}

func validateParameters(params SigV4Parameters) error {
	if params.AwsRegion == "" {
		return fmt.Errorf("AwsRegion is required")
	}
	if params.AwsCredentials.AccessKey == "" {
		return fmt.Errorf("AccessKey is required")
	}
	if params.AwsCredentials.SecretKey == "" {
		return fmt.Errorf("SecretKey is required")
	}
	if params.QueryParams == nil || len(params.QueryParams) == 0 {
		return fmt.Errorf("QueryParams is required")
	}
	if params.RequestTime.IsZero() {
		return fmt.Errorf("RequestTime is required")
	}
	return nil
}

func generateSignature(region, secretKey, formattedRequestDate, serviceName, stringToSign string) string {
	encodedKeySecret := []byte(SignatureSecretKeyPrefix + secretKey)
	hashDate := computeHmacSha256(encodedKeySecret, formattedRequestDate)
	hashRegion := computeHmacSha256(hashDate, region)
	hashService := computeHmacSha256(hashRegion, serviceName)
	signingKey := computeHmacSha256(hashService, TerminationString)

	signature := toHex(computeHmacSha256(signingKey, stringToSign))
	return signature
}

func generateSigV4QueryParameters(credential, formattedRequestDateTime, sessionToken, signature string) map[string]string {
	sigV4QueryParams := map[string]string{
		AuthorizationKey: AuthorizationValue,
		AmzAlgorithmKey:  Algorithm,
		AmzCredentialKey: credential,
		AmzDateKey:       formattedRequestDateTime,
		AmzSignatureKey:  signature,
	}

	if sessionToken != "" {
		sigV4QueryParams[AmzSecurityTokenHeadersKey] = sessionToken
	}

	return sigV4QueryParams
}

func toSortedEncodedQueryString(params map[string]string) string {
	var keys []string
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var queryParams []string
	for _, key := range keys {
		queryParams = append(queryParams, fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(params[key])))
	}
	return strings.Join(queryParams, "&")
}

func computeSha256Hash(data string) string {
	hash := sha256.New()
	hash.Write([]byte(data))
	return hex.EncodeToString(hash.Sum(nil))
}

func computeHmacSha256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func toHex(data []byte) string {
	return hex.EncodeToString(data)
}
