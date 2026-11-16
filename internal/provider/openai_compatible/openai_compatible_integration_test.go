//go:build integration

package openai_compatible

import (
	"os"
	"testing"

	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/provider/providertest"
)

func TestOpenAICompatibleConformance(t *testing.T) {
	if os.Getenv("SECOND_OPINION_API_KEY") == "" || os.Getenv("SECOND_OPINION_API_MODEL") == "" {
		t.Skip("set SECOND_OPINION_API_KEY and SECOND_OPINION_API_MODEL for endpoint conformance")
	}
	providertest.Conform(t, func(t *testing.T) provider.Provider {
		return New()
	})
}
