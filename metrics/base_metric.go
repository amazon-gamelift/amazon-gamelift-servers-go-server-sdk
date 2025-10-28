/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package metrics

import (
	"sync"
	"time"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/model"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/samplers"
)

// baseMetric contains the common functionality shared by all metric types.
type baseMetric struct {
	sampler           samplers.Sampler
	processor         MetricsProcessor
	tags              map[string]string
	key               string
	derivedMetrics    []model.DerivedMetric
	metricType        model.MetricType
	currentValue      float64
	tagsMutex         sync.RWMutex
	currentValueMutex sync.RWMutex
}

// newBaseMetric creates a new base metric with common functionality.
func newBaseMetric(key string, metricType model.MetricType, tags map[string]string, derivedMetrics []model.DerivedMetric, sampler samplers.Sampler, processor MetricsProcessor) *baseMetric {
	if sampler == nil {
		sampler = samplers.NewAll()
	}

	// Take a deep copy of tags to avoid shared state issues in the case where
	// two different metrics are built from the same base.
	baseTags := make(map[string]string)
	for k, v := range tags {
		baseTags[k] = v
	}

	return &baseMetric{
		key:            key,
		metricType:     metricType,
		tags:           baseTags,
		derivedMetrics: derivedMetrics,
		sampler:        sampler,
		processor:      processor,
	}
}

// Key returns the metric key.
func (b *baseMetric) Key() string {
	return b.key
}

// MetricType returns the metric type.
func (b *baseMetric) MetricType() model.MetricType {
	return b.metricType
}

// DerivedMetrics returns the derived metrics.
func (b *baseMetric) DerivedMetrics() []model.DerivedMetric {
	return b.derivedMetrics
}

// CurrentValue returns the current value of the metric.
func (b *baseMetric) CurrentValue() float64 {
	b.currentValueMutex.Lock()
	defer b.currentValueMutex.Unlock()

	return b.currentValue
}

// updateCurrentValue updates a value based on the passed in operation.
func (b *baseMetric) updateCurrentValue(value float64, operation model.MetricOperation) {
	if b.sampler != nil && !b.sampler.ShouldSample() {
		return
	}

	b.currentValueMutex.Lock()
	defer b.currentValueMutex.Unlock()

	switch operation {
	case model.MetricOperationSet:
		b.currentValue = value
	case model.MetricOperationAdjust:
		b.currentValue += value
	}
}

// Tags returns a copy of the tags.
func (b *baseMetric) Tags() map[string]string {
	b.tagsMutex.RLock()
	defer b.tagsMutex.RUnlock()

	result := make(map[string]string, len(b.tags))
	for k, v := range b.tags {
		result[k] = v
	}
	return result
}

// SetTag sets a tag.
func (b *baseMetric) SetTag(key, value string) error {
	return b.SetTags(map[string]string{key: value})
}

// SetTags sets multiple tags on this metric.
func (b *baseMetric) SetTags(tags map[string]string) error {
	b.tagsMutex.Lock()
	defer b.tagsMutex.Unlock()

	for key, value := range tags {
		if err := model.ValidateTagKey(key); err != nil {
			return err //nolint:wrapcheck
		}
		if err := model.ValidateTagValue(value); err != nil {
			return err //nolint:wrapcheck
		}

		b.tags[key] = value
	}

	return nil
}

// RemoveTag removes a tag.
func (b *baseMetric) RemoveTag(key string) {
	b.tagsMutex.Lock()
	defer b.tagsMutex.Unlock()
	delete(b.tags, key)
}

// enqueueMessage creates and enqueues a metric message.
func (b *baseMetric) enqueueMessage(value float64) {
	if b.sampler != nil && !b.sampler.ShouldSample() {
		return
	}

	if b.processor != nil {
		message := model.MetricMessage{
			Key:        b.key,
			Type:       b.metricType,
			Value:      value,
			Tags:       b.Tags(),
			SampleRate: b.getSampleRate(),
			Timestamp:  time.Now(),
		}

		// Handle derived metrics immediately at the metric level
		// This works for both base metrics and WithTags variants since they share derivedMetrics
		for _, derived := range b.derivedMetrics {
			derived.HandleMessage(message)
		}

		b.processor.enqueueMetric(message)
	}
}

// getSampleRate returns the sample rate from the sampler.
func (b *baseMetric) getSampleRate() float64 {
	if sampler, ok := b.sampler.(samplers.SampleRateProvider); ok {
		return sampler.GetSampleRate()
	}
	return 1.0
}

