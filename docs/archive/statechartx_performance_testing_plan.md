# StatechartX Performance Testing Plan

**Document Version**: 1.0  
**Date**: January 2, 2026  
**Status**: Planning Phase  
**Current State**: 32/32 tests passing, Phase 5 complete

---

## Executive Summary

This document outlines a comprehensive performance testing strategy for statechartx to identify bottlenecks, measure scalability, and ensure production readiness under extreme load. The plan covers stress testing, profiling, benchmarking, and breaking point analysis.

**Testing Goals**:
1. Validate performance with millions of states and events
2. Identify memory allocations in hot paths
3. Find CPU bottlenecks and optimize
4. Measure memory usage and detect leaks
5. Establish performance baselines and limits
6. Ensure linear scalability where expected

---

## Testing Environment

### Hardware Requirements
- **CPU**: Multi-core (4+ cores recommended)
- **RAM**: 16GB+ for extreme stress tests
- **Disk**: SSD for profiling data storage
- **OS**: Linux (Ubuntu) for consistent profiling

### Software Requirements
```bash
# Go toolchain
go version  # 1.21+

# Profiling tools
go install github.com/google/pprof@latest

# Visualization tools
sudo apt-get install graphviz  # For pprof graph generation

# Benchmarking tools
go install golang.org/x/perf/cmd/benchstat@latest
```

### Repository Setup
```bash
cd /home/ubuntu/github_repos/statechartx
git checkout phase1-runtime
git pull origin phase1-runtime
```

---

## Part 1: Stress Testing

### Objective
Test system behavior under extreme load to find breaking points and validate robustness.

### Test Suite: `statechart_stress_test.go`

#### Test 1: Million States Test
**Goal**: Validate state machine with 1,000,000 states

```go
func TestMillionStates(t *testing.T) {
    const numStates = 1_000_000
    
    // Build state machine with 1M states
    root := &State{
        ID:       "root",
        Children: make(map[StateID]*State, numStates),
    }
    
    // Create states in batches to avoid memory spike
    for i := 0; i < numStates; i++ {
        stateID := StateID(fmt.Sprintf("state_%d", i))
        root.Children[stateID] = &State{
            ID: stateID,
        }
        
        // Progress indicator
        if i%100_000 == 0 {
            t.Logf("Created %d states", i)
        }
    }
    
    // Create machine
    start := time.Now()
    m, err := NewMachine(root)
    if err != nil {
        t.Fatalf("Failed to create machine: %v", err)
    }
    creationTime := time.Since(start)
    t.Logf("Machine creation time: %v", creationTime)
    
    // Create runtime
    rt := NewRuntime(m, nil)
    ctx := context.Background()
    
    // Start runtime
    start = time.Now()
    err = rt.Start(ctx)
    if err != nil {
        t.Fatalf("Failed to start runtime: %v", err)
    }
    startTime := time.Since(start)
    t.Logf("Runtime start time: %v", startTime)
    
    // Measure memory usage
    var memStats runtime.MemStats
    runtime.ReadMemStats(&memStats)
    t.Logf("Memory allocated: %d MB", memStats.Alloc/1024/1024)
    t.Logf("Memory total: %d MB", memStats.TotalAlloc/1024/1024)
    t.Logf("Heap objects: %d", memStats.HeapObjects)
    
    // Transition to random states
    for i := 0; i < 1000; i++ {
        targetState := StateID(fmt.Sprintf("state_%d", rand.Intn(numStates)))
        err = rt.TransitionTo(ctx, targetState)
        if err != nil {
            t.Fatalf("Transition failed: %v", err)
        }
    }
    
    // Cleanup
    rt.Stop()
    
    // Success criteria
    if creationTime > 10*time.Second {
        t.Errorf("Machine creation too slow: %v", creationTime)
    }
    if memStats.Alloc/1024/1024 > 1000 { // 1GB limit
        t.Errorf("Memory usage too high: %d MB", memStats.Alloc/1024/1024)
    }
}
```

**Success Criteria**:
- Machine creation < 10 seconds
- Memory usage < 1GB
- No crashes or panics
- All transitions succeed

#### Test 2: Million Events Test
**Goal**: Process 1,000,000 events without loss or degradation

```go
func TestMillionEvents(t *testing.T) {
    const numEvents = 1_000_000
    
    // Simple state machine
    stateA := &State{ID: "A"}
    stateB := &State{ID: "B"}
    stateA.Transitions = []*Transition{
        {Event: "PING", Target: "B"},
    }
    stateB.Transitions = []*Transition{
        {Event: "PONG", Target: "A"},
    }
    
    m, _ := NewMachine(stateA)
    rt := NewRuntime(m, nil)
    ctx := context.Background()
    rt.Start(ctx)
    
    // Track processed events
    var processed atomic.Int64
    
    // Send events
    start := time.Now()
    for i := 0; i < numEvents; i++ {
        event := Event{
            ID: EventID("PING"),
        }
        err := rt.SendEvent(ctx, event)
        if err != nil {
            t.Fatalf("Failed to send event %d: %v", i, err)
        }
        processed.Add(1)
        
        // Progress indicator
        if i%100_000 == 0 {
            t.Logf("Sent %d events", i)
        }
    }
    duration := time.Since(start)
    
    // Wait for processing
    time.Sleep(5 * time.Second)
    
    // Metrics
    throughput := float64(numEvents) / duration.Seconds()
    t.Logf("Total time: %v", duration)
    t.Logf("Throughput: %.2f events/sec", throughput)
    t.Logf("Average latency: %v", duration/numEvents)
    
    // Memory stats
    var memStats runtime.MemStats
    runtime.ReadMemStats(&memStats)
    t.Logf("Memory allocated: %d MB", memStats.Alloc/1024/1024)
    
    rt.Stop()
    
    // Success criteria
    if throughput < 10_000 {
        t.Errorf("Throughput too low: %.2f events/sec", throughput)
    }
}
```

