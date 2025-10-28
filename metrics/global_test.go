/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package metrics

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/model"

	"github.com/golang/mock/gomock"
)

// resetGlobalProcessor resets the global processor state for testing.
// This should be called at the beginning and end of tests that use the global processor.
func resetGlobalProcessor() {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	globalProcessor = nil
	once = sync.Once{}
}

// InitIsolatedTestProcessor initializes a test processor and ensures cleanup after the test.
func InitIsolatedTestProcessor(t *testing.T, options ...ProcessorOption) error {
	t.Helper()

	// Reset any existing processor first.
	resetGlobalProcessor()

	// Initialize the processor with the provided options.
	err := InitMetricsProcessor(options...)
	if err != nil {
		return err
	}

	// Register cleanup to run after the test.
	t.Cleanup(func() {
		// Stop the processor if it's running.
		if processor := GetGlobalProcessor(); processor != nil {
			processor.Stop() //nolint:errcheck
		}
		resetGlobalProcessor()
	})

	return nil
}

// TestSetGlobalProcessor tests that only the first processor becomes global.
func TestSetGlobalProcessor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Initialize the global processor.
	err := InitIsolatedTestProcessor(t,
		WithTransport(NewMockTransport(ctrl)),
	)
	common.AssertEqual(t, nil, err)

	// Verify it was set as global.
	globalProc := GetGlobalProcessor()
	if globalProc == nil {
		t.Fatal("Expected global processor to not be nil")
	}
	common.AssertEqual(t, true, HasGlobalProcessor())
}

// TestGlobalProcessorSingleton tests that subsequent calls return errors after initialization.
func TestGlobalProcessorSingleton(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Initialize processor with non-tester function, so it doesn't internally reset.
	err := InitMetricsProcessor(
		WithTransport(NewMockTransport(ctrl)),
		WithGlobalTag("tag1", "value1"),
	)
	common.AssertEqual(t, nil, err)
	processor1 := GetGlobalProcessor()

	// Try to initialize second processor - should return error.
	err2 := InitMetricsProcessor(
		WithTransport(NewMockTransport(ctrl)),
		WithGlobalTag("tag2", "value2"),
	)
	if err2 == nil {
		t.Fatal("Expected error when initializing processor twice")
	}

	// Global processor should still be the first one.
	processor2 := GetGlobalProcessor()
	common.AssertEqual(t, processor1, processor2)

	// Global tags should be from the first processor.
	globalTags := processor2.GetGlobalTags()
	_, hasTag1 := globalTags["tag1"]
	_, hasTag2 := globalTags["tag2"]
	common.AssertEqual(t, true, hasTag1)
	common.AssertEqual(t, false, hasTag2)
}

// TestHasGlobalProcessor tests the HasGlobalProcessor function.
func TestHasGlobalProcessor(t *testing.T) {
	resetGlobalProcessor()
	// Initially should not have a global processor.
	common.AssertEqual(t, false, HasGlobalProcessor())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Initialize a processor.
	err := InitIsolatedTestProcessor(t,
		WithTransport(NewMockTransport(ctrl)),
	)
	common.AssertEqual(t, nil, err)

	// Now should have a global processor.
	common.AssertEqual(t, true, HasGlobalProcessor())
}

// TestGetGlobalProcessorNil tests GetGlobalProcessor when none is set.
func TestGetGlobalProcessorNil(t *testing.T) {
	resetGlobalProcessor()

	// Should return nil when no processor is set.
	processor := GetGlobalProcessor()
	common.AssertEqual(t, (*Processor)(nil), processor)
}

// TestGlobalProcessorConcurrency tests concurrent initialization attempts.
func TestGlobalProcessorConcurrency(t *testing.T) {
	resetGlobalProcessor()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var wg sync.WaitGroup
	numGoroutines := 10
	errors := make([]error, numGoroutines)

	// Try to initialize processors concurrently.
	// Initialize processor with non-tester function, so it doesn't internally reset.

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errors[idx] = InitMetricsProcessor(
				WithTransport(NewMockTransport(ctrl)),
			)
		}(i)
	}

	wg.Wait()

	// Only one should succeed, others should error.
	successCount := 0
	for _, err := range errors {
		if err == nil {
			successCount++
		}
	}
	common.AssertEqual(t, 1, successCount)

	// Global processor should be set.
	common.AssertEqual(t, true, HasGlobalProcessor())
}

