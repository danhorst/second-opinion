package process

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"testing"
)

func TestArgumentLimitDiagnostic(t *testing.T) {
	err := explainStartError(fmt.Errorf("exec: %w", syscall.E2BIG))
	if !errors.Is(err, syscall.E2BIG) || !strings.Contains(err.Error(), "reduce the material") {
		t.Fatalf("unhelpful error: %v", err)
	}
}