**Success Criteria**:
- Throughput > 10,000 events/sec
- No event loss
- Memory usage stable (no leaks)
- Average latency < 100μs

#### Test 3: Massive Parallel Regions Test
**Goal**: Test 1,000 parallel regions

```go
func TestMassiveParallelRegions(t *testing.T) {
    const numRegions = 1_000
    
    // Create parallel state with 1000 regions
    parallelState := &State{
        ID:         "parallel",
        IsParallel: true,
        Children:   make(map[StateID]*State, numRegions),
    }
    
    for i := 0; i < numRegions; i++ {
        regionID := StateID(fmt.Sprintf("region_%d", i))
        parallelState.Children[regionID] = &State{
            ID: regionID,
        }
    }
    
    m, _ := NewMachine(parallelState)
    rt := NewRuntime(m, nil)
    ctx := context.Background()
    
    // Track goroutines before
    goroutinesBefore := runtime.NumGoroutine()
    
    // Start runtime (spawns 1000 goroutines)
    start := time.Now()
    err := rt.Start(ctx)
    if err != nil {
        t.Fatalf("Failed to start: %v", err)
    }
    startTime := time.Since(start)
    t.Logf("Start time: %v", startTime)
    
    // Track goroutines after
    goroutinesAfter := runtime.NumGoroutine()
    goroutinesCreated := goroutinesAfter - goroutinesBefore
    t.Logf("Goroutines created: %d", goroutinesCreated)
    
    // Send broadcast event
    start = time.Now()
    rt.SendEvent(ctx, Event{ID: "BROADCAST", Address: 0})
    broadcastTime := time.Since(start)
    t.Logf("Broadcast time: %v", broadcastTime)
    
    // Memory stats
    var memStats runtime.MemStats
    runtime.ReadMemStats(&memStats)
    t.Logf("Memory allocated: %d MB", memStats.Alloc/1024/1024)
    
    // Stop runtime
    start = time.Now()
    rt.Stop()
    stopTime := time.Since(start)
    t.Logf("Stop time: %v", stopTime)
    
    // Wait for cleanup
    time.Sleep(1 * time.Second)
    
    // Verify goroutines cleaned up
    goroutinesFinal := runtime.NumGoroutine()
    if goroutinesFinal > goroutinesBefore+10 { // Allow some tolerance
        t.Errorf("Goroutine leak: before=%d, after=%d, final=%d",
            goroutinesBefore, goroutinesAfter, goroutinesFinal)
    }
    
    // Success criteria
    if startTime > 5*time.Second {
        t.Errorf("Start time too slow: %v", startTime)
    }
    if stopTime > 5*time.Second {
        t.Errorf("Stop time too slow: %v", stopTime)
    }
}
```

**Success Criteria**:
- Start time < 5 seconds
- Stop time < 5 seconds
- No goroutine leaks
- Memory usage < 500MB
- Broadcast delivery < 1 second

#### Test 4: Deep Hierarchy Stress Test
**Goal**: Test 1,000-level deep state hierarchy

```go
func TestDeepHierarchy(t *testing.T) {
    const depth = 1_000
    
    // Build deep hierarchy
    var root *State
    var current *State
    
    for i := 0; i < depth; i++ {
        state := &State{
            ID:       StateID(fmt.Sprintf("level_%d", i)),
            Children: make(map[StateID]*State),
        }
        
        if i == 0 {
            root = state
            current = state
        } else {
            current.Children[state.ID] = state
            current = state
        }
    }
    
    // Create machine
    start := time.Now()
    m, err := NewMachine(root)
    if err != nil {
        t.Fatalf("Failed to create machine: %v", err)
    }
    creationTime := time.Since(start)
    t.Logf("Machine creation time: %v", creationTime)
    
    // Create runtime
    rt := NewRuntime(m, nil)
    ctx := context.Background()
    rt.Start(ctx)
    
    // Transition to deepest state
    start = time.Now()
    err = rt.TransitionTo(ctx, current.ID)
    if err != nil {
        t.Fatalf("Transition failed: %v", err)
    }
    transitionTime := time.Since(start)
    t.Logf("Transition time: %v", transitionTime)
    
    // Memory stats
    var memStats runtime.MemStats
    runtime.ReadMemStats(&memStats)
    t.Logf("Memory allocated: %d MB", memStats.Alloc/1024/1024)
    
    rt.Stop()
    
    // Success criteria
    if transitionTime > 1*time.Second {
        t.Errorf("Transition too slow: %v", transitionTime)
    }
}
```

**Success Criteria**:
- Machine creation < 5 seconds
- Transition time < 1 second
- No stack overflow
- Memory usage < 100MB

#### Test 5: Concurrent State Machine Test
**Goal**: Run 10,000 state machines concurrently

