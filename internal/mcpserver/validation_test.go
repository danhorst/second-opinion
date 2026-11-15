package mcpserver

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/danhorst/second-opinion/internal/review"
)

func TestValidateJSONUnicodeAndDuplicateKeys(t *testing.T) {
	for _, raw := range []string{`{"x":"😀"}`, `{"x":"\ud83d\ude00"}`, `{"x":"\\ud800"}`, `{"x":"\""}`, `[true,42,null,{"x":1},{"x":2}]`} {
		if err := ValidateJSON([]byte(raw)); err != nil {
			t.Errorf("valid %q: %v", raw, err)
		}
	}
	for _, raw := range []string{`{"x":"\ud800"}`, `{"x":"\udc00"}`, `{"x":"\ud800\u0041"}`, `{"x":1,"\u0078":2}`, `[{"x":1,"x":2}]`, `{} {}`, "{\"x\":\"\xff\"}", `{"x":"\ud800x"}`} {
		if err := ValidateJSON([]byte(raw)); err == nil {
			t.Errorf("accepted invalid %q", raw)
		}
	}
}

func TestParseReviewInputRejectsBeforeExecution(t *testing.T) {
	cases := []string{`null`, `[]`, `{}`, `{"provider":"codex"}`, `{"material":"x"}`, `{"provider":"unknown","material":"x"}`, `{"provider":" codex","material":"x"}`, `{"provider":"codex","material":" "}`, `{"provider":"codex","material":null}`, `{"provider":"codex","material":1}`, `{"provider":"codex","material":"x","prompt":null}`, `{"provider":"codex","material":"x","prompt":""}`, `{"provider":"codex","material":"x","model":"\u2003"}`, `{"provider":"codex","material":"x","timeout_ms":null}`, `{"provider":"codex","material":"x","timeout_ms":1.5}`, `{"provider":"codex","material":"x","timeout_ms":0}`, `{"provider":"codex","material":"x","timeout_ms":300001}`, `{"provider":"codex","material":"x","extra":true}`}
	for _, raw := range cases {
		if _, err := ParseReviewInput([]byte(raw)); err == nil || err.Code != "invalid_input" || err.Execution != "not_started" {
			t.Errorf("%s: got %v", raw, err)
		}
	}
}

func TestParseReviewInputPreservesBytesAndDefaults(t *testing.T) {
	in, err := ParseReviewInput([]byte(`{"provider":"codex","material":" /tmp/a\n😀 ","model":" custom ","prompt":" p\n ","timeout_ms":300000}`))
	if err != nil {
		t.Fatal(err)
	}
	if in.Material != " /tmp/a\n😀 " || in.Model != " custom " || *in.Prompt != " p\n " || in.TimeoutMS != MaxTimeoutMS {
		t.Fatalf("normalized input: %#v", in)
	}
	t.Setenv("SECOND_OPINION_PROVIDER", "claude")
	in, err = ParseReviewInput([]byte(`{"provider":"codex","material":"x"}`))
	if err != nil || in.Provider != "codex" || in.Model != "" || in.Prompt != nil || in.TimeoutMS != DefaultTimeoutMS {
		t.Fatalf("defaults: %#v %v", in, err)
	}
}

func TestParseReviewInputByteBudgets(t *testing.T) {
	check := func(material, model string, prompt *string, want string) {
		t.Helper()
		args := map[string]any{"provider": "codex", "material": material}
		if model != "" {
			args["model"] = model
		}
		if prompt != nil {
			args["prompt"] = *prompt
		}
		raw, _ := json.Marshal(args)
		_, err := ParseReviewInput(raw)
		if want == "" {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.Code != want {
			t.Fatalf("got %v; want %s", err, want)
		}
	}
	p := "p"
	check(strings.Repeat("é", (MaxInputBytes-2)/2)+"x", "", &p, "")
	check(strings.Repeat("é", MaxInputBytes/2), "", &p, "input_too_large")
	check(strings.Repeat("x", MaxInputBytes-len(review.BakedPrompt)), "", nil, "")
	check(strings.Repeat("x", MaxInputBytes-len(review.BakedPrompt)+1), "", nil, "input_too_large")
	check("x", strings.Repeat("é", 128), nil, "")
	check("x", strings.Repeat("é", 129), nil, "invalid_input")
}

func TestEmptyInput(t *testing.T) {
	if err := ValidateEmptyInput([]byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`null`, `[]`, `{"unused":null}`, ``, `{"x":1,"x":2}`} {
		if ValidateEmptyInput([]byte(raw)) == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
}
