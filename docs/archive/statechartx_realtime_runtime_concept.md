# Real-Time Runtime Concept (Future Consideration)

**Status**: Earmarked for later exploration  
**Created**: 2026-01-02

---

## 1. Core Concept

An alternate runtime model designed for deterministic, tick-based execution:

- **Tick-based execution**: Game loop style, fixed time steps
- **Event ordering guarantees**: Events processed in deterministic order each tick
- **Macro state resolution per tick**: Complete state transitions resolved within single tick
- **Temporal separation**: All states read from tick N-1 context, write to tick N
- **No collision/read-write issues**: Temporal buffering eliminates race conditions
- **Goroutine usage**: Uncertain if needed (TBD)

---

## 2. Comparison with Current Event-Driven Model

| Aspect | Event-Driven (Current) | Real-Time (Future) |
|--------|------------------------|-------------------|
| Execution Model | Asynchronous, event-driven | Synchronous per tick, deterministic |
| Concurrency | Goroutines for parallel states | TBD (may not need goroutines) |
| State Updates | Immediate, as events arrive | Batched per tick |
| Determinism | Best-effort | Guaranteed |
| Use Cases | General async workflows | Games, simulations, real-time systems |

---

## 3. Open Questions to Explore Later

- **Goroutines**: Do we need them? Could tick processing be entirely sequential?
- **API differences**: How would the API differ from event-driven runtime?
- **Performance**: Latency vs throughput tradeoffs? Tick rate considerations?
- **Use cases**: What specific domains benefit most? (games, robotics, simulations, real-time control systems)
- **Hybrid approach**: Could both runtimes share the same statechart definition?
- **Time handling**: How to handle time-based transitions and delays in tick model?

---

## 4. Recommendation on Timing

**Proceed with event-driven implementation first.**

### Why Wait?

1. **Learning from implementation**: Building the event-driven runtime will surface design patterns, edge cases, and API decisions that will inform the real-time design
2. **Core abstractions**: The statechart model, transition logic, and action handling will be validated and stabilized
3. **Avoid premature optimization**: Real-time requirements may become clearer after seeing event-driven limitations in practice
4. **Incremental complexity**: Adding a second runtime model is easier once the first is proven

### What We'll Learn

- How to structure runtime interfaces for pluggability
- Common patterns in state transition handling
- Action execution and side-effect management
- Testing strategies for statechart behavior
- Performance bottlenecks and optimization opportunities

### When to Revisit

**Suggested timing**: After Phase 3 (Parallel States) or Phase 4 (History States)

At that point:
- Core runtime will be stable and tested
- Parallel state semantics will be well-understood
- We'll have real-world feedback on event-driven model limitations
- Runtime abstraction boundaries will be clear

---

## Notes

This document is intentionally brief—just enough to capture the concept without derailing current work. The real-time runtime remains a valid and interesting direction for future exploration, particularly for domains requiring deterministic execution and temporal consistency.
