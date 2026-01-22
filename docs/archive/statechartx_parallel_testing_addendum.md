# StatechartX Parallel State Testing Addendum

## Executive Summary

This document provides a comprehensive testing strategy for parallel states in statechartx to **guarantee correct operation with absolute certainty** and prevent indefinite hangs or deadlocks. The parallel state implementation uses goroutines (one per region) with channel-based event routing, making it critical to test all concurrency scenarios exhaustively.

**Architecture Context**: 
- Synchronous FSM for compound states (no locks needed)
- Goroutines for parallel states (one goroutine per parallel region)
- Event addressing via `Event.address` field (0 = broadcast, non-zero = targeted)
- Channel-based event queues for inter-region communication

**Testing Philosophy**: Every operation that could potentially block must have a timeout. Every goroutine spawned must have a verified cleanup path. Every channel operation must be proven non-blocking or have timeout protection.

---

## 1. Current Implementation Analysis

### 1.1 Existing Code (statechart.go)

The current implementation (254 lines) provides:
- Basic `State` and `Machine` types with int-based `StateID` and `EventID`
- Synchronous transition execution via `doTransition()`
- Entry/exit actions with `enterState()` and `exitState()`
- Simple transition selection via `pickTransition()`
- No parallel state support yet (all tests requiring parallel states are skipped)

**Key Observation**: The synchronous FSM has no concurrency issues because it's single-threaded. All complexity and risk comes from adding parallel states.

### 1.2 Planned Parallel State Architecture (Phase 5)

From the incremental plan:

```go
type StateType int
const (
    ATOMIC StateType = iota
    COMPOUND
    PARALLEL
)

type State struct {
    ID          StateID
    Type        StateType  // NEW: Distinguishes parallel from compound
    Parent      *State
    Children    map[StateID]*State
    // ... existing fields
}

type Event struct {
    ID      EventID
    Payload any
    address StateID  // NEW: 0 = broadcast, non-zero = targeted
}
```

**Parallel State Behavior**:
1. When entering a parallel state, spawn one goroutine per child region
2. Each region has its own event queue (channel)
3. Events with `address == 0` are broadcast to all regions
4. Events with `address != 0` are routed to specific region
5. Exiting parallel state waits for all regions to complete
6. Context cancellation propagates to all regions

---

## 2. Hang/Deadlock Scenarios Identified

### 2.1 Goroutine Lifecycle Issues

| Scenario | Risk | Impact |
|----------|------|--------|
| **Goroutine leak on error** | Region goroutine spawned but never cleaned up after error | Memory leak, resource exhaustion |
| **Orphaned goroutines on context cancel** | Context cancelled but goroutines don't exit | Indefinite hang, resource leak |
| **Double-spawn on re-entry** | Parallel state re-entered without cleaning up previous goroutines | Duplicate event processing, race conditions |
| **Panic in region goroutine** | Unrecovered panic crashes goroutine, parent waits forever | Indefinite hang |

### 2.2 Channel Blocking Issues

| Scenario | Risk | Impact |
|----------|------|--------|
| **Unbuffered channel send blocks** | Sending event to region with no receiver | Indefinite hang |
| **Channel full (buffered)** | Buffered channel fills up, sender blocks | Indefinite hang or event loss |
| **Receive on closed channel** | Region tries to receive after channel closed | Panic or incorrect behavior |
| **Send on closed channel** | Parent tries to send to closed region channel | Panic |
| **No receiver on broadcast** | Broadcasting to regions that haven't started yet | Lost events or hang |

### 2.3 Event Routing Deadlocks

| Scenario | Risk | Impact |
|----------|------|--------|
| **Circular event routing** | Region A sends to B, B sends to A, both block | Deadlock |
| **Broadcast during exit** | Broadcasting while regions are shutting down | Partial delivery, inconsistent state |
| **Targeted event to wrong region** | Event addressed to non-existent or inactive region | Lost event, hang waiting for response |
| **Event queue ordering violation** | Concurrent sends violate FIFO ordering | Non-deterministic behavior |

### 2.4 Shutdown/Cleanup Problems

| Scenario | Risk | Impact |
|----------|------|--------|
| **Exit waits forever** | WaitGroup never completes because region hung | Indefinite hang |
| **Partial cleanup on error** | Some regions cleaned up, others left running | Resource leak, inconsistent state |
| **Cleanup order violation** | Children cleaned up before parent, or vice versa | Panic, resource leak |
| **Context cancel doesn't propagate** | Child contexts not derived from parent | Regions don't stop on cancel |

### 2.5 Race Conditions

| Scenario | Risk | Impact |
|----------|------|--------|
| **Concurrent state access** | Multiple regions read/write shared state | Data corruption |
| **Event ordering across regions** | Events processed in different order than sent | Non-deterministic behavior |
| **Configuration update race** | Parent updates configuration while regions running | Inconsistent state view |
| **Transition during shutdown** | Transition fires while parallel state exiting | Panic, inconsistent state |

### 2.6 Resource Leaks

| Scenario | Risk | Impact |
|----------|------|--------|
| **Unclosed channels** | Channels not closed on exit | Memory leak, goroutine leak |
| **Unreleased WaitGroups** | WaitGroup.Done() not called | Indefinite hang |
| **Context leak** | Child contexts not cancelled | Resource leak |
| **Timer/ticker leak** | Timers not stopped on exit | Memory leak, CPU waste |

---

## 3. Testing Categories and Guarantees

### 3.1 Goroutine Lifecycle Tests

**Goal**: Guarantee every spawned goroutine has a verified cleanup path.

#### Test 3.1.1: Basic Parallel Region Spawn
```go
func TestParallelRegionSpawn(t *testing.T)
```
**Scenario**: Enter parallel state with 2 regions, verify both goroutines start.

**What could go wrong**: 
- Goroutines never start
- Only one goroutine starts
- Goroutines start but don't enter event loop

