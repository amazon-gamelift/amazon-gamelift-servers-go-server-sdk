/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package metrics

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/model"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server/log"
)

const (
	// DefaultBufferSize is the default size for the internal message buffer.
	DefaultBufferSize = 10000
	// DefaultIngressChannelSize is the default size for the ingress channel.
	DefaultIngressChannelSize = 4096
	// DefaultProcessorInterval is the default interval for processing metrics.
	DefaultProcessorInterval = 10 * time.Second
	// DefaultMaxWorkers is the default maximum number of worker routines.
	DefaultMaxWorkers = 10
	// EnableDimensionalMetricsEnvVar is the environment variable to enable dimensional metrics.
	EnableDimensionalMetricsEnvVar = "GAMELIFT_ENABLE_DIMENSIONAL_METRICS"
)

// MetricsProcessor is the main interface for processing metrics.
//
//go:generate mockgen -destination ./processor_mock_test.go -package=metrics . MetricsProcessor
type MetricsProcessor interface { //nolint:revive,interfacebloat
	// SetGlobalTag sets a global tag that will be applied to all metrics
	SetGlobalTag(key, value string) error
	// RemoveGlobalTag removes a global tag
	RemoveGlobalTag(key string)
	// GetGlobalTags returns all current global tags
	GetGlobalTags() map[string]string
	// Start starts the metrics processor
	Start(ctx context.Context) error
	// Stop stops the metrics processor
	Stop() error
	// GetMetric retrieves a registered metric by key
	GetMetric(key string) (model.Metric, bool)
	// ListMetrics returns all registered metrics
	ListMetrics() []model.Metric
	// UnregisterMetric unregisters a metric from the processor
	UnregisterMetric(key string)
	// OnGameSessionStarted sets the session_id global tag when a game session starts
	OnGameSessionStarted(sessionID string)
	// enqueueMetric adds a metric message to the processing queue
	enqueueMetric(message model.MetricMessage)
	// registerMetric registers a metric with the processor
	registerMetric(metric model.Metric)
	// dimensionalMetricsEnabled returns whether dimensional metrics are enabled
	dimensionalMetricsEnabled() bool
	// getLogger returns the processor's logger for internal use
	getLogger() log.ILogger
}

// ProcessorConfig contains configuration for the metrics processor.
type ProcessorConfig struct {
	Transport                Transport
	GlobalTags               map[string]string
	ProcessInterval          time.Duration
	BufferSize               int
	IngressChannelSize       int
	MaxNumWorkers            int
	EnableDerivedMetrics     bool
	EnableDimensionalMetrics bool
}

// DefaultProcessorConfig returns a default processor configuration.
func DefaultProcessorConfig() (*ProcessorConfig, error) {
	transport, err := NewStatsDTransport().Build()
	if err != nil {
		return nil, common.NewGameLiftError(common.MetricConfigurationException, "", fmt.Sprintf("failed to create default transport: %v", err)) //
	}

	return &ProcessorConfig{
		ProcessInterval:          DefaultProcessorInterval,
		EnableDerivedMetrics:     true,
		EnableDimensionalMetrics: DimensionalMetricsEnabled(),
		GlobalTags:               make(map[string]string),
		BufferSize:               DefaultBufferSize,
		IngressChannelSize:       DefaultIngressChannelSize,
		Transport:                transport,
		MaxNumWorkers:            DefaultMaxWorkers,
	}, nil
}

// ProcessorOption is a functional option for configuring the metrics processor.
type ProcessorOption func(*ProcessorConfig) error

// WithTransport sets the transport for the processor.
func WithTransport(transport Transport) ProcessorOption {
	return func(cfg *ProcessorConfig) error {
		if transport == nil {
			return common.NewGameLiftError(common.MetricConfigurationException, "", "transport cannot be nil")
		}
		cfg.Transport = transport
		return nil
	}
}

