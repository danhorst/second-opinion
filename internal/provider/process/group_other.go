//go:build !darwin && !linux && !freebsd && !openbsd && !netbsd && !dragonfly

package process

import "os/exec"

// CommandContext retains parent cancellation. Process-tree cleanup on these
// platforms needs a dedicated implementation (e.g. Windows Job Objects).
func configure(cmd *exec.Cmd) func() { return func() {} }