**How to test**:
- Use sync.WaitGroup or channel to signal goroutine start
- Verify both regions process a test event within timeout
- Check goroutine count before/after (runtime.NumGoroutine())

**Guarantees needed**:
- All N regions spawn exactly N goroutines
- All goroutines enter event loop within 100ms
- Goroutine count increases by exactly N

**Specific test cases**:
- 2 regions (basic case)
- 5 regions (multiple regions)
- 1 region (edge case - parallel with single child)
- 0 regions (error case - should fail gracefully)

---

#### Test 3.1.2: Parallel Region Cleanup on Exit
```go
func TestParallelRegionCleanupOnExit(t *testing.T)
```
**Scenario**: Enter parallel state, then transition out, verify all goroutines exit.

**What could go wrong**:
- Goroutines never exit
- Some goroutines exit, others hang
- Goroutines exit but channels not closed
- WaitGroup never completes

**How to test**:
- Track goroutine count before/after
- Use done channels to verify each goroutine exits
- Set timeout (1 second) for exit operation
- Verify no goroutine leaks with runtime.NumGoroutine()

**Guarantees needed**:
- All goroutines exit within 500ms of transition
- Goroutine count returns to baseline
- All channels closed
- WaitGroup completes

**Specific test cases**:
- Normal exit (transition to non-parallel state)
- Exit via final state in one region
- Exit via external transition from parallel state
- Exit with pending events in queues

---

#### Test 3.1.3: Parallel Region Cleanup on Context Cancel
```go
func TestParallelRegionCleanupOnContextCancel(t *testing.T)
```
**Scenario**: Enter parallel state, cancel context, verify all goroutines exit.

**What could go wrong**:
- Goroutines ignore context cancellation
- Some goroutines exit, others ignore cancel
- Goroutines exit but leave resources uncleaned
- Cancel takes too long (indefinite hang)

**How to test**:
- Start parallel state with cancellable context
- Cancel context after 100ms
- Verify all goroutines exit within 500ms
- Check for resource leaks (channels, timers)

**Guarantees needed**:
- Context cancellation propagates to all regions within 50ms
- All goroutines exit within 500ms of cancel
- No resource leaks
- Machine enters error/stopped state

**Specific test cases**:
- Cancel during idle (no events)
- Cancel during event processing
- Cancel during transition
- Cancel during nested parallel state

---

#### Test 3.1.4: Parallel Region Panic Recovery
```go
func TestParallelRegionPanicRecovery(t *testing.T)
```
**Scenario**: Region goroutine panics, verify other regions continue and cleanup happens.

**What could go wrong**:
- Panic crashes entire machine
- Other regions hang waiting for panicked region
- WaitGroup never completes (Done() not called)
- Parent never notified of error

**How to test**:
- Inject panic in one region's entry action
- Verify other regions continue processing
- Verify parent receives error notification
- Verify all goroutines eventually exit

**Guarantees needed**:
- Panics are recovered within region goroutine
- Other regions unaffected
- Parent notified via error channel
- All goroutines cleaned up within 1 second

**Specific test cases**:
- Panic in entry action
- Panic in exit action
- Panic in transition action
- Panic in guard evaluation

---

#### Test 3.1.5: Parallel Region Re-entry
```go
func TestParallelRegionReentry(t *testing.T)
```
**Scenario**: Exit parallel state, then re-enter, verify old goroutines cleaned up and new ones spawned.

**What could go wrong**:
- Old goroutines not cleaned up before re-entry
- New goroutines spawned while old ones still running
- Double event processing
- Goroutine count grows unbounded

**How to test**:
- Enter parallel state, exit, re-enter
- Track goroutine count at each step
- Verify old goroutines exit before new ones start
- Send events and verify processed only once

**Guarantees needed**:
- Old goroutines fully cleaned up before re-entry
- Goroutine count stable across re-entries
- No event duplication
- No resource leaks

**Specific test cases**:
- Single re-entry
- Multiple re-entries (10 times)
- Rapid re-entry (exit and re-enter immediately)
- Re-entry with pending events

---

### 3.2 Channel Operation Tests

**Goal**: Guarantee all channel operations are non-blocking or have timeout protection.

#### Test 3.2.1: Non-blocking Event Send
```go
func TestNonBlockingEventSend(t *testing.T)
```
**Scenario**: Send events to parallel regions, verify sends never block indefinitely.

**What could go wrong**:
- Unbuffered channel blocks sender
- Buffered channel fills up, blocks sender
- Receiver not ready, sender hangs

**How to test**:
- Send 100 events rapidly to parallel state
- Use timeout (100ms per send) to detect blocking
- Verify all sends complete within reasonable time
- Check for dropped events

**Guarantees needed**:
- Every send completes within 100ms
- No events dropped (or explicit drop policy)
- Buffered channels sized appropriately
- Backpressure mechanism if needed

**Specific test cases**:
- Broadcast events (to all regions)
- Targeted events (to specific region)
- Mixed broadcast and targeted
- High volume (1000 events/second)

---

#### Test 3.2.2: Channel Close Safety
```go
func TestChannelCloseSafety(t *testing.T)
```
**Scenario**: Close region channels during shutdown, verify no panics or hangs.

**What could go wrong**:
- Send on closed channel (panic)
- Receive on closed channel (unexpected behavior)
- Double close (panic)
- Close while send in progress (race)

**How to test**:
- Close channels in controlled order
- Attempt send after close (should fail gracefully)
- Attempt receive after close (should return immediately)
- Use race detector

**Guarantees needed**:
- Channels closed exactly once
- No send after close
- Receivers handle closed channel correctly
- No panics

**Specific test cases**:
- Close during idle
- Close during event processing
- Close with pending events in queue
- Close during concurrent sends

---

#### Test 3.2.3: Buffered Channel Sizing
```go
func TestBufferedChannelSizing(t *testing.T)
```
**Scenario**: Verify channel buffer sizes prevent blocking under normal load.

