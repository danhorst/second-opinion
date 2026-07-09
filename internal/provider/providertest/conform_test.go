package providertest

import (
	"errors"
	"strings"
	"testing"

	"github.com/danhorst/second-opinion/internal/provider"
)

// The honest reference provider passes the whole suite.
func TestConformHonestLoopback(t *testing.T) {
	Conform(t, func(t *testing.T) provider.Provider {
		return NewLoopback()
	})
}

// Each rigged violation is caught by the corresponding check.
func TestConformCatchesViolations(t *testing.T) {
	t.Run("instruction-file-leak", func(t *testing.T) {
		err := checkInstructionFileCold(t, NewLoopback(LeakInstructionFile()))
		if err == nil || !strings.Contains(err.Error(), "cold-reviewer violation") {
			t.Errorf("expected cold-reviewer violation, got %v", err)
		}
	})
	t.Run("repo-read", func(t *testing.T) {
		err := checkRepoReadCold(t, NewLoopback(ReadReferencedFiles()))
		if err == nil || !strings.Contains(err.Error(), "cold-reviewer violation") {
			t.Errorf("expected cold-reviewer violation, got %v", err)
		}
	})
	t.Run("omitted-provenance", func(t *testing.T) {
		err := checkContract(t.Context(), NewLoopback(OmitProvenance()))
		if err == nil || !strings.Contains(err.Error(), "provenance incomplete") {
			t.Errorf("expected provenance failure, got %v", err)
		}
	})
	t.Run("ignored-cancellation", func(t *testing.T) {
		err := checkCancellation(NewLoopback(IgnoreCancellation()))
		if err == nil || !strings.Contains(err.Error(), "expected error") {
			t.Errorf("expected cancellation failure, got %v", err)
		}
	})
}

// Did-not-run yields an error and no result; ran-and-found-nothing yields a
// result and no error.
func TestRanVersusDidNotRun(t *testing.T) {
	res, err := NewLoopback(Unavailable()).Review(t.Context(), provider.Request{Prompt: "p", Material: "m"})
	if !errors.Is(err, provider.ErrUnavailable) {
		t.Errorf("expected ErrUnavailable, got %v", err)
	}
	if res != nil {
		t.Errorf("did-not-run returned a result: %+v", res)
	}

	res, err = NewLoopback().Review(t.Context(), provider.Request{Prompt: "p", Material: "m"})
	if err != nil {
		t.Fatalf("honest review failed: %v", err)
	}
	if res.Findings != "" {
		t.Errorf("expected empty findings from a clean run, got %q", res.Findings)
	}
}