```go
func TestConcurrentStateMachines(t *testing.T) {
    const numMachines = 10_000
    
    // Simple state machine template
    stateA := &State{ID: "A"}
    stateB := &State{ID: "B"}
    stateA.Transitions = []*Transition{
        {Event: "GO", Target: "B"},
    }
    
    var wg sync.WaitGroup
    errors := make(chan error, numMachines)
    
    start := time.Now()
    
    // Launch 10k state machines
    for i := 0; i < numMachines; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            // Create machine
            m, err := NewMachine(stateA)
            if err != nil {
                errors <- fmt.Errorf("machine %d: %v", id, err)
                return
            }
            
            // Create runtime
            rt := NewRuntime(m, nil)
            ctx := context.Background()
            
            // Start
            if err := rt.Start(ctx); err != nil {
                errors <- fmt.Errorf("machine %d start: %v", id, err)
                return
            }
            
            // Send events
            for j := 0; j < 100; j++ {
                rt.SendEvent(ctx, Event{ID: "GO"})
            }
            
            // Stop
            rt.Stop()
        }(i)
        
        // Progress indicator
        if i%1000 == 0 {
            t.Logf("Launched %d machines", i)
        }
    }
    
    // Wait for completion
    wg.Wait()
    close(errors)
    
    duration := time.Since(start)
    t.Logf("Total time: %v", duration)
    t.Logf("Time per machine: %v", duration/numMachines)
    
    // Check for errors
    errorCount := 0
    for err := range errors {
        t.Errorf("Error: %v", err)
        errorCount++
    }
    
    // Memory stats
    var memStats runtime.MemStats
    runtime.ReadMemStats(&memStats)
    t.Logf("Memory allocated: %d MB", memStats.Alloc/1024/1024)
    
    // Success criteria
    if errorCount > 0 {
        t.Errorf("Failed machines: %d", errorCount)
    }
    if duration > 60*time.Second {
        t.Errorf("Execution too slow: %v", duration)
    }
}
```

**Success Criteria**:
- All machines complete successfully
- Total time < 60 seconds
- No race conditions (run with `-race`)
- Memory usage < 2GB

---

## Part 2: Allocation Profiling

### Objective
Identify memory allocations in hot paths and optimize to reduce GC pressure.

### Profiling Strategy

#### Step 1: Enable Allocation Profiling
```go
// Add to test file: statechart_alloc_profile_test.go

func TestAllocationProfile(t *testing.T) {
    // Create profile file
    f, err := os.Create("/home/ubuntu/statechartx_alloc.prof")
    if err != nil {
        t.Fatal(err)
    }
    defer f.Close()
    
    // Start memory profiling
    runtime.GC() // Clear existing allocations
    
    // Run workload
    runWorkload(t)
    
    // Write memory profile
    if err := pprof.WriteHeapProfile(f); err != nil {
        t.Fatal(err)
    }
}

func runWorkload(t *testing.T) {
    // Create state machine
    stateA := &State{ID: "A"}
    stateB := &State{ID: "B"}
    stateA.Transitions = []*Transition{
        {Event: "GO", Target: "B"},
    }
    stateB.Transitions = []*Transition{
        {Event: "BACK", Target: "A"},
    }
    
    m, _ := NewMachine(stateA)
    rt := NewRuntime(m, nil)
    ctx := context.Background()
    rt.Start(ctx)
    
    // Send 100k events (hot path)
    for i := 0; i < 100_000; i++ {
        rt.SendEvent(ctx, Event{ID: "GO"})
        rt.SendEvent(ctx, Event{ID: "BACK"})
    }
    
    rt.Stop()
}
```

#### Step 2: Run Allocation Profiling
```bash
# Run test with memory profiling
cd /home/ubuntu/github_repos/statechartx
go test -run TestAllocationProfile -memprofile=/home/ubuntu/statechartx_alloc.prof

# Analyze with pprof
go tool pprof -alloc_space /home/ubuntu/statechartx_alloc.prof

# Interactive commands:
# (pprof) top10          # Top 10 allocation sources
# (pprof) list SendEvent # Show allocations in SendEvent function
# (pprof) web            # Generate visual graph (requires graphviz)
```

#### Step 3: Analyze Allocation Hotspots
```bash
# Generate allocation report
go tool pprof -top -alloc_space /home/ubuntu/statechartx_alloc.prof > /home/ubuntu/statechartx_alloc_report.txt

# Generate visual graph
go tool pprof -png -alloc_space /home/ubuntu/statechartx_alloc.prof > /home/ubuntu/statechartx_alloc_graph.png

# Focus on specific function
go tool pprof -list=SendEvent /home/ubuntu/statechartx_alloc.prof
```

### Key Areas to Profile

1. **Event Sending Path**
   - `SendEvent()` function
   - Event struct allocations
   - Channel operations
   - Context allocations

2. **State Transition Path**
   - `executeTransition()` function
   - LCA computation
   - State entry/exit actions
   - Transition selection

3. **Parallel Region Management**
   - Goroutine spawning
   - Channel creation
   - Context creation
   - Event distribution

4. **Machine Creation**
   - State tree building
   - Map allocations
   - Slice allocations

### Optimization Targets

**Target 1: Reduce Event Allocations**
- Use object pools for Event structs
- Reuse event buffers
- Avoid unnecessary copying

**Target 2: Optimize State Lookups**
- Cache frequently accessed states
- Use more efficient data structures
- Reduce map lookups

**Target 3: Minimize Channel Allocations**
- Reuse channels where possible
- Use buffered channels efficiently
- Avoid creating channels in hot path

