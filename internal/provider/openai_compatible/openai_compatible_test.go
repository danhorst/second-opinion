package openai_compatible

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/provider/providertest"
)

func TestLocalEndpointConformance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"model":"fixture-model","choices":[{"message":{"role":"assistant","content":"NONE"}}]}`))
	}))
	defer server.Close()

	providertest.Conform(t, func(t *testing.T) provider.Provider {
		return New(WithBaseURL(server.URL), WithAPIKey("fixture-key"), WithModel("fixture-model"), WithHTTPClient(server.Client()))
	})
}

func TestReviewRequestShapeAndProvenance(t *testing.T) {
	var got chatRequest
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		authorization = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"model":"router/actual-model","choices":[{"message":{"role":"assistant","content":"Finding: stale cache."}}]}`))
	}))
	defer server.Close()

	req := provider.Request{Prompt: "review this", Material: "diff text"}
	res, err := New(WithBaseURL(server.URL+"/v1"), WithAPIKey("secret"), WithModel("requested-model"), WithHTTPClient(server.Client())).Review(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if authorization != "Bearer secret" {
		t.Errorf("authorization = %q", authorization)
	}
	if got.Model != "requested-model" || got.Stream || len(got.Messages) != 2 {
		t.Errorf("request = %+v", got)
	}
	if got.Messages[0].Role != "system" || got.Messages[0].Content != req.Prompt || got.Messages[1].Role != "user" || got.Messages[1].Content != req.Material {
		t.Errorf("messages = %+v", got.Messages)
	}
	if res.Findings != "Finding: stale cache." || res.Provenance.Model != "router/actual-model" {
		t.Errorf("result = %+v", res)
	}
	if res.Provenance.PromptHash != provider.HashPrompt(req.Prompt) || res.Provenance.MaterialHash != provider.HashMaterial(req.Material) {
		t.Errorf("provenance = %+v", res.Provenance)
	}
}

func TestReviewUsesUnknownModelMarker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":"OK"}}]}`))
	}))
	defer server.Close()

	res, err := New(WithBaseURL(server.URL), WithAPIKey("key"), WithModel("requested"), WithHTTPClient(server.Client())).Review(context.Background(), provider.Request{Prompt: "p", Material: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Provenance.Model != unknownModelMarker {
		t.Errorf("model = %q", res.Provenance.Model)
	}
}

func TestReviewRejectsMissingConfigurationBeforeTransport(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer server.Close()

	for name, p := range map[string]*Provider{
		"key":   New(WithBaseURL(server.URL), WithModel("model"), WithHTTPClient(server.Client())),
		"model": New(WithBaseURL(server.URL), WithAPIKey("key"), WithHTTPClient(server.Client())),
	} {
		if _, err := p.Review(context.Background(), provider.Request{Prompt: "p", Material: "m"}); err == nil {
			t.Errorf("%s: expected configuration error", name)
		}
	}
	if called {
		t.Error("missing configuration must not make an HTTP request")
	}
}

func TestReviewUsesEnvironmentModel(t *testing.T) {
	t.Setenv("SECOND_OPINION_API_MODEL", "env-model")
	t.Setenv("SECOND_OPINION_API_KEY", "env-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body chatRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.Model != "env-model" {
			t.Errorf("model = %q", body.Model)
		}
		w.Write([]byte(`{"model":"env-model","choices":[{"message":{"content":"OK"}}]}`))
	}))
	defer server.Close()
	t.Setenv("SECOND_OPINION_API_BASE_URL", server.URL)

	if _, err := New(WithHTTPClient(server.Client())).Review(context.Background(), provider.Request{Prompt: "p", Material: "m"}); err != nil {
		t.Fatal(err)
	}
}

func TestReviewClassifiesHTTPFailures(t *testing.T) {
	for _, tc := range []struct {
		status int
		match  error
	}{
		{http.StatusUnauthorized, provider.ErrAuth},
		{http.StatusForbidden, provider.ErrAuth},
		{http.StatusRequestTimeout, provider.ErrUnavailable},
		{http.StatusTooManyRequests, provider.ErrUnavailable},
		{http.StatusBadGateway, provider.ErrUnavailable},
	} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				w.Write([]byte("diagnostic"))
			}))
			defer server.Close()
			_, err := New(WithBaseURL(server.URL), WithAPIKey("key"), WithModel("model"), WithHTTPClient(server.Client())).Review(context.Background(), provider.Request{Prompt: "p", Material: "m"})
			if !errors.Is(err, tc.match) {
				t.Errorf("error = %v, want %v", err, tc.match)
			}
		})
	}
}

func TestReviewClassifiesTransportFailure(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	})}
	_, err := New(WithBaseURL("http://endpoint.invalid"), WithAPIKey("key"), WithModel("model"), WithHTTPClient(client)).Review(context.Background(), provider.Request{Prompt: "p", Material: "m"})
	if !errors.Is(err, provider.ErrUnavailable) {
		t.Errorf("error = %v, want ErrUnavailable", err)
	}
}

func TestReviewBoundsDiagnostics(t *testing.T) {
	diagnostic := strings.Repeat("x", maxDiagnosticBytes+100)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(diagnostic))
	}))
	defer server.Close()

	_, err := New(WithBaseURL(server.URL), WithAPIKey("key"), WithModel("model"), WithHTTPClient(server.Client())).Review(context.Background(), provider.Request{Prompt: "p", Material: "m"})
	if err == nil || len(err.Error()) > maxDiagnosticBytes+100 {
		t.Errorf("diagnostic was not bounded: %v", err)
	}
}

func TestReviewRejectsMalformedSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	res, err := New(WithBaseURL(server.URL), WithAPIKey("key"), WithModel("model"), WithHTTPClient(server.Client())).Review(context.Background(), provider.Request{Prompt: "p", Material: "m"})
	if err == nil || res != nil {
		t.Errorf("malformed success: result=%+v error=%v", res, err)
	}
}

func TestReviewCancellation(t *testing.T) {
	started := make(chan struct{})
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		close(started)
		<-r.Context().Done()
		return nil, r.Context().Err()
	})}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := New(WithBaseURL("http://endpoint.invalid"), WithAPIKey("key"), WithModel("model"), WithHTTPClient(client)).Review(ctx, provider.Request{Prompt: "p", Material: "m"})
		done <- err
	}()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
