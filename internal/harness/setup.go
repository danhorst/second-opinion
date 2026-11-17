// Package harness manages explicit, opt-in MCP registrations.
package harness

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/danhorst/second-opinion/internal/mcpclient"
	"github.com/danhorst/second-opinion/internal/mcpserver"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Options struct {
	Harness, Scope, ConfigPath, Executable, Transport, URL, TokenFile string
	Replace, DryRun                                                   bool
}
type Registration struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}
type Report struct {
	Path         string         `json:"path"`
	Harness      string         `json:"harness"`
	Changed      bool           `json:"changed"`
	DryRun       bool           `json:"dry_run"`
	Registration Registration   `json:"registration"`
	Entry        map[string]any `json:"entry"`
	Operation    string         `json:"operation"`
	Backup       string         `json:"backup,omitempty"`
}

func Resolve(o Options, getenv func(string) string) (string, Registration, error) {
	if o.Scope == "" {
		o.Scope = "user"
	}
	if o.Scope != "user" && o.Scope != "project" {
		return "", Registration{}, errors.New("scope must be user or project")
	}
	home := getenv("HOME")
	if home == "" && o.Scope == "user" && o.ConfigPath == "" {
		return "", Registration{}, errors.New("HOME or --config is required")
	}
	var p string
	switch o.Harness {
	case "codex":
		root := getenv("CODEX_HOME")
		if root == "" {
			root = filepath.Join(home, ".codex")
		}
		p = filepath.Join(root, "config.toml")
		if o.Scope == "project" {
			p = ".codex/config.toml"
		}
	case "claude":
		p = filepath.Join(home, ".claude.json")
		if getenv("CLAUDE_CONFIG_DIR") != "" && o.Scope == "user" && o.ConfigPath == "" {
			return "", Registration{}, errors.New("custom CLAUDE_CONFIG_DIR requires an explicit --config path")
		}
		if o.Scope == "project" {
			p = ".mcp.json"
		}
	case "antigravity":
		p = filepath.Join(home, ".gemini", "config", "mcp_config.json")
		if o.Scope == "project" {
			p = ".agents/mcp_config.json"
		}

	default:
		return "", Registration{}, errors.New("harness must be codex, claude, or antigravity")
	}
	if o.ConfigPath != "" {
		p = o.ConfigPath
	}
	p, e := filepath.Abs(p)
	if e != nil {
		return "", Registration{}, e
	}
	exe := o.Executable
	if exe == "" {
		exe, e = os.Executable()
		if e != nil {
			return "", Registration{}, e
		} // Keep Homebrew opt stable across upgrades.
		for _, prefix := range []string{"/opt/homebrew", "/usr/local", "/home/linuxbrew/.linuxbrew"} {
			if strings.HasPrefix(exe, prefix+"/Cellar/second-opinion/") {
				exe = prefix + "/opt/second-opinion/bin/second-opinion"
			}
		}
	}
	exe, e = filepath.Abs(exe)
	if e != nil {
		return "", Registration{}, e
	}
	r := Registration{Command: exe, Args: []string{"mcp", "serve"}}
	if o.Transport != "" && o.Transport != "stdio" && o.Transport != "http" {
		return "", r, errors.New("transport must be stdio or http")
	}
	if o.Transport != "http" && (o.URL != "" || o.TokenFile != "") {
		return "", r, errors.New("--url and --token-file require --transport http")
	}
	if o.Transport == "http" {
		if o.URL == "" {
			o.URL = "http://127.0.0.1:8765/mcp"
		}
		if o.TokenFile == "" {
			o.TokenFile = getenv("SECOND_OPINION_MCP_TOKEN_FILE")
			if o.TokenFile == "" && home != "" {
				o.TokenFile = filepath.Join(home, ".config", "second-opinion", "mcp-token")
			}
		}
		if o.URL == "" || !filepath.IsAbs(o.TokenFile) {
			return "", r, errors.New("HTTP registration requires --url and an absolute --token-file")
		}
		if e := mcpclient.ValidateURL(o.URL); e != nil {
			return "", r, e
		}
		r.Args = []string{"mcp", "connect", "--url", o.URL, "--token-file", o.TokenFile}
	}
	return p, r, nil
}
func read(p, h string) (map[string]any, []byte, error) {
	info, e := os.Lstat(p)
	if errors.Is(e, os.ErrNotExist) {
		return map[string]any{}, nil, nil
	}
	if e != nil {
		return nil, nil, e
	}
	if !info.Mode().IsRegular() {
		return nil, nil, errors.New("configuration must be a regular file, not a symlink")
	}
	if info.Size() > 4<<20 {
		return nil, nil, errors.New("configuration exceeds 4 MiB")
	}
	f, e := os.Open(p)
	if e != nil {
		return nil, nil, e
	}
	defer f.Close()
	opened, e := f.Stat()
	if e != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, nil, errors.New("configuration changed while opening")
	}
	b, e := io.ReadAll(io.LimitReader(f, (4<<20)+1))
	if e != nil {
		return nil, nil, e
	}
	if len(b) > 4<<20 {
		return nil, nil, errors.New("configuration exceeds 4 MiB")
	}
	m := map[string]any{}
	if h == "codex" {
		e = toml.Unmarshal(b, &m)
	} else {
		if mcpserver.ValidateJSON(b) != nil {
			return nil, nil, errors.New("malformed or ambiguous harness configuration")
		}
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.UseNumber()
		e = dec.Decode(&m)
		if e == nil {
			var extra any
			if dec.Decode(&extra) != io.EOF {
				e = errors.New("multiple JSON values")
			}
		}
	}
	if e != nil || m == nil {
		return nil, nil, errors.New("malformed harness configuration")
	}
	return m, b, nil
}
func key(h string) string {
	if h == "codex" {
		return "mcp_servers"
	}
	return "mcpServers"
}
func entries(m map[string]any, h string) (map[string]any, error) {
	k := key(h)
	v, ok := m[k]
	if !ok {
		v = map[string]any{}
		m[k] = v
	}
	s, ok := v.(map[string]any)
	if !ok {
		return nil, errors.New("MCP configuration must be a table or object")
	}
	return s, nil
}
func desired(o Options, r Registration) map[string]any {
	v := map[string]any{"command": r.Command, "args": r.Args}
	if o.Harness == "codex" {
		v["tool_timeout_sec"] = int64(310)
	} else if o.Harness == "claude" {
		v["timeout"] = int64(310000)
	} else {
		v["timeoutSeconds"] = int64(310)
	}
	return v
}
func decode(v any) (Registration, error) {
	b, e := json.Marshal(v)
	if e != nil {
		return Registration{}, e
	}
	var r Registration
	e = json.Unmarshal(b, &r)
	return r, e
}