// WithProcessInterval sets how often metrics are processed and flushed.
func WithProcessInterval(interval time.Duration) ProcessorOption {
	return func(cfg *ProcessorConfig) error {
		if interval <= 0 {
			return common.NewGameLiftError(common.MetricConfigurationException, "", "process interval must be positive")
		}
		cfg.ProcessInterval = interval
		return nil
	}
}

// WithEnableDerivedMetrics controls whether derived metrics are computed.
func WithEnableDerivedMetrics(enable bool) ProcessorOption {
	return func(cfg *ProcessorConfig) error {
		cfg.EnableDerivedMetrics = enable
		return nil
	}
}

// WithEnableDimensionalMetrics controls whether tags create unique metric instances.
func WithEnableDimensionalMetrics(enable bool) ProcessorOption {
	return func(cfg *ProcessorConfig) error {
		cfg.EnableDimensionalMetrics = enable
		return nil
	}
}

// WithGlobalTag adds a single global tag to the processor.
func WithGlobalTag(key, value string) ProcessorOption {
	return func(cfg *ProcessorConfig) error {
		if key == "" {
			return common.NewGameLiftError(common.ValidationException, "", "tag key cannot be empty")
		}
		if cfg.GlobalTags == nil {
			cfg.GlobalTags = make(map[string]string)
		}
		cfg.GlobalTags[key] = value
		return nil
	}
}

// WithGlobalTags sets multiple global tags at once.
func WithGlobalTags(tags map[string]string) ProcessorOption {
	return func(cfg *ProcessorConfig) error {
		if cfg.GlobalTags == nil {
			cfg.GlobalTags = make(map[string]string)
		}
		for k, v := range tags {
			if k == "" {
				return common.NewGameLiftError(common.ValidationException, "", "tag key cannot be empty")
			}
			cfg.GlobalTags[k] = v
		}
		return nil
	}
}

// WithBufferSize sets the size of the internal message buffer.
func WithBufferSize(size int) ProcessorOption {
	return func(cfg *ProcessorConfig) error {
		if size <= 0 {
			return common.NewGameLiftError(common.MetricConfigurationException, "", "buffer size must be positive")
		}
		cfg.BufferSize = size
		return nil
	}
}

// WithIngressChannelSize sets the size of the ingress channel for incoming metrics.
func WithIngressChannelSize(size int) ProcessorOption {
	return func(cfg *ProcessorConfig) error {
		if size <= 0 {
			return common.NewGameLiftError(common.MetricConfigurationException, "", "ingress channel size must be positive")
		}
		cfg.IngressChannelSize = size
		return nil
	}
}

// WithMaxWorkers sets the maximum number of worker routines for processing metrics.
func WithMaxWorkers(maxWorkers int) ProcessorOption {
	return func(cfg *ProcessorConfig) error {
		if maxWorkers <= 0 {
			return common.NewGameLiftError(common.MetricConfigurationException, "", "max workers must be positive")
		}
		cfg.MaxNumWorkers = maxWorkers
		return nil
	}
}

