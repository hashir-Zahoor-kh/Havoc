package health

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHealthz_AlwaysOK(t *testing.T) {
	t.Parallel()
	s := New(":0")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	s.handleHealthz(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
}

func TestReadyz_503BeforeReady_200After(t *testing.T) {
	t.Parallel()
	s := New(":0")

	rec := httptest.NewRecorder()
	s.handleReadyz(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("before MarkReady: got %d, want 503", rec.Code)
	}

	s.MarkReady()
	rec = httptest.NewRecorder()
	s.handleReadyz(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("after MarkReady: got %d, want 200", rec.Code)
	}

	s.MarkNotReady()
	rec = httptest.NewRecorder()
	s.handleReadyz(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("after MarkNotReady: got %d, want 503", rec.Code)
	}
}

func TestStart_RoundTrip(t *testing.T) {
	t.Parallel()

	// Bind to an ephemeral port so the test doesn't collide with anything.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	s := New(addr)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	startErr := make(chan error, 1)
	go func() { startErr <- s.Start(ctx) }()

	// Wait briefly for the server to come up.
	deadline := time.Now().Add(2 * time.Second)
	var resp *http.Response
	for {
		resp, err = http.Get("http://" + addr + "/healthz")
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server never came up: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "ok") {
		t.Fatalf("/healthz: got %d %q, want 200 ok", resp.StatusCode, string(body))
	}

	// /readyz should be 503 until MarkReady is called.
	resp, err = http.Get("http://" + addr + "/readyz")
	if err != nil {
		t.Fatalf("readyz: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("/readyz before ready: got %d, want 503", resp.StatusCode)
	}

	s.MarkReady()
	resp, err = http.Get("http://" + addr + "/readyz")
	if err != nil {
		t.Fatalf("readyz after MarkReady: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/readyz after ready: got %d, want 200", resp.StatusCode)
	}

	cancel()
	select {
	case err := <-startErr:
		if err != nil {
			t.Fatalf("Start returned: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after ctx cancel")
	}
}
