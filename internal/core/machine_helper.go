// Helper functions for machine precomputation and path calculations.
// Placed in separate file to organize code.

package core

import (
	"github.com/comalice/statechartx/internal/primitives"
)

// precomputePaths recursively traverses the state hierarchy starting from a state with given prefix.
// Builds stateCache[path] = stateConfig and ancestorCache[path] = []ancestorPaths including self.
func precomputePaths(state *primitives.StateConfig, prefix string, stateCache map[string]*primitives.StateConfig, ancestorCache map[string][]string) {
	fullpath := prefix
	if prefix != "" {
		fullpath += "."
	}
	fullpath += state.ID

	stateCache[fullpath] = state

	// Compute ancestors incrementally
	var ancestors []string
	if prefix == "" {
		ancestors = []string{fullpath}
	} else {
		prefixAncestors, exists := ancestorCache[prefix]
		if !exists {
			return // should not happen
		}
		ancestors = append(make([]string, 0, len(prefixAncestors)+1), prefixAncestors...)
		ancestors = append(ancestors, fullpath)
	}
	ancestorCache[fullpath] = ancestors

	// Recurse children
	for _, child := range state.Children {
		precomputePaths(child, fullpath, stateCache, ancestorCache)
	}
}
