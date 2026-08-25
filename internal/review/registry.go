package review

import (
	"fmt"
	"sort"

	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/provider/antigravity"
	"github.com/danhorst/second-opinion/internal/provider/claude"
	"github.com/danhorst/second-opinion/internal/provider/codex"
	"github.com/danhorst/second-opinion/internal/provider/openai_compatible"
)

// registry maps provider names to constructors. Adding an adapter means
// adding one line. model is the forced model; empty means the provider's
// default.
var registry = map[string]func(model string) provider.Provider{
	"antigravity": func(model string) provider.Provider {
		if model != "" {
			return antigravity.New(antigravity.WithModel(model))
		}
		return antigravity.New()
	},
	"claude": func(model string) provider.Provider {
		if model != "" {
			return claude.New(claude.WithModel(model))
		}
		return claude.New()
	},
	"codex": func(model string) provider.Provider {
		if model != "" {
			return codex.New(codex.WithModel(model))
		}
		return codex.New()
	},
	"openai-compatible": func(model string) provider.Provider {
		if model != "" {
			return openai_compatible.New(openai_compatible.WithModel(model))
		}
		return openai_compatible.New()
	},
}

// Providers returns the registered provider names, sorted.
func Providers() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// NewProvider constructs the named provider with an optional forced model.
func NewProvider(name, model string) (provider.Provider, error) {
	construct, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("review: unknown provider %q (registered: %v)", name, Providers())
	}
	return construct(model), nil
}