**What could go wrong**:
- Buffer too small, frequent blocking
- Buffer too large, memory waste
- Buffer size not configurable
- No backpressure mechanism

**How to test**:
- Send events at various rates
- Measure blocking frequency
- Test with different buffer sizes
- Verify memory usage reasonable

**Guarantees needed**:
- Buffer size >= 10 (minimum)
- No blocking under normal load (< 100 events/sec)
- Configurable buffer size
- Backpressure or drop policy for overload

**Specific test cases**:
- Buffer size 1 (minimal)
- Buffer size 10 (default)
- Buffer size 100 (high throughput)
- Buffer size 0 (unbuffered - should fail or warn)

---

#### Test 3.2.4: Channel Receive Timeout
```go
func TestChannelReceiveTimeout(t *testing.T)
```
**Scenario**: Region goroutine receives from channel with timeout, never hangs.

**What could go wrong**:
- Receive blocks forever if no events
- Timeout not implemented
- Timeout too long (appears hung)
- Timeout too short (misses events)

**How to test**:
- Start region with no events
- Verify goroutine doesn't hang (use context cancel)
- Send event after delay, verify received
- Measure receive latency

**Guarantees needed**:
- Receive uses select with context.Done()
- No indefinite blocking
- Events received within 10ms of send
- Graceful shutdown on context cancel

**Specific test cases**:
- No events (idle timeout)
- Events arrive after delay
- Context cancelled during receive
- High-frequency events (no timeout needed)

---

### 3.3 Deadlock Prevention Tests

**Goal**: Guarantee no circular dependencies or indefinite waits.

#### Test 3.3.1: No Circular Event Routing
```go
func TestNoCircularEventRouting(t *testing.T)
```
**Scenario**: Region A sends event to region B, B sends to A, verify no deadlock.

**What could go wrong**:
- Both regions block waiting for each other
- Event queues fill up, both block on send
- Infinite event loop

**How to test**:
- Create parallel state with 2 regions
- Region A sends event to B on entry
- Region B sends event to A on receiving event
- Verify both regions process events without hanging
- Use timeout (1 second) to detect deadlock

**Guarantees needed**:
- No deadlock (test completes within 1 second)
- Event loop terminates (use counter to limit)
- Both regions remain responsive
- Clear error if circular dependency detected

**Specific test cases**:
- A → B → A (simple cycle)
- A → B → C → A (3-way cycle)
- A → B, B → A (simultaneous)
- Cycle with broadcast events

---

#### Test 3.3.2: Broadcast During Transition
```go
func TestBroadcastDuringTransition(t *testing.T)
```
**Scenario**: Broadcast event while one region is transitioning, verify no deadlock.

**What could go wrong**:
- Transitioning region blocks broadcast
- Broadcast waits for transition to complete
- Other regions blocked waiting for transitioning region
- Inconsistent state across regions

**How to test**:
- Start parallel state with 2 regions
- Region A starts slow transition (100ms)
- Broadcast event during transition
- Verify region B receives event immediately
- Verify region A receives event after transition

**Guarantees needed**:
- Broadcast doesn't wait for all regions
- Each region processes broadcast independently
- No region blocks others
- Events delivered in order per region

**Specific test cases**:
- Broadcast during entry action
- Broadcast during exit action
- Broadcast during transition action
- Broadcast during guard evaluation

---

#### Test 3.3.3: Targeted Event to Inactive Region
```go
func TestTargetedEventToInactiveRegion(t *testing.T)
```
**Scenario**: Send event with address to region that doesn't exist or is inactive.

**What could go wrong**:
- Sender blocks waiting for non-existent receiver
- Event lost silently
- Panic on invalid address
- Indefinite hang

**How to test**:
- Send event with address to non-existent StateID
- Send event to region that has exited
- Verify sender doesn't block (timeout 100ms)
- Verify error returned or event dropped

**Guarantees needed**:
- Invalid address returns error immediately
- No blocking on invalid address
- Clear error message
- Event not lost silently (logged or error)

**Specific test cases**:
- Address to non-existent state
- Address to inactive region
- Address to parent state (not region)
- Address to sibling parallel state

---

#### Test 3.3.4: Exit Waits for All Regions
```go
func TestExitWaitsForAllRegions(t *testing.T)
```
**Scenario**: Transition out of parallel state, verify exit waits for all regions to complete.

**What could go wrong**:
- Exit returns before all regions stopped
- One slow region blocks exit indefinitely
- WaitGroup never completes
- Timeout not implemented

**How to test**:
- Create parallel state with 2 regions
- Make one region slow to exit (100ms)
- Transition out of parallel state
- Verify exit waits for slow region
- Verify exit completes within timeout (500ms)

**Guarantees needed**:
- Exit waits for all regions (WaitGroup)
- Exit has timeout (500ms default)
- Timeout returns error, doesn't hang
- Partial cleanup on timeout (best effort)

**Specific test cases**:
- All regions exit quickly
- One region slow (within timeout)
- One region hung (exceeds timeout)
- Multiple regions with varying exit times

---

### 3.4 Event Routing Tests

**Goal**: Guarantee events are delivered correctly and in order.

#### Test 3.4.1: Broadcast Event Delivery
```go
func TestBroadcastEventDelivery(t *testing.T)
```
**Scenario**: Send broadcast event (address == 0), verify all regions receive it.

**What could go wrong**:
- Only some regions receive event
- Event received multiple times by same region
- Event received in wrong order
- Event lost during broadcast

**How to test**:
- Create parallel state with 3 regions
- Each region increments counter on event
- Send broadcast event
- Verify all 3 counters incremented exactly once
- Verify delivery within 100ms

**Guarantees needed**:
- All active regions receive broadcast
- Each region receives exactly once
- Delivery within 100ms
- Order preserved per region

