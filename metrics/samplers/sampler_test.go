/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package samplers

import (
	"testing"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
)

func TestAllSampler(t *testing.T) {
	sampler := NewAll()

	// The "All" sampler always returns true, unlike others which can vary.
	common.AssertEqual(t, true, sampler.ShouldSample())
	common.AssertEqual(t, true, sampler.ShouldSample())
	common.AssertEqual(t, true, sampler.ShouldSample())

	common.AssertEqual(t, 1.0, sampler.GetSampleRate())
}

func TestNoneSampler(t *testing.T) {
	sampler := NewNone()

	// The "None" sampler always returns false, unlike others which can vary.
	common.AssertEqual(t, false, sampler.ShouldSample())
	common.AssertEqual(t, false, sampler.ShouldSample())
	common.AssertEqual(t, false, sampler.ShouldSample())

	common.AssertEqual(t, 0.0, sampler.GetSampleRate())
}

func TestFractionSampler_GetSampleRate(t *testing.T) {
	tests := map[string]struct {
		rate         float64
		expectedRate float64
	}{
		"negative rate clamped": {
			rate:         -0.5,
			expectedRate: 0.0,
		},
		"rate greater than 1.0 clamped": {
			rate:         1.5,
			expectedRate: 1.0,
		},
		"valid rate": {
			rate:         0.7,
			expectedRate: 0.7,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			sampler := NewFraction(test.rate)
			common.AssertEqual(t, test.expectedRate, sampler.GetSampleRate())
		})
	}
}

func TestFractionSampler_ShouldSample(t *testing.T) {
	t.Run("rate 0.0 always false", func(t *testing.T) {
		sampler := NewFraction(0.0)

		common.AssertEqual(t, false, sampler.ShouldSample())
		common.AssertEqual(t, false, sampler.ShouldSample())
		common.AssertEqual(t, false, sampler.ShouldSample())
	})

	t.Run("rate 1.0 always true", func(t *testing.T) {
		sampler := NewFraction(1.0)

		common.AssertEqual(t, true, sampler.ShouldSample())
		common.AssertEqual(t, true, sampler.ShouldSample())
		common.AssertEqual(t, true, sampler.ShouldSample())
	})

	t.Run("fractional rate distribution", func(t *testing.T) {
		sampler := NewFraction(0.5)
		var samples int
		iterations := 1000

		for i := 0; i < iterations; i++ {
			if sampler.ShouldSample() {
				samples++
			}
		}

		// With a large sample size, we should get close to 50% true results
		// Allow for some variance (40% to 60%)
		ratio := float64(samples) / float64(iterations)
		if ratio < 0.4 || ratio > 0.6 {
			t.Errorf("Expected ratio around 0.5, got %f (samples: %d, iterations: %d)",
				ratio, samples, iterations)
		}
	})

	t.Run("thread-safe", func(_ *testing.T) {
		sampler := NewFraction(0.5)
		done := make(chan bool, 10)

		// Start multiple goroutines calling ShouldSample concurrently. Should
		// not fail when tests are run with the race detector enabled.
		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 100; j++ {
					sampler.ShouldSample()
				}
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
