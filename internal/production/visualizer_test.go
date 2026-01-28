// Tests for DefaultVisualizer DOT export and hierarchy rendering.
package production

import (
	"strings"
	"testing"

	"github.com/comalice/statechartx/internal/primitives"
)

func TestDefaultVisualizer_ExportDOT_Simple(t *testing.T) {
	v := &DefaultVisualizer{}
	config := primitives.MachineConfig{
		ID:      "simple",
		Initial: "s1",
		States: map[string]*primitives.StateConfig{
			"s1": {
				ID:   "s1",
				Type: primitives.Atomic,
				On: map[string][]primitives.TransitionConfig{
					"e1": {{Target: "s2"}},
				},
			},
			"s2": {ID: "s2", Type: primitives.Atomic},
		},
	}
	dot := v.ExportDOT(config, []string{"s2"})

	if !strings.Contains(dot, `digraph Statechart {`) {
		t.Error("Missing DOT header")
	}
	if !strings.Contains(dot, `"s1"`) || !strings.Contains(dot, `"s2"`) {
		t.Error("Missing state nodes")
	}
	if !strings.Contains(dot, `"s1" -> "s2" [label="e1"]`) {
		t.Error("Missing transition edge")
	}
	if !strings.Contains(dot, `fillcolor=lightgreen`) {
		t.Error("Missing active state highlight")
	}
}

func TestDefaultVisualizer_ExportDOT_Hierarchy(t *testing.T) {
	v := &DefaultVisualizer{}
	config := primitives.MachineConfig{
		ID:      "hierarchical",
		Initial: "parent",
		States: map[string]*primitives.StateConfig{
			"parent": {
				ID:      "parent",
				Type:    primitives.Compound,
				Initial: "child1",
				Children: []*primitives.StateConfig{
					{ID: "child1", Type: primitives.Atomic},
					{ID: "child2", Type: primitives.Atomic},
				},
			},
		},
	}
	dot := v.ExportDOT(config, []string{"parent.child1"})

	if !strings.Contains(dot, `subgraph cluster_parent {`) {
		t.Error("Missing compound cluster")
	}
	if !strings.Contains(dot, `"parent"`) && strings.Contains(dot, `"child1"`) && strings.Contains(dot, `"child2"`) {
		t.Error("Missing hierarchical states")
	}
	if !strings.Contains(dot, `fillcolor=orange`) {
		t.Error("Missing parent active highlight")
	}
}

func TestDefaultVisualizer_ExportDOT_Parallel(t *testing.T) {
	v := &DefaultVisualizer{}
	config := primitives.MachineConfig{
		ID:      "parallel",
		Initial: "parallel",
		States: map[string]*primitives.StateConfig{
			"parallel": {
				ID:      "parallel",
				Type:    primitives.Parallel,
				Initial: "r1",
				Children: []*primitives.StateConfig{
					{
						ID:       "r1",
						Type:     primitives.Compound,
						Children: []*primitives.StateConfig{{ID: "r1.s1", Type: primitives.Atomic}},
					},
					{
						ID:       "r2",
						Type:     primitives.Compound,
						Children: []*primitives.StateConfig{{ID: "r2.s1", Type: primitives.Atomic}},
					},
				},
			},
		},
	}
	dot := v.ExportDOT(config, []string{"parallel.r1.r1.s1", "parallel.r2.r2.s1"})

	if !strings.Contains(dot, `cluster_parallel`) {
		t.Error("Missing parallel cluster")
	}
	if !strings.Contains(dot, `fillcolor=lightblue`) {
		t.Error("Missing parallel style")
	}
}

func TestDefaultVisualizer_ExportJSON(t *testing.T) {
	v := &DefaultVisualizer{}
	config := primitives.MachineConfig{ID: "json-test", Initial: "s1", States: nil}
	data, err := v.ExportJSON(config)
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}
	if !strings.Contains(string(data), `"id": "json-test"`) {
		t.Error("JSON missing expected field")
	}
}