**Target 4: Reduce Context Allocations**
- Reuse contexts where safe
- Avoid unnecessary context wrapping
- Use context pools

### Success Criteria
- Identify top 10 allocation sources
- Reduce allocations in hot path by 50%
- Document optimization opportunities
- Create optimization roadmap

---

## Part 3: CPU Profiling

### Objective
Identify CPU bottlenecks and hot spots in the codebase.

### Profiling Strategy

#### Step 1: Enable CPU Profiling
```go
// Add to test file: statechart_cpu_profile_test.go

func TestCPUProfile(t *testing.T) {
    // Create profile file
    f, err := os.Create("/home/ubuntu/statechartx_cpu.prof")
    if err != nil {
        t.Fatal(err)
    }
    defer f.Close()
    
    // Start CPU profiling
    if err := pprof.StartCPUProfile(f); err != nil {
        t.Fatal(err)
    }
    defer pprof.StopCPUProfile()
    
    // Run CPU-intensive workload
    runCPUWorkload(t)
}

func runCPUWorkload(t *testing.T) {
    // Create complex state machine
    root := createComplexStateMachine(100) // 100 states
    
    m, _ := NewMachine(root)
    rt := NewRuntime(m, nil)
    ctx := context.Background()
    rt.Start(ctx)
    
    // Send 1M events
    for i := 0; i < 1_000_000; i++ {
        rt.SendEvent(ctx, Event{ID: "EVENT"})
    }
    
    rt.Stop()
}
```

#### Step 2: Run CPU Profiling
```bash
# Run test with CPU profiling
cd /home/ubuntu/github_repos/statechartx
go test -run TestCPUProfile -cpuprofile=/home/ubuntu/statechartx_cpu.prof

# Analyze with pprof
go tool pprof /home/ubuntu/statechartx_cpu.prof

# Interactive commands:
# (pprof) top10          # Top 10 CPU consumers
# (pprof) list SendEvent # Show CPU usage in SendEvent
# (pprof) web            # Generate visual graph
```

#### Step 3: Analyze CPU Hotspots
```bash
# Generate CPU report
go tool pprof -top /home/ubuntu/statechartx_cpu.prof > /home/ubuntu/statechartx_cpu_report.txt

# Generate flame graph (requires go-torch or pprof web)
go tool pprof -http=:8080 /home/ubuntu/statechartx_cpu.prof
# Open browser to http://localhost:8080

# Focus on specific function
go tool pprof -list=executeTransition /home/ubuntu/statechartx_cpu.prof
```

### Key Areas to Profile

1. **Event Processing Loop**
   - Event queue management
   - Event dispatching
   - Transition selection

2. **LCA Computation**
   - Path finding algorithms
   - State hierarchy traversal
   - Comparison operations

3. **Lock Contention**
   - Mutex lock/unlock
   - RWMutex usage
   - Channel blocking

4. **Goroutine Scheduling**
   - Context switching overhead
   - Goroutine creation/destruction
   - Channel operations

### Optimization Targets

**Target 1: Optimize LCA Computation**
- Cache LCA results
- Use more efficient algorithms
- Reduce state hierarchy traversals

**Target 2: Reduce Lock Contention**
- Use lock-free data structures where possible
- Reduce critical section size
- Use RWMutex for read-heavy operations

**Target 3: Optimize Event Dispatching**
- Batch event processing
- Reduce channel operations
- Use more efficient event queues

**Target 4: Minimize Goroutine Overhead**
- Use goroutine pools
- Reduce goroutine creation
- Optimize channel buffer sizes

### Success Criteria
- Identify top 10 CPU consumers
- Reduce CPU usage in hot path by 30%
- Document bottlenecks
- Create optimization roadmap

---

## Part 4: Memory Profiling

### Objective
Track memory usage, identify leaks, and optimize memory footprint.

### Profiling Strategy

#### Step 1: Memory Usage Tracking
```go
// Add to test file: statechart_memory_profile_test.go

func TestMemoryUsage(t *testing.T) {
    // Track memory over time
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    done := make(chan bool)
    
    // Memory tracking goroutine
    go func() {
        for {
            select {
            case <-ticker.C:
                var m runtime.MemStats
                runtime.ReadMemStats(&m)
                t.Logf("Alloc=%d MB, TotalAlloc=%d MB, Sys=%d MB, NumGC=%d",
                    m.Alloc/1024/1024,
                    m.TotalAlloc/1024/1024,
                    m.Sys/1024/1024,
                    m.NumGC)
            case <-done:
                return
            }
        }
    }()
    
    // Run workload
    runMemoryWorkload(t)
    
    done <- true
}

func runMemoryWorkload(t *testing.T) {
    // Create and destroy 1000 state machines
    for i := 0; i < 1000; i++ {
        root := createStateMachine()
        m, _ := NewMachine(root)
        rt := NewRuntime(m, nil)
        ctx := context.Background()
        rt.Start(ctx)
        
        // Send events
        for j := 0; j < 1000; j++ {
            rt.SendEvent(ctx, Event{ID: "EVENT"})
        }
        
        rt.Stop()
        
        // Force GC every 100 iterations
        if i%100 == 0 {
            runtime.GC()
            t.Logf("Completed %d iterations", i)
        }
    }
}
```

