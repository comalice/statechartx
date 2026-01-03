#!/bin/bash

# Quick profiling script for StatechartX
set -e

echo "========================================"
echo "StatechartX Quick Profiling"
echo "========================================"
echo ""

# Create output directory
OUTPUT_DIR="./profile_results"
mkdir -p "$OUTPUT_DIR"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RUN_DIR="$OUTPUT_DIR/run_$TIMESTAMP"
mkdir -p "$RUN_DIR"
echo "Results: $RUN_DIR"
echo ""

# CPU Profiling
echo "[1/5] CPU profiling..."
go test -run=XXX -bench=BenchmarkStateTransition -cpuprofile="$RUN_DIR/cpu.prof" -benchtime=1s > "$RUN_DIR/bench_cpu.txt" 2>&1
if [ -f "$RUN_DIR/cpu.prof" ]; then
    go tool pprof -text "$RUN_DIR/cpu.prof" > "$RUN_DIR/cpu_report.txt" 2>&1
    echo "  ✓ CPU profile complete"
fi

# Memory Profiling
echo "[2/5] Memory profiling..."
go test -run=XXX -bench=BenchmarkMemoryAllocation -memprofile="$RUN_DIR/mem.prof" -benchtime=1s > "$RUN_DIR/bench_mem.txt" 2>&1
if [ -f "$RUN_DIR/mem.prof" ]; then
    go tool pprof -text "$RUN_DIR/mem.prof" > "$RUN_DIR/mem_report.txt" 2>&1
    echo "  ✓ Memory profile complete"
fi

# Allocation Profiling
echo "[3/5] Allocation profiling..."
go test -run=XXX -bench=BenchmarkMemoryAllocation -benchmem -benchtime=1s > "$RUN_DIR/alloc_report.txt" 2>&1
echo "  ✓ Allocation report complete"

# Benchmarks
echo "[4/5] Running benchmarks..."
go test -run=XXX -bench=. -benchmem -benchtime=1s > "$RUN_DIR/benchmarks_full.txt" 2>&1
echo "  ✓ Benchmarks complete"

# Race Detection (quick test)
echo "[5/5] Race detection..."
go test -race -run=TestConcurrentStateMachines -timeout=5m > "$RUN_DIR/race_report.txt" 2>&1 || echo "  ⚠ Race detection completed with warnings"
echo "  ✓ Race detection complete"

# Generate Summary
echo ""
echo "Generating summary..."
SUMMARY_FILE="$RUN_DIR/SUMMARY.md"

cat > "$SUMMARY_FILE" << EOF
# StatechartX Performance Profile Summary

**Generated:** $(date)
**Run ID:** $TIMESTAMP

---

## CPU Profile Top Functions

\`\`\`
EOF

if [ -f "$RUN_DIR/cpu_report.txt" ]; then
    head -20 "$RUN_DIR/cpu_report.txt" >> "$SUMMARY_FILE"
fi

cat >> "$SUMMARY_FILE" << EOF
\`\`\`

---

## Memory Profile Top Allocations

\`\`\`
EOF

if [ -f "$RUN_DIR/mem_report.txt" ]; then
    head -20 "$RUN_DIR/mem_report.txt" >> "$SUMMARY_FILE"
fi

cat >> "$SUMMARY_FILE" << EOF
\`\`\`

---

## Benchmark Results

\`\`\`
EOF

if [ -f "$RUN_DIR/benchmarks_full.txt" ]; then
    grep "^Benchmark" "$RUN_DIR/benchmarks_full.txt" >> "$SUMMARY_FILE" || echo "No results" >> "$SUMMARY_FILE"
fi

cat >> "$SUMMARY_FILE" << EOF
\`\`\`

---

## Race Detection

\`\`\`
EOF

if [ -f "$RUN_DIR/race_report.txt" ]; then
    if grep -q "WARNING: DATA RACE" "$RUN_DIR/race_report.txt"; then
        echo "⚠️  DATA RACES DETECTED" >> "$SUMMARY_FILE"
        grep -A 10 "WARNING: DATA RACE" "$RUN_DIR/race_report.txt" | head -30 >> "$SUMMARY_FILE"
    else
        echo "✓ No data races detected" >> "$SUMMARY_FILE"
    fi
fi

cat >> "$SUMMARY_FILE" << EOF
\`\`\`

---

## Files Generated

EOF

ls -lh "$RUN_DIR" >> "$SUMMARY_FILE"

echo ""
echo "========================================"
echo "Profiling Complete!"
echo "========================================"
echo "Results: $RUN_DIR"
echo "Summary: $SUMMARY_FILE"
echo ""
