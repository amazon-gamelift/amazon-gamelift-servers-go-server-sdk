/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// GIVEN empty host WHEN NewCrashReporterClient THEN error should be returned
func TestCrashReporterClient_NewCrashReporterClient_EmptyHost(t *testing.T) {
	_, err := NewCrashReporterClient("", 8080)
	if err == nil || !strings.Contains(err.Error(), "host cannot be empty") {
		t.Fatalf("expected host cannot be empty, got %v", err)
	}
}

// GIVEN invalid port WHEN NewCrashReporterClient THEN error should be returned
func TestCrashReporterClient_NewCrashReporterClient_InvalidPort(t *testing.T) {
	_, err := NewCrashReporterClient("localhost", 0)
	if err == nil || !strings.Contains(err.Error(), "port must be greater than zero") {
		t.Fatalf("expected port must be greater than zero, got %v", err)
	}
}

// GIVEN nil httpClient WHEN NewCrashReporterClientWithHTTPClient THEN error should be returned
func TestCrashReporterClient_NewCrashReporterClientWithHTTPClient_NilClient(t *testing.T) {
	_, err := NewCrashReporterClientWithHTTPClient(nil, "http://localhost:8080")
	if err == nil || !strings.Contains(err.Error(), "httpClient cannot be nil") {
		t.Fatalf("expected httpClient cannot be nil, got %v", err)
	}
}

// GIVEN empty baseURL WHEN NewCrashReporterClientWithHTTPClient THEN error should be returned
func TestCrashReporterClient_NewCrashReporterClientWithHTTPClient_EmptyBaseURL(t *testing.T) {
	client := &http.Client{}
	_, err := NewCrashReporterClientWithHTTPClient(client, "")
	if err == nil || !strings.Contains(err.Error(), "baseURL cannot be empty") {
		t.Fatalf("expected baseURL cannot be empty, got %v", err)
	}
}

// GIVEN valid server WHEN RegisterProcess THEN should succeed
func TestCrashReporterClient_RegisterProcess_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "register") {
			t.Errorf("expected register endpoint, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client, err := NewCrashReporterClientWithHTTPClient(ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := client.RegisterProcess(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// GIVEN server returning 500 WHEN RegisterProcess THEN error should be returned
func TestCrashReporterClient_RegisterProcess_Failure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer ts.Close()

	client, err := NewCrashReporterClientWithHTTPClient(ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = client.RegisterProcess()
	if err == nil || !strings.Contains(err.Error(), "Http response: 500") {
		t.Fatalf("expected error for 500 response, got %v", err)
	}
}

// GIVEN empty sessionID WHEN TagGameSession THEN error should be returned
func TestCrashReporterClient_TagGameSession_EmptySessionID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not be called for empty sessionID")
	}))
	defer ts.Close()

	client, err := NewCrashReporterClientWithHTTPClient(ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = client.TagGameSession("")
	if err == nil || !strings.Contains(err.Error(), "sessionID cannot be empty") {
		t.Fatalf("expected sessionID cannot be empty, got %v", err)
	}
}

// GIVEN server returning 500 WHEN TagGameSession THEN error should be returned
func TestCrashReporterClient_TagGameSession_Failure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer ts.Close()

	client, err := NewCrashReporterClientWithHTTPClient(ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = client.TagGameSession("session-123")
	if err == nil || !strings.Contains(err.Error(), "Http response: 400") {
		t.Fatalf("expected error for 400 response, got %v", err)
	}
}

// GIVEN server returning success WHEN DeregisterProcess THEN should succeed
func TestCrashReporterClient_DeregisterProcess_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "deregister") {
			t.Errorf("expected deregister endpoint, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client, err := NewCrashReporterClientWithHTTPClient(ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := client.DeregisterProcess(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// GIVEN server returning 503 WHEN DeregisterProcess THEN error should be returned
func TestCrashReporterClient_DeregisterProcess_Failure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	client, err := NewCrashReporterClientWithHTTPClient(ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = client.DeregisterProcess()
	if err == nil || !strings.Contains(err.Error(), "Http response: 503") {
		t.Fatalf("expected error for 503 response, got %v", err)
	}
}