// InitMetricsProcessor initializes the global metrics processor with functional options.
// Subsequent calls after the first init will return an error.
func InitMetricsProcessor(options ...ProcessorOption) error {
	// Check if already initialized before doing any work
	if HasGlobalProcessor() {
		return common.NewGameLiftError(common.MetricConfigurationException, "", "metrics processor already initialized")
	}

	// Start with default configuration
	config := &ProcessorConfig{
		ProcessInterval:          DefaultProcessorInterval,
		EnableDerivedMetrics:     true,
		EnableDimensionalMetrics: DimensionalMetricsEnabled(),
		GlobalTags:               make(map[string]string),
		BufferSize:               DefaultBufferSize,
		IngressChannelSize:       DefaultIngressChannelSize,
		MaxNumWorkers:            DefaultMaxWorkers,
	}

	// Apply all functional options
	for _, option := range options {
		if err := option(config); err != nil {
			return common.NewGameLiftError(common.MetricConfigurationException, "", fmt.Sprintf("failed to apply option: %v", err))
		}
	}

	// Validate required fields
	if config.Transport == nil {
		return common.NewGameLiftError(common.MetricConfigurationException, "", "transport is required")
	}

	// Use sync.Once to ensure only one processor is created even under concurrent access
	// The once variable is defined in global.go
	var initErr error
	var didInitialize bool

	once.Do(func() {
		didInitialize = true
		// Deep copy global tags to avoid shared state
		globalTags := make(map[string]string)
		for k, v := range config.GlobalTags {
			globalTags[k] = v
		}

		processor := &Processor{
			GlobalTags:               globalTags,
			MetricMap:                make(map[string]model.Metric),
			Transport:                config.Transport,
			enableDimensionalMetrics: config.EnableDimensionalMetrics,
			interval:                 config.ProcessInterval,
			messageQueue:             make(chan model.MetricMessage, config.BufferSize),
			ingressChannel:           make(chan model.MetricMessage, config.IngressChannelSize),
			logger:                   log.GetDefaultLogger(common.GetEnvStringOrDefault(common.EnvironmentKeyProcessID, "metrics_processor")),
			maxWorkers:               config.MaxNumWorkers,
		}

		// Initialize default global tags
		processor.setDefaultGlobalTags()

		// Set as the global processor (defined in global.go)
		SetGlobalProcessor(processor)
	})

	if !didInitialize {
		return common.NewGameLiftError(common.MetricConfigurationException, "", "metrics processor already initialized")
	}

	return initErr
}

// MustInitMetricsProcessor initializes the global metrics processor with functional options.
// Panics if initialization fails.
func MustInitMetricsProcessor(options ...ProcessorOption) {
	if err := InitMetricsProcessor(options...); err != nil {
		panic(fmt.Sprintf("failed to initialize metrics processor: %v", err))
	}
}

// DimensionalMetricsEnabled reads the environment variable to determine if dimensional metrics are enabled.
func DimensionalMetricsEnabled() bool {
	envValue := os.Getenv(EnableDimensionalMetricsEnvVar)
	if envValue == "" {
		return false // Default to disabled for backward compatibility
	}
	enabled, err := strconv.ParseBool(envValue)
	if err != nil {
		return false // Default to disabled if invalid value
	}
	return enabled
}

// Processor implements MetricsProcessor, ingesting metrics in a non-blocking manner.
type Processor struct {
	Transport                Transport
	logger                   log.ILogger
	messageQueue             chan model.MetricMessage
	MetricMap                map[string]model.Metric
	ingressChannel           chan model.MetricMessage
	GlobalTags               map[string]string
	shutdown                 chan struct{}
	serverUpGauge            *Gauge
	wg                       sync.WaitGroup
	interval                 time.Duration
	maxWorkers               int
	mtx                      sync.Mutex
	enableDimensionalMetrics bool
	started                  bool
}

// Public interface methods.

// SetGlobalTag sets a global tag that will be applied to all metrics.
func (p *Processor) SetGlobalTag(key, value string) error {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	if err := model.ValidateTagKey(key); err != nil {
		p.logger.Errorf("failed to validate tag key: %v", err)
		return err //nolint:wrapcheck
	}
	if err := model.ValidateTagValue(value); err != nil {
		p.logger.Errorf("failed to validate tag value: %v", err)
		return err //nolint:wrapcheck
	}

	p.GlobalTags[key] = value
	return nil
}

// RemoveGlobalTag removes a global tag.
func (p *Processor) RemoveGlobalTag(key string) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	delete(p.GlobalTags, key)
}

// GetGlobalTags returns all current global tags.
func (p *Processor) GetGlobalTags() map[string]string {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	// Return a copy to prevent external mutation
	tags := make(map[string]string)
	for k, v := range p.GlobalTags {
		tags[k] = v
	}
	return tags
}

