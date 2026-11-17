package mcpserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/review"
)

// ValidateJSON rejects malformed JSON, duplicate keys and lossy Unicode before
// encoding/json has an opportunity to silently replace invalid input.
func ValidateJSON(raw []byte) error {
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return errors.New("invalid JSON or UTF-8")
	}
	// JSON syntax is already checked, so every escape has enough bytes and every
	// unescaped quote is a string boundary. Check escaped UTF-16 scalar values.
	inString := false
	for i := 0; i < len(raw); i++ {
		if raw[i] == '"' {
			inString = !inString
			continue
		}
		if !inString || raw[i] != '\\' {
			continue
		}
		i++
		if raw[i] != 'u' {
			continue
		}
		n, _ := strconv.ParseUint(string(raw[i+1:i+5]), 16, 16)
		i += 4
		if n >= 0xdc00 && n <= 0xdfff {
			return errors.New("unpaired Unicode surrogate")
		}
		if n < 0xd800 || n > 0xdbff {
			continue
		}
		if i+6 >= len(raw) || raw[i+1] != '\\' || raw[i+2] != 'u' {
			return errors.New("unpaired Unicode surrogate")
		}
		low, err := strconv.ParseUint(string(raw[i+3:i+7]), 16, 16)
		if err != nil || low < 0xdc00 || low > 0xdfff {
			return errors.New("unpaired Unicode surrogate")
		}
		i += 6
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	return uniqueJSONValue(d)
}

func uniqueJSONValue(d *json.Decoder) error {
	t, err := d.Token()
	if err != nil {
		return err
	}
	switch t {
	case json.Delim('{'):
		seen := make(map[string]bool)
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return err
			}
			name := key.(string)
			if seen[name] {
				return errors.New("duplicate JSON key")
			}
			seen[name] = true
			if err := uniqueJSONValue(d); err != nil {
				return err
			}
		}
		_, err = d.Token()
	case json.Delim('['):
		for d.More() {
			if err := uniqueJSONValue(d); err != nil {
				return err
			}
		}
		_, err = d.Token()
	}
	return err
}

func invalidInput() *ToolError {
	return &ToolError{"invalid_input", "Arguments must match the tool input schema and contain valid Unicode text.", "not_started"}
}

func parseObject(raw json.RawMessage) (map[string]json.RawMessage, *ToolError) {
	if len(raw) > MaxRequestBytes || ValidateJSON(raw) != nil {
		return nil, invalidInput()
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return nil, invalidInput()
	}
	return fields, nil
}

func ValidateEmptyInput(raw json.RawMessage) *ToolError {
	fields, err := parseObject(raw)
	if err != nil {
		return err
	}
	if len(fields) != 0 {
		return invalidInput()
	}
	return nil
}

func ParseReviewInput(raw json.RawMessage) (ReviewInput, *ToolError) {
	in := ReviewInput{TimeoutMS: DefaultTimeoutMS}
	fields, err := parseObject(raw)
	if err != nil {
		return ReviewInput{}, err
	}
	for key, value := range fields {
		switch key {
		case "provider", "material", "model", "prompt":
			var s string
			if bytes.Equal(bytes.TrimSpace(value), []byte("null")) || json.Unmarshal(value, &s) != nil || strings.TrimSpace(s) == "" {
				return ReviewInput{}, invalidInput()
			}
			switch key {
			case "provider":
				in.Provider = s
			case "material":
				in.Material = s
			case "model":
				in.Model = s
			case "prompt":
				in.Prompt = &s
			}
		case "require_isolation":
			if json.Unmarshal(value, &in.RequireIsolation) != nil || in.RequireIsolation != provider.IsolationLocalMaterialOnly {
				return ReviewInput{}, invalidInput()
			}
		case "timeout_ms":
			if json.Unmarshal(value, &in.TimeoutMS) != nil || in.TimeoutMS < 1 || in.TimeoutMS > MaxTimeoutMS || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return ReviewInput{}, invalidInput()
			}
		default:
			return ReviewInput{}, invalidInput()
		}
	}
	known := false
	for _, name := range review.Providers() {
		if name == in.Provider {
			known = true
		}
	}
	if !known || in.Material == "" || len(in.Model) > MaxModelBytes {
		return ReviewInput{}, invalidInput()
	}
	prompt := review.BakedPrompt
	if in.Prompt != nil {
		prompt = *in.Prompt
	}
	if len(in.Material)+len(prompt) > MaxInputBytes {
		return ReviewInput{}, &ToolError{"input_too_large", "Material plus resolved prompt exceeds the 1048576-byte input limit.", "not_started"}
	}
	return in, nil
}
