package provider

import (
	"context"
	"errors"
	"fmt"
)

// Isolation describes a client-side boundary, never endpoint-side context or
// model certification. Empty means no execution requirement was supplied.
type Isolation string

const (
	IsolationUnverified        Isolation = "unverified"
	IsolationLocalMaterialOnly Isolation = "local-material-only"
)

// IsolationProvider is optional. Implementations must document and test their
// enforcement boundary; a successful cold probe is insufficient evidence.
type IsolationProvider interface {
	Isolation() Isolation
}

var ErrIsolationUnsupported = errors.New("provider: required isolation unsupported")

// EffectiveIsolation defaults closed for absent or unrecognized capabilities.
func EffectiveIsolation(p Provider) Isolation {
	if capable, ok := p.(IsolationProvider); ok && capable.Isolation() == IsolationLocalMaterialOnly {
		return IsolationLocalMaterialOnly
	}
	return IsolationUnverified
}

// ReviewWithIsolation gates execution before Review, without changing content
// or allowing a fallback. It records adapter policy separately from provenance.
func ReviewWithIsolation(ctx context.Context, p Provider, req Request, required Isolation) (*Result, error) {
	effective := EffectiveIsolation(p)
	if required != "" && (required != IsolationLocalMaterialOnly || effective != required) {
		return nil, fmt.Errorf("%w: requested %q, available %q", ErrIsolationUnsupported, required, effective)
	}
	result, err := p.Review(ctx, req)
	if err != nil {
		return result, &reviewInvocationError{err}
	}
	if result != nil && err == nil {
		result.Isolation = effective
	}
	return result, err
}

// reviewInvocationError distinguishes policy rejection from provider errors after
// invocation. Wrapping preserves classification without claiming remote execution.
type reviewInvocationError struct{ err error }

func (e *reviewInvocationError) Error() string { return e.err.Error() }
func (e *reviewInvocationError) Unwrap() error { return e.err }

// IsolationRejectedBeforeReview is true only for a policy rejection outside a
// provider invocation, including when a provider forwards a nested gate error.
func IsolationRejectedBeforeReview(err error) bool {
	var invoked *reviewInvocationError
	return errors.Is(err, ErrIsolationUnsupported) && !errors.As(err, &invoked)
}
