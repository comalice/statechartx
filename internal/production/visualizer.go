// Package production provides production integrations: persistence, event publishing, visualization.
// Implements core interfaces using stdlib where possible.
package production

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/comalice/statechartx/internal/primitives"
)

// DefaultVisualizer is the stdlib-only implementation of Visualizer.
type DefaultVisualizer struct{}

// ExportDOT generates Graphviz DOT source for the statechart.
func (v *DefaultVisualizer) ExportDOT(config primitives.MachineConfig, current []string) string {
	var buf bytes.Buffer
	buf.WriteString(`digraph Statechart {
  rankdir=LR;
  node [shape=box, fontsize=10, style=rounded];
  edge [fontsize=9];
`)

	active := getActiveStates(current)
	edges := collectEdges(config)
	roots := findRoots(config)

	for _, root := range roots {
		renderState(&buf, root, config, active)
	}

	for _, edge := range edges {
		label := edge.Label
		buf.WriteString(fmt.Sprintf(`  "%s" -> "%s" [label="%s"];\n`, edge.From, edge.To, label))
	}

	buf.WriteString("}\n")
	return buf.String()
}

// ExportJSON serializes the machine config to JSON.
func (v *DefaultVisualizer) ExportJSON(config primitives.MachineConfig) ([]byte, error) {
	return json.MarshalIndent(config, "", "  ")
}

// getActiveStates returns map of active state IDs from current paths.
func getActiveStates(current []string) map[string]bool {
	active := make(map[string]bool)
	for _, path := range current {
		segments := strings.Split(path, ".")
		for _, id := range segments {
			if id != "" {
				active[id] = true
			}
		}
	}
	return active
}

// Edge represents a transition edge.
type Edge struct {
	From  string
	To    string
	Label string
}

// collectEdges collects all transitions.
func collectEdges(config primitives.MachineConfig) []Edge {
	var edges []Edge
	for _, state := range config.States {
		if state.On != nil {
			for event, transList := range state.On {
				for _, trans := range transList {
					if trans.Target != "" {
						targetState, err := config.FindState(trans.Target)
						if err == nil && targetState != nil {
							edges = append(edges, Edge{
								From:  state.ID,
								To:    targetState.ID,
								Label: event,
							})
						}
					}
				}
			}
		}
	}
	return edges
}

// findRoots finds top-level states (not children of any state).
func findRoots(config primitives.MachineConfig) []*primitives.StateConfig {
	childIDs := make(map[string]bool)
	for _, s := range config.States {
		for _, c := range s.Children {
			childIDs[c.ID] = true
		}
	}
	var roots []*primitives.StateConfig
	for _, s := range config.States {
		if !childIDs[s.ID] {
			roots = append(roots, s)
		}
	}
	return roots
}

// renderState recursively renders states and subgraphs.
func renderState(buf *bytes.Buffer, state *primitives.StateConfig, config primitives.MachineConfig, active map[string]bool) {
	if len(state.Children) > 0 {
		// Compound or parallel: cluster
		clusterID := fmt.Sprintf(`cluster_%s`, state.ID)
		buf.WriteString(fmt.Sprintf("  subgraph %s {\n", clusterID))
		parentLabel := fmt.Sprintf("%s (%s)", state.ID, state.Type)
		parentStyle := ""
		if active[state.ID] {
			parentStyle = ` style=filled fillcolor=orange`
		}
		buf.WriteString(fmt.Sprintf(`    label="%s"%s;\n`, parentLabel, parentStyle))
		if state.Type == primitives.Parallel {
			buf.WriteString(` style=filled fillcolor=lightblue;`)
		}
		buf.WriteString("\n")

		// Parent node
		buf.WriteString(fmt.Sprintf(`    "%s" [label="%s" shape=ellipse%s];\n`, state.ID, state.ID, parentStyle))

		// Children
		for _, child := range state.Children {
			renderState(buf, child, config, active)
		}

		buf.WriteString("  }\n")
	} else {
		// Atomic leaf
		style := ""
		if active[state.ID] {
			style = ` style=filled fillcolor=lightgreen`
		}
		buf.WriteString(fmt.Sprintf(`  "%s" [label="%s"%s];\n`, state.ID, state.ID, style))
		buf.WriteString("\n")
	}
}
