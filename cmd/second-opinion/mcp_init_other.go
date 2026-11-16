//go:build !darwin && !linux

package main

import (
	"errors"
	"os"
)

func serviceUserSupported() error {
	return errors.New("mcp init supports the macOS and Linux service launcher only")
}
func serviceFileOwned(os.FileInfo) bool { return false }
