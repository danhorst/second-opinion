package buildinfo

import (
	"runtime/debug"
	"testing"
)

func TestVersionMetadata(t *testing.T) {
	for _, tt := range []struct {
		info *debug.BuildInfo
		want string
	}{
		{nil, "0.2.0"}, {&debug.BuildInfo{}, "0.2.0"},
		{&debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc123"}}}, "0.2.0+gabc123"},
		{&debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc123"}, {Key: "vcs.modified", Value: "true"}}}, "0.2.0+gabc123.dirty"},
	} {
		if got := formatVersion("0.2.0", tt.info); got != tt.want {
			t.Errorf("got %s want %s", got, tt.want)
		}
	}
}
