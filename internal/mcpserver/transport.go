package mcpserver

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var errTransportLimit = errors.New("MCP transport limit exceeded")

// cancellationContext survives the SDK legacy connection context, which
// deliberately detaches cancellation while retaining context values.
type cancellationContext struct{}

func bindCancellation(server *mcp.Server) {
	server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if parent, ok := ctx.Value(cancellationContext{}).(context.Context); ok {
				child, cancel := context.WithCancel(ctx)
				stop := context.AfterFunc(parent, cancel)
				defer stop()
				defer cancel()
				if parent.Err() != nil {
					cancel()
				}
				ctx = child
			}
			return next(ctx, method, req)
		}
	})
}

type stdioTransport struct{ conn *stdioConnection }

func (t *stdioTransport) Connect(context.Context) (mcp.Connection, error) { return t.conn, nil }
func (*stdioTransport) SupportsProtocolVersion(v string) bool {
	return v == "2026-07-28" || v == "2025-11-25"
}

type stdioConnection struct {
	reader      io.ReadCloser
	writer      io.WriteCloser
	scanner     *bufio.Scanner
	mu          sync.Mutex
	writeMu     sync.Mutex
	active      map[jsonrpc.ID]bool // whether explicit client cancellation was received
	modern      bool
	initialized bool
	rejected    chan *jsonrpc.Response
	closed      chan struct{}
	closeOnce   sync.Once
	cancel      context.CancelFunc
}

func newStdioConnection(r io.ReadCloser, w io.WriteCloser, cancel context.CancelFunc) *stdioConnection {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), MaxRequestBytes+2)
	c := &stdioConnection{reader: r, writer: w, scanner: scanner, active: make(map[jsonrpc.ID]bool), modern: true, rejected: make(chan *jsonrpc.Response, MaxInflightMessages), closed: make(chan struct{}), cancel: cancel}
	go func() {
		for {
			select {
			case response := <-c.rejected:
				if c.write(response) != nil {
					c.Close()
					return
				}
			case <-c.closed:
				return
			}
		}
	}()
	return c
}

func (c *stdioConnection) SessionID() string { return "" }
func (c *stdioConnection) Close() error {
	var err error
	c.closeOnce.Do(func() { close(c.closed); c.cancel(); err = errors.Join(c.reader.Close(), c.writer.Close()) })
	return err
}
func (c *stdioConnection) Read(ctx context.Context) (message jsonrpc.Message, readErr error) {
	defer func() {
		if readErr != nil {
			c.cancel()
		}
	}()
	for c.scanner.Scan() {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		raw := c.scanner.Bytes()
		if len(raw) > MaxRequestBytes {
			return nil, errTransportLimit
		}
		if ValidateJSON(raw) != nil {
			return nil, errors.New("invalid MCP JSON frame")
		}
		msg, err := jsonrpc.DecodeMessage(raw)
		if err != nil {
			return nil, errors.New("invalid MCP protocol frame")
		}
		req, ok := msg.(*jsonrpc.Request)
		if !ok {
			continue
		} // The gateway never makes client requests.
		c.mu.Lock()
		if !req.IsCall() {
			switch req.Method {
			case "notifications/cancelled":
				var p struct {
					RequestID any `json:"requestId"`
				}
				if json.Unmarshal(req.Params, &p) != nil {
					c.mu.Unlock()
					continue
				}
				id, e := jsonrpc.MakeID(p.RequestID)
				cancelled, exists := c.active[id]
				if e != nil || !exists || cancelled {
					c.mu.Unlock()
					continue
				}
				c.active[id] = true
			case "notifications/initialized":
				if c.initialized {
					c.mu.Unlock()
					continue
				}
				c.initialized = true
			default:
				c.mu.Unlock()
				continue
			}
			c.mu.Unlock()
			return msg, nil
		}
		if _, exists := c.active[req.ID]; exists {
			c.mu.Unlock()
			return nil, errors.New("duplicate active MCP request identifier")
		}
		if len(c.active) >= MaxInflightMessages {
			c.mu.Unlock()
			response := &jsonrpc.Response{ID: req.ID, Error: &jsonrpc.Error{Code: -32000, Message: "MCP transport is busy"}}
			select {
			case c.rejected <- response:
				continue
			default:
				return nil, errTransportLimit
			}
		}
		c.active[req.ID] = false
		if req.Method == "initialize" {
			var p struct {
				ProtocolVersion string `json:"protocolVersion"`
			}
			if json.Unmarshal(req.Params, &p) == nil {
				c.modern = p.ProtocolVersion != "2025-11-25"
			}
		}
		c.mu.Unlock()
		return msg, nil
	}
	c.cancel() // EOF cancels reviews before waiting for the SDK to drain.
	if err := c.scanner.Err(); err != nil {
		return nil, errors.New("invalid or oversized MCP frame")
	}
	return nil, io.EOF
}
func (c *stdioConnection) Write(ctx context.Context, msg jsonrpc.Message) error {
	if response, ok := msg.(*jsonrpc.Response); ok {
		c.mu.Lock()
		cancelled := c.active[response.ID]
		modern := c.modern
		c.mu.Unlock()
		defer func() { c.mu.Lock(); delete(c.active, response.ID); c.mu.Unlock() }()
		if cancelled && modern {
			return nil
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return c.write(msg)
}
func (c *stdioConnection) write(msg jsonrpc.Message) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	raw, err := jsonrpc.EncodeMessage(msg)
	if err != nil {
		return errors.New("cannot encode MCP response")
	}
	if len(raw) > MaxResponseBytes {
		return errTransportLimit
	}
	_, err = c.writer.Write(append(raw, '\n'))
	return err
}

// ServeStdio owns its streams until EOF or shutdown. Request admission happens
// before the SDK queue; cancellation notifications bypass saturated call slots.
func ServeStdio(ctx context.Context, server *mcp.Server, r io.ReadCloser, w io.WriteCloser) error {
	bindCancellation(server)
	lifetime, cancel := context.WithCancel(ctx)
	defer cancel()
	conn := newStdioConnection(r, w, cancel)
	defer conn.Close()
	session, err := server.Connect(context.WithValue(lifetime, cancellationContext{}, lifetime), &stdioTransport{conn}, nil)
	if err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- session.Wait() }()
	select {
	case err = <-done:
		return err
	case <-lifetime.Done():
		go session.Close()
		conn.Close()
		select {
		case err = <-done:
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		case <-time.After(5 * time.Second):
			return errors.New("MCP shutdown deadline exceeded")
		}
	}
}