// Start starts the metrics processor.
func (p *Processor) Start(ctx context.Context) error {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	if p.started {
		return common.NewGameLiftError(common.MetricConfigurationException, "", "processor already started")
	}

	p.shutdown = make(chan struct{})

	p.wg.Add(1)
	go p.run(ctx)

	p.started = true

	if p.serverUpGauge == nil {
		// up=1 will be sent automatically on every flush cycle
		// we pass nil for tags since server_up always uses fresh GlobalTags on each send
		baseMetric := newBaseMetric("up", model.MetricTypeGauge, nil, nil, nil, p)
		p.serverUpGauge = &Gauge{baseMetric: baseMetric}

		p.MetricMap["up"] = baseMetric
	}
	return nil
}

// Stop stops the metrics processor.
func (p *Processor) Stop() error {
	p.mtx.Lock()

	if !p.started {
		p.mtx.Unlock()
		return nil
	}

	// Send up=0 synchronously before any shutdown activities
	if p.serverUpGauge != nil {
		// Copy tags while holding lock
		globalTags := make(map[string]string)
		for k, v := range p.GlobalTags {
			globalTags[k] = v
		}

		finalMessage := []model.MetricMessage{{
			Key:        "up",
			Type:       model.MetricTypeGauge,
			Value:      0,
			Tags:       globalTags,
			SampleRate: 1.0,
			Timestamp:  time.Now(),
		}}

		// Send synchronously directly to transport
		// This ensures it's sent before we close anything
		if err := p.Transport.Send(finalMessage); err != nil {
			p.logger.Errorf("failed to send up=0 on shutdown: %v", err)
		}
	}

	close(p.shutdown)
	close(p.ingressChannel)

	p.mtx.Unlock()
	p.wg.Wait()
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.messageQueue = make(chan model.MetricMessage, cap(p.messageQueue))
	p.ingressChannel = make(chan model.MetricMessage, cap(p.ingressChannel))

	err := p.Transport.Close()
	if err != nil {
		p.logger.Errorf("failed to close transport during stop: %v", err)
	}
	p.started = false
	return nil
}

// GetMetric retrieves a registered metric by key.
func (p *Processor) GetMetric(key string) (model.Metric, bool) { //nolint:ireturn
	p.mtx.Lock()
	defer p.mtx.Unlock()

	metric, exists := p.MetricMap[key]
	return metric, exists
}

// ListMetrics returns all registered metrics.
func (p *Processor) ListMetrics() []model.Metric {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	metrics := make([]model.Metric, 0, len(p.MetricMap))
	for _, metric := range p.MetricMap {
		metrics = append(metrics, metric)
	}

	return metrics
}

// UnregisterMetric unregisters a metric from the processor.
func (p *Processor) UnregisterMetric(key string) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	delete(p.MetricMap, key)
}

// enqueueMetric adds a metric message to the processing queue.
func (p *Processor) enqueueMetric(message model.MetricMessage) {
	p.ingressChannel <- message
}

// registerMetric registers a metric with the processor.
func (p *Processor) registerMetric(metric model.Metric) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	compositeKey := p.createCompositeKey(metric.Key(), metric.Tags())
	if _, exists := p.MetricMap[compositeKey]; exists {
		return
	}

	p.MetricMap[compositeKey] = metric
}

// dimensionalMetricsEnabled returns whether dimensional metrics are enabled.
func (p *Processor) dimensionalMetricsEnabled() bool {
	return p.enableDimensionalMetrics
}

// getLogger returns the processor's logger for internal use by metrics.
func (p *Processor) getLogger() log.ILogger { //nolint:ireturn
	return p.logger
}

func (p *Processor) run(ctx context.Context) {
	defer p.wg.Done()

	p.wg.Add(p.maxWorkers)
	for i := 0; i < p.maxWorkers; i++ {
		go p.processWorker(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			close(p.messageQueue)
			return

		case <-p.shutdown:
			close(p.messageQueue)
			return

		case message, ok := <-p.ingressChannel:
			if !ok {
				close(p.messageQueue)
				return
			}

			select {
			case p.messageQueue <- message:
			case <-ctx.Done():
				close(p.messageQueue)
				return
			case <-p.shutdown:
				close(p.messageQueue)
				return
			default:
			}
		}
	}
}

