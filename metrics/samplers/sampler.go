/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package samplers

import (
	"math/rand"
	"sync"
	"time"
)

// Sampler controls whether a metric should be sampled.
type Sampler interface {
	// ShouldSample returns true if the metric should be recorded
	ShouldSample() bool
}

// SampleRateProvider provides the sample rate for samplers that support it.
type SampleRateProvider interface {
	// GetSampleRate returns the sample rate (0.0 to 1.0)
	GetSampleRate() float64
}

var _ Sampler = (*AllSampler)(nil)
var _ SampleRateProvider = (*AllSampler)(nil)

// AllSampler samples every metric.
type AllSampler struct{}

// NewAll creates a sampler that samples everything.
func NewAll() *AllSampler {
	return &AllSampler{}
}

// ShouldSample always returns true.
func (s *AllSampler) ShouldSample() bool {
	return true
}

// GetSampleRate returns 1.0 (100% sampling).
func (s *AllSampler) GetSampleRate() float64 {
	return 1.0
}

var _ Sampler = (*NoneSampler)(nil)
var _ SampleRateProvider = (*NoneSampler)(nil)

// NoneSampler samples no metrics.
type NoneSampler struct{}

// NewNone creates a sampler that samples nothing.
func NewNone() *NoneSampler {
	return &NoneSampler{}
}

// ShouldSample always returns false.
func (s *NoneSampler) ShouldSample() bool {
	return false
}

// GetSampleRate returns 0.0 (0% sampling).
func (s *NoneSampler) GetSampleRate() float64 {
	return 0.0
}

var _ Sampler = (*FractionSampler)(nil)
var _ SampleRateProvider = (*FractionSampler)(nil)

// FractionSampler samples a fraction of metrics.
type FractionSampler struct {
	random *rand.Rand
	rate   float64
	mutex  sync.Mutex
}

// NewFraction creates a sampler that samples a fraction of metrics.
// The rate should be between 0.0 and 1.0, values outside this range will be
// clamped.
func NewFraction(rate float64) *FractionSampler {
	if rate < 0.0 {
		rate = 0.0
	}
	if rate > 1.0 {
		rate = 1.0
	}

	return &FractionSampler{
		rate: rate,
		// Using math/rand for performance in metrics sampling, cryptographic randomness not required.
		random: rand.New(rand.NewSource(time.Now().UnixNano())), //nolint:gosec
	}
}

// ShouldSample returns true based on the configured rate.
func (s *FractionSampler) ShouldSample() bool {
	if s.rate == 0.0 {
		return false
	}
	if s.rate == 1.0 {
		return true
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.random.Float64() < s.rate
}

// GetSampleRate returns the configured sample rate.
func (s *FractionSampler) GetSampleRate() float64 {
	return s.rate
}
