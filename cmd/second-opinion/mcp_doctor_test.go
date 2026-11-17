package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMCPDoctorMissingRegistration(t *testing.T) {
	home := t.TempDir()
	var out, stderr bytes.Buffer
	code := runMCPDoctor(context.Background(), []string{"--harness", "claude", "--json"}, func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	}, &out, &stderr)
	if code != 1 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	var report doctorReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Configuration != "failed" || report.Connection != "not_checked" || report.Review != "not_performed" {
		t.Fatalf("%+v", report)
	}
}

func TestMCPDoctorRejectsArbitraryCommand(t *testing.T) {
	home := t.TempDir()
	config := filepath.Join(home, "config.json")
	if err := os.WriteFile(config, []byte(`{"mcpServers":{"second-opinion":{"command":"/bin/sh","args":["-c","exit 0"]}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	if code := runMCPDoctor(t.Context(), []string{"--harness", "claude", "--config", config, "--json"}, func(string) string { return home }, &out, &stderr); code != 1 {
		t.Fatalf("exit %d", code)
	}
	var report doctorReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Configuration != "failed" || report.Connection != "not_checked" {
		t.Fatalf("%+v", report)
	}
}

func TestMCPDoctorUsage(t *testing.T) {
	for _, args := range [][]string{{}, {"--harness", "unknown"}, {"--harness", "codex", "extra"}} {
		var out, stderr bytes.Buffer
		if code := runMCPDoctor(t.Context(), args, func(string) string { return t.TempDir() }, &out, &stderr); code != 2 {
			t.Fatalf("args %v exit %d", args, code)
		}
	}
}
