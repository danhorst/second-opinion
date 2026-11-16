//go:build darwin || linux

package main

import (
	"errors"
	"os"
	"syscall"
)

func serviceUserSupported() error {
	if os.Geteuid() == 0 {
		return errors.New("run mcp init as your normal service user, without sudo")
	}
	return nil
}

func serviceFileOwned(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid())
}