// TestGlobalProcessorThreadSafety tests thread-safe operations on global processor.
func TestGlobalProcessorThreadSafety(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Initialize a processor.
	err := InitIsolatedTestProcessor(t,
		WithTransport(NewMockTransport(ctrl)),
	)
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrently access global processor.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Get global processor.
			proc := GetGlobalProcessor()
			if proc != nil {
				// Set a tag.
				tagKey := fmt.Sprintf("tag_%d", idx)
				tagValue := fmt.Sprintf("value_%d", idx)
				proc.SetGlobalTag(tagKey, tagValue) //nolint:errcheck

				// Get tags.
				proc.GetGlobalTags()

				// Check if processor exists.
				HasGlobalProcessor()
			}
		}(i)
	}

	wg.Wait()

	// Should still have the global processor.
	common.AssertEqual(t, true, HasGlobalProcessor())
	common.AssertEqual(t, processor, GetGlobalProcessor())
}

func TestGlobalIntegration(t *testing.T) {
	// Reset global state, as some tests use non-testing processor.
	resetGlobalProcessor()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 1. Initialize processor with some initial tags
	err := InitIsolatedTestProcessor(t,
		WithTransport(NewMockTransport(ctrl)),
		WithGlobalTags(map[string]string{
			"service": "test-service",
			"env":     "test",
		}),
	)
	common.AssertEqual(t, nil, err)
	processor := GetGlobalProcessor()

	// 2. Verify initial tags are set
	globalTags := processor.GetGlobalTags()
	common.AssertEqual(t, "test-service", globalTags["service"])
	common.AssertEqual(t, "test", globalTags["env"])
}

// TestInitMetricsProcessorReturnsError tests that InitMetricsProcessor returns error gracefully.
func TestInitMetricsProcessorReturnsError(t *testing.T) {
	resetGlobalProcessor()

	// Test with missing transport - should return error, not panic.
	err := InitMetricsProcessor()
	if err == nil {
		t.Fatal("Expected error when transport is missing")
	}

	// Verify no processor was created.
	common.AssertEqual(t, false, HasGlobalProcessor())
}

// TestMustInitMetricsProcessorPanics tests that MustInitMetricsProcessor panics on failure.
func TestMustInitMetricsProcessorPanics(t *testing.T) {
	resetGlobalProcessor()

	// Test that MustInitMetricsProcessor panics when transport is missing.
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected MustInitMetricsProcessor to panic, but it didn't")
		}
	}()

	MustInitMetricsProcessor() // Should panic due to missing transport
}

// TestMustInitMetricsProcessorSuccess tests that MustInitMetricsProcessor works when valid.
func TestMustInitMetricsProcessorSuccess(t *testing.T) {
	resetGlobalProcessor()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Should not panic with valid transport.
	MustInitMetricsProcessor(WithTransport(NewMockTransport(ctrl)))

	// Verify processor was created.
	common.AssertEqual(t, true, HasGlobalProcessor())
}

// TestGlobalProcessorDefaultTags tests that default tags are set automatically.
func TestGlobalProcessorDefaultTags(t *testing.T) {
	resetGlobalProcessor()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Set environment variables for testing.
	t.Setenv(common.EnvironmentKeyProcessID, "test-process-id")

	// Create mock transport with expectations.
	mockTransport := NewMockTransport(ctrl)
	mockTransport.EXPECT().Send(gomock.Any()).Return(nil).AnyTimes()
	mockTransport.EXPECT().Close().Return(nil).AnyTimes()

	// Initialize processor.
	err := InitIsolatedTestProcessor(t, WithTransport(mockTransport))
	common.AssertEqual(t, nil, err)

	// Get global processor.
	globalProc := GetGlobalProcessor()
	if globalProc == nil {
		t.Fatal("Expected globalProc to not be nil")
	}

	// Check default tags were set.
	globalTags := globalProc.GetGlobalTags()

	// Check process_pid is set and matches the actual pid.
	common.AssertEqual(t, strconv.Itoa(os.Getpid()), globalTags["process_pid"])

	// Check gamelift_process_id from environment.
	common.AssertEqual(t, "test-process-id", globalTags["gamelift_process_id"])
}

