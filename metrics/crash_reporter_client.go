/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package metrics

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// CrashReporterClient is a client to communicate with OTEL Collector Crash Reporter.
type CrashReporterClient struct {
	httpClient *http.Client
	baseURL    string
}

const (
	httpDefaultTimeout       = 5 * time.Second
	registerProcessUrlPath   = "register"
	updateProcessUrlPath     = "update"
	deregisterProcessUrlPath = "deregister"
	processPidParameterName  = "process_pid"
	sessionIdParameterName   = "session_id"
)

// NewCrashReporterClient creates a new client using host and port.
func NewCrashReporterClient(host string, port int) (*CrashReporterClient, error) {
	if host == "" {
		return nil, errors.New("host cannot be empty")
	}
	if port <= 0 {
		return nil, errors.New("port must be greater than zero")
	}

	baseURL := fmt.Sprintf("http://%s:%d", host, port)
	return &CrashReporterClient{
		httpClient: &http.Client{
			Timeout: httpDefaultTimeout,
		},
		baseURL: baseURL,
	}, nil
}

// NewCrashReporterClientWithHTTPClient allows using a custom http.Client.
func NewCrashReporterClientWithHTTPClient(httpClient *http.Client, baseURL string) (*CrashReporterClient, error) {
	if httpClient == nil {
		return nil, errors.New("httpClient cannot be nil")
	}
	if baseURL == "" {
		return nil, errors.New("baseURL cannot be empty")
	}

	return &CrashReporterClient{
		httpClient: httpClient,
		baseURL:    baseURL,
	}, nil
}

// RegisterProcess registers the current process with the crash reporter.
func (c *CrashReporterClient) RegisterProcess() error {
	pid := os.Getpid()
	url := fmt.Sprintf("%s/%s?%s=%d", c.baseURL, registerProcessUrlPath, processPidParameterName, pid)

	log.Printf("Registering process with %s=%d in OTEL Collector Crash Reporter", processPidParameterName, pid)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to register %s=%d to OTEL Collector Crash Reporter due to error: %w", processPidParameterName, pid, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to register %s=%d to OTEL Collector Crash Reporter, Http response: %s", processPidParameterName, pid, resp.Status)
	}

	return nil
}

// TagGameSession tags the process with a game session ID.
func (c *CrashReporterClient) TagGameSession(sessionID string) error {
	if sessionID == "" {
		return errors.New("sessionID cannot be empty")
	}

	pid := os.Getpid()
	url := fmt.Sprintf("%s/%s?%s=%d&%s=%s",
		c.baseURL, updateProcessUrlPath, processPidParameterName, pid, sessionIdParameterName, sessionID)

	log.Printf("Tagging process %d with %s=%s in OTEL Collector Crash Reporter", pid, sessionIdParameterName, sessionID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to tag %s=%s for process %s=%d in the OTEL Collector Crash Reporter due to error: %w", sessionIdParameterName, sessionID, processPidParameterName, pid, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to tag %s=%s for process %s=%d in the OTEL Collector Crash Reporter, Http response: %s",
			sessionIdParameterName, sessionID, processPidParameterName, pid, resp.Status)
	}

	return nil
}

// DeregisterProcess removes the process from the crash reporter.
func (c *CrashReporterClient) DeregisterProcess() error {
	pid := os.Getpid()
	url := fmt.Sprintf("%s/%s?%s=%d", c.baseURL, deregisterProcessUrlPath, processPidParameterName, pid)

	log.Printf("Unregistering process with %s=%d in OTEL Collector Crash Reporter", processPidParameterName, pid)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to deregister %s=%d in the OTEL Collector Crash Reporter due to error: %w", processPidParameterName, pid, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to deregister %s=%d in the OTEL Collector Crash Reporter, Http response: %s", processPidParameterName, pid, resp.Status)
	}

	return nil
}
