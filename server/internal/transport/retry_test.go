/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package transport_test

import (
	"errors"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"go.uber.org/goleak"

	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/server/internal/mock"
	"github.com/jamesstow/amazon-gamelift-servers-go-server-sdk/v5/server/internal/transport"
)

var testError = errors.New("test error")

const TestRetryInterval = "5ms"

func setRetryEnvironmentVariables(t *testing.T) {
	envErr := os.Setenv(common.RetryInterval, TestRetryInterval)
	if envErr != nil {
		t.Fatalf("Failed to set RetryInterval environment variable: %s", envErr)
	}
}

func TestRetryTransportWrite(t *testing.T) {
	setRetryEnvironmentVariables(t)
	defer goleak.VerifyNone(t)

	ctrl := gomock.NewController(t)

	logger := mock.NewMockILogger(ctrl)
	transportMock := mock.NewMockITransport(ctrl)

	transportMock.
		EXPECT().
		Write([]byte(testMessage)).
		Return(testError)

	logger.
		EXPECT().
		Debugf("Call Failed: %s. Retrying attempt: %d of %d", testError.Error(), 1, common.MaxRetryDefault)

	transportMock.
		EXPECT().
		Write([]byte(testMessage)).
		Return(nil)

	retryTransport := transport.WithRetry(transportMock, logger)

	t.Logf("Tests are running, please wait")

	err := retryTransport.Write([]byte(testMessage))
	if err != nil {
		t.Fatalf("fall to write to retry transport: %v", err)
	}
}

func TestRetryTransportWriteMaxAttempts(t *testing.T) {
	setRetryEnvironmentVariables(t)
	defer goleak.VerifyNone(t)

	ctrl := gomock.NewController(t)

	logger := mock.NewMockILogger(ctrl)
	transportMock := mock.NewMockITransport(ctrl)

	for i := 0; i < common.MaxRetryDefault; i++ {
		transportMock.
			EXPECT().
			Write([]byte(testMessage)).
			Return(testError)

		logger.
			EXPECT().
			Debugf("Call Failed: %s. Retrying attempt: %d of %d", testError.Error(), i+1, common.MaxRetryDefault)
	}

	retryTransport := transport.WithRetry(transportMock, logger)

	t.Logf("Tests are running, please wait")

	err := retryTransport.Write([]byte(testMessage))
	if err == nil || common.GetErrorTypeFromMessage(err.Error()) != common.WebsocketRetriableSendMessageFailure {
		t.Fatalf("unexpected error: %v", err)
	}
}
