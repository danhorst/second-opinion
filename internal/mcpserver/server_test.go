package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/review"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type testReviewer func(context.Context, provider.Request) (*provider.Result, error)

func (f testReviewer) Review(ctx context.Context, r provider.Request) (*provider.Result, error) {
	return f(ctx, r)
}
func goodResult(r provider.Request) *provider.Result {
	return &provider.Result{Findings: r.Material, Provenance: provider.Provenance{Provider: "codex", Model: "actual-model", PromptHash: provider.HashPrompt(r.Prompt), MaterialHash: provider.HashMaterial(r.Material)}}
}
func sdkSession(t *testing.T, e review.Executor, version string) *mcp.ClientSession {
	t.Helper()
	st, ct := mcp.NewInMemoryTransports()
	ss, err := NewServer(e).Connect(t.Context(), st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ss.Close() })
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(t.Context(), ct, &mcp.ClientSessionOptions{ProtocolVersion: version})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}
func callTool(t *testing.T, c *mcp.ClientSession, name string, args any) *mcp.CallToolResult {
	t.Helper()
	r, e := c.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func mirrored(t *testing.T, r *mcp.CallToolResult) map[string]any {
	t.Helper()
	if len(r.Content) != 1 {
		t.Fatalf("content count %d", len(r.Content))
	}
	text, ok := r.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content %T", r.Content[0])
	}
	var a, b map[string]any
	if err := json.Unmarshal([]byte(text.Text), &a); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(r.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &b); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatal("text/structured mismatch")
	}
	return a
}
func failureCode(t *testing.T, r *mcp.CallToolResult, want string) {
	t.Helper()
	v := mirrored(t, r)
	if !r.IsError || v["kind"] != "error" || v["error"].(map[string]any)["code"] != want {
		t.Fatalf("want %s: %+v", want, v)
	}
}

