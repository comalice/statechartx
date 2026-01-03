#!/bin/bash

# Comprehensive profiling script for StatechartX
# This script runs CPU, memory, heap, and allocation profiling
# and generates reports for analysis

set -e

echo "========================================"
echo "StatechartX Comprehensive Profiling"
echo "========================================"
echo ""

# Create output directory
OUTPUT_DIR="./profile_results"
mkdir -p "$OUTPUT_DIR"
echo "Output directory: $OUTPUT_DIR"
echo ""

# Timestamp for this run
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RUN_DIR="$OUTPUT_DIR/run_$TIMESTAMP"
mkdir -p "$RUN_DIR"
echo "Results will be saved to: $RUN_DIR"
echo ""

# ========================================
# 1. CPU Profiling
# ========================================
echo "[1/7] Running CPU profiling..."
go test -run=XXX -bench=. -cpuprofile="$RUN_DIR/cpu.prof" -benchtime=5s > "$RUN_DIR/bench_cpu.txt" 2>&1
echo "  ✓ CPU profile saved to cpu.prof"

# Generate CPU profile report
if [ -f "$RUN_DIR/cpu.prof" ]; then
    go tool pprof -text "$RUN_DIR/cpu.prof" > "$RUN_DIR/cpu_report.txt" 2>&1
    go tool pprof -pdf "$RUN_DIR/cpu.prof" > "$RUN_DIR/cpu_graph.pdf" 2>&1 || echo "  (PDF generation skipped - graphviz not installed)"
    echo "  ✓ CPU report generated"
fi
echo ""

# ========================================
# 2. Memory Profiling
# ========================================
echo "[2/7] Running memory profiling..."
go test -run=XXX -bench=. -memprofile="$RUN_DIR/mem.prof" -benchtime=5s > "$RUN_DIR/bench_mem.txt" 2>&1
echo "  ✓ Memory profile saved to mem.prof"

# Generate memory profile report
if [ -f "$RUN_DIR/mem.prof" ]; then
    go tool pprof -text "$RUN_DIR/mem.prof" > "$RUN_DIR/mem_report.txt" 2>&1
    go tool pprof -pdf "$RUN_DIR/mem.prof" > "$RUN_DIR/mem_graph.pdf" 2>&1 || echo "  (PDF generation skipped - graphviz not installed)"
    echo "  ✓ Memory report generated"
fi
echo ""

# ========================================
# 3. Allocation Profiling
# ========================================
echo "[3/7] Running allocation profiling..."
go test -run=XXX -bench=BenchmarkMemoryAllocation -benchmem -benchtime=5s > "$RUN_DIR/alloc_report.txt" 2>&1
echo "  ✓ Allocation report saved"
echo ""

# ========================================
# 4. Heap Profiling (via memory profile)
# ========================================
echo "[4/7] Analyzing heap allocations..."
if [ -f "$RUN_DIR/mem.prof" ]; then
    go tool pprof -alloc_space -text "$RUN_DIR/mem.prof" > "$RUN_DIR/heap_alloc_space.txt" 2>&1
    go tool pprof -alloc_objects -text "$RUN_DIR/mem.prof" > "$RUN_DIR/heap_alloc_objects.txt" 2>&1
    echo "  ✓ Heap analysis complete"
fi
echo ""

# ========================================
# 5. Race Detection
# ========================================
echo "[5/7] Running race detection..."
go test -race -run=TestConcurrent -timeout=10m > "$RUN_DIR/race_report.txt" 2>&1 || echo "  ⚠ Race detection completed with warnings (see race_report.txt)"
echo "  ✓ Race detection complete"
echo ""

# ========================================
# 6. Benchmark Comparison
# ========================================
echo "[6/7] Running comprehensive benchmarks..."
go test -run=XXX -bench=. -benchmem -benchtime=3s > "$RUN_DIR/benchmarks_full.txt" 2>&1
echo "  ✓ Benchmarks complete"
echo ""

# ========================================
# 7. Generate Summary Report
# ========================================
echo "[7/7] Generating summary report..."

SUMMARY_FILE="$RUN_DIR/SUMMARY.md"

cat > "$SUMMARY_FILE" << EOF
# StatechartX Performance Profile Summary

**Generated:** $(date)
**Run ID:** $TIMESTAMP

---

## Quick Links

- [CPU Profile Report](cpu_report.txt)
- [Memory Profile Report](mem_report.txt)
- [Allocation Report](alloc_report.txt)
- [Heap Analysis - Space](heap_alloc_space.txt)
- [Heap Analysis - Objects](heap_alloc_objects.txt)
- [Race Detection Report](race_report.txt)
- [Full Benchmarks](benchmarks_full.txt)

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

## Benchmark Results Summary

\`\`\`
EOF

if [ -f "$RUN_DIR/benchmarks_full.txt" ]; then
    grep "^Benchmark" "$RUN_DIR/benchmarks_full.txt" >> "$SUMMARY_FILE" || echo "No benchmark results found" >> "$SUMMARY_FILE"
fi

cat >> "$SUMMARY_FILE" << EOF
\`\`\`

---

## Race Detection Results

\`\`\`
EOF

if [ -f "$RUN_DIR/race_report.txt" ]; then
    if grep -q "WARNING: DATA RACE" "$RUN_DIR/race_report.txt"; then
        echo "⚠️  DATA RACES DETECTED - See race_report.txt for details" >> "$SUMMARY_FILE"
        grep -A 10 "WARNING: DATA RACE" "$RUN_DIR/race_report.txt" | head -30 >> "$SUMMARY_FILE"
    else
        echo "✓ No data races detected" >> "$SUMMARY_FILE"
    fi
fi

cat >> "$SUMMARY_FILE" << EOF
\`\`\`

---

## Allocation Statistics

\`\`\`
EOF

if [ -f "$RUN_DIR/alloc_report.txt" ]; then
    grep -E "(Benchmark|allocs/op)" "$RUN_DIR/alloc_report.txt" >> "$SUMMARY_FILE" || echo "No allocation data" >> "$SUMMARY_FILE"
fi

cat >> "$SUMMARY_FILE" << EOF
\`\`\`

---

## Analysis Commands

To interactively explore profiles:

\`\`\`bash
# CPU profile
go tool pprof $RUN_DIR/cpu.prof

# Memory profile
go tool pprof $RUN_DIR/mem.prof

# Web interface (requires graphviz)
go tool pprof -http=:8080 $RUN_DIR/cpu.prof
\`\`\`

---

## Files Generated

EOF

ls -lh "$RUN_DIR" >> "$SUMMARY_FILE"

echo "  ✓ Summary report generated: $SUMMARY_FILE"
echo ""

# ========================================
# Completion
# ========================================
echo "========================================"
echo "Profiling Complete!"
echo "========================================"
echo ""
echo "Results location: $RUN_DIR"
echo "Summary report: $SUMMARY_FILE"
echo ""
echo "To view the summary:"
echo "  cat $SUMMARY_FILE"
echo ""
echo "To explore CPU profile interactively:"
echo "  go tool pprof $RUN_DIR/cpu.prof"
echo ""
echo "To explore memory profile interactively:"
echo "  go tool pprof $RUN_DIR/mem.prof"
echo ""
