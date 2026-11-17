package mcpserver

import (
	"encoding/json"
	"testing"

	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/review"
	"github.com/google/jsonschema-go/jsonschema"
)

func TestSchemasValidateContract(t *testing.T) {
	failure := ErrorOutput{1, "error", ToolError{"busy", "Both review slots are occupied.", "not_started"}}
	cases := []struct {
		name           string
		schema         map[string]any
		valid, invalid []any
	}{
		{"empty", EmptyInputSchema, []any{map[string]any{}}, []any{nil, map[string]any{"x": 1}}},
		{"input", ReviewInputSchema, []any{map[string]any{"provider": "openai-compatible", "material": "x", "require_isolation": "local-material-only"}, map[string]any{"provider": "codex", "material": "x"}, map[string]any{"provider": "claude", "material": "x", "timeout_ms": 300000}}, []any{map[string]any{"provider": "codex", "material": "x", "require_isolation": nil}, map[string]any{"provider": "codex", "material": "x", "require_isolation": "unverified"}, map[string]any{"provider": "bad", "material": "x"}, map[string]any{"provider": "codex", "material": "x", "unknown": 1}, map[string]any{"provider": "codex", "material": "x", "prompt": nil}}},
		{"review", ReviewOutputSchema, []any{ReviewOutput{"unverified", 1, "review", "", Provenance{"codex", "model", provider.HashPrompt("p"), provider.HashMaterial("x")}, Receipt{provider.HashPrompt("x"), 1, 120000}}, ErrorOutput{1, "error", ToolError{"isolation_unsupported", "Unsupported local boundary", "not_started"}}, failure}, []any{map[string]any{"schema_version": 1, "kind": "review", "raw": "x"}, map[string]any{"schema_version": 2, "kind": "error", "error": failure.Error}}},
		{"providers", ProvidersOutputSchema, []any{ProvidersOutput{review.ProviderIsolation(), 1, "providers", review.Providers(), EffectiveLimits()}, failure}, []any{map[string]any{"schema_version": 1, "kind": "providers", "providers": review.Providers()}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.schema)
			if err != nil {
				t.Fatal(err)
			}
			var schema jsonschema.Schema
			if err = json.Unmarshal(raw, &schema); err != nil {
				t.Fatal(err)
			}
			resolved, err := schema.Resolve(nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, valid := range tc.valid {
				b, _ := json.Marshal(valid)
				var v any
				json.Unmarshal(b, &v)
				if err := resolved.Validate(v); err != nil {
					t.Errorf("valid %s: %v", b, err)
				}
			}
			for _, invalid := range tc.invalid {
				b, _ := json.Marshal(invalid)
				var v any
				json.Unmarshal(b, &v)
				if err := resolved.Validate(v); err == nil {
					t.Errorf("accepted invalid %s", b)
				}
			}
		})
	}
}
