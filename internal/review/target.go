package review

import (
	"bytes"
	"context"
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
		if len(bytes.TrimSpace(content)) > 0 {
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
	return FromDiffContext(context.Background(), base)
}

// FromDiffContext assembles a built-in, uncolored patch without invoking
// external diff or textconv helpers. The trailing -- makes base a revision,
// never a path; rejecting leading dashes prevents option injection.
func FromDiffContext(ctx context.Context, base string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if base == "" || strings.HasPrefix(base, "-") {
		return "", fmt.Errorf("review: invalid diff base %q", base)
	}
	out, err := exec.CommandContext(ctx, "git", "diff", "--no-ext-diff", "--no-textconv", "--no-color", base, "--").Output()
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
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
