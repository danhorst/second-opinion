// Package process controls invocation-owned subprocesses. It does not inspect
// requests, read configuration, or alter authentication.
package process

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

// Run executes a CommandContext command with bounded output-pipe drain and
// platform-specific descendant cleanup. Callers retain ownership of streams,
// environment, directory, and the cancellation context.
func Run(cmd *exec.Cmd) error {
	cmd.WaitDelay = 2 * time.Second
	cleanup := configure(cmd)
	defer cleanup()
	return explainStartError(cmd.Run())
}

func explainStartError(err error) error {
	if errors.Is(err, syscall.E2BIG) {
		return fmt.Errorf("provider arguments exceed the operating system limit; reduce the material or prompt size: %w", err)
	}
	return err
}
