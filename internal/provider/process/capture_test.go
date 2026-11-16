package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/danhorst/second-opinion/internal/provider"
)

// This runs the test executable as a real OS child; it does not impersonate
// any provider binary or make claims about provider transport behavior.
func TestCaptureHelper(t *testing.T) {
	mode := os.Getenv("SECOND_OPINION_CAPTURE_TEST")
	if mode == "" {
		return
	}
	switch mode {
	case "stdout", "stderr":
		stream := os.Stdout
		if mode == "stderr" {
			stream = os.Stderr
		}
		_, _ = stream.Write(bytes.Repeat([]byte("x"), 32768))
		time.Sleep(30 * time.Second)
	case "complete":
		_, _ = os.Stdout.Write(bytes.Repeat([]byte("o"), 32768))
		_, _ = os.Stderr.Write(bytes.Repeat([]byte("e"), 16384))
	case "concurrent":
		var wg sync.WaitGroup
		for _, stream := range []*os.File{os.Stdout, os.Stderr} {
			wg.Add(1)
			go func(stream *os.File) {
				defer wg.Done()
				for i := 0; i < 100; i++ {
					_, _ = stream.Write(bytes.Repeat([]byte("x"), 256))
				}
			}(stream)
		}
		wg.Wait()
	case "environment":
		_, present := os.LookupEnv("SECOND_OPINION_MCP_TOKEN")
		fmt.Fprintf(os.Stdout, "%t:%s", present, os.Getenv("SECOND_OPINION_KEEP_TEST"))
	}
	os.Exit(0)
}

func captureCommand(t *testing.T, mode string) *exec.Cmd {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCaptureHelper$")
	cmd.Env = append(os.Environ(), "SECOND_OPINION_CAPTURE_TEST="+mode)
	return cmd
}

func TestCaptureOverflow(t *testing.T) {
	for _, stream := range []string{"stdout", "stderr"} {
		t.Run(stream, func(t *testing.T) {
			cmd := captureCommand(t, stream)
			var out, diag bytes.Buffer
			cmd.Stdout, cmd.Stderr = &out, &diag
			start := time.Now()
			err := Run(cmd, provider.CaptureLimits{StdoutBytes: 1024, StderrBytes: 512})
			if !errors.Is(err, provider.ErrOutputLimit) {
				t.Fatalf("want output limit, got %v", err)
			}
			if time.Since(start) > 3*time.Second {
				t.Fatal("overflow did not terminate child promptly")
			}
			if cmd.ProcessState == nil {
				t.Fatal("child was not waited for")
			}
			if out.Len() > 1024 || diag.Len() > 512 {
				t.Fatalf("capture exceeded limits: %d/%d", out.Len(), diag.Len())
			}
			if stream == "stdout" && out.Len() != 1024 || stream == "stderr" && diag.Len() != 512 {
				t.Fatal("capture did not reach boundary")
			}
		})
	}
}

func TestCaptureDefaultsAndExactBoundary(t *testing.T) {
	for _, tc := range []struct {
		name   string
		limits []provider.CaptureLimits
	}{
		{name: "omitted"},
		{name: "zero", limits: []provider.CaptureLimits{{}}},
		{name: "exact", limits: []provider.CaptureLimits{{StdoutBytes: 32768, StderrBytes: 16384}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := captureCommand(t, "complete")
			var out, diag bytes.Buffer
			cmd.Stdout, cmd.Stderr = &out, &diag
			if err := Run(cmd, tc.limits...); err != nil {
				t.Fatal(err)
			}
			if out.String() != strings.Repeat("o", 32768) || diag.String() != strings.Repeat("e", 16384) {
				t.Fatal("output changed")
			}
		})
	}
}

func TestGatewayTokenNeverInherited(t *testing.T) {
	t.Setenv("SECOND_OPINION_MCP_TOKEN", "secret inherited token")
	t.Setenv("SECOND_OPINION_KEEP_TEST", "preserved")
	t.Setenv("SECOND_OPINION_CAPTURE_TEST", "environment")
	for _, explicit := range []bool{false, true} {
		t.Run(fmt.Sprint(explicit), func(t *testing.T) {
			cmd := captureCommand(t, "environment")
			if !explicit {
				cmd.Env = nil
			} else {
				cmd.Env = append(cmd.Env, "SECOND_OPINION_MCP_TOKEN=explicit secret")
			}
			var out bytes.Buffer
			cmd.Stdout = &out
			if err := Run(cmd); err != nil {
				t.Fatal(err)
			}
			if out.String() != "false:preserved" {
				t.Fatalf("unexpected environment result %q", out.String())
			}
		})
	}
}

// Wrapping stdout and stderr must not remove os/exec's shared-writer safety.
func TestCaptureSharedWriter(t *testing.T) {
	cmd := captureCommand(t, "concurrent")
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := Run(cmd, provider.CaptureLimits{StdoutBytes: 25600, StderrBytes: 25600}); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 51200 {
		t.Fatalf("lost concurrent output: %d", output.Len())
	}
}

func TestCaptureDiscardStillEnforcesLimit(t *testing.T) {
	cmd := captureCommand(t, "stdout")
	// A nil destination means discard, not an exemption from the byte budget.
	if err := Run(cmd, provider.CaptureLimits{StdoutBytes: 128}); !errors.Is(err, provider.ErrOutputLimit) {
		t.Fatalf("discarded overflow = %v", err)
	}
}
