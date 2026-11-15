// Package buildinfo identifies installed builds independently of wire schemas.
package buildinfo

import (
	_ "embed"
	"runtime/debug"
	"strings"
)

//go:embed VERSION
var baseVersion string

// Version reports the source version plus available Go VCS build metadata.
func Version() string {
	info, _ := debug.ReadBuildInfo()
	return formatVersion(strings.TrimSpace(baseVersion), info)
}

func formatVersion(base string, info *debug.BuildInfo) string {
	if info == nil {
		return base
	}
	var revision string
	var dirty bool
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			dirty = setting.Value == "true"
		}
	}
	var metadata []string
	if revision != "" {
		metadata = append(metadata, "g"+revision)
	}
	if dirty {
		metadata = append(metadata, "dirty")
	}
	if len(metadata) == 0 {
		return base
	}
	return base + "+" + strings.Join(metadata, ".")
}
