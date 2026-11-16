package mcpserver

import (
	"context"
	"net"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/review"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRejectedHTTPStressDoesNotQueue(t *testing.T) {
	token, authority := strings.Repeat("t", 32), "127.0.0.1:12345"
	h, err := NewHTTPHandler(NewServer(review.Executor{}), authority, token)
	if err != nil {
		t.Fatal(err)
	}
	started, release, done := make(chan struct{}, 4), make(chan struct{}), make(chan struct{}, 4)
	for range 4 {
		go func() {
			r := httptest.NewRequest("POST", "http://"+authority+"/mcp", nil)
			r.Header.Set("Authorization", "Bearer "+token)
			r.Body = blockingBody{started, release}
			h.ServeHTTP(httptest.NewRecorder(), r)
			done <- struct{}{}
		}()
	}
	for range 4 {
		<-started
	}
	defer func() {
		close(release)
		for range 4 {
			<-done
		}
	}()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	goroutines := runtime.NumGoroutine()
	for range 2000 {
		reads := new(atomic.Int32)
		r := httptest.NewRequest("POST", "http://"+authority+"/mcp", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		r.Body = observedBody{reads}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 503 || reads.Load() != 0 {
			t.Fatal("rejected request read or queued")
		}
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	if after.HeapAlloc > before.HeapAlloc+(8<<20) || runtime.NumGoroutine() > goroutines+4 {
		t.Fatal("rejected calls retained unbounded work")
	}
}

func TestStdioShutdownDeadline(t *testing.T) {
	for _, stubborn := range []bool{false, true} {
		t.Run(map[bool]string{false: "cooperative", true: "noncooperative"}[stubborn], func(t *testing.T) {
			started, release, stopped := make(chan struct{}), make(chan struct{}), make(chan struct{})
			defer close(release)
			e := review.Executor{NewProvider: func(_, _ string, _ provider.CaptureLimits) (provider.Provider, error) {
				return testReviewer(func(ctx context.Context, _ provider.Request) (*provider.Result, error) {
					close(started)
					if stubborn {
						<-release
					} else {
						<-ctx.Done()
					}
					close(stopped)
					return nil, ctx.Err()
				}), nil
			}}
			a, b := net.Pipe()
			defer b.Close()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- ServeStdio(ctx, NewServer(e), a, a) }()
			cs, err := mcp.NewClient(&mcp.Implementation{Name: "shutdown", Version: "1"}, nil).Connect(t.Context(), &mcp.IOTransport{Reader: b, Writer: b}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer cs.Close()
			go func() {
				_, _ = cs.CallTool(t.Context(), &mcp.CallToolParams{Name: "review", Arguments: map[string]any{"provider": "codex", "material": "shutdown"}})
			}()
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("not started")
			}
			at := time.Now()
			cancel()
			select {
			case <-done:
			case <-time.After(6 * time.Second):
				t.Fatal("unbounded shutdown")
			}
			if stubborn && time.Since(at) < 4*time.Second {
				t.Fatal("did not drain active call")
			}
			if !stubborn {
				select {
				case <-stopped:
				case <-time.After(time.Second):
					t.Fatal("provider not cancelled")
				}
			}
		})
	}
}
