package mcpserver

import "github.com/danhorst/second-opinion/internal/review"

func objectSchema(properties map[string]any, required ...string) map[string]any {
	if required == nil {
		required = []string{}
	}
	return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
}

func textSchema() map[string]any { return map[string]any{"type": "string", "minLength": 1} }
func integerSchema(minimum, maximum int) map[string]any {
	return map[string]any{"type": "integer", "minimum": minimum, "maximum": maximum}
}
func constantSchema(value any) map[string]any { return map[string]any{"const": value} }
func hashSchema() map[string]any {
	return map[string]any{"type": "string", "pattern": "^[0-9a-f]{64}$"}
}

// The runtime validator additionally enforces UTF-8 byte budgets and nonblank
// Unicode text; JSON Schema string lengths count code points, not bytes.
var ReviewInputSchema = objectSchema(map[string]any{
	"require_isolation": map[string]any{"type": "string", "enum": []string{"local-material-only"}, "description": "Require a client-side material boundary. The configured endpoint and transport remain trusted; this does not certify endpoint context or OS isolation."},
	"provider":          map[string]any{"type": "string", "enum": review.Providers(), "description": "Exact registered provider; no inherited default."},
	"material":          map[string]any{"type": "string", "minLength": 1, "description": "Complete nonblank text to review. Combined with the resolved prompt, at most 1048576 UTF-8 bytes. Paths are text, never opened."},
	"model":             map[string]any{"type": "string", "minLength": 1, "maxLength": MaxModelBytes, "description": "Nonblank model name, at most 256 UTF-8 bytes."},
	"prompt":            map[string]any{"type": "string", "minLength": 1, "description": "Nonblank replacement prompt; omitted selects the baked prompt."},
	"timeout_ms":        integerSchema(1, MaxTimeoutMS),
}, "provider", "material")

var EmptyInputSchema = objectSchema(map[string]any{})

func errorSchema() map[string]any {
	return objectSchema(map[string]any{
		"schema_version": constantSchema(2), "kind": constantSchema("error"),
		"error": objectSchema(map[string]any{
			"code":      map[string]any{"type": "string", "enum": []string{"invalid_input", "input_too_large", "busy", "unavailable", "auth", "timeout", "cancelled", "output_too_large", "provider_failed", "isolation_unsupported"}},
			"message":   textSchema(),
			"execution": map[string]any{"type": "string", "enum": []string{"not_started", "unknown"}},
		}, "code", "message", "execution"),
	}, "schema_version", "kind", "error")
}

var ReviewOutputSchema = map[string]any{"type": "object", "oneOf": []any{
	objectSchema(map[string]any{
		"schema_version": constantSchema(2), "kind": constantSchema("review"), "raw": map[string]any{"type": "string"},
		"isolation":  isolationSchema(),
		"provenance": objectSchema(map[string]any{"provider": textSchema(), "model": textSchema(), "prompt_hash": hashSchema(), "material_hash": hashSchema()}, "provider", "model", "prompt_hash", "material_hash"),
		"request":    objectSchema(map[string]any{"material_hash": hashSchema(), "material_bytes": integerSchema(1, MaxInputBytes), "timeout_ms": integerSchema(1, MaxTimeoutMS)}, "material_hash", "material_bytes", "timeout_ms"),
	}, "schema_version", "kind", "raw", "provenance", "request", "isolation"),
	errorSchema(),
}}

func limitsSchema() map[string]any {
	properties := map[string]any{
		"max_request_bytes": constantSchema(MaxRequestBytes), "max_input_bytes": constantSchema(MaxInputBytes),
		"max_model_bytes": constantSchema(MaxModelBytes), "default_timeout_ms": constantSchema(DefaultTimeoutMS),
		"max_timeout_ms": constantSchema(MaxTimeoutMS), "max_concurrent_reviews": constantSchema(MaxConcurrentReviews),
		"max_stdout_bytes": constantSchema(MaxStdoutBytes), "max_stderr_bytes": constantSchema(MaxStderrBytes),
		"max_raw_bytes": constantSchema(MaxRawBytes), "max_response_bytes": constantSchema(MaxResponseBytes),
		"max_inflight_messages": constantSchema(MaxInflightMessages),
	}
	return objectSchema(properties, "max_request_bytes", "max_input_bytes", "max_model_bytes", "default_timeout_ms", "max_timeout_ms", "max_concurrent_reviews", "max_stdout_bytes", "max_stderr_bytes", "max_raw_bytes", "max_response_bytes", "max_inflight_messages")
}

var ProvidersOutputSchema = map[string]any{"type": "object", "oneOf": []any{
	objectSchema(map[string]any{
		"schema_version": constantSchema(2), "kind": constantSchema("providers"),
		"providers":          map[string]any{"type": "array", "items": textSchema(), "uniqueItems": true},
		"limits":             limitsSchema(),
		"provider_isolation": providerIsolationSchema(),
	}, "schema_version", "kind", "providers", "limits", "provider_isolation"),
	errorSchema(),
}}

func isolationSchema() map[string]any {
	return map[string]any{"type": "string", "enum": []string{"unverified", "local-material-only"}, "description": "Local client boundary only; endpoint and transport remain trusted. Unverified does not certify a read boundary."}
}

func providerIsolationSchema() map[string]any {
	properties := make(map[string]any)
	for _, name := range review.Providers() {
		properties[name] = isolationSchema()
	}
	return objectSchema(properties, review.Providers()...)
}