func (p *Processor) processWorker(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	messages := make([]model.MetricMessage, 0, 1024)

	for {
		select {
		case <-ctx.Done():
			if err := p.flushMessages(messages); err != nil {
				p.logger.Errorf("failed to flush messages on shutdown: %v", err)
			}
			return

		case <-p.shutdown:
			if err := p.flushMessages(messages); err != nil {
				p.logger.Errorf("failed to flush messages on shutdown: %v", err)
			}
			return

		case <-ticker.C:
			// Always send up heartbeat on every tick
			p.sendServerUpHeartbeat()

			if err := p.flushMessages(messages); err != nil {
				p.logger.Errorf("failed to flush messages on interval: %v", err)
			}
			messages = messages[:0]

		case message, ok := <-p.messageQueue:
			if !ok {
				if err := p.flushMessages(messages); err != nil {
					p.logger.Errorf("failed to flush messages on channel close: %v", err)
				}
				return
			}

			if len(messages) >= cap(messages) {
				if err := p.flushMessages(messages); err != nil {
					p.logger.Errorf("failed to flush messages on overflow: %v", err)
				}
				messages = messages[:0]
			}

			messages = append(messages, message)
		}
	}
}

func (p *Processor) sendServerUpHeartbeat() {
	p.mtx.Lock()
	if !p.started || p.serverUpGauge == nil {
		p.mtx.Unlock()
		return
	}
	globalTags := make(map[string]string)
	for k, v := range p.GlobalTags {
		globalTags[k] = v
	}
	p.mtx.Unlock()

	heartbeat := []model.MetricMessage{{
		Key:        "up",
		Type:       model.MetricTypeGauge,
		Value:      1,
		Tags:       globalTags,
		SampleRate: 1.0,
		Timestamp:  time.Now(),
	}}

	if err := p.Transport.Send(heartbeat); err != nil {
		p.logger.Errorf("failed to send up heartbeat: %v", err)
	}
}

func (p *Processor) flushMessages(messages []model.MetricMessage) error {
	if len(messages) == 0 {
		return nil // nothing to flush
	}

	globalTags := p.GetGlobalTags()

	derivedMessages, err := p.emitDerivedMetrics()
	if err != nil {
		p.logger.Errorf("failed to emit derived metrics: %v", err)
	} else {
		messages = append(messages, derivedMessages...)
	}

	for _, message := range messages {
		message.Tags = p.mergeTags(message.Tags, globalTags)
	}
	// Transport errors are handled with context by all callers.
	if err := p.Transport.Send(messages); err != nil {
		return err //nolint:wrapcheck
	}

	return nil
}

func (p *Processor) mergeTags(perMetricTags map[string]string, globalTags map[string]string) map[string]string {
	if perMetricTags == nil {
		perMetricTags = make(map[string]string)
	}

	// global tags overwrite any conflicting per-metric tags
	for k, v := range globalTags {
		perMetricTags[k] = v
	}

	return perMetricTags
}

func (p *Processor) emitDerivedMetrics() ([]model.MetricMessage, error) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	var allDerivedMessages []model.MetricMessage

	for _, metric := range p.MetricMap {
		derivedMetrics := metric.DerivedMetrics()
		if len(derivedMetrics) == 0 {
			continue
		}

		var derivedMessages []model.MetricMessage
		for _, derived := range derivedMetrics {
			messages := derived.EmitMetrics(metric)
			sourceTags := metric.Tags()
			for idx := range messages {
				messages[idx].Tags = copyTags(sourceTags)
			}
			derivedMessages = append(derivedMessages, messages...)
			derived.Reset()
		}

		if len(derivedMessages) > 0 {
			allDerivedMessages = append(allDerivedMessages, derivedMessages...)
		}
	}

	return allDerivedMessages, nil
}