#### Step 2: Memory Leak Detection
```go
func TestMemoryLeak(t *testing.T) {
    // Baseline memory
    runtime.GC()
    var m1 runtime.MemStats
    runtime.ReadMemStats(&m1)
    baseline := m1.Alloc
    
    // Run workload multiple times
    for i := 0; i < 10; i++ {
        runWorkloadIteration(t)
        runtime.GC()
        
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        current := m.Alloc
        growth := current - baseline
        
        t.Logf("Iteration %d: Memory growth = %d MB",
            i, growth/1024/1024)
        
        // Check for leak
        if growth > 100*1024*1024 { // 100MB threshold
            t.Errorf("Possible memory leak: %d MB growth", growth/1024/1024)
        }
    }
}
```

#### Step 3: Heap Profiling
```bash
# Run test with heap profiling
go test -run TestMemoryUsage -memprofile=/home/ubuntu/statechartx_heap.prof

# Analyze heap usage
go tool pprof -inuse_space /home/ubuntu/statechartx_heap.prof

# Interactive commands:
# (pprof) top10          # Top 10 memory consumers
# (pprof) list NewRuntime # Show memory usage in NewRuntime
# (pprof) web            # Generate visual graph
```

### Key Areas to Profile

1. **Runtime Lifecycle**
   - Runtime creation
   - Runtime destruction
   - Goroutine cleanup
   - Channel cleanup

2. **State Machine Storage**
   - State tree memory
   - Transition storage
   - Map overhead
   - Slice overhead

3. **Event Queues**
   - Channel buffers
   - Event struct storage
   - Queue growth

4. **Parallel Regions**
   - Per-region overhead
   - Goroutine stack memory
   - Channel memory

### Optimization Targets

**Target 1: Reduce Runtime Overhead**
- Minimize per-runtime allocations
- Reuse runtime instances
- Optimize cleanup

**Target 2: Optimize State Storage**
- Use more compact data structures
- Share immutable state data
- Reduce pointer overhead

**Target 3: Optimize Event Queues**
- Use ring buffers instead of channels
- Limit queue growth
- Implement backpressure

**Target 4: Reduce Parallel Region Overhead**
- Use goroutine pools
- Share channels where safe
- Optimize per-region memory

### Success Criteria
- No memory leaks detected
- Memory usage < 1GB for 1M states
- Memory usage < 500MB for 1000 parallel regions
- GC pressure minimized
- Document memory characteristics

---

## Part 5: Benchmark Tests

### Objective
Establish performance baselines and track improvements over time.

### Benchmark Suite: `statechart_bench_test.go`

#### Benchmark 1: State Transition
```go
func BenchmarkStateTransition(b *testing.B) {
    stateA := &State{ID: "A"}
    stateB := &State{ID: "B"}
    stateA.Transitions = []*Transition{
        {Event: "GO", Target: "B"},
    }
    
    m, _ := NewMachine(stateA)
    rt := NewRuntime(m, nil)
    ctx := context.Background()
    rt.Start(ctx)
    defer rt.Stop()
    
    event := Event{ID: "GO"}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        rt.SendEvent(ctx, event)
    }
}
```

#### Benchmark 2: Event Sending
```go
func BenchmarkEventSend(b *testing.B) {
    state := &State{ID: "A"}
    m, _ := NewMachine(state)
    rt := NewRuntime(m, nil)
    ctx := context.Background()
    rt.Start(ctx)
    defer rt.Stop()
    
    event := Event{ID: "EVENT"}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        rt.SendEvent(ctx, event)
    }
}
```

#### Benchmark 3: LCA Computation
```go
func BenchmarkLCAComputation(b *testing.B) {
    // Create deep hierarchy
    root := createDeepHierarchy(100) // 100 levels
    m, _ := NewMachine(root)
    rt := NewRuntime(m, nil)
    
    // Get two deep states
    state1 := getStateAtDepth(root, 50)
    state2 := getStateAtDepth(root, 75)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        rt.computeLCA(state1, state2)
    }
}
```

#### Benchmark 4: Parallel Region Spawn
```go
func BenchmarkParallelRegionSpawn(b *testing.B) {
    parallelState := &State{
        ID:         "parallel",
        IsParallel: true,
        Children:   make(map[StateID]*State),
    }
    
    for i := 0; i < 10; i++ {
        parallelState.Children[StateID(fmt.Sprintf("r%d", i))] = &State{
            ID: StateID(fmt.Sprintf("r%d", i)),
        }
    }
    
    m, _ := NewMachine(parallelState)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        rt := NewRuntime(m, nil)
        ctx := context.Background()
        rt.Start(ctx)
        rt.Stop()
    }
}
```

#### Benchmark 5: Machine Creation
```go
func BenchmarkMachineCreation(b *testing.B) {
    root := createStateMachine() // Standard state machine
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        NewMachine(root)
    }
}
```

### Running Benchmarks
```bash
# Run all benchmarks
cd /home/ubuntu/github_repos/statechartx
go test -bench=. -benchmem -benchtime=10s > /home/ubuntu/statechartx_bench_results.txt

# Run specific benchmark
go test -bench=BenchmarkStateTransition -benchmem -benchtime=10s

# Compare benchmarks (before/after optimization)
go test -bench=. -benchmem > /home/ubuntu/bench_before.txt
# ... make optimizations ...
go test -bench=. -benchmem > /home/ubuntu/bench_after.txt
benchstat /home/ubuntu/bench_before.txt /home/ubuntu/bench_after.txt
```

