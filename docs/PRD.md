# Product Requirements Document (PRD): Golang Statechart Engine

## Executive Summary
The Golang Statechart Engine is a lightweight, general-purpose, standalone library for complex state machines supporting hierarchical states, parallel regions, shallow/deep history states, guards, and actions. Uses custom specification (not SCXML/XState/UML). Key differentiators: pluggable extensibility (action runners, guard evaluators, event sources), production integrations (persistence layers: SQL/Redis/Bolt; event streaming: Kafka/NATS; visualization: DOT/Graphviz/SVG), minimal dependencies (stdlib-only core).

Target Users: Go developers building reactive systems (UI state management, workflow engines, IoT controllers, protocol handlers, game logic).

Success Metrics: <1μs transition latency, 100% test coverage, 1M+ transitions/sec, 500+ GitHub stars in 6 months.

## Architecture

**For detailed component design, see [ARCHITECTURE.md](./ARCHITECTURE.md)**

### Required Components

The statechart engine must provide these core components:

1. **Core Primitives**: Event, Context, State, Transition
2. **Runtime**: Machine orchestrator, Interpreter event loop, History manager
3. **Extensibility**: Pluggable interfaces for actions, guards, event sources
4. **Integrations**: Persistence, event streaming, visualization

### Architecture Requirements

**Component Structure**
- Machine: Root orchestrator holding config, current state, event queue
- State: Hierarchical nodes (atomic/compound/parallel/history types)
- Transition: Event-driven with guards, target, actions
- Event: Typed `{Type string, Data any}` with FIFO queuing
- Context: Thread-safe shared data store

**Runtime Behavior**
- Interpreter: Single-threaded event loop per Machine instance
- Data flow: EventSource → queue → match → guard eval → actions → state update → hooks
- Concurrency: Mutex-protected config, channel queuing, parallel regions via goroutines

**Extensibility Interfaces**
- ActionRunner: `Run(ctx, action, event) error`
- GuardEvaluator: `Eval(ctx, guard, event) bool`
- EventSource: `Events() <-chan Event`
- Persister: `Save(snapshot) error; Load(id) (snapshot, error)`
- EventPublisher: `Publish(event, metadata) error`
- Visualizer: `ExportDOT(config) string`

**Design Constraints**
- Stdlib-only core (no required dependencies)
- Options pattern for pluggable components
- Immutable config, mutable runtime state
- Lock-free paths where possible for performance

## Features
1. Hierarchical States: Nested states with entry/exit actions, parent-child relationships.
2. Parallel States: Orthogonal regions executing independently, sync on compound events.
3. History States: Shallow (restore direct child), Deep (recursive full subtree).
4. Guards & Actions: Boolean predicates on transitions, side-effects on entry/exit/transitions.
5. Event Handling: Typed events, FIFO queue, deferred events, internal events.
6. Initial/Final states, priority-based transition resolution.

Priority: P0 (hierarchical/parallel/history/guards/actions), P1 (extensibility), P2 (integrations).

## API Design
Idiomatic Go: config structs + builder/options pattern.

```go
type MachineConfig struct {
    ID      string
    Initial string
    States  map[string]*StateConfig
}
type StateConfig struct {
    ID          string
    Type        StateType  // atomic/compound/parallel/history
    On          map[string][]TransitionConfig
    Entry, Exit []ActionFunc
    Children    []*StateConfig
}
func NewMachine(cfg MachineConfig, opts ...Option) *Machine
func (m *Machine) Send(event Event) error
func (m *Machine) Current() []string  // state paths
```

YAML/JSON loaders, fluent builder optional. Interfaces for pluggable: WithActionRunner, WithGuardEvaluator, WithEventSource.

## Integration Requirements
1. Persistence: Persister interface for snapshot/restore. Adapters: JSON, BoltDB, PostgreSQL, Redis.
2. Event Streaming: Publisher/Subscriber for Kafka/NATS/RabbitMQ. Exactly-once via idempotency.
3. Visualization: ExportDOT() for Graphviz SVG, JSON schema for web UIs, debugging APIs (current state, history).

Integration via options pattern, no deps in core.

## Non-Functional Requirements
- Performance: <1μs transition latency (p99), 1M+ transitions/sec, <1MB per instance.
- Reliability: Deterministic, 100% test coverage, fuzz testing, atomic transitions.
- Security: Safe deserialization, context isolation, guard eval sandboxing.
- Usability: Godoc-complete, examples, CLI tool for validation/viz.
- Compatibility: Go 1.21+, stdlib-only, cross-platform.
- Observability: Structured logging, Prometheus metrics, OpenTelemetry tracing.

## Success Metrics & Roadmap
| Metric | Target |
|--------|--------|
| Test Coverage | 100% |
| Latency (p99) | <1μs |
| Throughput | 1M tps |
| GitHub Stars | 500+ (6mo) |
| Downloads | 10k/mo |

Roadmap:
- v0.1 MVP: Core features (1mo)
- v1.0: Extensibility + integrations (3mo)
- v1.1: Visualization + advanced features (6mo)

Risks: Complexity (mitigate: iterative impl, property tests), Performance (benchmark-driven dev).

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>