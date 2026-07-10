package review

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ErrNothingToReview is returned when target assembly produces empty
// material — a review of nothing burns money to say nothing.
var ErrNothingToReview = errors.New("review: nothing to review")

// FromFiles assembles material from files, concatenated in argument order,
// each preceded by a header naming its path so multi-file reviews can cite
// locations.
func FromFiles(paths []string) (string, error) {
	var b strings.Builder
	empty := true
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("review: %w", err)
		}
		if len(strings.TrimSpace(string(content))) > 0 {
			empty = false
		}
		fmt.Fprintf(&b, "=== %s ===\n%s\n", path, content)
	}
	if empty {
		return "", ErrNothingToReview
	}
	return b.String(), nil
}

// FromDiff assembles material from the caller's git diff against base.
// Assembly runs in the caller's working directory — the caller's explicit
// act, which is what the nothing-implicit invariant permits.
func FromDiff(base string) (string, error) {
	out, err := exec.Command("git", "diff", base).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("review: git diff %s: %s", base, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("review: git diff: %w", err)
	}
	if strings.TrimSpace(string(out)) == "" {
		return "", ErrNothingToReview
	}
	return string(out), nil
}
