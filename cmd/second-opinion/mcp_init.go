package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func runMCPInit(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mcp init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	path := fs.String("token-file", "", "service token file (default: $SECOND_OPINION_MCP_TOKEN_FILE or ~/.config/second-opinion/mcp-token)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	explicit := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "token-file" {
			explicit = true
		}
	})
	if fs.NArg() != 0 || (explicit && *path == "") {
		fmt.Fprintln(stderr, "second-opinion: mcp init accepts only --token-file PATH")
		return 2
	}
	if err := serviceUserSupported(); err != nil {
		fmt.Fprintln(stderr, "second-opinion:", err)
		return 1
	}
	if !explicit {
		*path = getenv("SECOND_OPINION_MCP_TOKEN_FILE")
	}
	if *path == "" {
		home := getenv("HOME")
		if home == "" {
			fmt.Fprintln(stderr, "second-opinion: HOME or --token-file is required")
			return 1
		}
		*path = filepath.Join(home, ".config", "second-opinion", "mcp-token")
	}
	absolute, err := filepath.Abs(*path)
	if err != nil {
		fmt.Fprintln(stderr, "second-opinion: cannot resolve token path")
		return 1
	}
	created, err := initializeMCPToken(absolute)
	if err != nil {
		fmt.Fprintln(stderr, "second-opinion:", err)
		return 1
	}
	status := "Validated existing token (unchanged)"
	if created {
		status = "Created private token"
	}
	_, err = fmt.Fprintf(stdout, "%s at %q.\nToken value was not printed.\nRegister a harness with: second-opinion mcp install --harness codex --transport http --token-file %q\nIf using a custom token path, configure SECOND_OPINION_MCP_TOKEN_FILE in the service environment.\nThen run: brew services start danhorst/tap/second-opinion\n", status, absolute, absolute)
	if err != nil {
		return 1
	}
	return 0
}

// Publish a complete credential atomically without replacing an existing name.
func initializeMCPToken(path string) (bool, error) {
	if _, err := os.Lstat(path); err == nil {
		return false, validateMCPTokenFile(path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, errors.New("cannot inspect token file")
	}
	var entropy [32]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return false, errors.New("cannot generate a secure token")
	}
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return false, errors.New("cannot create token directory")
	}
	temp, err := os.CreateTemp(parent, ".mcp-token-*")
	if err != nil {
		return false, errors.New("cannot create private token file")
	}
	defer os.Remove(temp.Name())
	defer temp.Close()
	if err := temp.Chmod(0600); err != nil {
		return false, errors.New("cannot set private token permissions")
	}
	if _, err := io.WriteString(temp, hex.EncodeToString(entropy[:])+"\n"); err != nil {
		return false, errors.New("cannot write token file")
	}
	if err := temp.Sync(); err != nil {
		return false, errors.New("cannot flush token file")
	}
	if err := temp.Close(); err != nil {
		return false, errors.New("cannot close token file")
	}
	if err := os.Link(temp.Name(), path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return false, validateMCPTokenFile(path)
		}
		return false, errors.New("cannot publish token file without replacement")
	}
	return true, nil
}

func validateMCPTokenFile(path string) error {
	_, err := readMCPTokenFile(path)
	return err
}

func readMCPTokenFile(path string) (string, error) {
	invalid := errors.New("existing token must be a private user-owned regular nonsymlink file with a valid 32–4096-character bearer token; left unchanged")
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() {
		return "", invalid
	}
	if before.Mode().Perm() != 0400 && before.Mode().Perm() != 0600 {
		return "", invalid
	}
	if before.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return "", invalid
	}
	if !serviceFileOwned(before) || before.Size() < 32 || before.Size() > 4097 {
		return "", invalid
	}
	f, err := os.Open(path)
	if err != nil {
		return "", invalid
	}
	defer f.Close()
	after, err := f.Stat()
	if err != nil || !os.SameFile(before, after) || !after.Mode().IsRegular() || !serviceFileOwned(after) || (after.Mode().Perm() != 0400 && after.Mode().Perm() != 0600) {
		return "", invalid
	}
	data, err := io.ReadAll(io.LimitReader(f, 4098))
	if err != nil || len(data) > 4097 {
		return "", invalid
	}
	token := strings.TrimSuffix(string(data), "\n")
	if len(token) < 32 || len(token) > 4096 {
		return "", invalid
	}
	for _, c := range []byte(token) {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("._~+/=-", rune(c))) {
			return "", invalid
		}
	}
	return token, nil
}
