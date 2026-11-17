// Package mcpclient connects newline JSON RPC clients to the private stateless server.
package mcpclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/danhorst/second-opinion/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

const MaxFrame = mcpserver.MaxRequestBytes

func ValidateURL(raw string) error {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Path != "/mcp" || u.RawPath != "" {
		return errors.New("endpoint must be literal loopback HTTP /mcp")
	}
	if (u.Hostname() != "127.0.0.1" && u.Hostname() != "::1") || u.Opaque != "" {
		return errors.New("endpoint must be literal loopback HTTP /mcp")
	}
	if u.Port() != "" {
		port, err := strconv.Atoi(u.Port())
		if err != nil || port < 1 || port > 65535 {
			return errors.New("invalid local MCP port")
		}
	} else if strings.HasSuffix(u.Host, ":") {
		return errors.New("invalid local MCP port")
	}
	return nil
}

// Bridge forwards bounded frames. Closing input or cancelling ctx stops outstanding requests.
func Bridge(ctx context.Context, endpoint, token string, input io.ReadCloser, output io.Writer) error {
	if err := ValidateURL(endpoint); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 310 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect refused") }}
	defer client.CloseIdleConnections()
	var wg sync.WaitGroup
	var mu, out sync.Mutex
	var asyncErr error
	version := "2026-07-28"
	fail := func() { mu.Lock(); asyncErr = errors.New("MCP transport failed"); mu.Unlock(); cancel() }
	active := map[jsonrpc.ID]context.CancelFunc{}
	slots := make(chan struct{}, mcpserver.MaxInflightMessages)
	cleanupDone := make(chan struct{})
	go func() {
		defer close(cleanupDone)
		<-ctx.Done()
		input.Close()
		if closer, ok := output.(io.Closer); ok {
			closer.Close()
		}
	}()
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), MaxFrame+2)
	var failure error
	for scanner.Scan() {
		frame := append([]byte(nil), scanner.Bytes()...)
		var msg struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Method  string          `json:"method"`
			Params  struct {
				ID json.RawMessage `json:"requestId"`
			} `json:"params"`
		}
		if len(frame) > MaxFrame || mcpserver.ValidateJSON(frame) != nil || json.Unmarshal(frame, &msg) != nil || msg.JSONRPC != "2.0" {
			failure = errors.New("invalid JSON RPC frame")
			break
		}
		decoded, decodeErr := jsonrpc.DecodeMessage(frame)
		request, requestOK := decoded.(*jsonrpc.Request)
		if decodeErr != nil || !requestOK {
			failure = errors.New("invalid MCP request")
			break
		}
		if msg.Method == "notifications/cancelled" {
			// Decode IDs through JSON RPC itself to preserve integers beyond float64 precision.
			cancelMessage, idErr := jsonrpc.DecodeMessage(append(append([]byte(`{"jsonrpc":"2.0","result":{},"id":`), msg.Params.ID...), '}'))
			cancelResponse, ok := cancelMessage.(*jsonrpc.Response)
			if idErr != nil || !ok {
				continue
			}
			cancelID := cancelResponse.ID
			mu.Lock()
			if stop := active[cancelID]; stop != nil {
				stop()
			}
			mu.Unlock()
			continue
		}
		select {
		case slots <- struct{}{}:
		default:
			failure = errors.New("too many outstanding requests")
		}
		if failure != nil {
			break
		}
		requestCtx, stop := context.WithCancel(ctx)
		key := request.ID
		hasID := request.IsCall()
		mu.Lock()
		if hasID {
			if _, ok := active[key]; ok {
				mu.Unlock()
				stop()
				<-slots
				failure = errors.New("duplicate request ID")
				break
			}
			active[key] = stop
		}
		mu.Unlock()
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				<-slots
				stop()
				mu.Lock()
				if hasID {
					delete(active, key)
				}
				mu.Unlock()
			}()
			req, err := http.NewRequestWithContext(requestCtx, "POST", endpoint, bytes.NewReader(frame))
			if err != nil {
				return
			}
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			mu.Lock()
			wireVersion := version
			if msg.Method == "initialize" {
				var init struct {
					Params struct {
						Version string `json:"protocolVersion"`
					} `json:"params"`
				}
				_ = json.Unmarshal(frame, &init)
				wireVersion = init.Params.Version
			}
			req.Header.Set("MCP-Protocol-Version", wireVersion)
			req.Header.Set("Mcp-Method", msg.Method)
			var named struct {
				Params struct {
					Name string `json:"name"`
					URI  string `json:"uri"`
				} `json:"params"`
			}
			_ = json.Unmarshal(frame, &named)
			if msg.Method == "resources/read" {
				req.Header.Set("Mcp-Name", named.Params.URI)
			} else if named.Params.Name != "" {
				req.Header.Set("Mcp-Name", named.Params.Name)
			}
			mu.Unlock()
			resp, err := client.Do(req)
			var body []byte
			if err == nil {
				body, err = io.ReadAll(io.LimitReader(resp.Body, mcpserver.MaxResponseBytes+1))
				resp.Body.Close()
				if resp.StatusCode < 200 || resp.StatusCode >= 300 || len(body) > mcpserver.MaxResponseBytes {
					err = errors.New("HTTP request failed")
				}
			}
			if !hasID {
				if err != nil && ctx.Err() == nil {
					fail()
				}
				return
			}
			var reply struct {
				ID     json.RawMessage `json:"id"`
				Result struct {
					Version string `json:"protocolVersion"`
				} `json:"result"`
			}
			if err == nil && (mcpserver.ValidateJSON(body) != nil || json.Unmarshal(body, &reply) != nil || !matchingResponse(body, key)) {
				err = errors.New("invalid response")
			}
			if err == nil && msg.Method == "initialize" && reply.Result.Version != "" {
				mu.Lock()
				version = reply.Result.Version
				mu.Unlock()
			}
			mu.Lock()
			modern := version != "2025-11-25"
			mu.Unlock()
			if ctx.Err() != nil || (requestCtx.Err() != nil && modern) {
				return
			}
			if err != nil {
				body, _ = json.Marshal(struct {
					JSONRPC string          `json:"jsonrpc"`
					ID      json.RawMessage `json:"id"`
					Error   any             `json:"error"`
				}{"2.0", msg.ID, map[string]any{"code": -32000, "message": "local MCP request failed"}})
			}
			compact := new(bytes.Buffer)
			if json.Compact(compact, body) != nil {
				return
			}
			out.Lock()
			if ctx.Err() != nil || (requestCtx.Err() != nil && modern) {
				out.Unlock()
				return
			}
			var n int
			line := append(compact.Bytes(), '\n')
			n, err = output.Write(line)
			if err == nil && n != len(line) {
				err = io.ErrShortWrite
			}
			out.Unlock()
			if err != nil && ctx.Err() == nil {
				fail()
			}
		}()
	}
	readErr := scanner.Err()
	wasCancelled := ctx.Err() != nil
	cancel()
	wg.Wait()
	<-cleanupDone
	if asyncErr != nil {
		return asyncErr
	}
	if failure != nil {
		return failure
	}
	if readErr != nil && !wasCancelled {
		return errors.New("cannot read MCP frame")
	}
	return nil
}

func matchingResponse(body []byte, id jsonrpc.ID) bool {
	msg, err := jsonrpc.DecodeMessage(body)
	if err != nil {
		return false
	}
	resp, ok := msg.(*jsonrpc.Response)
	return ok && resp.ID == id
}