### Success Criteria
- State transition: < 1μs per transition
- Event send: < 500ns per event
- LCA computation: < 100ns
- Parallel region spawn: < 1ms for 10 regions
- Machine creation: < 10μs

---

## Part 6: Breaking Point Tests

### Objective
Find the limits where the system fails or degrades significantly.

### Test Suite: `statechart_breaking_point_test.go`

#### Test 1: Maximum States
**Goal**: Find maximum number of states before failure

```go
func TestMaximumStates(t *testing.T) {
    sizes := []int{1_000, 10_000, 100_000, 1_000_000, 10_000_000}
    
    for _, size := range sizes {
        t.Run(fmt.Sprintf("States_%d", size), func(t *testing.T) {
            success := testStateCount(t, size)
            if !success {
                t.Logf("Breaking point: %d states", size)
                return
            }
            t.Logf("Success with %d states", size)
        })
    }
}

func testStateCount(t *testing.T, count int) bool {
    defer func() {
        if r := recover(); r != nil {
            t.Logf("Panic at %d states: %v", count, r)
        }
    }()
    
    root := &State{
        ID:       "root",
        Children: make(map[StateID]*State, count),
    }
    
    for i := 0; i < count; i++ {
        root.Children[StateID(fmt.Sprintf("s%d", i))] = &State{
            ID: StateID(fmt.Sprintf("s%d", i)),
        }
    }
    
    m, err := NewMachine(root)
    if err != nil {
        t.Logf("Machine creation failed: %v", err)
        return false
    }
    
    rt := NewRuntime(m, nil)
    ctx := context.Background()
    err = rt.Start(ctx)
    if err != nil {
        t.Logf("Runtime start failed: %v", err)
        return false
    }
    
    rt.Stop()
    return true
}
```

#### Test 2: Maximum Events Per Second
**Goal**: Find maximum event throughput

```go
func TestMaximumEventThroughput(t *testing.T) {
    rates := []int{1_000, 10_000, 100_000, 1_000_000, 10_000_000}
    
    for _, rate := range rates {
        t.Run(fmt.Sprintf("Rate_%d", rate), func(t *testing.T) {
            throughput := testEventRate(t, rate)
            t.Logf("Achieved throughput: %.2f events/sec", throughput)
        })
    }
}

func testEventRate(t *testing.T, targetRate int) float64 {
    state := &State{ID: "A"}
    m, _ := NewMachine(state)
    rt := NewRuntime(m, nil)
    ctx := context.Background()
    rt.Start(ctx)
    defer rt.Stop()
    
    event := Event{ID: "EVENT"}
    duration := 10 * time.Second
    targetCount := targetRate * 10 // 10 seconds
    
    start := time.Now()
    successCount := 0
    
    for i := 0; i < targetCount; i++ {
        err := rt.SendEvent(ctx, event)
        if err == nil {
            successCount++
        }
        
        if time.Since(start) > duration {
            break
        }
    }
    
    elapsed := time.Since(start)
    return float64(successCount) / elapsed.Seconds()
}
```

#### Test 3: Maximum Parallel Regions
**Goal**: Find maximum number of parallel regions

```go
func TestMaximumParallelRegions(t *testing.T) {
    counts := []int{10, 100, 1_000, 10_000, 100_000}
    
    for _, count := range counts {
        t.Run(fmt.Sprintf("Regions_%d", count), func(t *testing.T) {
            success := testParallelRegionCount(t, count)
            if !success {
                t.Logf("Breaking point: %d regions", count)
                return
            }
            t.Logf("Success with %d regions", count)
        })
    }
}
```

#### Test 4: Maximum Hierarchy Depth
**Goal**: Find maximum state hierarchy depth

```go
func TestMaximumHierarchyDepth(t *testing.T) {
    depths := []int{10, 100, 1_000, 10_000, 100_000}
    
    for _, depth := range depths {
        t.Run(fmt.Sprintf("Depth_%d", depth), func(t *testing.T) {
            success := testHierarchyDepth(t, depth)
            if !success {
                t.Logf("Breaking point: %d levels", depth)
                return
            }
            t.Logf("Success with %d levels", depth)
        })
    }
}
```

### Success Criteria
- Document breaking points for all dimensions
- Identify failure modes (OOM, timeout, panic, etc.)
- Establish safe operating limits
- Create scaling guidelines

---

## Part 7: Scalability Tests

### Objective
Verify linear vs exponential scaling characteristics.

### Test Suite: `statechart_scalability_test.go`

#### Test 1: State Count Scalability
**Goal**: Verify O(1) or O(log n) state lookup

```go
func TestStateCountScalability(t *testing.T) {
    sizes := []int{100, 1_000, 10_000, 100_000}
    results := make(map[int]time.Duration)
    
    for _, size := range sizes {
        duration := measureStateTransitionTime(t, size)
        results[size] = duration
        t.Logf("Size %d: %v per transition", size, duration)
    }
    
    // Analyze scaling
    analyzeScaling(t, results)
}

func analyzeScaling(t *testing.T, results map[int]time.Duration) {
    // Check if scaling is linear, logarithmic, or exponential
    // Compare ratios: if time doubles when size doubles, it's linear
    // If time stays constant, it's O(1)
    // If time quadruples when size doubles, it's exponential
    
    sizes := []int{100, 1_000, 10_000, 100_000}
    for i := 1; i < len(sizes); i++ {
        sizeRatio := float64(sizes[i]) / float64(sizes[i-1])
        timeRatio := float64(results[sizes[i]]) / float64(results[sizes[i-1]])
        
        t.Logf("Size ratio: %.2f, Time ratio: %.2f", sizeRatio, timeRatio)
        
        if timeRatio > sizeRatio*1.5 {
            t.Errorf("Scaling worse than linear: size %.2fx, time %.2fx",
                sizeRatio, timeRatio)
        }
    }
}
```

