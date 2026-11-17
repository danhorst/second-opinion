package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMCPInstallCLI(t *testing.T) {
	home := t.TempDir()
	env := func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	}
	config := filepath.Join(home, "config.json")
	args := []string{"--harness", "claude", "--config", config, "--executable", "/bin/second-opinion", "--json"}
	var out, stderr bytes.Buffer
	if code := runMCPInstall(append(args, "--dry-run"), env, &out, &stderr); code != 0 {
		t.Fatalf("%d %s", code, stderr.String())
	}
	if _, err := os.Stat(config); !os.IsNotExist(err) {
		t.Fatal("dry run created config")
	}
	var report map[string]any
	if err := json.Unmarshal(out.Bytes(), &report); err != nil || report["entry"] == nil {
		t.Fatal("preview missing entry", err)
	}
	out.Reset()
	if code := runMCPInstall(args, env, &out, &stderr); code != 0 {
		t.Fatalf("%d %s", code, stderr.String())
	}
	if code := runMCPUninstall(args, env, &out, &stderr); code != 0 {
		t.Fatalf("%d %s", code, stderr.String())
	}
}

func TestMCPInstallInvalidFlags(t *testing.T) {
	for _, args := range [][]string{{}, {"--harness", "bogus"}, {"--harness", "claude", "positional"}, {"--harness", "codex", "--url", "http://127.0.0.1:8765/mcp"}} {
		var out, stderr bytes.Buffer
		if code := runMCPInstall(args, os.Getenv, &out, &stderr); code != 2 {
			t.Fatalf("%v: %d %s", args, code, stderr.String())
		}
	}
}

func TestMCPHTTPInstallDoesNotCopyToken(t *testing.T) {
	home := t.TempDir()
	token := filepath.Join(home, "token")
	if _, err := initializeMCPToken(token); err != nil {
		t.Fatal(err)
	}
	secret, err := os.ReadFile(token)
	if err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(home, "config.json")
	var out, stderr bytes.Buffer
	args := []string{"--harness", "claude", "--config", config, "--executable", "/bin/second-opinion", "--transport", "http", "--token-file", token, "--json"}
	if code := runMCPInstall(args, os.Getenv, &out, &stderr); code != 0 {
		t.Fatalf("%d %s", code, stderr.String())
	}
	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range [][]byte{data, out.Bytes(), stderr.Bytes()} {
		if bytes.Contains(b, bytes.TrimSpace(secret)) {
			t.Fatal("token leaked")
		}
	}
}
