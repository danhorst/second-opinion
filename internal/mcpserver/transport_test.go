package mcpserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/review"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func transportSession(t *testing.T, kind, version string, e review.Executor) *mcp.ClientSession {
	t.Helper()
	server := NewServer(e)
	var tr mcp.Transport
	if kind == "stdio" {
		a, b := net.Pipe()
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)
		go func() { done <- ServeStdio(ctx, server, a, a) }()
		t.Cleanup(func() {
			cancel()
			b.Close()
			select {
			case <-done:
			case <-time.After(6 * time.Second):
				t.Error("stdio shutdown stuck")
			}
		})
		tr = &mcp.IOTransport{Reader: b, Writer: b}
	} else {
		ts := httptest.NewUnstartedServer(nil)
		token := strings.Repeat("x", 32)
		handler, err := NewHTTPHandler(server, ts.Listener.Addr().String(), token)
		if err != nil {
			t.Fatal(err)
		}
		ts.Config.Handler = handler
		ts.Start()
		t.Cleanup(ts.Close)
		tr = &mcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp", HTTPClient: &http.Client{Transport: bearerRoundTripper{token: token, t: t}}, DisableStandaloneSSE: true, MaxRetries: -1}
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "transport-test", Version: "1"}, nil).Connect(t.Context(), tr, &mcp.ClientSessionOptions{ProtocolVersion: version})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

type bearerRoundTripper struct {
	token string
	t     *testing.T
}

func (b bearerRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	response, err := http.DefaultTransport.RoundTrip(r)
	if err == nil && response.Header.Get("Mcp-Session-Id") != "" {
		b.t.Error("stateless server emitted session ID")
	}
	return response, err
}

func TestTransportVersions(t *testing.T) {
	for _, kind := range []string{"stdio", "http"} {
		for _, version := range []string{"2026-07-28", "2025-11-25"} {
			t.Run(kind+version, func(t *testing.T) {
				e := review.Executor{NewProvider: func(_, _ string, _ provider.CaptureLimits) (provider.Provider, error) {
					return testReviewer(func(_ context.Context, r provider.Request) (*provider.Result, error) { return goodResult(r), nil }), nil
				}}
				cs := transportSession(t, kind, version, e)
				tools, err := cs.ListTools(t.Context(), nil)
				if err != nil {
					t.Fatal(err)
				}
				if len(tools.Tools) != 2 {
					t.Fatalf("tools %d", len(tools.Tools))
				}
				result := callTool(t, cs, "review", map[string]any{"provider": "codex", "material": "exact bytes"})
				if got := mirrored(t, result)["raw"]; got != "exact bytes" {
					t.Fatalf("raw: %v", got)
				}
				callTool(t, cs, "list_providers", map[string]any{})
			})
		}
	}
}

func TestTransportCancellationAndRecovery(t *testing.T) {
	for _, kind := range []string{"stdio", "http"} {
		for _, version := range []string{"2026-07-28", "2025-11-25"} {
			t.Run(kind+version, func(t *testing.T) {
				started := make(chan string, 2)
				stopped := make(chan string, 2)
				release := make(chan struct{})
				e := review.Executor{NewProvider: func(_, _ string, _ provider.CaptureLimits) (provider.Provider, error) {
					return testReviewer(func(ctx context.Context, r provider.Request) (*provider.Result, error) {
						if r.Material == "fast" {
							return goodResult(r), nil
						}
						started <- r.Material
						select {
						case <-ctx.Done():
							stopped <- r.Material
							return nil, ctx.Err()
						case <-release:
							return goodResult(r), nil
						}
					}), nil
				}}
				cs := transportSession(t, kind, version, e)
				first, cancel := context.WithCancel(t.Context())
				defer cancel()
				done := make(chan error, 2)
				go func() {
					_, err := cs.CallTool(first, &mcp.CallToolParams{Name: "review", Arguments: map[string]any{"provider": "codex", "material": "first"}})
					done <- err
				}()
				select {
				case <-started:
				case <-time.After(3 * time.Second):
					t.Fatal("first never started")
				}
				go func() {
					_, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: "review", Arguments: map[string]any{"provider": "codex", "material": "second"}})
					done <- err
				}()
				select {
				case <-started:
				case <-time.After(3 * time.Second):
					t.Fatal("second never started")
				}
				failureCode(t, callTool(t, cs, "review", map[string]any{"provider": "codex", "material": "third"}), "busy")
				cancel()
				select {
				case got := <-stopped:
					if got != "first" {
						t.Fatalf("cancelled %s", got)
					}
				case <-time.After(3 * time.Second):
					t.Fatal("provider did not receive cancellation")
				}
				// Provider cleanup is asynchronous; wait until its slot becomes reusable.
				deadline := time.Now().Add(3 * time.Second)
				for {
					r := callTool(t, cs, "review", map[string]any{"provider": "codex", "material": "fast"})
					if !r.IsError {
						break
					}
					if time.Now().After(deadline) {
						t.Fatal("capacity did not recover")
					}
					time.Sleep(time.Millisecond)
				}
				close(release)
				for range 2 {
					select {
					case <-done:
					case <-time.After(3 * time.Second):
						t.Fatal("call stuck")
					}
				}
			})
		}
	}
}

