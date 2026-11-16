//go:build darwin || linux

package main

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestMCPInitCreateAndPreserve(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("service setup refuses root")
	}
	home := t.TempDir()
	getenv := func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	}
	var out, diagnostic bytes.Buffer
	if code := runMCPInit(nil, getenv, &out, &diagnostic); code != 0 {
		t.Fatalf("exit %d: %s", code, diagnostic.String())
	}
	path := filepath.Join(home, ".config", "second-opinion", "mcp-token")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 65 || data[64] != '\n' {
		t.Fatal("bad token format")
	}
	if _, err := hex.DecodeString(string(data[:64])); err != nil {
		t.Fatal(err)
	}
	if out.Len() == 0 || strings.Contains(out.String(), string(data[:64])) || diagnostic.Len() != 0 {
		t.Fatal("unsafe output")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("token permissions")
	}
	dir, _ := os.Stat(filepath.Dir(path))
	if dir.Mode().Perm() != 0700 {
		t.Fatal("directory permissions")
	}
	out.Reset()
	if code := runMCPInit(nil, getenv, &out, &diagnostic); code != 0 {
		t.Fatal(diagnostic.String())
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(data, after) || !strings.Contains(out.String(), "unchanged") {
		t.Fatal("token replaced")
	}
	other := filepath.Join(home, "other")
	if _, err := initializeMCPToken(other); err != nil {
		t.Fatal(err)
	}
	otherData, _ := os.ReadFile(other)
	if bytes.Equal(data, otherData) {
		t.Fatal("reused token")
	}
}

func TestMCPInitInvalidExistingUnchanged(t *testing.T) {
	for _, tc := range []struct {
		name, data string
		mode       os.FileMode
	}{
		{"permissions", strings.Repeat("x", 64), 0644}, {"short", "secret", 0600},
		{"multiline", strings.Repeat("x", 64) + "\n\n", 0600}, {"nul", strings.Repeat("x", 64) + "\x00", 0600},
		{"oversized", strings.Repeat("x", 4098), 0600}, {"whitespace", strings.Repeat("x", 64) + " ", 0600},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "token")
			if err := os.WriteFile(path, []byte(tc.data), tc.mode); err != nil {
				t.Fatal(err)
			}
			if _, err := initializeMCPToken(path); err == nil {
				t.Fatal("accepted invalid token")
			}
			after, _ := os.ReadFile(path)
			if string(after) != tc.data {
				t.Fatal("modified token")
			}
		})
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	os.WriteFile(target, []byte(strings.Repeat("x", 64)), 0600)
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := initializeMCPToken(link); err == nil {
		t.Fatal("accepted symlink")
	}
	if _, err := initializeMCPToken(dir); err == nil {
		t.Fatal("accepted directory")
	}
}

func TestMCPInitConcurrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	var created atomic.Int32
	var wg sync.WaitGroup
	for range 12 {
		wg.Go(func() {
			new, err := initializeMCPToken(path)
			if err != nil {
				t.Error(err)
			}
			if new {
				created.Add(1)
			}
		})
	}
	wg.Wait()
	if created.Load() != 1 {
		t.Fatalf("created %d times", created.Load())
	}
	if err := validateMCPTokenFile(path); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Fatal("temporary credentials left behind")
	}
}

func TestMCPInitFlags(t *testing.T) {
	var out, diagnostic bytes.Buffer
	for _, args := range [][]string{{"--unknown"}, {"extra"}, {"--token-file", ""}} {
		if code := runMCPInit(args, func(string) string { return "" }, &out, &diagnostic); code != 2 {
			t.Fatalf("%v: %d", args, code)
		}
	}
	if os.Geteuid() == 0 {
		return
	}
	dir := t.TempDir()
	envPath := filepath.Join(dir, "environment")
	explicit := filepath.Join(dir, "explicit")
	getenv := func(k string) string {
		if k == "SECOND_OPINION_MCP_TOKEN_FILE" {
			return envPath
		}
		return ""
	}
	if code := runMCPInit([]string{"--token-file", explicit}, getenv, &out, &diagnostic); code != 0 {
		t.Fatal(diagnostic.String())
	}
	if _, err := os.Stat(envPath); !os.IsNotExist(err) {
		t.Fatal("flag did not override environment")
	}
	if code := runMCPInit(nil, getenv, &out, &diagnostic); code != 0 {
		t.Fatal(diagnostic.String())
	}
	if err := validateMCPTokenFile(envPath); err != nil {
		t.Fatal(err)
	}
}
