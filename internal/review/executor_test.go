package review

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/provider/providertest"
)

// These callbacks exercise orchestration and result validation only. They do
// not emulate provider executables or establish adapter conformance.
type engineProvider func(context.Context, provider.Request) (*provider.Result, error)

func (p engineProvider) Review(ctx context.Context, req provider.Request) (*provider.Result, error) {
	return p(ctx, req)
}

func TestExecutorRequestIsolation(t *testing.T) {
	t.Setenv("SECOND_OPINION_PROVIDER", "environment-must-not-select")
	prompt := "  custom ☃\r\n"
	requests := []ExecutionRequest{
		{Provider: "first", Model: "forced", Prompt: &prompt, Material: " /some/path\r\n☃  ", CaptureLimits: provider.CaptureLimits{StdoutBytes: 42, StderrBytes: 7}},
		{Provider: "second", Material: "B\n"},
	}
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "caller")
	constructions := 0
	executor := Executor{NewProvider: func(name, model string, limits provider.CaptureLimits) (provider.Provider, error) {
		index := constructions
		constructions++
		want := requests[index]
		if name != want.Provider || model != want.Model || limits != want.CaptureLimits {
			t.Fatalf("construction = %q %q %+v, want %+v", name, model, limits, want)
		}
		calls := 0
		return engineProvider(func(gotCtx context.Context, got provider.Request) (*provider.Result, error) {
			calls++
			if calls != 1 {
				t.Fatal("provider reused")
			}
			if gotCtx != ctx || got.Prompt != ResolvePrompt(want.Prompt) || got.Material != want.Material {
				t.Fatalf("request/context changed: %+v", got)
			}
			return providertest.NewLoopback().Review(gotCtx, got)
		}), nil
	}}
	for _, req := range requests {
		if _, err := executor.Review(ctx, req); err != nil {
			t.Fatal(err)
		}
	}
	if constructions != len(requests) {
		t.Fatalf("constructed %d providers", constructions)
	}
}

func TestResolvePrompt(t *testing.T) {
	if ResolvePrompt(nil) != BakedPrompt {
		t.Fatal("omission did not select baked prompt")
	}
	for _, prompt := range []string{"", " \t\n", "custom"} {
		if ResolvePrompt(&prompt) != prompt {
			t.Fatal("explicit prompt changed")
		}
	}
}

func TestExecutorResults(t *testing.T) {
	good := provider.Result{Findings: "  raw ☃\r\n", Provenance: provider.Provenance{Provider: "loopback", Model: "provider-default", PromptHash: provider.HashPrompt(BakedPrompt)}}
	for _, tc := range []struct {
		name      string
		mutate    func(*provider.Result)
		nilResult bool
		err       error
		wantErr   error
	}{
		{name: "preserves raw and provenance"},
		{name: "empty findings", mutate: func(r *provider.Result) { r.Findings = "" }},
		{name: "nil success", nilResult: true, wantErr: ErrInvalidResult},
		{name: "both result and error", err: provider.ErrAuth, wantErr: ErrInvalidResult},
		{name: "provider error", nilResult: true, err: provider.ErrUnavailable, wantErr: provider.ErrUnavailable},
		{name: "missing provider", mutate: func(r *provider.Result) { r.Provenance.Provider = " " }, wantErr: ErrInvalidResult},
		{name: "missing model", mutate: func(r *provider.Result) { r.Provenance.Model = "" }, wantErr: ErrInvalidResult},
		{name: "missing hash", mutate: func(r *provider.Result) { r.Provenance.PromptHash = "" }, wantErr: ErrInvalidResult},
		{name: "wrong hash", mutate: func(r *provider.Result) { r.Provenance.PromptHash = provider.HashPrompt("other") }, wantErr: ErrInvalidResult},
		{name: "invalid findings", mutate: func(r *provider.Result) { r.Findings = "\xff" }, wantErr: ErrInvalidResult},
		{name: "invalid provider", mutate: func(r *provider.Result) { r.Provenance.Provider = "\xff" }, wantErr: ErrInvalidResult},
		{name: "invalid model", mutate: func(r *provider.Result) { r.Provenance.Model = "\xff" }, wantErr: ErrInvalidResult},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := good
			if tc.mutate != nil {
				tc.mutate(&want)
			}
			e := Executor{NewProvider: func(string, string, provider.CaptureLimits) (provider.Provider, error) {
				return engineProvider(func(context.Context, provider.Request) (*provider.Result, error) {
					if tc.nilResult {
						return nil, tc.err
					}
					return &want, tc.err
				}), nil
			}}
			got, err := e.Review(context.Background(), ExecutionRequest{})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				if got != nil {
					t.Fatal("returned partial result on failure")
				}
			} else if !reflect.DeepEqual(got, &want) {
				t.Fatalf("result = %+v, want %+v", got, want)
			}
		})
	}
}

func TestExecutorCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	e := Executor{NewProvider: func(string, string, provider.CaptureLimits) (provider.Provider, error) {
		return providertest.NewLoopback(), nil
	}}
	if result, err := e.Review(ctx, ExecutionRequest{}); result != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled review = %+v, %v", result, err)
	}
}

func TestExecutorFactoryErrors(t *testing.T) {
	for _, factory := range []ProviderFactory{
		func(string, string, provider.CaptureLimits) (provider.Provider, error) {
			return nil, provider.ErrUnavailable
		},
		func(string, string, provider.CaptureLimits) (provider.Provider, error) { return nil, nil },
	} {
		if result, err := (Executor{NewProvider: factory}).Review(context.Background(), ExecutionRequest{}); err == nil || result != nil {
			t.Fatalf("invalid factory result = %+v, %v", result, err)
		}
	}
	if result, err := (Executor{}).Review(context.Background(), ExecutionRequest{Provider: "unknown"}); err == nil || result != nil {
		t.Fatalf("unknown registry provider = %+v, %v", result, err)
	}
}