func (p *Processor) createCompositeKey(metricKey string, tags map[string]string) string {
	if !p.enableDimensionalMetrics {
		// When dimensional metrics are disabled, ignore tags for metric identity
		return metricKey
	}

	tagsKey := createTagsKey(tags)
	if tagsKey == "" {
		return metricKey
	}
	return metricKey + "|" + tagsKey
}

func createTagsKey(tags map[string]string) string {
	if len(tags) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(tags))

	for k, v := range tags {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ",")
}

func copyTags(tags map[string]string) map[string]string {
	if tags == nil {
		return nil
	}
	copied := make(map[string]string, len(tags))
	for k, v := range tags {
		copied[k] = v
	}
	return copied
}

var _ MetricsProcessor = (*Processor)(nil)

// Global convenience functions that operate on the global processor.
// These provide a simpler API for common operations without needing a processor reference.

// SetGlobalTag adds a global tag to the global processor.
func SetGlobalTag(key, value string) error {
	processor := GetGlobalProcessor()
	if processor == nil {
		return common.NewGameLiftError(common.MetricConfigurationException, "", "metrics processor not initialized")
	}
	return processor.SetGlobalTag(key, value)
}

// RemoveGlobalTag removes a global tag from the global processor.
func RemoveGlobalTag(key string) {
	processor := GetGlobalProcessor()
	if processor != nil {
		processor.RemoveGlobalTag(key)
	}
}

// GetGlobalTags returns all global tags from the global processor.
func GetGlobalTags() map[string]string {
	processor := GetGlobalProcessor()
	if processor == nil {
		return make(map[string]string)
	}
	return processor.GetGlobalTags()
}

// Start starts the global processor.
func Start(ctx context.Context) error {
	processor := GetGlobalProcessor()
	if processor == nil {
		return common.NewGameLiftError(common.MetricConfigurationException, "", "metrics processor not initialized - call InitMetricsProcessor() first")
	}
	return processor.Start(ctx)
}

// Stop stops the global processor.
func Stop() error {
	processor := GetGlobalProcessor()
	if processor == nil {
		return nil
	}
	return processor.Stop()
}

// GetMetric retrieves a specific metric from the global processor.
func GetMetric(key string) (model.Metric, bool) { //nolint:ireturn
	processor := GetGlobalProcessor()
	if processor == nil {
		return nil, false
	}
	return processor.GetMetric(key)
}

// ListMetrics returns all registered metrics from the global processor.
func ListMetrics() []model.Metric {
	processor := GetGlobalProcessor()
	if processor == nil {
		return []model.Metric{}
	}
	return processor.ListMetrics()
}

// UnregisterMetric unregisters a metric from the global processor.
func UnregisterMetric(key string) {
	processor := GetGlobalProcessor()
	if processor != nil {
		processor.UnregisterMetric(key)
	}
}

// setDefaultGlobalTags sets default global tags that should always be present.
func (p *Processor) setDefaultGlobalTags() {
	// Set process_pid with OS process ID
	pid := os.Getpid()
	err := p.SetGlobalTag("process_pid", strconv.Itoa(pid))
	if err != nil {
		p.logger.Errorf("Error setting global tag process_id: %v", err)
	}

	// Set gamelift_process_id from environment variable if available
	if processID := os.Getenv(common.EnvironmentKeyProcessID); processID != "" {
		err = p.SetGlobalTag("gamelift_process_id", processID)
		if err != nil {
			p.logger.Errorf("Error setting global gamelift_process_id: %v", err)
		}
	}
}

// OnGameSessionStarted sets the session_id global tag when a game session starts.
// This should be called when the GameLift SDK receives a game session activation.
func (p *Processor) OnGameSessionStarted(sessionID string) {
	if sessionID != "" {
		err := p.SetGlobalTag("session_id", sessionID)
		if err != nil {
			p.logger.Errorf("Error setting global session_id: %v", err)
		}
	}
}
