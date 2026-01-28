package core

import (
	"testing"

	"github.com/comalice/statechartx/internal/primitives"
)

func TestComputeLCCA(t *testing.T) {
	tests := []struct {
		source, target, lcca string
	}{
		{"a.b.c", "a.b.d", "a.b"},
		{"a.b", "a.c", "a"},
		{"a", "b", ""},
		{"a.b.c", "a.b.c", "a.b.c"},
	}
	for _, tt := range tests {
		if got := computeLCCA(tt.source, tt.target); got != tt.lcca {
			t.Errorf("computeLCCA(%q, %q) = %q, want %q", tt.source, tt.target, got, tt.lcca)
		}
	}
}

func TestGetAncestors(t *testing.T) {
	tests := []struct {
		path string
		want []string
	}{
		{"a", []string{"a"}},
		{"a.b", []string{"a", "a.b"}},
		{"a.b.c", []string{"a", "a.b", "a.b.c"}},
	}
	for _, tt := range tests {
		if got := getAncestors(tt.path); !equalStringSlices(got, tt.want) {
			t.Errorf("getAncestors(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestResolveInitialLeaf(t *testing.T) {
	child1 := primitives.NewStateConfig("child1", primitives.Atomic)
	child2 := primitives.NewStateConfig("child2", primitives.Atomic)
	compound := primitives.NewStateConfig("compound", primitives.Compound).
		WithInitial("child1").
		WithChildren([]*primitives.StateConfig{child1, child2})

	config := primitives.MachineConfig{
		States: map[string]*primitives.StateConfig{"compound": compound},
	}

	if got := resolveInitialLeaf(&config, "compound"); got != "compound.child1" {
		t.Errorf("resolveInitialLeaf(compound) = %q, want compound.child1", got)
	}
}
