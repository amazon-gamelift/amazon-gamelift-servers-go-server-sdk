/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package metrics

import (
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/model"
)

// testDerivedMetric implements DerivedMetric for testing.
type testDerivedMetric struct {
	key string
}

func (m *testDerivedMetric) Key() string                                      { return m.key }
func (m *testDerivedMetric) HandleMessage(_ model.MetricMessage)              {}
func (m *testDerivedMetric) EmitMetrics(_ model.Metric) []model.MetricMessage { return nil }
func (m *testDerivedMetric) Reset()                                           {}

// testSampler implements Sampler for testing.
type testSampler struct {
	shouldSample bool
}

func (m *testSampler) ShouldSample() bool {
	return m.shouldSample
}

// testPatternSampler implements Sampler for testing fractional behavior deterministically.
type testPatternSampler struct {
	pattern []bool
	index   int
}

func (m *testPatternSampler) ShouldSample() bool {
	if len(m.pattern) == 0 {
		return false
	}
	result := m.pattern[m.index%len(m.pattern)]
	m.index++
	return result
}

// newTestPatternSampler creates a mock sampler with a deterministic pattern.
func newTestPatternSampler(pattern []bool) *testPatternSampler {
	return &testPatternSampler{pattern: pattern}
}
