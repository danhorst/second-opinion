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
type registration struct {
	construct func(model string, limits provider.CaptureLimits) provider.Provider
	isolation provider.Isolation
}

var registry = map[string]registration{
	"antigravity": {construct: func(model string, limits provider.CaptureLimits) provider.Provider {
		return antigravity.New(antigravity.WithModel(model), antigravity.WithCaptureLimits(limits))
	}},
	"claude": {construct: func(model string, limits provider.CaptureLimits) provider.Provider {
		return claude.New(claude.WithModel(model), claude.WithCaptureLimits(limits))
	}},
	"codex": {construct: func(model string, limits provider.CaptureLimits) provider.Provider {
		return codex.New(codex.WithModel(model), codex.WithCaptureLimits(limits))
	}},
	"openai-compatible": {isolation: provider.IsolationLocalMaterialOnly, construct: func(model string, limits provider.CaptureLimits) provider.Provider {
		opts := []openai_compatible.Option{openai_compatible.WithCaptureLimits(limits)}
		if model != "" {
			opts = append(opts, openai_compatible.WithModel(model))
		}
		return openai_compatible.New(opts...)
	}},
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

// ProviderIsolation reports local adapter capabilities without executing reviews
// or reading endpoint credentials. Execution checks the constructed provider
// independently; discovery metadata cannot authorize execution.
func ProviderIsolation() map[string]provider.Isolation {
	levels := make(map[string]provider.Isolation, len(registry))
	for name, entry := range registry {
		levels[name] = provider.IsolationUnverified
		if entry.isolation == provider.IsolationLocalMaterialOnly {
			levels[name] = entry.isolation
		}
	}
	return levels
}

// NewProvider constructs the named provider with an optional forced model.
func NewProvider(name, model string) (provider.Provider, error) {
	return NewProviderWithLimits(name, model, provider.CaptureLimits{})
}

// NewProviderWithLimits constructs a fresh provider with optional output budgets.
// Zero limits preserve the unbounded capture behavior of NewProvider.
func NewProviderWithLimits(name, model string, limits provider.CaptureLimits) (provider.Provider, error) {
	construct, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("review: unknown provider %q (registered: %v)", name, Providers())
	}
	return construct.construct(model, limits), nil
}