// TestGlobalProcessorOnGameSessionStarted tests that OnGameSessionStarted works on global processor.
func TestGlobalProcessorOnGameSessionStarted(t *testing.T) {
	resetGlobalProcessor()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Initialize processor.
	err := InitIsolatedTestProcessor(t, WithTransport(NewMockTransport(ctrl)))
	common.AssertEqual(t, nil, err)

	// Get global processor and call OnGameSessionStarted.
	globalProc := GetGlobalProcessor()
	if globalProc == nil {
		t.Fatal("Expected globalProc to not be nil")
	}

	// Call OnGameSessionStarted.
	sessionID := "test-session-123"
	globalProc.OnGameSessionStarted(sessionID)

	// Verify session_id was set as global tag.
	globalTags := globalProc.GetGlobalTags()
	common.AssertEqual(t, sessionID, globalTags["session_id"])
}

func TestGlobalOnGameSessionStarted(t *testing.T) {
	resetGlobalProcessor()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	err := InitIsolatedTestProcessor(t, WithTransport(NewMockTransport(ctrl)))
	common.AssertEqual(t, nil, err)

	// Test calling global OnGameSessionStarted.
	session := model.GameSession{
		GameSessionID: "test-session-456",
		FleetID:       "test-fleet",
	}

	OnGameSessionStarted(session)

	// Verify session_id was set in global processor.
	globalProc := GetGlobalProcessor()
	if globalProc == nil {
		t.Fatal("Expected globalProc to not be nil")
	}

	globalTags := globalProc.GetGlobalTags()
	common.AssertEqual(t, session.GameSessionID, globalTags["session_id"])
}

// TestGlobalOnGameSessionStartedNoProcessor tests the global function when no processor exists.
func TestGlobalOnGameSessionStartedNoProcessor(_ *testing.T) {
	resetGlobalProcessor()

	// Test calling global OnGameSessionStarted without a processor.
	session := model.GameSession{
		GameSessionID: "test-session-789",
	}

	// Should not panic when no processor exists.
	OnGameSessionStarted(session)
}

func TestGlobalIntegrationWithDefaults(t *testing.T) {
	resetGlobalProcessor()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Set environment variables to test default tags.
	t.Setenv(common.EnvironmentKeyProcessID, "integration-process-id")
	// Initialize processor with some custom tags (gets default tags automatically).
	err := InitIsolatedTestProcessor(t,
		WithTransport(NewMockTransport(ctrl)),
		WithGlobalTag("service", "test-service"),
	)
	common.AssertEqual(t, nil, err)

	processor := GetGlobalProcessor()
	if processor == nil {
		t.Fatal("Expected processor to not be nil")
	}

	// Verify both default and custom tags are set.
	globalTags := processor.GetGlobalTags()

	// Should have process_id (always set).
	_, hasProcessPid := globalTags["process_pid"]
	common.AssertEqual(t, true, hasProcessPid)

	// Should have gamelift_process_id from env var.
	common.AssertEqual(t, "integration-process-id", globalTags["gamelift_process_id"])

	// Should have custom tag.
	common.AssertEqual(t, "test-service", globalTags["service"])

	// Should NOT have session_id yet.
	_, hasSessionID := globalTags["session_id"]
	common.AssertEqual(t, false, hasSessionID)

	// Simulate game session start (what user would do in their callback).
	session := model.GameSession{
		GameSessionID: "end-to-end-session-123",
	}

	OnGameSessionStarted(session)

	// Verify session_id is now set.
	updatedTags := processor.GetGlobalTags()
	common.AssertEqual(t, "end-to-end-session-123", updatedTags["session_id"])

	// Verify all other tags are still present.
	common.AssertEqual(t, "integration-process-id", updatedTags["gamelift_process_id"])
	common.AssertEqual(t, "test-service", updatedTags["service"])
	_, hasProcessPidFinal := updatedTags["process_pid"]
	common.AssertEqual(t, true, hasProcessPidFinal)
}
