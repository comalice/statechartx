# StatechartX Documentation

This directory contains comprehensive documentation for the StatechartX state machine library.

## Getting Started

**New to StatechartX?** Start here:

1. [Core Package Guide](../README_CORE.md) - API patterns, code examples, common use cases
2. [Decision Guide](DECISION-GUIDE.md) - Choose runtime, state patterns, and implementation strategies
3. [Examples](../examples/README.md) - Runnable code samples

## Core Documentation

### [Architecture Overview](architecture.md)
System design, key concepts, and architectural decisions. Covers:
- Hierarchical state management
- Entry/exit ordering semantics
- Parallel state execution model
- Concurrency and thread safety
- Core transition engine analysis

### [Real-Time Runtime](../realtime/README.md)
Tick-based deterministic runtime for games, simulations, and robotics. See the [realtime package README](../realtime/README.md) which covers:
- Fixed time-step execution model
- Event batching and ordering guarantees
- Deterministic parallel region processing
- Performance characteristics vs event-driven runtime
- Use cases and examples

### [Performance Testing](performance.md)
Comprehensive performance benchmarks and optimization insights. Covers:
- Stress tests (million states, massive parallel regions, deep hierarchies)
- Benchmark results (transitions, LCA, event routing, history)
- Breaking point analysis
- Profiling results and CPU/memory hotspots
- Production performance limits

### [Decision Guide](DECISION-GUIDE.md)
Runtime and pattern selection guide with decision tables. Covers:
- Event-driven vs real-time runtime selection
- State patterns (parallel, history, sequential)
- Transition patterns (guarded, eventless, internal)
- Event and action design patterns
- Migration strategies and common pitfalls

### [SCXML Conformance](scxml-conformance.md)
W3C SCXML test suite integration and conformance testing. Covers:
- Test suite organization and structure
- SCXML to Go translation mapping
- Test downloading and execution
- Custom scxml-translator skill documentation

## Subpackage Documentation

- [Real-Time Package](../realtime/README.md) - Detailed API for tick-based runtime
- [Test Utilities](../testutil/) - Test adapter utilities

## Historical Documentation

The [archive/](archive/) directory contains historical implementation notes, phase summaries, and development plans from the project's evolution. These documents provide context on design decisions and implementation history but are not required for using the library.

### Archive Contents

- Implementation phase summaries (PHASE1-5)
- Historical design documents
- Development planning artifacts
- Performance testing evolution
- Incremental analysis reports

## Examples

See [../examples/README.md](../examples/README.md) for runnable code examples demonstrating:
- Basic state machine usage
- Real-time game loops
- Physics simulations
- Replay systems

## Contributing

See [../CONTRIBUTING.md](../CONTRIBUTING.md) for documentation contribution guidelines.

## Navigation

- [Back to Project Root](../)
- [View Examples](../examples/)
- [View Source Code](../)
