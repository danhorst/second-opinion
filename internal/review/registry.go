package review

import (
	"fmt"
	"sort"

	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/provider/antigravity"
	"github.com/danhorst/second-opinion/internal/provider/claude"
	"github.com/danhorst/second-opinion/internal/provider/codex"
)

// registry maps provider names to constructors. Adding an adapter means
// adding one line. model is the forced model; empty means the provider's
// default.
var registry = map[string]func(model string, limits provider.CaptureLimits) provider.Provider{
	"antigravity": func(model string, limits provider.CaptureLimits) provider.Provider {
		return antigravity.New(antigravity.WithModel(model), antigravity.WithCaptureLimits(limits))
	},
	"claude": func(model string, limits provider.CaptureLimits) provider.Provider {
		return claude.New(claude.WithModel(model), claude.WithCaptureLimits(limits))
	},
	"codex": func(model string, limits provider.CaptureLimits) provider.Provider {
		return codex.New(codex.WithModel(model), codex.WithCaptureLimits(limits))
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
	return NewProviderWithLimits(name, model, provider.CaptureLimits{})
}

// NewProviderWithLimits constructs a fresh provider with optional output budgets.
// Zero limits preserve the unbounded capture behavior of NewProvider.
func NewProviderWithLimits(name, model string, limits provider.CaptureLimits) (provider.Provider, error) {
	construct, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("review: unknown provider %q (registered: %v)", name, Providers())
	}
	return construct(model, limits), nil
}
