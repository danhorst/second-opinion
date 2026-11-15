// Package mcpserver provides the stateless MCP review gateway.
package mcpserver

const (
	MaxRequestBytes      = 8 << 20
	MaxInputBytes        = 1 << 20
	MaxModelBytes        = 256
	DefaultTimeoutMS     = 120000
	MaxTimeoutMS         = 300000
	MaxConcurrentReviews = 2
	MaxStdoutBytes       = 8 << 20
	MaxStderrBytes       = 64 << 10
	MaxRawBytes          = 1 << 20
	MaxResponseBytes     = 16 << 20
	MaxInflightMessages  = 4
)

type ReviewInput struct {
	Provider  string
	Material  string
	Model     string
	Prompt    *string
	TimeoutMS int
}

type ToolError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Execution string `json:"execution"`
}

type Provenance struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	PromptHash string `json:"prompt_hash"`
}

type Receipt struct {
	MaterialHash  string `json:"material_hash"`
	MaterialBytes int    `json:"material_bytes"`
	TimeoutMS     int    `json:"timeout_ms"`
}

type ReviewOutput struct {
	SchemaVersion int        `json:"schema_version"`
	Kind          string     `json:"kind"`
	Raw           string     `json:"raw"`
	Provenance    Provenance `json:"provenance"`
	Request       Receipt    `json:"request"`
}

type ErrorOutput struct {
	SchemaVersion int       `json:"schema_version"`
	Kind          string    `json:"kind"`
	Error         ToolError `json:"error"`
}

type ProvidersOutput struct {
	SchemaVersion int      `json:"schema_version"`
	Kind          string   `json:"kind"`
	Providers     []string `json:"providers"`
	Limits        Limits   `json:"limits"`
}

type Limits struct {
	MaxRequestBytes      int `json:"max_request_bytes"`
	MaxInputBytes        int `json:"max_input_bytes"`
	MaxModelBytes        int `json:"max_model_bytes"`
	DefaultTimeoutMS     int `json:"default_timeout_ms"`
	MaxTimeoutMS         int `json:"max_timeout_ms"`
	MaxConcurrentReviews int `json:"max_concurrent_reviews"`
	MaxStdoutBytes       int `json:"max_stdout_bytes"`
	MaxStderrBytes       int `json:"max_stderr_bytes"`
	MaxRawBytes          int `json:"max_raw_bytes"`
	MaxResponseBytes     int `json:"max_response_bytes"`
	MaxInflightMessages  int `json:"max_inflight_messages"`
}

func EffectiveLimits() Limits {
	return Limits{MaxRequestBytes, MaxInputBytes, MaxModelBytes, DefaultTimeoutMS, MaxTimeoutMS, MaxConcurrentReviews, MaxStdoutBytes, MaxStderrBytes, MaxRawBytes, MaxResponseBytes, MaxInflightMessages}
}
