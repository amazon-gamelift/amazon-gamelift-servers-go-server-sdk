/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
package transport

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/gorilla/websocket"
)

type correctnessLogger struct{ warn, reconnectFailed func() }

func (correctnessLogger) Debugf(string, ...interface{}) {}
func (l correctnessLogger) Errorf(format string, _ ...interface{}) {
	if l.reconnectFailed != nil && strings.Contains(format, "Reconnect failed:") {
		l.reconnectFailed()
	}
}
func (l correctnessLogger) Warnf(format string, _ ...interface{}) {
	if l.warn != nil && strings.Contains(format, "Detected network interruption") {
		l.warn()
	}
}

type correctnessDialer func(string, http.Header) (Conn, *http.Response, error)

func (d correctnessDialer) Dial(u string, h http.Header) (Conn, *http.Response, error) {
	return d(u, h)
}

type correctnessConn struct {
	done    chan struct{}
	once    sync.Once
	write   func([]byte) error
	handler func(int, string) error
}

func newCorrectnessConn(write func([]byte) error) *correctnessConn {
	return &correctnessConn{done: make(chan struct{}), write: write, handler: func(int, string) error { return nil }}
}
func (c *correctnessConn) ReadMessage() (int, []byte, error) {
	<-c.done
	return 0, nil, &websocket.CloseError{Code: websocket.CloseNormalClosure}
}
func (c *correctnessConn) WriteMessage(_ int, b []byte) error {
	if c.write != nil {
		return c.write(b)
	}
	return nil
}
func (c *correctnessConn) Close() error                              { c.once.Do(func() { close(c.done) }); return nil }
func (c *correctnessConn) CloseHandler() func(int, string) error     { return c.handler }
func (c *correctnessConn) SetCloseHandler(h func(int, string) error) { c.handler = h }
func correctnessReceive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(20 * time.Second):
		t.Fatal("operation did not complete")
		var zero T
		return zero
	}
}
func correctnessConnect(t *testing.T, tr ITransport) {
	t.Helper()
	tr.(*websocketTransport).connectRetryInterval = time.Microsecond
	u, _ := url.Parse("wss://example.invalid")
	if err := tr.Connect(u); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tr.PreventAutoReconnect(); tr.Close() })
}
func correctnessHandshakeFailure() (Conn, *http.Response, error) {
	return nil, &http.Response{Status: "500 Internal Server Error", Body: io.NopCloser(strings.NewReader("injected handshake failure"))}, errors.New("injected handshake failure")
}

type correctnessWriteResult struct {
	err        error
	panicValue any
}

func correctnessWrite(tr ITransport, data string) (r correctnessWriteResult) {
	defer func() { r.panicValue = recover() }()
	r.err = tr.Write([]byte(data))
	return
}

func TestReconnectWaiterReceivesSharedHandshakeFailure(t *testing.T) {
	t.Setenv(common.DisconnectWebsocketTimeout, "1ms")
	first := newCorrectnessConn(nil)
	defer first.Close()
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	var calls atomic.Int32
	var recovered atomic.Bool
	next := newCorrectnessConn(nil)
	defer next.Close()
	tr := Websocket(correctnessLogger{}, correctnessDialer(func(string, http.Header) (Conn, *http.Response, error) {
		if calls.Add(1) == 1 {
			return first, nil, nil
		}
		if recovered.Load() {
			return next, nil, nil
		}
		select {
		case <-entered:
		default:
			close(entered)
			<-release
		}
		return correctnessHandshakeFailure()
	}))
	correctnessConnect(t, tr)
	leader, waiter := make(chan error, 1), make(chan error, 1)
	go func() { leader <- tr.Reconnect() }()
	correctnessReceive(t, entered)
	go func() { waiter <- tr.Reconnect() }()
	deadline := time.Now().Add(5 * time.Second)
	joined := false
	for time.Now().Before(deadline) {
		stacks := make([]byte, 1<<16)
		n := runtime.Stack(stacks, true)
		blocked := 0
		for _, stack := range strings.Split(string(stacks[:n]), "\n\n") {
			if strings.Contains(stack, "(*websocketTransport).Reconnect(") && (strings.Contains(stack, "[chan receive]") || strings.Contains(stack, "[select]") || strings.Contains(stack, "[sync.Mutex.Lock]") || strings.Contains(stack, "[sync.Cond.Wait]")) {
				blocked++
			}
		}
		if blocked >= 2 {
			joined = true
			break
		}
		runtime.Gosched()
	}
	releaseOnce.Do(func() { close(release) })
	leaderErr, waiterErr := correctnessReceive(t, leader), correctnessReceive(t, waiter)
	if !joined {
		t.Fatal("waiter did not join the pending reconnect")
	}
	if leaderErr == nil || !strings.Contains(leaderErr.Error(), "500") {
		t.Fatalf("leader failure=%v", leaderErr)
	}
	if waiterErr != leaderErr {
		t.Fatalf("waiter lost the shared failure: leader=%v waiter=%v", leaderErr, waiterErr)
	}
	recovered.Store(true)
	if err := tr.Reconnect(); err != nil {
		t.Fatalf("a failed attempt prevented a later reconnect: %v", err)
	}
}

