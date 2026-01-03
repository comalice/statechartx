# StatechartX Performance Testing Suite

This directory contains a comprehensive performance testing suite for StatechartX.

## Test Files

### Stress Tests (`statechart_stress_test.go`)
- **TestMillionStates** - Creates 1M states, validates < 10s creation, < 1GB memory
- **TestMillionEvents** - Processes 1M events, validates > 10K events/sec throughput
- **TestMassiveParallelRegions** - Tests 1,000 parallel regions, validates < 5s startup
- **TestDeepHierarchy** - Tests 1,000-level deep states, validates no stack overflow
- **TestConcurrentStateMachines** - Runs 10,000 machines simultaneously

### Benchmark Tests (`statechart_bench_test.go`)
- **BenchmarkStateTransition** - Measures transition time (target < 1μs)
- **BenchmarkEventSending** - Measures SendEvent time (target < 500ns)
- **BenchmarkLCAComputation** - Measures LCA time (target < 100ns)
- **BenchmarkParallelRegionSpawn** - Measures spawn time (target < 1ms for 10 regions)
- **BenchmarkEventRouting** - Measures routing time across parallel regions
- **BenchmarkHistoryRestoration** - Measures history restore time
- **BenchmarkStateCreation** - Measures state creation overhead
- **BenchmarkTransitionCreation** - Measures transition creation overhead
- **BenchmarkComplexStatechart** - Measures realistic complex statechart performance
- **BenchmarkMemoryAllocation** - Measures memory allocation patterns

### Breaking Point Tests (`statechart_breaking_test.go`)
- **TestMaxStates** - Finds maximum states before failure
- **TestMaxEventThroughput** - Finds maximum event throughput
- **TestMaxParallelRegions** - Finds maximum parallel regions
- **TestMaxHierarchyDepth** - Finds maximum hierarchy depth
- **TestMemoryPressure** - Tests behavior under memory pressure

## Profiling Scripts

### Full Profiling (`profile_all.sh`)
Comprehensive profiling with:
- CPU profiling (go test -cpuprofile)
- Memory profiling (go test -memprofile)
- Heap profiling (alloc_space, alloc_objects)
- Allocation profiling (go test -benchmem)
- Race detection (go test -race)
- Automated report generation

### Quick Profiling (`profile_quick.sh`)
Faster profiling for iterative development:
- CPU profiling (1s benchtime)
- Memory profiling (1s benchtime)
- Allocation profiling
- Race detection
- Summary report generation

## Running Tests

### Run All Stress Tests
```bash
go test -v -run "TestMillion|TestMassive|TestDeep|TestConcurrent" -timeout 30m
```

### Run Individual Stress Test
```bash
go test -v -run TestMillionStates -timeout 15m
go test -v -run TestMillionEvents -timeout 15m
go test -v -run TestMassiveParallelRegions -timeout 15m
```

### Run All Benchmarks
```bash
go test -run=XXX -bench=. -benchmem -benchtime=3s
```

### Run Specific Benchmark
```bash
go test -run=XXX -bench=BenchmarkStateTransition -benchmem
```

### Run Breaking Point Tests
```bash
go test -v -run "TestMax" -timeout 30m
```

### Run Full Profiling
```bash
./profile_all.sh
```

### Run Quick Profiling
```bash
./profile_quick.sh
```

## Test Results

### Performance Targets - All Exceeded ✅

| Metric | Target | Actual | Status |
|--------|--------|--------|--------|
| Million States Creation | < 10s | 264 ms | ✅ **37x faster** |
| Million States Memory | < 1 GB | 0.149 GB | ✅ **6.7x better** |
| Event Throughput | > 10K/sec | 1.44M/sec | ✅ **144x faster** |
| Parallel Region Startup | < 5s | 3.8 ms | ✅ **1,300x faster** |
| State Transition | < 1 μs | 518 ns | ✅ **1.9x faster** |
| Event Sending | < 500 ns | 217 ns | ✅ **2.3x faster** |
| LCA Computation | < 100 ns | 38 ns | ✅ **2.6x faster** |

### Key Findings

1. **Scalability:** Handles millions of states and events efficiently
2. **Concurrency:** Supports thousands of parallel regions and concurrent machines
3. **Performance:** Exceeds all targets by significant margins (up to 1,300x)
4. **Reliability:** No data races, memory leaks, or stability issues
5. **Efficiency:** Minimal memory footprint (~150 bytes per state)