**Specific test cases**:
- 2 regions (basic)
- 5 regions (multiple)
- 1 region (edge case)
- Nested parallel states (broadcast to all levels)

---

#### Test 3.4.2: Targeted Event Delivery
```go
func TestTargetedEventDelivery(t *testing.T)
```
**Scenario**: Send event with specific address, verify only target region receives it.

**What could go wrong**:
- Event delivered to all regions (broadcast)
- Event delivered to wrong region
- Event not delivered at all
- Event delivered multiple times

**How to test**:
- Create parallel state with 3 regions (IDs: 10, 20, 30)
- Send event with address = 20
- Verify only region 20 receives event
- Verify regions 10 and 30 don't receive it

**Guarantees needed**:
- Only target region receives event
- Other regions unaffected
- Delivery within 100ms
- Error if target doesn't exist

**Specific test cases**:
- Target each region individually
- Target non-existent region (error)
- Target parent state (should broadcast to children?)
- Target nested state within region

---

#### Test 3.4.3: Event Ordering Within Region
```go
func TestEventOrderingWithinRegion(t *testing.T)
```
**Scenario**: Send multiple events to same region, verify FIFO order.

**What could go wrong**:
- Events processed out of order
- Events lost
- Events duplicated
- Race condition in queue

**How to test**:
- Send events 1, 2, 3 to region
- Region appends event ID to slice
- Verify slice is [1, 2, 3]
- Use race detector

**Guarantees needed**:
- FIFO order within region
- No events lost
- No events duplicated
- Thread-safe queue

**Specific test cases**:
- Sequential sends (slow)
- Rapid sends (fast)
- Mixed broadcast and targeted
- Sends from multiple goroutines

---

#### Test 3.4.4: Cross-Region Event Ordering
```go
func TestCrossRegionEventOrdering(t *testing.T)
```
**Scenario**: Send events to different regions, verify independent processing.

**What could go wrong**:
- Regions process events in lockstep (serialized)
- One region blocks others
- Global ordering enforced (incorrect)
- Race conditions

**How to test**:
- Send event to region A (slow processing)
- Send event to region B (fast processing)
- Verify B completes before A
- Verify A not blocked by B

**Guarantees needed**:
- Regions process independently
- No global ordering (unless required by spec)
- No region blocks others
- Concurrent processing

**Specific test cases**:
- A slow, B fast
- Both slow
- Both fast
- 3+ regions with varying speeds

---

### 3.5 Shutdown Tests

**Goal**: Guarantee graceful termination in all scenarios.

#### Test 3.5.1: Graceful Shutdown (No Pending Events)
```go
func TestGracefulShutdownNoPendingEvents(t *testing.T)
```
**Scenario**: Stop machine with parallel state, no pending events.

**What could go wrong**:
- Goroutines don't exit
- Channels not closed
- Resources leaked
- Shutdown hangs

**How to test**:
- Start parallel state
- Call Stop() or cancel context
- Verify all goroutines exit within 500ms
- Verify no resource leaks

**Guarantees needed**:
- All goroutines exit within 500ms
- All channels closed
- No resource leaks
- Machine enters stopped state

**Specific test cases**:
- Stop() method
- Context cancellation
- Transition to final state
- External shutdown signal

---

#### Test 3.5.2: Shutdown With Pending Events
```go
func TestShutdownWithPendingEvents(t *testing.T)
```
**Scenario**: Stop machine with events still in queues.

**What could go wrong**:
- Pending events processed after shutdown
- Goroutines hang waiting for events to process
- Events lost without notification
- Inconsistent state

**How to test**:
- Send 100 events to parallel state
- Immediately call Stop()
- Verify shutdown completes within 500ms
- Verify pending events either processed or dropped (documented behavior)

**Guarantees needed**:
- Shutdown completes within 500ms regardless of pending events
- Clear policy: process pending or drop (documented)
- No goroutines left running
- No panics

**Specific test cases**:
- Few pending events (< 10)
- Many pending events (> 100)
- Events in multiple region queues
- Events being processed during shutdown

---

#### Test 3.5.3: Partial Shutdown on Error
```go
func TestPartialShutdownOnError(t *testing.T)
```
**Scenario**: One region errors during shutdown, verify others still cleaned up.

**What could go wrong**:
- Error in one region blocks shutdown of others
- Some regions left running
- WaitGroup never completes
- Resource leaks

**How to test**:
- Inject error in one region's exit action
- Trigger shutdown
- Verify other regions still exit
- Verify error propagated to caller
- Verify all goroutines eventually exit

**Guarantees needed**:
- Best-effort cleanup of all regions
- Errors collected and returned
- All goroutines exit within 1 second
- No resource leaks

**Specific test cases**:
- Error in one region
- Errors in multiple regions
- Panic in one region
- Timeout in one region

---

#### Test 3.5.4: Nested Parallel State Shutdown
```go
func TestNestedParallelStateShutdown(t *testing.T)
```
**Scenario**: Parallel state contains child parallel states, verify all levels cleaned up.

**What could go wrong**:
- Only top-level regions cleaned up
- Child parallel states left running
- Cleanup order violation (children before parents)
- Exponential goroutine leak

**How to test**:
- Create nested parallel states (2 levels)
- Count total goroutines (should be 4+)
- Trigger shutdown
- Verify all goroutines exit
- Verify cleanup order (children first)

**Guarantees needed**:
- All levels cleaned up
- Cleanup order: leaf → root
- All goroutines exit within 1 second
- No resource leaks

**Specific test cases**:
- 2-level nesting
- 3-level nesting
- Mixed parallel and compound states
- Asymmetric nesting (some branches deeper)

---

### 3.6 Race Condition Tests

**Goal**: Guarantee thread-safe operation under concurrent access.

#### Test 3.6.1: Concurrent State Access
```go
func TestConcurrentStateAccess(t *testing.T)
```
**Scenario**: Multiple regions read/write shared extended state concurrently.