// ValidateListenAddress accepts only literal IPv4/IPv6 loopback authorities.
func ValidateListenAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil || (host != "127.0.0.1" && host != "::1") {
		return errors.New("MCP HTTP requires a literal loopback address and port")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 0 || n > 65535 {
		return errors.New("invalid MCP HTTP port")
	}
	return nil
}

// NewHTTPHandler authenticates and bounds each request before the SDK sees it.
// The authority must be the actual listening socket authority, including port.
func NewHTTPHandler(server *mcp.Server, authority, token string) (http.Handler, error) {
	if err := ValidateListenAddress(authority); err != nil {
		return nil, err
	}
	if len(token) < 32 {
		return nil, errors.New("SECOND_OPINION_MCP_TOKEN must contain at least 32 bytes")
	}
	bindCancellation(server)
	sdk := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, PropagateRequestCancellation: true})
	expected := sha256.Sum256([]byte("Bearer " + token))
	slots := make(chan struct{}, MaxInflightMessages)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		credential := sha256.Sum256([]byte(r.Header.Get("Authorization")))
		if subtle.ConstantTimeCompare(credential[:], expected[:]) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if _, present := r.Header["Origin"]; present {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if r.Host != authority {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if r.URL.Path != "/mcp" || r.URL.RawQuery != "" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
		default:
			http.Error(w, "MCP transport is busy", http.StatusServiceUnavailable)
			return
		}
		// ReadTimeout also protects production sockets; setting this deadline keeps
		// the same bound when this handler is mounted on another HTTP server.
		controller := http.NewResponseController(w)
		_ = controller.SetReadDeadline(time.Now().Add(10 * time.Second))
		raw, err := io.ReadAll(io.LimitReader(r.Body, MaxRequestBytes+1))
		r.Body.Close()
		if err != nil {
			http.Error(w, "Cannot read MCP request", http.StatusBadRequest)
			return
		}
		if len(raw) > MaxRequestBytes {
			http.Error(w, "MCP request too large", http.StatusRequestEntityTooLarge)
			return
		}
		if ValidateJSON(raw) != nil {
			http.Error(w, "Invalid MCP JSON", http.StatusBadRequest)
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), cancellationContext{}, r.Context()))
		r.Body = io.NopCloser(bytes.NewReader(raw))
		r.ContentLength = int64(len(raw))
		r.Header.Del("Mcp-Session-Id") // No opt-in SDK legacy environment can add affinity.
		out := &boundedHTTPResponse{header: make(http.Header)}
		sdk.ServeHTTP(out, r)
		if out.overflow {
			http.Error(w, "MCP response too large", http.StatusInternalServerError)
			return
		}
		for key, values := range out.header {
			if strings.EqualFold(key, "Mcp-Session-Id") {
				continue
			}
			w.Header()[key] = values
		}
		if out.status == 0 {
			out.status = http.StatusOK
		}
		w.WriteHeader(out.status)
		_, _ = w.Write(out.body.Bytes())
	}), nil
}

type boundedHTTPResponse struct {
	header   http.Header
	body     bytes.Buffer
	status   int
	overflow bool
}

func (w *boundedHTTPResponse) Header() http.Header { return w.header }
func (w *boundedHTTPResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}
func (w *boundedHTTPResponse) Write(p []byte) (int, error) {
	if w.overflow || len(p) > MaxResponseBytes-w.body.Len() {
		w.overflow = true
		return 0, errTransportLimit
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(p)
}

// JSONResponse does not stream. Flush is supported to let SDK response
// controllers complete without sending unbounded partial output to the client.
func (w *boundedHTTPResponse) Flush() {}

func ServeHTTP(ctx context.Context, server *mcp.Server, address, token string) error {
	if err := ValidateListenAddress(address); err != nil {
		return err
	}
	if len(token) < 32 {
		return errors.New("SECOND_OPINION_MCP_TOKEN must contain at least 32 bytes")
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return errors.New("cannot listen on MCP loopback address")
	}
	defer listener.Close()
	handler, err := NewHTTPHandler(server, listener.Addr().String(), token)
	if err != nil {
		return err
	}
	httpServer := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 310 * time.Second, IdleTimeout: 30 * time.Second, BaseContext: func(net.Listener) context.Context { return ctx }}
	done := make(chan error, 1)
	go func() { done <- httpServer.Serve(listener) }()
	select {
	case err = <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		drain, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = httpServer.Shutdown(drain)
		if err != nil {
			_ = httpServer.Close()
		}
		return err
	}
}