type observedBody struct{ reads *atomic.Int32 }

func (b observedBody) Read([]byte) (int, error) { b.reads.Add(1); return 0, io.EOF }
func (observedBody) Close() error               { return nil }

func TestHTTPBoundaryBeforeBody(t *testing.T) {
	const authority = "127.0.0.1:12345"
	token := strings.Repeat("t", 32)
	h, err := NewHTTPHandler(NewServer(review.Executor{}), authority, token)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, auth, host, origin, method string
		status                           int
	}{
		{"no auth", "", authority, "", "POST", 401}, {"bad auth", "bad", authority, "", "POST", 401},
		{"host", token, "localhost:12345", "", "POST", 403}, {"origin", token, authority, "null", "POST", 403},
		{"GET", token, authority, "", "GET", 405}, {"DELETE", token, authority, "", "DELETE", 405},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reads := new(atomic.Int32)
			r := httptest.NewRequest(tc.method, "http://"+authority+"/mcp", nil)
			r.Body = observedBody{reads}
			r.Host = tc.host
			if tc.auth != "" {
				r.Header.Set("Authorization", "Bearer "+tc.auth)
			}
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.status || reads.Load() != 0 {
				t.Fatalf("status %d body reads %d", w.Code, reads.Load())
			}
		})
	}
	for _, raw := range []string{strings.Repeat(" ", MaxRequestBytes+1), `{"x":"\ud800"}`, "{\"x\":\"\xff\"}"} {
		r := httptest.NewRequest("POST", "http://"+authority+"/mcp", strings.NewReader(raw))
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 400 && w.Code != 413 {
			t.Fatalf("invalid input status %d", w.Code)
		}
	}
}

