package review

import (
	"bytes"
	"os"
	"testing"
)

// Assembly-only baseline; provider latency and model quality are not measured.
func BenchmarkFromFiles(b *testing.B) {
	for _, size := range []struct {
		name  string
		bytes int
	}{{"1MiB", 1 << 20}, {"10MiB", 10 << 20}} {
		b.Run(size.name, func(b *testing.B) {
			path := b.TempDir() + "/material.txt"
			if err := os.WriteFile(path, bytes.Repeat([]byte("x"), size.bytes), 0o600); err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(size.bytes))
			b.ReportAllocs()
			for b.Loop() {
				if _, err := FromFiles([]string{path}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