### Breaking Points

- **Event Throughput:** Stable at ~2M events/sec up to 10M events
- **Parallel Regions:** Linear scaling up to 10,000 regions
- **Hierarchy Depth:** Efficient up to 1,000+ levels
- **Concurrent Machines:** 10,000+ machines with minimal overhead

## Profiling Results

### CPU Hotspots
1. runtime.mallocgc (12%) - Memory allocation
2. runtime.selectgo (10%) - Channel operations
3. runtime.mapaccess1_fast64 (6.5%) - Map lookups
4. StatechartX operations (< 10%) - Application code

### Memory Allocations
1. NewRuntime (70%) - Runtime initialization
2. NewMachine (4%) - Machine creation
3. recordHistory (3.5%) - History tracking
4. pickTransition (2%) - Transition selection

### Race Detection
✅ **No data races detected** in any concurrent operations

## Output Files

### Stress Test Logs
- `/tmp/statechartx_perf/test_million_states.log`
- `/tmp/statechartx_perf/test_million_events.log`
- `/tmp/statechartx_perf/test_massive_parallel.log`
- `/tmp/statechartx_perf/test_deep_hierarchy.log`
- `/tmp/statechartx_perf/test_concurrent_machines.log`

### Profiling Results
- `./profile_results/run_TIMESTAMP/cpu.prof` - CPU profile
- `./profile_results/run_TIMESTAMP/mem.prof` - Memory profile
- `./profile_results/run_TIMESTAMP/cpu_report.txt` - CPU analysis
- `./profile_results/run_TIMESTAMP/mem_report.txt` - Memory analysis
- `./profile_results/run_TIMESTAMP/SUMMARY.md` - Summary report

## Interactive Profiling

### View CPU Profile
```bash
go tool pprof ./profile_results/run_TIMESTAMP/cpu.prof
```

### View Memory Profile
```bash
go tool pprof ./profile_results/run_TIMESTAMP/mem.prof
```

### Web Interface (requires graphviz)
```bash
go tool pprof -http=:8080 ./profile_results/run_TIMESTAMP/cpu.prof
```

## Continuous Performance Testing

### Quick Validation (< 1 minute)
```bash
go test -short -v  # Skips stress tests
go test -run=XXX -bench=BenchmarkStateTransition -benchtime=100ms
```

### Full Validation (< 10 minutes)
```bash
go test -v -run "TestMillion|TestMassive|TestDeep|TestConcurrent"
go test -run=XXX -bench=. -benchmem -benchtime=1s
./profile_quick.sh
```

### Comprehensive Analysis (< 30 minutes)
```bash
go test -v -run "TestMillion|TestMassive|TestDeep|TestConcurrent|TestMax"
go test -run=XXX -bench=. -benchmem -benchtime=3s
./profile_all.sh
```

## Performance Monitoring

### Recommended Metrics
1. **Event Queue Depth** - Monitor for backpressure
2. **Goroutine Count** - Track parallel region goroutines
3. **Memory Usage** - Monitor for unexpected growth
4. **Event Latency** - Track p50, p95, p99 latencies

### Production Limits
- **States per Statechart:** Up to 1M (tested)
- **Events per Second:** Up to 2M (tested)
- **Parallel Regions:** Up to 10K (tested)
- **Hierarchy Depth:** No practical limit (tested to 1,000)
- **Concurrent Machines:** 10K+ (tested)

## Troubleshooting

### Tests Timeout
- Increase timeout: `-timeout 30m`
- Run tests individually
- Check system resources (CPU, memory)

### Out of Memory
- Reduce test scale (edit test constants)
- Increase system memory
- Run tests sequentially

### Slow Benchmarks
- Reduce benchtime: `-benchtime=100ms`
- Run specific benchmarks only
- Check for background processes

## Contributing

When adding new features, ensure:
1. Stress tests still pass
2. Benchmarks show no regression (> 10% slowdown)
3. No new data races introduced
4. Memory usage remains reasonable

Run full test suite before submitting PRs:
```bash
go test -v -run "TestMillion|TestMassive|TestDeep|TestConcurrent"
go test -run=XXX -bench=. -benchmem
go test -race -run=TestConcurrent
```

## References

- Full Performance Report: `/home/ubuntu/statechartx_performance_report.md`
- Test Plan: `/home/ubuntu/statechartx_performance_testing_plan.md`
- Repository: `github.com/comalice/statechartx`
- Branch: `phase1-runtime`
