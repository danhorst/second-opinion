package harness

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	for _, h := range []string{"codex", "claude", "antigravity"} {
		t.Run(h, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "config")
			o := Options{Harness: h, ConfigPath: p, Executable: "/opt/bin/second-opinion"}
			r, e := Apply(o, os.Getenv, false)
			if e != nil || !r.Changed {
				t.Fatalf("%+v %v", r, e)
			}
			r, e = Apply(o, os.Getenv, false)
			if e != nil || r.Changed {
				t.Fatalf("repeat %+v %v", r, e)
			}
			got, ok, e := Inspect(o, os.Getenv)
			if e != nil || !ok || got.Command != o.Executable {
				t.Fatal(got, ok, e)
			}
			r, e = Apply(o, os.Getenv, true)
			if e != nil || !r.Changed {
				t.Fatal(r, e)
			}
		})
	}
}
func TestPreserveAndConflict(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config")
	original := []byte(`{"unrelated":{"x":1},"mcpServers":{"second-opinion":{"command":"other"}}}`)
	os.WriteFile(p, original, 0600)
	o := Options{Harness: "claude", ConfigPath: p, Executable: "/bin/second-opinion"}
	if _, e := Apply(o, os.Getenv, false); e == nil {
		t.Fatal("accepted conflict")
	}
	o.Replace = true
	o.DryRun = true
	if _, e := Apply(o, os.Getenv, false); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(p)
	if string(b) != string(original) {
		t.Fatal("dry run modified file")
	}
	o.DryRun = false
	if _, e := Apply(o, os.Getenv, false); e != nil {
		t.Fatal(e)
	}
	b, _ = os.ReadFile(p)
	if !contains(b, `"unrelated"`) {
		t.Fatal("lost unrelated setting")
	}
}
func contains(b []byte, s string) bool {
	for i := 0; i+len(s) <= len(b); i++ {
		if string(b[i:i+len(s)]) == s {
			return true
		}
	}
	return false
}
func TestUnsafeAndMalformed(t *testing.T) {
	for _, data := range []string{`{"mcpServers":{}} junk`, `{"mcpServers":[]}`, `null`} {
		p := filepath.Join(t.TempDir(), "config")
		os.WriteFile(p, []byte(data), 0600)
		if _, e := Apply(Options{Harness: "claude", ConfigPath: p, Executable: "/bin/second-opinion"}, os.Getenv, false); e == nil {
			t.Fatalf("accepted %s", data)
		}
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "link")
	os.Symlink(filepath.Join(dir, "missing"), p)
	if _, e := Apply(Options{Harness: "codex", ConfigPath: p, Executable: "/bin/second-opinion"}, os.Getenv, false); e == nil {
		t.Fatal("accepted symlink")
	}
}
func TestHTTPDefaultsAndRemoval(t *testing.T) {
	o := Options{Harness: "claude", ConfigPath: filepath.Join(t.TempDir(), "config"), Executable: "/bin/second-opinion", Transport: "http"}
	env := func(k string) string {
		if k == "HOME" {
			return "/test-home"
		}
		return ""
	}
	_, r, e := Resolve(o, env)
	if e != nil || r.Args[3] != "http://127.0.0.1:8765/mcp" || r.Args[5] != "/test-home/.config/second-opinion/mcp-token" {
		t.Fatal(r, e)
	}
	if _, e = Apply(o, env, false); e != nil {
		t.Fatal(e)
	}
	o.Transport = "stdio"
	if _, e = Apply(o, env, true); e != nil {
		t.Fatal(e)
	}
}

func TestPathsAndScopes(t *testing.T) {
	env := func(k string) string {
		if k == "HOME" {
			return "/example"
		}
		return ""
	}
	for h, want := range map[string]string{"codex": "/example/.codex/config.toml", "claude": "/example/.claude.json", "antigravity": "/example/.gemini/config/mcp_config.json"} {
		p, _, err := Resolve(Options{Harness: h, Executable: "/bin/second-opinion"}, env)
		if err != nil || p != want {
			t.Fatalf("%s: %s %v", h, p, err)
		}
	}
	for h, want := range map[string]string{"codex": ".codex/config.toml", "claude": ".mcp.json", "antigravity": ".agents/mcp_config.json"} {
		p, _, err := Resolve(Options{Harness: h, Scope: "project", Executable: "/bin/second-opinion"}, env)
		abs, _ := filepath.Abs(want)
		if err != nil || p != abs {
			t.Fatalf("%s: %s %v", h, p, err)
		}
	}
}

func TestTOMLPreservationAndBackup(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.toml")
	original := []byte("# user comment\nmodel = \"custom\"\n[features]\nexample = true\n[mcp_servers.other]\ncommand = \"other\"\n")
	if err := os.WriteFile(p, original, 0600); err != nil {
		t.Fatal(err)
	}
	o := Options{Harness: "codex", ConfigPath: p, Executable: "/bin/second-opinion"}
	report, err := Apply(o, os.Getenv, false)
	if err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(report.Backup)
	if err != nil || string(backup) != string(original) {
		t.Fatal("backup differs", err)
	}
	info, err := os.Stat(report.Backup)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("backup permissions", err)
	}
	parsed, _, err := read(p, "codex")
	if err != nil {
		t.Fatal(err)
	}
	if parsed["model"] != "custom" || parsed["features"].(map[string]any)["example"] != true {
		t.Fatal("unrelated settings changed")
	}
	servers := parsed["mcp_servers"].(map[string]any)
	if servers["other"].(map[string]any)["command"] != "other" {
		t.Fatal("other MCP lost")
	}
	if _, err = Apply(o, os.Getenv, true); err != nil {
		t.Fatal(err)
	}
	parsed, _, _ = read(p, "codex")
	if parsed["mcp_servers"].(map[string]any)["other"] == nil {
		t.Fatal("other MCP removed")
	}
}

func TestRejectDuplicateJSONKeys(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	original := []byte(`{"other":1,"other":2}`)
	os.WriteFile(p, original, 0600)
	if _, err := Apply(Options{Harness: "claude", ConfigPath: p, Executable: "/bin/second-opinion"}, os.Getenv, false); err == nil {
		t.Fatal("accepted ambiguous JSON")
	}
	got, _ := os.ReadFile(p)
	if string(got) != string(original) {
		t.Fatal("changed ambiguous config")
	}
}

func TestLongerTimeoutAccepted(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(p, []byte(`{"mcpServers":{"second-opinion":{"command":"/bin/second-opinion","args":["mcp","serve"],"timeout":600000}}}`), 0600)
	o := Options{Harness: "claude", ConfigPath: p, Executable: "/bin/second-opinion"}
	if _, ok, err := Inspect(o, os.Getenv); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if _, err := Apply(o, os.Getenv, true); err != nil {
		t.Fatal(err)
	}
}