#### Test 2: Event Rate Scalability
**Goal**: Verify linear throughput scaling

```go
func TestEventRateScalability(t *testing.T) {
    rates := []int{1_000, 10_000, 100_000, 1_000_000}
    
    for _, rate := range rates {
        throughput := measureEventThroughput(t, rate)
        t.Logf("Target rate: %d, Achieved: %.2f", rate, throughput)
    }
}
```

#### Test 3: Parallel Region Scalability
**Goal**: Verify linear scaling with region count

```go
func TestParallelRegionScalability(t *testing.T) {
    counts := []int{10, 100, 1_000}
    
    for _, count := range counts {
        duration := measureParallelRegionTime(t, count)
        t.Logf("Regions: %d, Time: %v", count, duration)
    }
}
```

### Success Criteria
- State lookup: O(1) or O(log n)
- Event processing: Linear throughput
- Parallel regions: Linear scaling
- Memory usage: Linear with state count
- No exponential degradation

---

## Part 8: Profiling Automation

### Objective
Automate profiling and reporting for continuous performance monitoring.

### Automation Script: `profile_all.sh`

```bash
#!/bin/bash
# profile_all.sh - Comprehensive profiling automation

set -e

REPO_DIR="/home/ubuntu/github_repos/statechartx"
OUTPUT_DIR="/home/ubuntu/statechartx_profiles"
DATE=$(date +%Y%m%d_%H%M%S)
PROFILE_DIR="$OUTPUT_DIR/$DATE"

mkdir -p "$PROFILE_DIR"

cd "$REPO_DIR"

echo "=== StatechartX Performance Profiling ==="
echo "Date: $DATE"
echo "Output: $PROFILE_DIR"
echo ""

# 1. CPU Profiling
echo "[1/7] Running CPU profiling..."
go test -run TestCPUProfile -cpuprofile="$PROFILE_DIR/cpu.prof" -v
go tool pprof -top "$PROFILE_DIR/cpu.prof" > "$PROFILE_DIR/cpu_report.txt"
go tool pprof -png "$PROFILE_DIR/cpu.prof" > "$PROFILE_DIR/cpu_graph.png"
echo "  ✓ CPU profile saved"

# 2. Memory Profiling
echo "[2/7] Running memory profiling..."
go test -run TestAllocationProfile -memprofile="$PROFILE_DIR/mem.prof" -v
go tool pprof -top -alloc_space "$PROFILE_DIR/mem.prof" > "$PROFILE_DIR/mem_report.txt"
go tool pprof -png -alloc_space "$PROFILE_DIR/mem.prof" > "$PROFILE_DIR/mem_graph.png"
echo "  ✓ Memory profile saved"

# 3. Heap Profiling
echo "[3/7] Running heap profiling..."
go test -run TestMemoryUsage -memprofile="$PROFILE_DIR/heap.prof" -v
go tool pprof -top -inuse_space "$PROFILE_DIR/heap.prof" > "$PROFILE_DIR/heap_report.txt"
echo "  ✓ Heap profile saved"

# 4. Benchmarks
echo "[4/7] Running benchmarks..."
go test -bench=. -benchmem -benchtime=10s > "$PROFILE_DIR/bench_results.txt"
echo "  ✓ Benchmark results saved"

# 5. Stress Tests
echo "[5/7] Running stress tests..."
go test -run "TestMillion|TestMassive|TestDeep|TestConcurrent" -v -timeout 30m > "$PROFILE_DIR/stress_results.txt" 2>&1
echo "  ✓ Stress test results saved"

# 6. Race Detection
echo "[6/7] Running race detection..."
go test ./... -race -timeout 30s > "$PROFILE_DIR/race_results.txt" 2>&1
echo "  ✓ Race detection results saved"

# 7. Breaking Point Tests
echo "[7/7] Running breaking point tests..."
go test -run "TestMaximum" -v -timeout 60m > "$PROFILE_DIR/breaking_point_results.txt" 2>&1
echo "  ✓ Breaking point results saved"

echo ""
echo "=== Profiling Complete ==="
echo "Results saved to: $PROFILE_DIR"
echo ""
echo "Summary:"
echo "  - CPU profile: $PROFILE_DIR/cpu_report.txt"
echo "  - Memory profile: $PROFILE_DIR/mem_report.txt"
echo "  - Heap profile: $PROFILE_DIR/heap_report.txt"
echo "  - Benchmarks: $PROFILE_DIR/bench_results.txt"
echo "  - Stress tests: $PROFILE_DIR/stress_results.txt"
echo "  - Race detection: $PROFILE_DIR/race_results.txt"
echo "  - Breaking points: $PROFILE_DIR/breaking_point_results.txt"
echo ""
```

### Usage
```bash
chmod +x /home/ubuntu/profile_all.sh
/home/ubuntu/profile_all.sh
```

---

## Part 9: Performance Metrics Dashboard

### Objective
Create a dashboard to visualize performance metrics over time.