// baseBuilder contains common builder functionality.
type baseBuilder struct {
	sampler        samplers.Sampler
	processor      MetricsProcessor
	tags           map[string]string
	key            string
	derivedMetrics []model.DerivedMetric
}

// newBaseBuilder creates a new base builder.
func newBaseBuilder(key string) *baseBuilder {
	return &baseBuilder{
		key:            key,
		tags:           make(map[string]string),
		derivedMetrics: make([]model.DerivedMetric, 0),
	}
}

// withTag adds a single tag to the builder.
func (b *baseBuilder) withTag(key, value string) {
	b.tags[key] = value
}

// withSampler sets the sampler for the builder.
func (b *baseBuilder) withSampler(sampler samplers.Sampler) {
	b.sampler = sampler
}

// withDerivedMetrics adds derived metrics to the builder.
func (b *baseBuilder) withDerivedMetrics(derived ...model.DerivedMetric) {
	b.derivedMetrics = append(b.derivedMetrics, derived...)
}

// withMetricsProcessor sets the MetricsProcessor for the builder.
func (b *baseBuilder) withMetricsProcessor(processor MetricsProcessor) {
	b.processor = processor
}

// withTags adds tags to the builder.
func (b *baseBuilder) withTags(tags map[string]string) {
	for k, v := range tags {
		b.tags[k] = v
	}
}

// getTags returns a copy of the tags.
func (b *baseBuilder) getTags() map[string]string {
	copiedTags := make(map[string]string)
	for k, v := range b.tags {
		copiedTags[k] = v
	}
	return copiedTags
}

// cloneDerivedMetrics creates a safe copy of derived metrics for dimensional variants.
func cloneDerivedMetrics(derivedMetrics []model.DerivedMetric) []model.DerivedMetric {
	if len(derivedMetrics) == 0 {
		return nil
	}

	clonedMetrics := make([]model.DerivedMetric, len(derivedMetrics))
	for i, derived := range derivedMetrics {
		clonedMetrics[i] = cloneDerivedMetric(derived)
	}
	return clonedMetrics
}

// cloneDerivedMetric creates a safe copy of a single derived metric for dimensional variants.
func cloneDerivedMetric(derived model.DerivedMetric) model.DerivedMetric { //nolint:ireturn
	cloner, ok := derived.(model.DerivedMetricCloner)
	if !ok {
		return derived
	}

	cloned := cloner.Clone()
	if cloned == nil {
		return derived
	}

	return cloned
}

// withTags contains the common WithTags implementation logic shared by all metric types.
// Returns a new baseMetric if a new instance was created, nil if the existing instance was modified.
func (b *baseMetric) withTags(tags map[string]string) *baseMetric {
	if len(tags) == 0 {
		return nil
	}

	// Handle non-dimensional case (update existing instance)
	if b.processor != nil && !b.processor.dimensionalMetricsEnabled() {
		for key, value := range tags {
			err := b.SetTag(key, value)
			if err != nil && b.processor.getLogger() != nil {
				b.processor.getLogger().Errorf("Failed to set tag %s=%s: %v", key, value, err)
			}
		}
		return nil // no new instance created
	}

	// Handle dimensional case (create new instance)
	newTags := copyTags(b.tags)
	for key, value := range tags {
		newTags[key] = value
	}

	clonedDerivedMetrics := cloneDerivedMetrics(b.derivedMetrics)

	newBase := &baseMetric{
		key:            b.key,
		metricType:     b.metricType,
		tags:           newTags,
		derivedMetrics: clonedDerivedMetrics,
		sampler:        b.sampler,
		processor:      b.processor,
		currentValue:   0,
	}
	b.processor.registerMetric(newBase)
	return newBase // new instance created
}

// createDimensionalMetric creates a temporary metric instance with dimensional tags (not registered).
func (b *baseMetric) createDimensionalMetric(dimensionalTags map[string]string) *baseMetric {
	if len(dimensionalTags) == 0 {
		return b
	}

	combinedTags := make(map[string]string, len(b.tags)+len(dimensionalTags))
	for k, v := range b.tags {
		combinedTags[k] = v
	}
	for k, v := range dimensionalTags {
		combinedTags[k] = v
	}

	clonedDerivedMetrics := cloneDerivedMetrics(b.derivedMetrics)
	return &baseMetric{
		key:            b.key,
		metricType:     b.metricType,
		tags:           combinedTags,
		derivedMetrics: clonedDerivedMetrics,
		sampler:        b.sampler,
		processor:      b.processor,
		currentValue:   0,
	}
}