// ValidateRegistration rejects arbitrary commands, environment injection, and unknown arguments.
func ValidateRegistration(r Registration) error {
	if !filepath.IsAbs(r.Command) || filepath.Base(r.Command) != "second-opinion" {
		return errors.New("registration must launch an absolute second-opinion executable")
	}
	if reflect.DeepEqual(r.Args, []string{"mcp", "serve"}) {
		return nil
	}
	if len(r.Args) == 6 && r.Args[0] == "mcp" && r.Args[1] == "connect" && r.Args[2] == "--url" && r.Args[4] == "--token-file" && filepath.IsAbs(r.Args[5]) {
		return mcpclient.ValidateURL(r.Args[3])
	}
	return errors.New("unrecognized Second Opinion arguments")
}
func Inspect(o Options, getenv func(string) string) (Registration, bool, error) {
	p, _, e := Resolve(o, getenv)
	if e != nil {
		return Registration{}, false, e
	}
	m, _, e := read(p, o.Harness)
	if e != nil {
		return Registration{}, false, e
	}
	s, e := entries(m, o.Harness)
	if e != nil {
		return Registration{}, false, e
	}
	v, ok := s["second-opinion"]
	if !ok {
		return Registration{}, false, nil
	}
	r, e := decode(v)
	if e != nil {
		return r, true, e
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return r, true, errors.New("invalid registration")
	}
	for k := range obj {
		if k != "command" && k != "args" && !(o.Harness == "codex" && k == "tool_timeout_sec") && !(o.Harness == "claude" && k == "timeout") && !(o.Harness == "antigravity" && k == "timeoutSeconds") {
			return r, true, fmt.Errorf("unsupported registration field %s", k)
		}
	}
	if e = ValidateRegistration(r); e != nil {
		return r, true, e
	}
	timeoutKey, wantTimeout := "tool_timeout_sec", "310"
	if o.Harness == "claude" {
		timeoutKey = "timeout"
		wantTimeout = "310000"
	}
	if o.Harness == "antigravity" {
		timeoutKey = "timeoutSeconds"
	}
	{
		b, _ := json.Marshal(obj[timeoutKey])
		value, err := strconv.ParseFloat(string(b), 64)
		minimum, _ := strconv.ParseFloat(wantTimeout, 64)
		if err != nil || value < minimum {
			return r, true, errors.New("tool timeout must allow 310 seconds")
		}
	}
	return r, true, nil
}
func Apply(o Options, getenv func(string) string, uninstall bool) (Report, error) {
	p, r, e := Resolve(o, getenv)
	report := Report{Path: p, Harness: o.Harness, DryRun: o.DryRun, Registration: r}
	if e == nil {
		e = ValidateRegistration(r)
	}
	if e != nil {
		return report, e
	}
	m, original, e := read(p, o.Harness)
	if e != nil {
		return report, e
	}
	s, e := entries(m, o.Harness)
	if e != nil {
		return report, e
	}
	old, exists := s["second-opinion"]
	want := desired(o, r)
	report.Entry = want
	report.Operation = "install"
	if uninstall {
		report.Operation = "uninstall"
	}
	a, _ := json.Marshal(old)
	b, _ := json.Marshal(want)
	same := bytes.Equal(a, b)
	if uninstall {
		if !exists {
			return report, nil
		}
		existing, ok, err := Inspect(o, getenv)
		if err != nil || !ok || existing.Command != r.Command {
			return report, errors.New("existing registration differs; refusing to remove it")
		}
		report.Registration = existing
		report.Entry = old.(map[string]any)
		delete(s, "second-opinion")
	} else {
		if same {
			return report, nil
		}
		if exists && !o.Replace {
			return report, errors.New("conflicting registration; use --replace to replace it")
		}
		s["second-opinion"] = want
	}
	report.Changed = true
	if o.DryRun {
		return report, nil
	}
	var out []byte
	if o.Harness == "codex" {
		out, e = toml.Marshal(m)
	} else {
		out, e = json.MarshalIndent(m, "", "  ")
		out = append(out, '\n')
	}
	if e != nil {
		return report, e
	}
	if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		return report, e
	}
	f, e := os.CreateTemp(filepath.Dir(p), ".second-opinion-*")
	if e != nil {
		return report, e
	}
	defer os.Remove(f.Name())
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(out)
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return report, e
	}
	_, current, e := read(p, o.Harness)
	if e != nil {
		return report, e
	}
	if !bytes.Equal(current, original) {
		return report, errors.New("configuration changed during setup; retry")
	}
	if original != nil {
		backup, e := os.CreateTemp(filepath.Dir(p), filepath.Base(p)+".second-opinion-backup-*")
		if e != nil {
			return report, e
		}
		report.Backup = backup.Name()
		if e = backup.Chmod(0600); e == nil {
			_, e = backup.Write(original)
		}
		ce := backup.Close()
		if e == nil {
			e = ce
		}
		if e != nil {
			return report, e
		}
	}
	return report, os.Rename(f.Name(), p)
}