**What could go wrong**:
- Data races
- Inconsistent reads
- Lost updates
- Panics

**How to test**:
- Create parallel state with 3 regions
- Each region increments shared counter 100 times
- Run with -race flag
- Verify final count is 300
- Verify no race conditions detected

**Guarantees needed**:
- Thread-safe access to extended state (mutex or channels)
- No data races
- Consistent state updates
- Clear documentation of thread-safety guarantees

**Specific test cases**:
- Shared counter (read-modify-write)
- Shared map (concurrent writes)
- Shared slice (concurrent appends)
- Complex shared state (struct with multiple fields)

---

#### Test 3.6.2: Configuration Update Race
```go
func TestConfigurationUpdateRace(t *testing.T)
```
**Scenario**: Parent updates configuration while regions query it.

**What could go wrong**:
- Regions see inconsistent configuration
- Data race on configuration map
- Panic on concurrent map access
- Stale configuration reads

**How to test**:
- Parent updates configuration every 10ms
- Regions query configuration every 5ms
- Run with -race flag
- Verify no races detected
- Verify regions see consistent snapshots

**Guarantees needed**:
- Thread-safe configuration access
- Consistent snapshots (no partial updates visible)
- No data races
- Clear documentation of configuration access patterns

**Specific test cases**:
- Frequent updates (high contention)
- Infrequent updates (low contention)
- Large configuration (many states)
- Small configuration (few states)

---

#### Test 3.6.3: Transition During Event Processing
```go
func TestTransitionDuringEventProcessing(t *testing.T)
```
**Scenario**: External transition fires while region processing event.

**What could go wrong**:
- Region continues processing after transition
- Event processing interrupted mid-action
- Inconsistent state
- Panic

**How to test**:
- Region starts slow event processing (100ms)
- External transition fires after 50ms
- Verify event processing cancelled or completed
- Verify transition completes correctly
- Verify no inconsistent state

**Guarantees needed**:
- Event processing cancelled on transition (context cancel)
- Or event processing completes before transition
- Clear policy documented
- No inconsistent state

**Specific test cases**:
- Transition during entry action
- Transition during transition action
- Transition during exit action
- Transition during guard evaluation

---

#### Test 3.6.4: Concurrent Event Sends
```go
func TestConcurrentEventSends(t *testing.T)
```
**Scenario**: Multiple goroutines send events to parallel state concurrently.

**What could go wrong**:
- Data races on event queue
- Events lost
- Events duplicated
- Panic on concurrent channel send

**How to test**:
- Spawn 10 goroutines
- Each sends 100 events
- Verify all 1000 events received
- Run with -race flag
- Verify no races detected

**Guarantees needed**:
- Thread-safe event sending
- No events lost
- No events duplicated
- No data races

**Specific test cases**:
- Many senders, one region
- Many senders, many regions
- Broadcast events from multiple senders
- Targeted events from multiple senders

---

### 3.7 Stress Tests

**Goal**: Guarantee correct operation under high load.

#### Test 3.7.1: Many Parallel Regions
```go
func TestManyParallelRegions(t *testing.T)
```
**Scenario**: Parallel state with 100 regions.

**What could go wrong**:
- Goroutine explosion
- Memory exhaustion
- Slow startup/shutdown
- Scheduler overhead

**How to test**:
- Create parallel state with 100 regions
- Verify all regions start within 1 second
- Send broadcast event, verify all receive within 1 second
- Shutdown, verify all exit within 2 seconds
- Monitor memory usage

**Guarantees needed**:
- Scales to at least 100 regions
- Startup time O(N) or better
- Shutdown time O(N) or better
- Memory usage reasonable (< 10MB for 100 regions)

**Specific test cases**:
- 10 regions
- 50 regions
- 100 regions
- 1000 regions (if feasible)

---

#### Test 3.7.2: High Event Volume
```go
func TestHighEventVolume(t *testing.T)
```
**Scenario**: Send 10,000 events to parallel state rapidly.

**What could go wrong**:
- Event queue overflow
- Memory exhaustion
- Slow processing
- Events lost

**How to test**:
- Send 10,000 events as fast as possible
- Verify all events processed
- Measure throughput (events/second)
- Monitor memory usage
- Verify no events lost

**Guarantees needed**:
- Handles at least 1,000 events/second
- No events lost (or explicit drop policy)
- Memory usage bounded
- Backpressure mechanism if needed

**Specific test cases**:
- 1,000 events
- 10,000 events
- 100,000 events
- Sustained load (1,000 events/sec for 60 seconds)

---

#### Test 3.7.3: Deep Nesting
```go
func TestDeepNesting(t *testing.T)
```
**Scenario**: Parallel states nested 10 levels deep.

**What could go wrong**:
- Stack overflow
- Exponential goroutine growth
- Slow entry/exit
- Memory exhaustion

**How to test**:
- Create 10-level nested parallel states
- Count total goroutines (should be 2^10 = 1024)
- Verify entry completes within 5 seconds
- Verify exit completes within 5 seconds
- Monitor memory usage

**Guarantees needed**:
- Supports at least 5 levels of nesting
- Goroutine count = 2^N (expected)
- Entry/exit time O(N) or O(N log N)
- Memory usage reasonable

**Specific test cases**:
- 2 levels (4 goroutines)
- 5 levels (32 goroutines)
- 10 levels (1024 goroutines)
- Asymmetric nesting (some branches deeper)

---

#### Test 3.7.4: Rapid State Changes
```go
func TestRapidStateChanges(t *testing.T)
```
**Scenario**: Enter/exit parallel state 1000 times rapidly.

**What could go wrong**:
- Goroutine leak on each cycle
- Memory leak on each cycle
- Slow cleanup
- Resource exhaustion

**How to test**:
- Loop 1000 times: enter parallel state, exit immediately
- Monitor goroutine count (should return to baseline)
- Monitor memory usage (should be stable)
- Verify no resource leaks
- Measure time (should complete within 10 seconds)

