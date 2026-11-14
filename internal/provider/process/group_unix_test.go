//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package process

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"
)

// Real subprocesses test OS lifecycle behavior, not a mocked reviewer.
func TestProcessHelper(t *testing.T) {
	mode := os.Getenv("SECOND_OPINION_PROCESS_TEST")
	if mode == "" {
		return
	}
	if mode == "child" {
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestProcessHelper$")
	child.Env = append(os.Environ(), "SECOND_OPINION_PROCESS_TEST=child")
	child.Stdout, child.Stderr = os.Stdout, os.Stderr
	if err := child.Start(); err != nil {
		os.Exit(3)
	}
	if err := os.WriteFile(os.Getenv("SECOND_OPINION_PROCESS_READY"), []byte("ready"), 0o600); err != nil {
		os.Exit(4)
	}
	if mode == "exit-parent" {
		os.Exit(0)
	}
	_ = child.Wait()
	os.Exit(0)
}

func TestProcessTreeLifecycle(t *testing.T) {
	for _, mode := range []string{"cancel", "exit-parent"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			ready := t.TempDir() + "/ready"
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestProcessHelper$")
			cmd.Env = append(os.Environ(), "SECOND_OPINION_PROCESS_TEST="+mode, "SECOND_OPINION_PROCESS_READY="+ready)
			var out bytes.Buffer
			cmd.Stdout, cmd.Stderr = &out, &out
			done := make(chan error, 1)
			go func() { done <- Run(cmd) }()
			for {
				if _, err := os.Stat(ready); err == nil {
					break
				}
				select {
				case err := <-done:
					t.Fatalf("exited before ready: %v: %s", err, out.String())
				case <-ctx.Done():
					t.Fatal("helper never became ready")
				case <-time.After(10 * time.Millisecond):
				}
			}
			if mode == "cancel" {
				cancel()
			}
			limit := 4 * time.Second
			if mode == "cancel" {
				limit = time.Second
			}
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("expected cancellation or pipe-drain error")
				}
				if mode == "exit-parent" && !errors.Is(err, exec.ErrWaitDelay) {
					t.Fatalf("want drain error: %v", err)
				}
			case <-time.After(limit):
				t.Fatal("subprocess tree did not terminate promptly")
			}
		})
	}
}