func TestSDKExactInputsAndStatelessness(t *testing.T) {
	for _, version := range []string{"2026-07-28", "2025-11-25"} {
		t.Run(version, func(t *testing.T) {
			var constructions int
			var requests []provider.Request
			e := review.Executor{NewProvider: func(name, model string, limits provider.CaptureLimits) (provider.Provider, error) {
				if name != "codex" || limits != (provider.CaptureLimits{StdoutBytes: MaxStdoutBytes, StderrBytes: MaxStderrBytes}) {
					t.Errorf("factory %s %+v", name, limits)
				}
				wantModel := ""
				if constructions == 0 {
					wantModel = " forced "
				}
				if model != wantModel {
					t.Errorf("model %q", model)
				}
				constructions++
				return testReviewer(func(_ context.Context, r provider.Request) (*provider.Result, error) {
					requests = append(requests, r)
					return goodResult(r), nil
				}), nil
			}}
			cs := sdkSession(t, e, version)
			listed, err := cs.ListTools(t.Context(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(listed.Tools) != 2 || listed.Tools[0].Name != "list_providers" || listed.Tools[1].Name != "review" {
				t.Fatalf("tools %+v", listed.Tools)
			}
			mirrored(t, callTool(t, cs, "list_providers", map[string]any{}))
			if constructions != 0 {
				t.Fatal("discovery invoked factory")
			}
			custom := "  prompt ☃\r\n"
			material := " /not/a/file ☃\r\n "
			result := callTool(t, cs, "review", map[string]any{"provider": "codex", "model": " forced ", "prompt": custom, "material": material, "timeout_ms": 4567})
			v := mirrored(t, result)
			if result.IsError || v["raw"] != material {
				t.Fatalf("result %+v", v)
			}
			p := v["provenance"].(map[string]any)
			receipt := v["request"].(map[string]any)
			if p["material_hash"] != provider.HashMaterial(material) || p["prompt_hash"] != provider.HashPrompt(custom) || p["model"] != "actual-model" || receipt["material_hash"] != provider.HashPrompt(material) || receipt["material_bytes"] != float64(len(material)) || receipt["timeout_ms"] != float64(4567) {
				t.Fatalf("receipt/provenance %+v", v)
			}
			v = mirrored(t, callTool(t, cs, "review", map[string]any{"provider": "codex", "material": "B"}))
			if len(requests) != 2 || requests[0] != (provider.Request{Prompt: custom, Material: material}) || requests[1] != (provider.Request{Prompt: review.BakedPrompt, Material: "B"}) || v["request"].(map[string]any)["timeout_ms"] != float64(DefaultTimeoutMS) {
				t.Fatalf("state leaked: %+v", requests)
			}
		})
	}
}

func TestSDKFailuresAndRecovery(t *testing.T) {
	cases := []struct {
		name, code string
		err        error
		raw        string
		panic      bool
		nilResult  bool
	}{
		{name: "auth", code: "auth", err: fmt.Errorf("secret material: %w", provider.ErrAuth)},
		{name: "unavailable", code: "unavailable", err: provider.ErrUnavailable},
		{name: "overflow", code: "output_too_large", err: provider.ErrOutputLimit},
		{name: "unknown", code: "provider_failed", err: errors.New("secret material")},
		{name: "nil", code: "provider_failed", nilResult: true},
		{name: "panic", code: "provider_failed", panic: true},
		{name: "raw too large", code: "output_too_large", raw: strings.Repeat("x", MaxRawBytes+1)},
		{name: "empty success", raw: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var n int
			cs := sdkSession(t, review.Executor{NewProvider: func(string, string, provider.CaptureLimits) (provider.Provider, error) {
				return testReviewer(func(_ context.Context, r provider.Request) (*provider.Result, error) {
					n++
					if n > 1 {
						return goodResult(r), nil
					}
					if tc.panic {
						panic("secret material")
					}
					if tc.err != nil || tc.nilResult {
						return nil, tc.err
					}
					v := goodResult(r)
					v.Findings = tc.raw
					return v, nil
				}), nil
			}}, "2026-07-28")
			r := callTool(t, cs, "review", map[string]any{"provider": "codex", "material": "test"})
			if tc.code != "" {
				failureCode(t, r, tc.code)
			} else {
				if r.IsError || mirrored(t, r)["raw"] != "" {
					t.Fatal("empty result failed")
				}
			}
			encoded, _ := json.Marshal(r)
			if strings.Contains(string(encoded), "secret material") {
				t.Fatal("diagnostic leaked")
			}
			for i := 0; i < 3; i++ {
				if callTool(t, cs, "review", map[string]any{"provider": "codex", "material": "recovered"}).IsError {
					t.Fatal("capacity leaked")
				}
			}
		})
	}
}

func TestSDKBusyCancellationAndTimeout(t *testing.T) {
	var active atomic.Int32
	entered := make(chan string, 8)
	exited := make(chan string, 8)
	release := make(chan struct{})
	cs := sdkSession(t, review.Executor{NewProvider: func(string, string, provider.CaptureLimits) (provider.Provider, error) {
		return testReviewer(func(ctx context.Context, r provider.Request) (*provider.Result, error) {
			active.Add(1)
			defer active.Add(-1)
			defer func() { exited <- r.Material }()
			entered <- r.Material
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-release:
				return goodResult(r), nil
			}
		}), nil
	}}, "2026-07-28")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 2)
	go func() {
		_, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "review", Arguments: map[string]any{"provider": "codex", "material": "A"}})
		done <- err
	}()
	<-entered
	go func() {
		_, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: "review", Arguments: map[string]any{"provider": "codex", "material": "B"}})
		done <- err
	}()
	<-entered
	failureCode(t, callTool(t, cs, "review", map[string]any{"provider": "codex", "material": "busy"}), "busy")
	cancel()
	select {
	case name := <-exited:
		if name != "A" {
			t.Fatal("sibling cancelled")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancel not delivered")
	}
	// Wait for slot release after the provider's deferred notification.
	deadline := time.Now().Add(time.Second)
	for {
		r := callTool(t, cs, "review", map[string]any{"provider": "codex", "material": "deadline", "timeout_ms": 5})
		v := mirrored(t, r)
		if v["error"].(map[string]any)["code"] == "timeout" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("capacity not recovered")
		}
	}
	if active.Load() != 1 {
		t.Fatalf("sibling active=%d", active.Load())
	}
	close(release)
	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("request stuck")
		}
	}
}

func TestSDKInvalidInputsNeverConstructProvider(t *testing.T) {
	for _, version := range []string{"2026-07-28", "2025-11-25"} {
		t.Run(version, func(t *testing.T) {
			cs := sdkSession(t, review.Executor{NewProvider: func(string, string, provider.CaptureLimits) (provider.Provider, error) {
				t.Error("invalid input reached factory")
				return nil, errors.New("unexpected factory")
			}}, version)
			for _, args := range []map[string]any{
				{}, {"provider": "codex", "material": " "}, {"provider": "unknown", "material": "test"},
				{"provider": "codex", "material": "test", "prompt": nil}, {"provider": "codex", "material": "test", "timeout_ms": 1.5},
				{"provider": "codex", "material": "test", "file": "do-not-read"},
			} {
				failureCode(t, callTool(t, cs, "review", args), "invalid_input")
			}
			failureCode(t, callTool(t, cs, "review", map[string]any{"provider": "codex", "material": strings.Repeat("x", MaxInputBytes)}), "input_too_large")
			failureCode(t, callTool(t, cs, "list_providers", map[string]any{"unexpected": true}), "invalid_input")
		})
	}
}

type observedTransport struct {
	mcp.Transport
	ids chan jsonrpc.ID
}

func (t observedTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	c, e := t.Transport.Connect(ctx)
	if e != nil {
		return nil, e
	}
	return observedConnection{c, t.ids}, nil
}

type observedConnection struct {
	mcp.Connection
	ids chan jsonrpc.ID
}