**Guarantees needed**:
- No goroutine leak
- No memory leak
- Stable performance across cycles
- Completes within 10 seconds

**Specific test cases**:
- 100 cycles
- 1,000 cycles
- 10,000 cycles
- With events during each cycle

---

### 3.8 Timeout Tests

**Goal**: Guarantee no operation waits indefinitely.

#### Test 3.8.1: Entry Timeout
```go
func TestEntryTimeout(t *testing.T)
```
**Scenario**: Region entry action hangs, verify timeout.

**What could go wrong**:
- Entry hangs indefinitely
- Other regions blocked
- Machine unusable
- No error reported

**How to test**:
- Create region with entry action that sleeps 10 seconds
- Set entry timeout to 1 second
- Verify entry fails with timeout error
- Verify other regions unaffected
- Verify machine still usable

**Guarantees needed**:
- Entry timeout configurable (default 5 seconds)
- Timeout returns error
- Other regions unaffected
- Machine remains usable

**Specific test cases**:
- Entry action hangs
- Entry action slow but completes
- Multiple regions, one hangs
- Nested parallel states, one hangs

---

#### Test 3.8.2: Exit Timeout
```go
func TestExitTimeout(t *testing.T)
```
**Scenario**: Region exit action hangs, verify timeout.

**What could go wrong**:
- Exit hangs indefinitely
- Transition never completes
- Machine stuck in parallel state
- Resource leak

**How to test**:
- Create region with exit action that sleeps 10 seconds
- Set exit timeout to 1 second
- Trigger transition out of parallel state
- Verify exit fails with timeout error
- Verify best-effort cleanup
- Verify machine transitions despite timeout

**Guarantees needed**:
- Exit timeout configurable (default 5 seconds)
- Timeout returns error
- Best-effort cleanup of other regions
- Machine transitions despite timeout

**Specific test cases**:
- Exit action hangs
- Exit action slow but completes
- Multiple regions, one hangs
- Nested parallel states, one hangs

---

#### Test 3.8.3: Event Processing Timeout
```go
func TestEventProcessingTimeout(t *testing.T)
```
**Scenario**: Transition action hangs, verify timeout.

**What could go wrong**:
- Action hangs indefinitely
- Region stuck in transition
- Other events blocked
- Machine unusable

**How to test**:
- Create transition with action that sleeps 10 seconds
- Set action timeout to 1 second
- Send event to trigger transition
- Verify action fails with timeout error
- Verify region returns to source state
- Verify region still processes events

**Guarantees needed**:
- Action timeout configurable (default 5 seconds)
- Timeout returns error
- Region returns to source state (rollback)
- Region remains usable

**Specific test cases**:
- Transition action hangs
- Guard evaluation hangs
- Entry action hangs
- Exit action hangs

---

#### Test 3.8.4: Shutdown Timeout
```go
func TestShutdownTimeout(t *testing.T)
```
**Scenario**: Region doesn't exit, verify shutdown timeout.

**What could go wrong**:
- Shutdown hangs indefinitely
- Machine never stops
- Resource leak
- Goroutines left running

**How to test**:
- Create region that ignores shutdown signal
- Set shutdown timeout to 1 second
- Trigger shutdown
- Verify shutdown completes within 1 second
- Verify error returned
- Verify best-effort cleanup

**Guarantees needed**:
- Shutdown timeout configurable (default 5 seconds)
- Timeout returns error
- Best-effort cleanup (force kill goroutines if needed)
- Machine enters stopped state

**Specific test cases**:
- One region ignores shutdown
- All regions ignore shutdown
- Nested parallel states ignore shutdown
- Shutdown during event processing

---

## 4. Implementation Recommendations

### 4.1 Goroutine Management

```go
type parallelRegion struct {
    state    *State
    events   chan Event
    done     chan struct{}
    ctx      context.Context
    cancel   context.CancelFunc
}

func (ps *ParallelState) enter(ctx context.Context) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(ps.regions))
    
    for _, region := range ps.regions {
        wg.Add(1)
        go func(r *parallelRegion) {
            defer wg.Done()
            defer recover() // Catch panics
            if err := r.run(); err != nil {
                errChan <- err
            }
        }(region)
    }
    
    // Wait with timeout
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        return nil
    case <-time.After(5 * time.Second):
        return errors.New("entry timeout")
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

**Key points**:
- Use WaitGroup for synchronization
- Recover from panics in each goroutine
- Use timeout for entry
- Propagate context cancellation
- Collect errors from all regions

---

### 4.2 Channel Design

```go
type parallelRegion struct {
    events chan Event  // Buffered, size 10+
    done   chan struct{}
}

func (r *parallelRegion) run() error {
    for {
        select {
        case event := <-r.events:
            r.processEvent(event)
        case <-r.done:
            return nil
        case <-r.ctx.Done():
            return r.ctx.Err()
        }
    }
}

func (ps *ParallelState) sendEvent(event Event) error {
    if event.address == 0 {
        // Broadcast
        for _, region := range ps.regions {
            select {
            case region.events <- event:
            case <-time.After(100 * time.Millisecond):
                return errors.New("send timeout")
            }
        }
    } else {
        // Targeted
        region := ps.findRegion(event.address)
        if region == nil {
            return errors.New("region not found")
        }
        select {
        case region.events <- event:
        case <-time.After(100 * time.Millisecond):
            return errors.New("send timeout")
        }
    }
    return nil
}
```

**Key points**:
- Buffered channels (size 10+)
- Timeout on send (100ms)
- Select with context.Done() for cancellation
- Separate done channel for clean shutdown
- Error on send timeout (don't block)

---

### 4.3 Cleanup Pattern

```go
func (ps *ParallelState) exit(ctx context.Context) error {
    // Signal all regions to stop
    for _, region := range ps.regions {
        close(region.done)
    }
    
    // Wait with timeout
    done := make(chan struct{})
    go func() {
        ps.wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        // Clean exit
    case <-time.After(5 * time.Second):
        // Timeout - force cleanup
        for _, region := range ps.regions {
            region.cancel() // Cancel context
        }
        return errors.New("exit timeout")
    }
    
    // Close channels
    for _, region := range ps.regions {
        close(region.events)
    }
    
    return nil
}
```

**Key points**:
- Signal stop via done channel
- Wait with timeout
- Force cleanup on timeout (cancel contexts)
- Close channels after goroutines exit
- Return error on timeout (don't hang)

---

### 4.4 Testing Utilities

```go
// Test helper: count goroutines
func countGoroutines() int {
    return runtime.NumGoroutine()
}

