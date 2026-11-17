package mcpclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/danhorst/second-opinion/internal/mcpserver"
	"github.com/danhorst/second-opinion/internal/review"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestValidateURL(t *testing.T) {
	for _, s := range []string{"https://127.0.0.1/mcp", "http://localhost/mcp", "http://127.0.0.1/mcp?secret", "http://u@127.0.0.1/mcp", "http://127.0.0.1/mcp#x", "http://127.0.0.1/other", "http://127.0.0.2/mcp", "http://[::ffff:127.0.0.2]/mcp", "http://[::ffff:127.0.0.1]/mcp", "http://127.0.0.1:0/mcp", "http://127.0.0.1:65536/mcp", "http://127.0.0.1:/mcp"} {
		if ValidateURL(s) == nil {
			t.Errorf("accepted %s", s)
		}
	}
}

func TestBridgeRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing auth")
		}
		var msg map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&msg)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": msg["id"], "result": map[string]any{"tools": []any{}}})
	}))
	defer server.Close()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Bridge(ctx, server.URL+"/mcp", "test-token", inR, outW); outW.Close() }()
	io.WriteString(inW, "{\"jsonrpc\":\"2.0\",\"id\":\"abc\",\"method\":\"tools/list\"}\n")
	line, err := bufio.NewReader(outR).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	var msg struct {
		ID string `json:"id"`
	}
	if json.Unmarshal([]byte(line), &msg) != nil || msg.ID != "abc" {
		t.Fatalf("invalid response %s", line)
	}
	inW.Close()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestSDKDiscovery(t *testing.T) {
	for _, version := range []string{"2026-07-28", "2025-11-25"} {
		t.Run(version, func(t *testing.T) {
			token := strings.Repeat("x", 32)
			server := httptest.NewUnstartedServer(nil)
			h, err := mcpserver.NewHTTPHandler(mcpserver.NewServer(review.Executor{}), server.Listener.Addr().String(), token)
			if err != nil {
				t.Fatal(err)
			}
			server.Config.Handler = h
			server.Start()
			defer server.Close()
			inR, inW := io.Pipe()
			outR, outW := io.Pipe()
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- Bridge(ctx, server.URL+"/mcp", token, inR, outW) }()
			client, err := mcp.NewClient(&mcp.Implementation{Name: "bridge-test", Version: "1"}, nil).Connect(ctx, &mcp.IOTransport{Reader: outR, Writer: inW}, &mcp.ClientSessionOptions{ProtocolVersion: version})
			if err != nil {
				t.Fatal(err)
			}
			result, err := client.ListTools(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Tools) == 0 {
				t.Fatal("no tools")
			}
			client.Close()
			inW.Close()
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-ctx.Done():
				t.Fatal("bridge did not stop")
			}
		})
	}
}

func TestRedactedFailure(t *testing.T) {
	for _, kind := range []string{"wrong-id", "oversize", "redirect"} {
		t.Run(kind, func(t *testing.T) {
			leaked := make(chan bool, 1)
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked <- true }))
			defer target.Close()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch kind {
				case "wrong-id":
					io.WriteString(w, `{"jsonrpc":"2.0","id":999,"result":{"secret":"sensitive"}}`)
				case "oversize":
					io.WriteString(w, strings.Repeat("x", mcpserver.MaxResponseBytes+1))
				case "redirect":
					http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
				}
			}))
			defer server.Close()
			inR, inW := io.Pipe()
			outR, outW := io.Pipe()
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- Bridge(ctx, server.URL+"/mcp", "secret-token", inR, outW) }()
			io.WriteString(inW, `{"jsonrpc":"2.0","id":"original","method":"tools/list"}`+"\n")
			line, err := bufio.NewReader(outR).ReadString('\n')
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(line, `"id":"original"`) || !strings.Contains(line, "local MCP request failed") || strings.Contains(line, "secret") {
				t.Fatalf("bad failure: %s", line)
			}
			select {
			case <-leaked:
				t.Fatal("redirect followed")
			default:
			}
			inW.Close()
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestActiveCancellation(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		close(started)
		<-r.Context().Done()
		close(cancelled)
	}))
	defer server.Close()
	inR, inW := io.Pipe()
	var output bytes.Buffer
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Bridge(ctx, server.URL+"/mcp", "token", inR, &output) }()
	io.WriteString(inW, `{"jsonrpc":"2.0","id":9007199254740993,"method":"tools/list"}`+"\n")
	<-started
	io.WriteString(inW, `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":9007199254740993}}`+"\n")
	select {
	case <-cancelled:
	case <-ctx.Done():
		t.Fatal("HTTP not cancelled")
	}
	inW.Close()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatalf("cancelled response emitted: %s", output.String())
	}
}

func TestAdmissionAndEOF(t *testing.T) {
	for _, kind := range []string{"duplicate", "saturation", "EOF"} {
		t.Run(kind, func(t *testing.T) {
			started := make(chan struct{}, 8)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				started <- struct{}{}
				<-r.Context().Done()
			}))
			defer server.Close()
			inR, inW := io.Pipe()
			var output bytes.Buffer
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- Bridge(ctx, server.URL+"/mcp", "token", inR, &output) }()
			count := 1
			if kind == "saturation" {
				count = mcpserver.MaxInflightMessages
			}
			for i := 0; i < count; i++ {
				fmt.Fprintf(inW, `{"jsonrpc":"2.0","id":%d,"method":"tools/list"}`+"\n", i)
				<-started
			}
			if kind == "duplicate" {
				io.WriteString(inW, `{"jsonrpc":"2.0","id":0,"method":"tools/list"}`+"\n")
			} else if kind == "saturation" {
				io.WriteString(inW, `{"jsonrpc":"2.0","id":99,"method":"tools/list"}`+"\n")
			} else {
				inW.Close()
			}
			select {
			case err := <-done:
				if (err != nil) != (kind != "EOF") {
					t.Fatalf("unexpected error: %v", err)
				}
			case <-ctx.Done():
				t.Fatal("bridge did not stop")
			}
			inW.Close()
			if output.Len() != 0 {
				t.Fatal("shutdown emitted response")
			}
		})
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func TestOutputFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":{}}`)
	}))
	defer server.Close()
	inR, inW := io.Pipe()
	defer inW.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Bridge(ctx, server.URL+"/mcp", "token", inR, brokenWriter{}) }()
	io.WriteString(inW, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("output failure accepted")
		}
	case <-ctx.Done():
		t.Fatal("bridge did not stop")
	}
}