func (c observedConnection) Write(ctx context.Context, msg jsonrpc.Message) error {
	if r, ok := msg.(*jsonrpc.Request); ok && r.Method == "tools/call" {
		c.ids <- r.ID
	}
	return c.Connection.Write(ctx, msg)
}

func TestSDKSeparateConnectionsReuseIDsWithoutState(t *testing.T) {
	for _, version := range []string{"2026-07-28", "2025-11-25"} {
		t.Run(version, func(t *testing.T) {
			var requests []provider.Request
			executor := review.Executor{NewProvider: func(string, string, provider.CaptureLimits) (provider.Provider, error) {
				return testReviewer(func(_ context.Context, r provider.Request) (*provider.Result, error) {
					requests = append(requests, r)
					return goodResult(r), nil
				}), nil
			}}
			shared := NewServer(executor)
			connect := func(server *mcp.Server) (*mcp.ClientSession, chan jsonrpc.ID) {
				st, ct := mcp.NewInMemoryTransports()
				ids := make(chan jsonrpc.ID, 1)
				ss, e := server.Connect(t.Context(), st, nil)
				if e != nil {
					t.Fatal(e)
				}
				t.Cleanup(func() { ss.Close() })
				cs, e := mcp.NewClient(&mcp.Implementation{Name: "independent", Version: "1"}, nil).Connect(t.Context(), observedTransport{ct, ids}, &mcp.ClientSessionOptions{ProtocolVersion: version})
				if e != nil {
					t.Fatal(e)
				}
				t.Cleanup(func() { cs.Close() })
				return cs, ids
			}
			a, aIDs := connect(shared)
			b, bIDs := connect(shared)
			fresh, _ := connect(NewServer(executor))
			callTool(t, a, "review", map[string]any{"provider": "codex", "model": "forced", "prompt": "A prompt", "material": "A"})
			bResult := mirrored(t, callTool(t, b, "review", map[string]any{"provider": "codex", "material": "B"}))
			freshResult := mirrored(t, callTool(t, fresh, "review", map[string]any{"provider": "codex", "material": "B"}))
			if idA, idB := <-aIDs, <-bIDs; idA != idB {
				t.Fatalf("test needs reused IDs: %v vs %v", idA, idB)
			}
			if len(requests) != 3 || requests[1] != requests[2] || requests[1] != (provider.Request{Prompt: review.BakedPrompt, Material: "B"}) || !reflect.DeepEqual(bResult, freshResult) {
				t.Fatalf("connection state leaked: %+v", requests)
			}
		})
	}
}

func TestSDKBusyStressDoesNotQueueOrRetainMaterial(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var constructions atomic.Int32
	entered := make(chan struct{}, 2)
	cs := sdkSession(t, review.Executor{NewProvider: func(string, string, provider.CaptureLimits) (provider.Provider, error) {
		constructions.Add(1)
		return testReviewer(func(ctx context.Context, _ provider.Request) (*provider.Result, error) {
			entered <- struct{}{}
			<-ctx.Done()
			return nil, ctx.Err()
		}), nil
	}}, "2026-07-28")
	done := make(chan struct{}, 2)
	for i := 0; i < 2; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			_, _ = cs.CallTool(ctx, &mcp.CallToolParams{Name: "review", Arguments: map[string]any{"provider": "codex", "material": "occupy"}})
		}()
	}
	<-entered
	<-entered
	failureCode(t, callTool(t, cs, "review", map[string]any{"provider": "codex", "material": "warmup"}), "busy")
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < 256; i++ {
		failureCode(t, callTool(t, cs, "review", map[string]any{"provider": "codex", "material": fmt.Sprintf("%d:%s", i, strings.Repeat("x", 64<<10))}), "busy")
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	if constructions.Load() != 2 {
		t.Fatalf("busy requests started work: %d", constructions.Load())
	}
	// More than 16 MiB of distinct material crossed the seam. Allow generous
	// runtime/SDK bookkeeping slack, but reject retention of the busy inputs.
	if after.HeapAlloc > before.HeapAlloc+(8<<20) {
		t.Fatalf("live heap grew by %d bytes", after.HeapAlloc-before.HeapAlloc)
	}
	cancel()
	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("active request did not cancel")
		}
	}
}

func TestResultEncodingBounds(t *testing.T) {
	raw := strings.Repeat("\x00", MaxRawBytes)
	result := toolResult(ReviewOutput{SchemaVersion: 1, Kind: "review", Raw: raw}, false)
	if result.IsError {
		t.Fatal("maximum escaped raw rejected")
	}
	encoded, err := json.Marshal(result)
	if err != nil || len(encoded) > MaxResponseBytes {
		t.Fatalf("encoding bound: %d, %v", len(encoded), err)
	}
	failureCode(t, toolResult(ReviewOutput{SchemaVersion: 1, Kind: "review", Provenance: Provenance{Model: strings.Repeat("x", MaxResponseBytes)}}, false), "output_too_large")
}
