# StatechartX Benchmarks

This directory contains performance benchmarks for the StatechartX state machine library.

## Running Benchmarks

### Run All Benchmarks
```bash
go test -bench=. -benchmem ./benchmarks ./internal/core
```

### Run Specific Benchmark
```bash
go test -bench=BenchmarkRealtimeThroughput -benchmem ./benchmarks
```

### Using Make Targets
```bash
make bench              # Run all benchmarks with memory stats
make bench-vs-baseline  # Compare against baseline
make bench-baseline     # Update baseline with current results
make bench-snapshot     # Capture dated snapshot for documentation
```

## Benchmark Result Storage

Benchmark results are stored in `benchmarks/results/`:

- **`baseline.txt`**: Current performance baseline (tracked in git)
- **`YYYY-MM-DD.txt`**: Dated snapshots for documentation/PRs
- **`old.txt`/`new.txt`**: Temporary comparison files (gitignored)

## Comparing Benchmarks

### Compare Against Baseline

The most common workflow - compare your current changes against the established baseline:

```bash
make bench-vs-baseline
```

This will:
1. Run current benchmarks
2. Compare against `benchmarks/results/baseline.txt`
3. Show performance differences using `benchstat`

**Example output:**
```
name                          old time/op    new time/op    delta
RealtimeThroughput-4           10.2µs ± 2%    9.8µs ± 1%  -3.92%  (p=0.000 n=10+10)
SimpleTransition-4              113ns ± 1%    108ns ± 2%  -4.42%  (p=0.000 n=10+10)

name                          old alloc/op   new alloc/op   delta
RealtimeThroughput-4            0.00B          0.00B          ~   (all equal)

name                          old allocs/op  new allocs/op  delta
RealtimeThroughput-4             0.00           0.00          ~   (all equal)
```

### Manual Comparison

For comparing two specific benchmark runs:

```bash
# Capture "before" state
go test -bench=. -benchmem ./benchmarks > old.txt

# Make your changes...

# Capture "after" state
go test -bench=. -benchmem ./benchmarks > new.txt

# Compare
make bench-cmp
```

## Updating the Baseline

After verifying performance improvements, update the baseline:

```bash
make bench-vs-baseline  # Verify improvements
make bench-baseline     # Update baseline
git add benchmarks/results/baseline.txt
git commit -m "perf: update benchmark baseline after optimization"
```

**When to update:**
- After significant performance improvements
- After fixing benchmark issues (e.g., honest benchmark fixes)
- When establishing a new reference point for a major version

## Capturing Snapshots for Documentation

For PRs, issues, or release notes, capture a dated snapshot:

```bash
make bench-snapshot
```

This creates `benchmarks/results/YYYY-MM-DD.txt` which you can:
- Include in PR descriptions
- Reference in issue comments
- Keep for historical tracking

**Example:**
```bash
make bench-snapshot
# Creates: benchmarks/results/2026-01-22.txt
# Include in PR: "See benchmarks/results/2026-01-22.txt for performance impact"
```

## Interpreting benchstat Output

### Performance Changes

- **Negative delta (-)**: Improvement (faster/less memory)
- **Positive delta (+)**: Regression (slower/more memory)
- **~**: No significant change

### Statistical Significance

- **p-value < 0.05**: Statistically significant change
- **p-value > 0.05**: Change may be noise

### Confidence

- **n=10+10**: Number of samples (more is better)
- **±2%**: Standard deviation (lower is more consistent)

## Prerequisites

### Install benchstat

```bash
make install-benchstat
```

Or manually:
```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

## Benchmark Types

### Realtime Runtime Benchmarks

- **BenchmarkRealtimeThroughput**: Events processed per second (with verification)
- **BenchmarkRealtimeLatency**: End-to-end event latency
- **BenchmarkRealtimeQueueCapacity**: Queue limits before backpressure
- **BenchmarkRealtimeTickProcessing**: Batch processing time per tick

### Core Runtime Benchmarks

- **BenchmarkEventThroughput**: Concurrent event processing (8 workers)
- **BenchmarkSimpleTransition**: Self-loop transitions
- **BenchmarkHierarchicalTransition**: Parent-child state transitions
- **BenchmarkParallelTransition**: Orthogonal region transitions
- **BenchmarkGuardedTransition**: Conditional transitions
- **BenchmarkTransition** (internal/core): Basic state transitions

### Other Benchmarks

- **BenchmarkMemoryFootprint**: Memory per machine instance

## Best Practices

### Before Committing

```bash
make bench-vs-baseline  # Check for regressions
```

### For Pull Requests

```bash
make bench-snapshot     # Capture snapshot
# Include benchmarks/results/YYYY-MM-DD.txt in PR description
```

### For CI/CD

```bash
# Add to your CI pipeline
make install-benchstat
make bench-vs-baseline
```

### Benchmark Stability

For consistent results:
- Close unnecessary applications
- Run multiple times: `-benchtime=1s` or `-count=5`
- Use same hardware/environment for comparisons
- Disable CPU frequency scaling if possible:
  ```bash
  # Linux
  sudo cpupower frequency-set --governor performance
  ```

## Honest Benchmarking Philosophy

These benchmarks measure reality, not hide weaknesses:

- ✅ Stop when backpressure occurs (measure where system breaks)
- ✅ Verify work was done (action counters, not just queue insertion)
- ✅ Report actual events processed (not attempts)
- ✅ Zero allocations under load (no GC pressure)
- ✅ Predictable failure points (queue size determines capacity)

See [BENCHMARKS.md](../BENCHMARKS.md) in the project root for detailed performance analysis and benchmark results explanation.

## Troubleshooting

### "benchstat not found"

```bash
make install-benchstat
```

### "No baseline found"

```bash
make bench-baseline
```

### "Benchmarks failing"

Some benchmarks intentionally hit backpressure to measure system limits. This is expected behavior for honest benchmarking. See the benchmark output logs for "Stopped at backpressure" messages.

### Performance Varies

Benchmark results can vary based on:
- System load
- CPU frequency scaling
- Background processes
- Thermal throttling

Run benchmarks multiple times and look for consistent patterns rather than single-run results.
