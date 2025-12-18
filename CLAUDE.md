READ A FILE BEFORE ATTEMPTING TO EDIT

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**statechartx** is a minimal, composable, concurrent-ready hierarchical state machine implementation in Go (~520 LOC). It provides:

- Hierarchical state nesting with proper entry/exit order
- Initial states and shallow history support
- Guarded transitions with actions
- Thread-safe event dispatch via `sync.RWMutex`
- Explicit composition for concurrent state machines via goroutines/channels
- No built-in parallel regions; parallelism achieved through composition

## Key Architecture

### Core Types (statechart.go)

- **State**: Hierarchical state node with ID, parent/children, transitions, initial/history states, entry/exit actions
- **Runtime**: Executable state machine instance managing active state configuration, thread-safe event processing
- **Transition**: Event-triggered edges with optional guards and actions
- **Event**: Any comparable Go type used to trigger transitions

### Key Concepts

1. **Hierarchical State Management**: States form a tree. `Runtime.current` tracks all active states (compound states + their active leaf descendants).

2. **Entry/Exit Order**: Transitions compute LCA (Lowest Common Ancestor) to exit states bottom-up and enter states top-down, preserving SCXML-like semantics.

3. **History**: Shallow history only. Parent states remember their last active child via `State.History`.

4. **Concurrency Model**: Single `Runtime` uses mutex for thread-safe `SendEvent`. For orthogonal regions, run multiple `Runtime` instances via `RunAsActor()` in separate goroutines.

5. **Extended State**: User context passed to all actions/guards via `ext any` parameter for application-specific data.

### File Structure

```
statechart.go              - Core state machine implementation
statechart_test.go         - Unit tests for basic functionality
statechart_scxml_tests.go  - SCXML conformance tests (currently empty)
cmd/scxml_dowloader/       - Utility to download W3C SCXML test suite
pkg/scxml_test_suite/      - Downloaded SCXML test files from W3C
```

## Development Commands

### Building
```bash
go build ./...
```

### Testing
```bash
make test           # Run all tests
make test-race      # Run with race detector (recommended)
go test -v ./...    # Verbose test output
```

### Test Coverage
```bash
make coverage       # Generates coverage.html
```

### Linting & Analysis
```bash
make lint           # Run revive linter
make vet            # Run go vet
make staticcheck    # Run staticcheck
make check          # All-in-one: format, vet, staticcheck, lint, test-race
```

### Formatting
```bash
make format         # Run gofmt and goimports
```

### Benchmarking
```bash
make bench          # Run benchmarks with memory stats
```

### Fuzzing
```bash
make fuzz           # Run fuzz tests for 30s each
```

### SCXML Test Suite
```bash
# Download W3C SCXML conformance tests
go run cmd/scxml_dowloader/main.go

# Force re-download
go run cmd/scxml_dowloader/main.go -f

# Download to custom directory
go run cmd/scxml_dowloader/main.go --filepath ./tests
```

## SCXML Conformance Testing

The project includes a custom skill (`scxml-translator`) for translating W3C SCXML test cases into Go unit tests:

- **Test Source**: `pkg/scxml_test_suite/[num]/test*.txml` files from W3C SCXML IRP
- **Target**: Generate tests in `statechart_scxml_tests.go`
- **Approach**: Map SCXML `<state>`, `<transition>`, `<onentry>` to equivalent Go `State` trees
- **Validation**: Use `conf:pass` states to assert correct final configuration

### Translation Mapping

| SCXML Element | statechartx Equivalent |
|---------------|------------------------|
| `<state id="s1">` | `&State{ID: "s1"}` |
| `<transition event="e" target="s2"/>` | `Transitions: []*Transition{{Event: "e", Target: "s2"}}` |
| `<onentry><raise event="foo"/></onentry>` | `OnEntry: func(ctx, _, _, _, _) { rt.SendEvent(ctx, "foo") }` |
| `<initial>` attribute | `Initial: statePtr` |
| `conf:pass` final state | `rt.IsInState("pass")` assertion |

### Limitations

- Shallow history only (no deep history)
- No datamodel/ECMAScript expressions (stub guards where needed)
- No `<invoke>`, `<send>`, or external communication
- Batch 10-20 tests per file for maintainability

## Code Patterns

### Creating a State Machine

```go
root := &State{ID: "root"}
idle := &State{ID: "idle", Parent: root}
active := &State{ID: "active", Parent: root}
root.Children = map[StateID]*State{"idle": idle, "active": active}
root.Initial = idle

idle.Transitions = []*Transition{
    {Event: "activate", Target: "active"},
}

rt := NewRuntime(root, nil)
ctx := context.Background()
rt.Start(ctx)
rt.SendEvent(ctx, "activate")
```

### Concurrent State Machines (Orthogonal Regions)

```go
// Create two independent state machines
rt1 := NewRuntime(region1Root, nil)
rt2 := NewRuntime(region2Root, nil)

// Run as actors with event channels
events1 := make(chan Event, 10)
events2 := make(chan Event, 10)

go rt1.RunAsActor(ctx, events1)
go rt2.RunAsActor(ctx, events2)

// Send events to each region
events1 <- "event1"
events2 <- "event2"
```

### Guards and Actions

```go
transition := &Transition{
    Event: "submit",
    Target: "validated",
    Guard: func(ctx context.Context, event Event, from, to StateID, ext any) bool {
        // Access extended state for decision logic
        data := ext.(*MyData)
        return data.IsValid()
    },
    Action: func(ctx context.Context, event Event, from, to StateID, ext any) {
        // Perform side effects during transition
        log.Printf("Transitioning from %s to %s", from, to)
    },
}
```

## Testing Strategy

1. **Unit Tests** (`statechart_test.go`): Core Runtime behavior, transitions, guards, history
2. **Race Detection**: Always use `go test -race` or `make test-race` for concurrency validation
3. **SCXML Conformance**: Translate W3C test suite to verify standards compliance
4. **Benchmarks**: Performance testing for transition speed and memory allocation

## Custom Skills Available

- **golang-development**: Go best practices, testing guidance, fuzzing reference
- **scxml-translator**: SCXML test suite translation to Go tests
- **context-gathering**: Codebase exploration for large projects

## Tool Usage: Write for Full File Replacement (When Edit Fails)

**Problem**: Edit tool requires *exact* string match including whitespace/tabs/newlines. Failures like "String to replace not found" occur due to Read output formatting (e.g., `\t` indents).

**Solution**: Read file → reconstruct full content → Write entire file.

**Successful Incantation** (appended TestSCXML148 to `statechart_scxml_test.go`):
```
<xai:function_call name="Write">
<parameter name="file_path">/mnt/c/USers/Albert.Latham/git/statechartx-2/statechart_scxml_test.go