// Test helper: wait for goroutine count
func waitForGoroutineCount(t *testing.T, expected int, timeout time.Duration) {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if countGoroutines() == expected {
            return
        }
        time.Sleep(10 * time.Millisecond)
    }
    t.Fatalf("goroutine count: got %d, want %d", countGoroutines(), expected)
}

// Test helper: verify no goroutine leak
func verifyNoGoroutineLeak(t *testing.T, baseline int) {
    time.Sleep(100 * time.Millisecond) // Allow cleanup
    current := countGoroutines()
    if current > baseline {
        t.Errorf("goroutine leak: baseline %d, current %d", baseline, current)
    }
}

// Test helper: run with timeout
func runWithTimeout(t *testing.T, timeout time.Duration, fn func()) {
    done := make(chan struct{})
    go func() {
        fn()
        close(done)
    }()
    select {
    case <-done:
        // Success
    case <-time.After(timeout):
        t.Fatal("test timeout")
    }
}
```

---

## 5. Test Execution Strategy

### 5.1 Test Phases

**Phase 1: Basic Functionality** (Must pass before proceeding)
- Test 3.1.1: Basic Parallel Region Spawn
- Test 3.1.2: Parallel Region Cleanup on Exit
- Test 3.2.1: Non-blocking Event Send
- Test 3.4.1: Broadcast Event Delivery
- Test 3.5.1: Graceful Shutdown (No Pending Events)

**Phase 2: Error Handling** (Must pass before production)
- Test 3.1.3: Parallel Region Cleanup on Context Cancel
- Test 3.1.4: Parallel Region Panic Recovery
- Test 3.2.2: Channel Close Safety
- Test 3.5.2: Shutdown With Pending Events
- Test 3.5.3: Partial Shutdown on Error

**Phase 3: Advanced Features** (Must pass for full SCXML compliance)
- Test 3.1.5: Parallel Region Re-entry
- Test 3.4.2: Targeted Event Delivery
- Test 3.4.3: Event Ordering Within Region
- Test 3.5.4: Nested Parallel State Shutdown

**Phase 4: Concurrency** (Must pass with -race flag)
- Test 3.6.1: Concurrent State Access
- Test 3.6.2: Configuration Update Race
- Test 3.6.3: Transition During Event Processing
- Test 3.6.4: Concurrent Event Sends

**Phase 5: Stress Testing** (Must pass for production readiness)
- Test 3.7.1: Many Parallel Regions
- Test 3.7.2: High Event Volume
- Test 3.7.3: Deep Nesting
- Test 3.7.4: Rapid State Changes

**Phase 6: Timeout Guarantees** (Must pass for absolute certainty)
- Test 3.8.1: Entry Timeout
- Test 3.8.2: Exit Timeout
- Test 3.8.3: Event Processing Timeout
- Test 3.8.4: Shutdown Timeout

---

### 5.2 Continuous Testing

```bash
# Run all tests with race detector
go test -race -v ./...

# Run specific test category
go test -race -v -run TestParallel

# Run with timeout (fail if any test hangs)
go test -race -v -timeout 30s ./...

# Run stress tests separately (longer timeout)
go test -race -v -run TestStress -timeout 5m ./...

# Check for goroutine leaks
go test -race -v -run TestGoroutine ./...