func TestListenAddress(t *testing.T) {
	for _, address := range []string{"127.0.0.1:0", "127.0.0.1:8765", "[::1]:8765"} {
		if err := ValidateListenAddress(address); err != nil {
			t.Fatal(err)
		}
	}
	for _, address := range []string{"localhost:8765", ":8765", "0.0.0.0:8765", "192.168.1.1:8765", "127.0.0.2:8765", "[::]:8765", "127.0.0.1:-1", "127.0.0.1:65536"} {
		if ValidateListenAddress(address) == nil {
			t.Errorf("accepted %s", address)
		}
	}
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

func TestStdioAdmissionBypassesCancellation(t *testing.T) {
	var input strings.Builder
	for i := 1; i <= 5; i++ {
		fmt.Fprintf(&input, "{\"jsonrpc\":\"2.0\",\"id\":%d,\"method\":\"tools/list\"}\n", i)
	}
	input.WriteString("{\"jsonrpc\":\"2.0\",\"method\":\"notifications/cancelled\",\"params\":{\"requestId\":1}}\n")
	c := newStdioConnection(io.NopCloser(strings.NewReader(input.String())), nopWriteCloser{io.Discard}, func() {})
	defer c.Close()
	for range 4 {
		if _, err := c.Read(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	msg, err := c.Read(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if got := msg.(*jsonrpc.Request).Method; got != "notifications/cancelled" {
		t.Fatalf("read %s", got)
	}
	if len(c.active) != 4 {
		t.Fatalf("active %d", len(c.active))
	}
	id, _ := jsonrpc.MakeID(float64(1))
	if err := c.Write(t.Context(), &jsonrpc.Response{ID: id, Result: json.RawMessage(`{}`)}); err != nil {
		t.Fatal(err)
	}
	if len(c.active) != 3 {
		t.Fatal("cancelled request capacity leaked")
	}
}

func TestStdioFrameAndResponseBounds(t *testing.T) {
	for _, raw := range []string{strings.Repeat("x", MaxRequestBytes+1), `{"jsonrpc":"2.0","id":1,"method":"x","params":{"x":"\ud800"}}`, "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"\xff\"}"} {
		c := newStdioConnection(io.NopCloser(strings.NewReader(raw+"\n")), nopWriteCloser{io.Discard}, func() {})
		if _, err := c.Read(t.Context()); err == nil {
			t.Fatal("accepted invalid frame")
		}
		c.Close()
	}
	var out bytes.Buffer
	c := newStdioConnection(io.NopCloser(strings.NewReader("")), nopWriteCloser{&out}, func() {})
	defer c.Close()
	raw, _ := json.Marshal(strings.Repeat("x", MaxResponseBytes))
	if c.Write(t.Context(), &jsonrpc.Response{Result: raw}) == nil || out.Len() != 0 {
		t.Fatal("oversized response written")
	}
	w := &boundedHTTPResponse{header: make(http.Header)}
	if _, err := w.Write(make([]byte, MaxResponseBytes+1)); err == nil || w.body.Len() != 0 {
		t.Fatal("HTTP response not bounded")
	}
}

type blockedWriter struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (w *blockedWriter) Write(p []byte) (int, error) {
	w.once.Do(func() { close(w.started) })
	<-w.release
	return len(p), nil
}
func (w *blockedWriter) Close() error { return nil }

func TestStdioBlockedOutputKeepsAdmission(t *testing.T) {
	w := &blockedWriter{started: make(chan struct{}), release: make(chan struct{})}
	c := newStdioConnection(io.NopCloser(strings.NewReader("")), w, func() {})
	defer c.Close()
	done := make(chan struct{}, 4)
	for i := range 4 {
		id, _ := jsonrpc.MakeID(float64(i))
		c.active[id] = false
	}
	for i := range 4 {
		go func(i int) {
			id, _ := jsonrpc.MakeID(float64(i))
			_ = c.Write(t.Context(), &jsonrpc.Response{ID: id, Result: json.RawMessage(`{}`)})
			done <- struct{}{}
		}(i)
	}
	<-w.started
	c.mu.Lock()
	count := len(c.active)
	c.mu.Unlock()
	if count != MaxInflightMessages {
		t.Errorf("blocked responses released admission: %d", count)
	}
	close(w.release)
	for range 4 {
		<-done
	}
	c.mu.Lock()
	count = len(c.active)
	c.mu.Unlock()
	if count != 0 {
		t.Error("completed responses leaked admission")
	}
}

func TestTransportDisconnectCancelsReview(t *testing.T) {
	for _, kind := range []string{"stdio", "http"} {
		for _, version := range []string{"2026-07-28", "2025-11-25"} {
			t.Run(kind+version, func(t *testing.T) {
				started, stopped := make(chan struct{}), make(chan struct{})
				e := review.Executor{NewProvider: func(_, _ string, _ provider.CaptureLimits) (provider.Provider, error) {
					return testReviewer(func(ctx context.Context, _ provider.Request) (*provider.Result, error) {
						close(started)
						<-ctx.Done()
						close(stopped)
						return nil, ctx.Err()
					}), nil
				}}
				server := NewServer(e)
				done := make(chan struct{})
				var disconnect func()
				if kind == "stdio" {
					a, b := net.Pipe()
					ctx, cancel := context.WithCancel(t.Context())
					defer cancel()
					go func() { _ = ServeStdio(ctx, server, a, a) }()
					client, err := mcp.NewClient(&mcp.Implementation{Name: "disconnect", Version: "1"}, nil).Connect(ctx, &mcp.IOTransport{Reader: b, Writer: b}, &mcp.ClientSessionOptions{ProtocolVersion: version})
					if err != nil {
						t.Fatal(err)
					}
					defer client.Close()
					disconnect = func() { b.Close() }
					go func() {
						_, _ = client.CallTool(ctx, &mcp.CallToolParams{Name: "review", Arguments: map[string]any{"provider": "codex", "material": "disconnect"}})
						close(done)
					}()
				} else {
					ts := httptest.NewUnstartedServer(nil)
					token := strings.Repeat("x", 32)
					h, err := NewHTTPHandler(server, ts.Listener.Addr().String(), token)
					if err != nil {
						t.Fatal(err)
					}
					ts.Config.Handler = h
					ts.Start()
					defer ts.Close()
					ctx, cancel := context.WithCancel(t.Context())
					defer cancel()
					disconnect = cancel
					payload := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"review","arguments":{"provider":"codex","material":"disconnect"}}}`
					if version == "2026-07-28" {
						payload = strings.Replace(payload, `"params":{`, `"params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}},`, 1)
					}
					req, err := http.NewRequestWithContext(ctx, "POST", ts.URL+"/mcp", strings.NewReader(payload))
					if err != nil {
						t.Fatal(err)
					}
					req.Header.Set("Authorization", "Bearer "+token)
					req.Header.Set("MCP-Protocol-Version", version)
					req.Header.Set("Mcp-Method", "tools/call")
					req.Header.Set("Mcp-Name", "review")
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Accept", "application/json, text/event-stream")
					go func() {
						response, _ := http.DefaultClient.Do(req)
						if response != nil {
							response.Body.Close()
						}
						close(done)
					}()
				}
				defer disconnect()
				select {
				case <-started:
				case <-time.After(3 * time.Second):
					t.Fatal("not started")
				}
				disconnect()
				select {
				case <-stopped:
				case <-time.After(3 * time.Second):
					t.Fatal("disconnect did not cancel provider")
				}
				select {
				case <-done:
				case <-time.After(3 * time.Second):
					t.Fatal("call did not stop")
				}
			})
		}
	}
}

type blockingBody struct {
	started chan<- struct{}
	release <-chan struct{}
}

func (b blockingBody) Read([]byte) (int, error) {
	b.started <- struct{}{}
	<-b.release
	return 0, io.EOF
}
func (blockingBody) Close() error { return nil }

func TestHTTPTransportAdmissionBeforeBody(t *testing.T) {
	token := strings.Repeat("x", 32)
	authority := "127.0.0.1:12345"
	h, err := NewHTTPHandler(NewServer(review.Executor{}), authority, token)
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{}, 4)
	release := make(chan struct{})
	done := make(chan struct{}, 4)
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
	reads := new(atomic.Int32)
	r := httptest.NewRequest("POST", "http://"+authority+"/mcp", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	r.Body = observedBody{reads}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 503 || reads.Load() != 0 {
		t.Errorf("saturated status %d body reads %d", w.Code, reads.Load())
	}
	close(release)
	for range 4 {
		<-done
	}
}

func TestHTTPSlowBodyDeadline(t *testing.T) {
	ts := httptest.NewUnstartedServer(nil)
	token := strings.Repeat("x", 32)
	h, err := NewHTTPHandler(NewServer(review.Executor{}), ts.Listener.Addr().String(), token)
	if err != nil {
		t.Fatal(err)
	}
	ts.Config.Handler = h
	ts.Start()
	defer ts.Close()
	conn, err := net.Dial("tcp", ts.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(13 * time.Second))
	started := time.Now()
	fmt.Fprintf(conn, "POST /mcp HTTP/1.1\r\nHost: %s\r\nAuthorization: Bearer %s\r\nContent-Length: 100\r\n\r\n{", ts.Listener.Addr(), token)
	response, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 400 {
		t.Fatalf("slow body status %d", response.StatusCode)
	}
	if elapsed := time.Since(started); elapsed < 9*time.Second || elapsed > 12*time.Second {
		t.Fatalf("body deadline %s", elapsed)
	}
}

func TestHTTPSubscriptionsCannotHoldStream(t *testing.T) {
	token := strings.Repeat("x", 32)
	authority := "127.0.0.1:12345"
	h, err := NewHTTPHandler(NewServer(review.Executor{}), authority, token)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "http://"+authority+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"subscriptions/listen","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}},"subscriptions":[]}}`))
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json, text/event-stream")
	r.Header.Set("MCP-Protocol-Version", "2026-07-28")
	r.Header.Set("Mcp-Method", "subscriptions/listen")
	done := make(chan struct{})
	w := httptest.NewRecorder()
	go func() { h.ServeHTTP(w, r); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("subscription retained a stream")
	}
	if !strings.Contains(w.Body.String(), `"code":-32601`) {
		t.Fatalf("subscription did not fail: %s", w.Body.String())
	}
}
