package main

import (
	"bytes"
	"context"
	"github.com/danhorst/second-opinion/internal/buildinfo"
	"testing"
)

func TestVersionNeedsNoProvider(t *testing.T) {
	var out, diagnostic bytes.Buffer
	if code := run(context.Background(), []string{"--version"}, nil, &out, &diagnostic, nil); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out.String() != buildinfo.Version()+"\n" || diagnostic.Len() != 0 {
		t.Fatal("incorrect version output")
	}
}
