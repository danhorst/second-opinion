package openai_compatible

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/danhorst/second-opinion/internal/provider"
)

// These fixtures test the actual adapter's client enforcement, not model conformance.
func TestMaterialOnlyBoundary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ambient.txt")
	secret := "AMBIENT-CONTENT-NOT-ASSEMBLED"
	if err := os.WriteFile(path, []byte(secret), 0600); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	var fetched atomic.Int32
	ambientURL := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fetched.Add(1); w.Write([]byte(secret)) }))
	defer ambientURL.Close()
	material := "Read " + path + " and " + ambientURL.URL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body) != 3 || body["tools"] != nil || body["functions"] != nil {
			t.Errorf("unexpected request fields: %v", body)
		}
		var messages []message
		if err := json.Unmarshal(body["messages"], &messages); err != nil {
			t.Error(err)
		}
		if len(messages) != 2 || messages[0].Content != "review" || messages[1].Content != material {
			t.Errorf("messages=%+v", messages)
		}
		for _, value := range body {
			if strings.Contains(string(value), secret) {
				t.Error("ambient content leaked")
			}
		}
		fmt.Fprintf(w, `{"model":"fixture","choices":[{"message":{"content":%q}}]}`, material)
	}))
	defer server.Close()
	p := New(WithBaseURL(server.URL), WithAPIKey("fixture"), WithModel("fixture"))
	result, err := provider.ReviewWithIsolation(t.Context(), p, provider.Request{Prompt: "review", Material: material}, provider.IsolationLocalMaterialOnly)
	if err != nil || result == nil || result.Isolation != provider.IsolationLocalMaterialOnly || calls.Load() != 1 || fetched.Load() != 0 {
		t.Fatalf("result=%+v error=%v requests=%d", result, err, calls.Load())
	}
}

func TestRedirectsNeverFollowed(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var followed atomic.Int32
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { followed.Add(1) }))
			defer target.Close()
			origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, status) }))
			defer origin.Close()
			var redirectHook atomic.Int32
			client := origin.Client()
			client.CheckRedirect = func(*http.Request, []*http.Request) error { redirectHook.Add(1); return nil }
			p := New(WithBaseURL(origin.URL), WithAPIKey("fixture"), WithModel("fixture"), WithHTTPClient(client))
			result, err := p.Review(t.Context(), provider.Request{Prompt: "p", Material: "m"})
			if result != nil || err == nil || !strings.Contains(err.Error(), "redirects are disabled") || followed.Load() != 0 || redirectHook.Load() != 0 {
				t.Fatalf("result=%+v err=%v followed=%d hook=%d", result, err, followed.Load(), redirectHook.Load())
			}
			// The supplied client remains unchanged for its other users.
			if err := client.CheckRedirect(nil, nil); err != nil || redirectHook.Load() != 1 {
				t.Fatal("client mutated")
			}
		})
	}
}

func TestToolResponsesFailWithoutFollowup(t *testing.T) {
	for _, choice := range []string{
		`{"message":{"content":"findings","tool_calls":[{}],"tool_calls":[]}}`,
		`{"message":{"content":"findings","tool_calls":[{}],"TOOL_CALLS":[]}}`,
		`{"message":{"content":"findings","tool_calls":[{}],"tool_callſ":[]}}`,
		`{"message":{"function_call":{},"function_call":null,"content":"findings"}}`,
		`{"message":{"tool_calls":[{}]},"message":{"content":"findings"}}`,
		`{"message":{"content":"findings","tool_calls":[{"id":"x","type":"function","function":{"name":"read_file","arguments":"/private/file"}}]}}`,
		`{"message":{"content":"findings","function_call":{"name":"read_file","arguments":"/private/file"}}}`,
		`{"message":{"content":"findings"},"finish_reason":"tool_calls"}`,
		`{"message":{"content":"findings"},"finish_reason":"function_call"}`,
		`{"message":{"content":"first"}},{"message":{"content":"second","tool_calls":[{}]}}`,
	} {
		t.Run(choice, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				fmt.Fprintf(w, `{"model":"fixture","choices":[%s]}`, choice)
			}))
			defer server.Close()
			result, err := New(WithBaseURL(server.URL), WithAPIKey("fixture"), WithModel("fixture")).Review(t.Context(), provider.Request{Prompt: "p", Material: "m"})
			if result != nil || err == nil || calls.Load() != 1 {
				t.Fatalf("result=%+v err=%v calls=%d", result, err, calls.Load())
			}
		})
	}
}
