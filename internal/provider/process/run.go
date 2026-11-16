// Package process controls invocation-owned subprocesses. It does not inspect
// requests or read configuration. Gateway-only credentials are not inherited.
package process

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/danhorst/second-opinion/internal/provider"
)

// Run executes a CommandContext command with bounded output-pipe drain and
// platform-specific descendant cleanup. Positive capture limits bound bytes
// written to the caller's output streams. SECOND_OPINION_MCP_TOKEN is always
// removed from the child's environment, including when limits are omitted.
func Run(cmd *exec.Cmd, limits ...provider.CaptureLimits) error {
	cmd.WaitDelay = 2 * time.Second
	cleanup := configure(cmd)
	defer cleanup()
	env := cmd.Environ()
	cmd.Env = make([]string, 0, len(env))
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		if !strings.EqualFold(key, "SECOND_OPINION_MCP_TOKEN") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	var overflow atomic.Bool
	if len(limits) > 0 {
		// A shared mutex also supports callers directing both streams into one
		// non-concurrent writer, which os/exec normally serializes for them.
		var mu sync.Mutex
		abort := func() {
			if overflow.CompareAndSwap(false, true) {
				if cmd.Cancel != nil {
					_ = cmd.Cancel()
				} else if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
			}
		}
		cmd.Stdout = boundedWriter(cmd.Stdout, limits[0].StdoutBytes, &mu, abort)
		cmd.Stderr = boundedWriter(cmd.Stderr, limits[0].StderrBytes, &mu, abort)
	}
	err := cmd.Run()
	// Run has joined its copying goroutines before we inspect overflow. Preserve
	// this reason even when killing the child also caused an exit/context error.
	if overflow.Load() {
		return provider.ErrOutputLimit
	}
	return explainStartError(err)
}

type captureWriter struct {
	dst       io.Writer
	remaining int
	limited   bool
	mu        *sync.Mutex
	overflow  func()
}

func boundedWriter(dst io.Writer, limit int, mu *sync.Mutex, overflow func()) io.Writer {
	if dst == nil {
		dst = io.Discard
	}
	return &captureWriter{dst: dst, remaining: limit, limited: limit > 0, mu: mu, overflow: overflow}
}

func (w *captureWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	exceeded := w.limited && len(p) > w.remaining
	if exceeded {
		p = p[:w.remaining]
	}
	n, err := w.dst.Write(p)
	if w.limited {
		w.remaining -= n
	}
	if exceeded {
		w.overflow()
		return n, provider.ErrOutputLimit
	}
	return n, err
}

func explainStartError(err error) error {
	if errors.Is(err, syscall.E2BIG) {
		return fmt.Errorf("provider arguments exceed the operating system limit; reduce the material or prompt size: %w", err)
	}
	return err
}
