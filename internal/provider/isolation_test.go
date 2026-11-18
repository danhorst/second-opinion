package provider

import (
	"context"
	"errors"
	"testing"
)

type isolationReference struct {
	level Isolation
	calls int
}

func (p *isolationReference) Isolation() Isolation { return p.level }
func (p *isolationReference) Review(_ context.Context, r Request) (*Result, error) {
	p.calls++
	return &Result{Findings: r.Material, Isolation: IsolationLocalMaterialOnly}, nil
}

type noIsolation struct{ Provider }

func TestIsolationExecutionPolicy(t *testing.T) {
	for _, tc := range []struct {
		name            string
		level, required Isolation
		hide, allowed   bool
	}{
		{"supported", IsolationLocalMaterialOnly, IsolationLocalMaterialOnly, false, true},
		{"unsupported", IsolationUnverified, IsolationLocalMaterialOnly, false, false},
		{"absent", IsolationLocalMaterialOnly, IsolationLocalMaterialOnly, true, false},
		{"unknown capability", Isolation("future"), IsolationLocalMaterialOnly, false, false},
		{"unknown requirement", IsolationLocalMaterialOnly, Isolation("future"), false, false},
		{"unverified is not a requirement", IsolationUnverified, IsolationUnverified, false, false},
		{"default", IsolationUnverified, "", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ref := &isolationReference{level: tc.level}
			var p Provider = ref
			if tc.hide {
				p = noIsolation{p}
			}
			result, err := ReviewWithIsolation(context.Background(), p, Request{Material: "exact material"}, tc.required)
			if !tc.allowed {
				if !errors.Is(err, ErrIsolationUnsupported) || result != nil || ref.calls != 0 {
					t.Fatalf("result=%+v err=%v calls=%d", result, err, ref.calls)
				}
			} else if err != nil || result == nil || ref.calls != 1 || result.Findings != "exact material" || result.Isolation != EffectiveIsolation(p) {
				t.Fatalf("result=%+v err=%v calls=%d", result, err, ref.calls)
			}
		})
	}
}