### Metrics to Track

1. **Throughput Metrics**
   - Events per second
   - Transitions per second
   - State changes per second

2. **Latency Metrics**
   - Average event latency
   - P50, P95, P99 latency
   - Maximum latency

3. **Resource Metrics**
   - Memory usage (MB)
   - CPU usage (%)
   - Goroutine count
   - GC pause time

4. **Scalability Metrics**
   - Time vs state count
   - Time vs event rate
   - Time vs parallel regions

5. **Reliability Metrics**
   - Error rate
   - Panic count
   - Goroutine leaks
   - Race conditions

### Dashboard Implementation

```go
// metrics.go - Performance metrics collection

package statechart

import (
    "sync/atomic"
    "time"
)

type Metrics struct {
    EventsSent       atomic.Int64
    EventsProcessed  atomic.Int64
    TransitionsCount atomic.Int64
    ErrorCount       atomic.Int64
    
    TotalLatency     atomic.Int64 // nanoseconds
    MaxLatency       atomic.Int64 // nanoseconds
    
    StartTime        time.Time
}

func (m *Metrics) RecordEvent(latency time.Duration) {
    m.EventsProcessed.Add(1)
    m.TotalLatency.Add(int64(latency))
    
    // Update max latency
    for {
        current := m.MaxLatency.Load()
        if int64(latency) <= current {
            break
        }
        if m.MaxLatency.CompareAndSwap(current, int64(latency)) {
            break
        }
    }
}

func (m *Metrics) GetThroughput() float64 {
    elapsed := time.Since(m.StartTime).Seconds()
    return float64(m.EventsProcessed.Load()) / elapsed
}

func (m *Metrics) GetAverageLatency() time.Duration {
    total := m.TotalLatency.Load()
    count := m.EventsProcessed.Load()
    if count == 0 {
        return 0
    }
    return time.Duration(total / count)
}

func (m *Metrics) GetMaxLatency() time.Duration {
    return time.Duration(m.MaxLatency.Load())
}

func (m *Metrics) Report() string {
    return fmt.Sprintf(
        "Events: %d, Throughput: %.2f/s, Avg Latency: %v, Max Latency: %v, Errors: %d",
        m.EventsProcessed.Load(),
        m.GetThroughput(),
        m.GetAverageLatency(),
        m.GetMaxLatency(),
        m.ErrorCount.Load(),
    )
}
```

---

## Part 10: Optimization Roadmap

### Phase 1: Quick Wins (1-2 weeks)
1. **Object Pooling**
   - Pool Event structs
   - Pool transition objects
   - Reduce allocations by 30-50%

2. **Cache Optimization**
   - Cache LCA results
   - Cache state lookups
   - Reduce CPU by 20-30%

3. **Lock Optimization**
   - Use RWMutex for read-heavy operations
   - Reduce critical section size
   - Improve concurrency by 20-40%

### Phase 2: Structural Improvements (2-4 weeks)
1. **Event Queue Optimization**
   - Replace channels with ring buffers
   - Implement backpressure
   - Improve throughput by 50-100%

2. **Goroutine Pooling**
   - Use worker pools for parallel regions
   - Reduce goroutine creation overhead
   - Improve startup time by 50%

3. **State Storage Optimization**
   - Use more compact data structures
   - Reduce pointer overhead
   - Reduce memory by 30-50%

### Phase 3: Advanced Optimizations (4-8 weeks)
1. **Lock-Free Data Structures**
   - Implement lock-free event queues
   - Use atomic operations where possible
   - Improve concurrency by 100%+

2. **SIMD Optimizations**
   - Vectorize state comparisons
   - Optimize batch operations
   - Improve CPU by 50%+

3. **Zero-Copy Event Passing**
   - Eliminate event copying
   - Use shared memory where safe
   - Reduce allocations by 80%+

---

## Summary

This comprehensive performance testing plan covers:

1. ✅ **Stress Testing**: Millions of states, events, parallel regions
2. ✅ **Allocation Profiling**: pprof-based allocation analysis
3. ✅ **CPU Profiling**: Bottleneck identification and optimization
4. ✅ **Memory Profiling**: Leak detection and memory optimization
5. ✅ **Benchmark Tests**: Performance baselines and tracking
6. ✅ **Breaking Point Tests**: System limits and failure modes
7. ✅ **Scalability Tests**: Linear vs exponential growth analysis
8. ✅ **Automation**: Scripted profiling and reporting
9. ✅ **Metrics Dashboard**: Performance visualization
10. ✅ **Optimization Roadmap**: Phased improvement plan

### Execution Timeline

**Week 1-2**: Setup and baseline testing
- Implement all test suites
- Run initial profiling
- Establish baselines

**Week 3-4**: Stress testing and breaking point analysis
- Run extreme load tests
- Identify breaking points
- Document limitations

**Week 5-6**: Profiling and optimization
- Deep dive into allocations
- Identify CPU bottlenecks
- Create optimization plan

**Week 7-8**: Implementation and validation
- Implement optimizations
- Re-run benchmarks
- Validate improvements

### Success Criteria

- ✅ All stress tests pass or document breaking points
- ✅ Allocation hotspots identified and documented
- ✅ CPU bottlenecks identified and documented
- ✅ No memory leaks detected
- ✅ Performance baselines established
- ✅ Optimization roadmap created
- ✅ Automated profiling pipeline working

---

**End of Performance Testing Plan**