# Profile memory usage
go test -memprofile=mem.prof -run TestManyParallelRegions
go tool pprof mem.prof
```

---

### 5.3 Success Criteria

**Absolute Certainty Checklist**:

- [ ] All Phase 1-6 tests pass
- [ ] All tests pass with `-race` flag (no data races)
- [ ] All tests pass with `-timeout 30s` (no indefinite hangs)
- [ ] No goroutine leaks detected (baseline == final count)
- [ ] No memory leaks detected (stable memory usage)
- [ ] All operations have timeouts (no indefinite waits)
- [ ] All goroutines have verified cleanup paths
- [ ] All channels closed properly on shutdown
- [ ] All panics recovered and propagated
- [ ] All errors collected and returned
- [ ] Stress tests pass (100 regions, 10k events, 1000 cycles)
- [ ] Nested parallel states work (5+ levels)
- [ ] Context cancellation propagates correctly
- [ ] Shutdown completes within 5 seconds (worst case)
- [ ] Documentation complete (all guarantees documented)

---

## 6. Documentation Requirements

### 6.1 API Documentation

Every parallel state API must document:

1. **Goroutine Lifecycle**: When goroutines are spawned and cleaned up
2. **Timeout Behavior**: Default timeouts and how to configure them
3. **Error Handling**: What errors can occur and how they're reported
4. **Thread Safety**: What operations are thread-safe
5. **Resource Cleanup**: What resources are allocated and when they're freed
6. **Event Ordering**: Guarantees about event delivery order
7. **Shutdown Behavior**: What happens to pending events on shutdown

Example:
```go
// EnterParallelState enters a parallel state, spawning one goroutine per region.
// Each region runs independently with its own event queue (buffered, size 10).
//
// Goroutine Lifecycle:
//   - Spawns N goroutines (one per region) on entry
//   - All goroutines cleaned up on exit (verified via WaitGroup)
//   - Panics recovered and propagated via error channel
//
// Timeouts:
//   - Entry timeout: 5 seconds (configurable via context)
//   - Exit timeout: 5 seconds (configurable via context)
//   - Event send timeout: 100ms per region
//
// Error Handling:
//   - Returns error if any region fails to start
//   - Returns error if entry timeout exceeded
//   - Collects errors from all regions on exit
//
// Thread Safety:
//   - Safe to call from multiple goroutines
//   - Extended state access must be synchronized by caller
//
// Resource Cleanup:
//   - All goroutines exit within 5 seconds of exit call
//   - All channels closed after goroutines exit
//   - Context cancelled on error or timeout
//
// Event Ordering:
//   - FIFO order within each region
//   - No ordering guarantee across regions
//   - Broadcast events delivered to all regions concurrently
//
// Shutdown Behavior:
//   - Pending events processed if time permits
//   - Events dropped after 5 second timeout
//   - Best-effort cleanup on timeout
func (m *Machine) EnterParallelState(ctx context.Context, state *State) error
```

---

### 6.2 Testing Documentation

Each test must document:

1. **Scenario**: What is being tested
2. **What Could Go Wrong**: Potential failure modes
3. **How to Test**: Testing approach
4. **Guarantees**: What the test proves
5. **Timeout**: Maximum test duration
6. **Cleanup**: How test cleans up resources

Example:
```go
// TestParallelRegionCleanupOnExit verifies that all region goroutines exit
// when transitioning out of a parallel state.
//
// Scenario:
//   - Enter parallel state with 3 regions
//   - Transition to non-parallel state
//   - Verify all goroutines exit
//
// What Could Go Wrong:
//   - Goroutines never exit (indefinite hang)
//   - Some goroutines exit, others hang
//   - Channels not closed (resource leak)
//   - WaitGroup never completes
//
// How to Test:
//   - Track goroutine count before/after
//   - Use done channels to verify each goroutine exits
//   - Set timeout (1 second) for exit operation
//   - Verify no goroutine leaks with runtime.NumGoroutine()
//
// Guarantees:
//   - All goroutines exit within 500ms of transition
//   - Goroutine count returns to baseline
//   - All channels closed
//   - WaitGroup completes
//
// Timeout: 2 seconds (test fails if exceeded)
//
// Cleanup: Context cancelled, all goroutines verified exited
func TestParallelRegionCleanupOnExit(t *testing.T) {
    // ...
}
```

---

## 7. Summary

This testing addendum provides **absolute certainty** that parallel states work correctly by:

1. **Comprehensive Coverage**: 32 test cases covering all failure modes
2. **Timeout Guarantees**: Every operation has a timeout, no indefinite waits
3. **Goroutine Verification**: Every spawned goroutine has a verified cleanup path
4. **Channel Safety**: All channel operations proven non-blocking or timeout-protected
5. **Race Detection**: All tests run with `-race` flag to detect data races
6. **Stress Testing**: Proven to handle 100 regions, 10k events, 1000 cycles
7. **Error Recovery**: All error paths tested, including panics and timeouts
8. **Documentation**: Every guarantee explicitly documented

**Implementation Priority**:
1. Implement Phase 1 tests first (basic functionality)
2. Don't proceed to Phase 2 until Phase 1 passes
3. Run all tests with `-race` flag continuously
4. Add timeout to every blocking operation
5. Verify goroutine cleanup in every test
6. Document all guarantees in code

**Success Metric**: All 32 tests pass with `-race` flag and `-timeout 30s`, proving parallel states work correctly with absolute certainty and will never hang indefinitely.

---

## Appendix A: Quick Reference

### Timeout Values
- Entry timeout: 5 seconds (default)
- Exit timeout: 5 seconds (default)
- Event send timeout: 100ms per region
- Action timeout: 5 seconds (default)
- Shutdown timeout: 5 seconds (default)
- Test timeout: 30 seconds (global)

### Channel Buffer Sizes
- Region event queue: 10 (minimum)
- Error channel: N (number of regions)
- Done channel: unbuffered

### Goroutine Counts
- Parallel state with N regions: N goroutines
- Nested parallel (2 levels): 2^2 = 4 goroutines
- Nested parallel (5 levels): 2^5 = 32 goroutines

### Test Execution Order
1. Phase 1: Basic (5 tests)
2. Phase 2: Error Handling (5 tests)
3. Phase 3: Advanced (4 tests)
4. Phase 4: Concurrency (4 tests) - with `-race`
5. Phase 5: Stress (4 tests) - longer timeout
6. Phase 6: Timeouts (4 tests)

### Common Failure Patterns
- Goroutine leak: `runtime.NumGoroutine()` doesn't return to baseline
- Channel leak: Channels not closed on exit
- Deadlock: Test hangs, timeout exceeded
- Race condition: `-race` flag reports data race
- Panic: Unrecovered panic crashes goroutine
- Resource leak: Memory usage grows unbounded

---

## Appendix B: Test Template

```go
// Test[Category][Scenario] verifies [what is being tested].
//
// Scenario: [description]
// What Could Go Wrong: [failure modes]
// How to Test: [approach]
// Guarantees: [what this proves]
// Timeout: [max duration]
// Cleanup: [how resources are cleaned up]
func Test[Category][Scenario](t *testing.T) {
    // Setup
    baseline := runtime.NumGoroutine()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    
    // Test body
    // ...
    
    // Verify no goroutine leak
    time.Sleep(100 * time.Millisecond)
    if runtime.NumGoroutine() > baseline {
        t.Errorf("goroutine leak: baseline %d, current %d", 
            baseline, runtime.NumGoroutine())
    }
    
    // Verify test completed within timeout
    select {
    case <-ctx.Done():
        t.Fatal("test timeout")
    default:
        // Success
    }
}
```

---

**End of Parallel State Testing Addendum**
