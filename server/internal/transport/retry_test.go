/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package transport_test

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"go.uber.org/goleak"

	"aws/amazon-gamelift-go-sdk/common"
	"aws/amazon-gamelift-go-sdk/server/internal/mock"
	"aws/amazon-gamelift-go-sdk/server/internal/transport"
)

var testError = errors.New("test error")

func TestRetryTransportWrite(t *testing.T) {
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
	if err == nil || err.Error() != "[GameLiftError: ErrorType={25}, ErrorName={Failed write retry}, ErrorMessage={write attempt overflow}]" {
		t.Fatalf("unexpected error: %v", err)
	}
}
