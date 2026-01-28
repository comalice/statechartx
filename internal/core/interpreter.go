package core

import (
	"fmt"
	"strings"

	"github.com/comalice/statechartx/internal/primitives"
)

// candidateTransition represents a valid transition candidate from hierarchical search.
type candidateTransition struct {
	sourcePath string
	trans      primitives.TransitionConfig
	priority   int
}

// computeLCCA returns the least common compound ancestor path of source and target paths.
func computeLCCA(sourcePath, targetPath string) string {
	source := strings.Split(sourcePath, ".")
	target := strings.Split(targetPath, ".")

	minLen := len(source)
	if len(target) < minLen {
		minLen = len(target)
	}

	lcaIndex := 0
	for lcaIndex < minLen && source[lcaIndex] == target[lcaIndex] {
		lcaIndex++
	}

	if lcaIndex == 0 {
		return "" // No common ancestor
	}

	return strings.Join(source[:lcaIndex], ".")
}

// getAncestors returns all ancestor paths of a leaf path (including self).
func getAncestors(leafPath string) []string {
	segments := strings.Split(leafPath, ".")
	ancestors := make([]string, len(segments))

	current := ""
	for i, seg := range segments {
		if current != "" {
			current += "."
		}
		current += seg
		ancestors[i] = current
	}
	return ancestors
}

// expandStatePath returns the full path ancestors for a state path.
func expandStatePath(path string) []string {
	return getAncestors(path)
}

// getExitStates returns the states to exit: innermost to LCCA (reverse for execution).
func getExitStates(sourcePath, lccaPath string) []string {
	if lccaPath == "" {
		return []string{sourcePath}
	}

	source := strings.Split(sourcePath, ".")
	if !strings.HasPrefix(sourcePath, lccaPath+".") {
		return nil
	}

	lccaSegs := strings.Split(lccaPath, ".")
	exitSegs := source[len(lccaSegs):]

	paths := []string{}
	current := lccaPath
	for _, seg := range exitSegs {
		if current != "" {
			current += "."
		}
		current += seg
		paths = append(paths, current)
	}

	return paths
}

// getEntryStates returns the states to enter: LCCA to target (outer first).
func getEntryStates(lccaPath, targetPath string) []string {
	if lccaPath == "" {
		return []string{targetPath}
	}

	lccaSegs := strings.Split(lccaPath, ".")
	targetSegs := strings.Split(targetPath, ".")

	if len(targetSegs) <= len(lccaSegs) {
		return nil
	}

	entrySegs := targetSegs[len(lccaSegs):]

	paths := []string{}
	current := lccaPath
	for _, seg := range entrySegs {
		if current != "" {
			current += "."
		}
		current += seg
		paths = append(paths, current)
	}

	return paths
}

// resolveInitialLeaf recurses to the leaf initial state path for compound/parallel.
func resolveInitialLeaf(config *primitives.MachineConfig, path string) string {
	state, err := config.FindState(path)
	if err != nil {
		return path
	}

	if state.Type == primitives.Compound || state.Type == primitives.Parallel {
		if state.Initial == "" {
			return path
		}
		childPath := path
		if path != "" {
			childPath += "."
		}
		childPath += state.Initial
		return resolveInitialLeaf(config, childPath)
	}

	return path
}

// defaultGuardEval provides default guard evaluation for Phase 2.
func defaultGuardEval(ctx *primitives.Context, guard primitives.GuardRef, event primitives.Event) bool {
	if guard == nil {
		return true
	}
	if g, ok := guard.(func(*primitives.Context, primitives.Event) bool); ok {
		return g(ctx, event)
	}
	return false
}

// defaultActionRun provides default action execution for Phase 2.
func defaultActionRun(ctx *primitives.Context, action primitives.ActionRef, event primitives.Event) error {
	if action == nil {
		return nil
	}
	if a, ok := action.(func(*primitives.Context, primitives.Event)); ok {
		a(ctx, event)
		return nil
	}
	return fmt.Errorf("unregistered action: %v", action)
}

// syncParallelRegions stub for parallel region synchronization (Phase 2.4).
func syncParallelRegions(regions []*Machine) error {
	return nil // stub - Phase 2.4
}
