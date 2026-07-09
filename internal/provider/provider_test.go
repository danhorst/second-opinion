package provider

import "testing"

func TestHashPrompt(t *testing.T) {
	a := HashPrompt("find the flaws")
	b := HashPrompt("find the flaws")
	c := HashPrompt("find the flaws.")

	if a != b {
		t.Errorf("same prompt hashed differently: %q vs %q", a, b)
	}
	if a == c {
		t.Errorf("different prompts hashed identically: %q", a)
	}
	if len(a) != 64 {
		t.Errorf("expected 64 hex chars, got %d: %q", len(a), a)
	}
}

func TestHashPromptEmpty(t *testing.T) {
	if HashPrompt("") == "" {
		t.Error("empty prompt must still produce an identity")
	}
}