func TestConcurrentWriteRetainsItsOwnRetryProgress(t *testing.T) {
	t.Setenv(common.DisconnectWebsocketTimeout, "1ms")
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	var warned, allowSuccess atomic.Bool
	logger := correctnessLogger{warn: func() {
		if warned.CompareAndSwap(false, true) {
			close(entered)
			<-release
		}
	}}
	var successfulFrames atomic.Int32
	first := newCorrectnessConn(func([]byte) error { return io.ErrClosedPipe })
	next := newCorrectnessConn(func(b []byte) error {
		if string(b) != "first" {
			return fmt.Errorf("unexpected frame %q", b)
		}
		successfulFrames.Add(1)
		return nil
	})
	defer first.Close()
	defer next.Close()
	var calls atomic.Int32
	tr := Websocket(logger, correctnessDialer(func(string, http.Header) (Conn, *http.Response, error) {
		if calls.Add(1) == 1 {
			return first, nil, nil
		}
		if allowSuccess.Load() {
			return next, nil, nil
		}
		return correctnessHandshakeFailure()
	}))
	correctnessConnect(t, tr)
	pending := make(chan correctnessWriteResult, 1)
	go func() { pending <- correctnessWrite(tr, "first") }()
	correctnessReceive(t, entered)
	second := correctnessWrite(tr, "second")
	if second.panicValue != nil || second.err == nil {
		t.Fatalf("second writer: error=%v panic=%v", second.err, second.panicValue)
	}
	allowSuccess.Store(true)
	releaseOnce.Do(func() { close(release) })
	result := correctnessReceive(t, pending)
	if result.panicValue != nil || result.err != nil || successfulFrames.Load() != 1 {
		t.Fatalf("first writer lost retry progress: error=%v panic=%v successful frames=%d", result.err, result.panicValue, successfulFrames.Load())
	}
}

func TestSuccessfulReconnectDoesNotResetFailedFrameBudget(t *testing.T) {
	t.Setenv(common.DisconnectWebsocketTimeout, "1ms")
	var frames, dials atomic.Int32
	var mu sync.Mutex
	var sockets []*correctnessConn
	defer func() {
		mu.Lock()
		defer mu.Unlock()
		for _, c := range sockets {
			c.Close()
		}
	}()
	tr := Websocket(correctnessLogger{}, correctnessDialer(func(string, http.Header) (Conn, *http.Response, error) {
		// NOTE: Bound the broken implementation's loop so a regression reports a failure.
		if dials.Add(1) > int32(common.MaxReadWriteRetry+1) {
			return correctnessHandshakeFailure()
		}
		c := newCorrectnessConn(func([]byte) error { frames.Add(1); return io.ErrClosedPipe })
		mu.Lock()
		sockets = append(sockets, c)
		mu.Unlock()
		return c, nil, nil
	}))
	correctnessConnect(t, tr)
	result := correctnessWrite(tr, "never delivered")
	if result.panicValue != nil || result.err == nil {
		t.Fatalf("failed frame reported success/panic: error=%v panic=%v", result.err, result.panicValue)
	}
	if !strings.Contains(result.err.Error(), io.ErrClosedPipe.Error()) {
		t.Fatalf("last frame error was lost: %v", result.err)
	}
	if got := frames.Load(); got > int32(common.MaxReadWriteRetry) {
		t.Fatalf("successful handshakes reset failed frame budget: writes=%d limit=%d", got, common.MaxReadWriteRetry)
	}
}

func TestRetiredCloseHandlerLeavesReplacementUsable(t *testing.T) {
	t.Setenv(common.DisconnectWebsocketTimeout, "1ms")
	first, next := newCorrectnessConn(nil), newCorrectnessConn(nil)
	defer first.Close()
	defer next.Close()
	var calls atomic.Int32
	tr := Websocket(correctnessLogger{}, correctnessDialer(func(string, http.Header) (Conn, *http.Response, error) {
		if calls.Add(1) == 1 {
			return first, nil, nil
		}
		return next, nil, nil
	}))
	correctnessConnect(t, tr)
	retiredHandler := first.CloseHandler()
	u, _ := url.Parse("wss://example.invalid/refreshed")
	if err := tr.Connect(u); err != nil {
		t.Fatal(err)
	}
	if err := retiredHandler(websocket.CloseNormalClosure, "old connection"); err != nil {
		t.Fatal(err)
	}
	if err := tr.Write([]byte("replacement")); err != nil {
		t.Fatalf("old close handler disabled the replacement: %v", err)
	}
	select {
	case <-next.done:
		t.Fatal("old close handler closed the replacement")
	default:
	}
}

func TestWriteRetriesReplacementAfterFailedReconnect(t *testing.T) {
	t.Setenv(common.DisconnectWebsocketTimeout, "1ms")
	first := newCorrectnessConn(func([]byte) error { return io.ErrClosedPipe })
	var frames atomic.Int32
	next := newCorrectnessConn(func([]byte) error { frames.Add(1); return nil })
	defer first.Close()
	defer next.Close()
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	logger := correctnessLogger{reconnectFailed: func() {
		close(entered)
		<-release
	}}
	var calls atomic.Int32
	tr := Websocket(logger, correctnessDialer(func(address string, _ http.Header) (Conn, *http.Response, error) {
		if calls.Add(1) == 1 {
			return first, nil, nil
		}
		if address == "wss://example.invalid/fresh" {
			return next, nil, nil
		}
		return correctnessHandshakeFailure()
	}))
	correctnessConnect(t, tr)
	pending := make(chan correctnessWriteResult, 1)
	go func() { pending <- correctnessWrite(tr, "heartbeat") }()
	correctnessReceive(t, entered)
	u, _ := url.Parse("wss://example.invalid/fresh")
	if err := tr.Connect(u); err != nil {
		t.Fatal(err)
	}
	releaseOnce.Do(func() { close(release) })
	result := correctnessReceive(t, pending)
	if result.panicValue != nil || result.err != nil || frames.Load() != 1 {
		t.Fatalf("write did not use replacement: error=%v panic=%v frames=%d", result.err, result.panicValue, frames.Load())
	}
}
