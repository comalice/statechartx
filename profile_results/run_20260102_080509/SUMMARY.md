# StatechartX Performance Profile Summary

**Generated:** Fri Jan  2 08:05:45 UTC 2026
**Run ID:** 20260102_080509

---

## CPU Profile Top Functions

```
File: statechartx.test
Type: cpu
Time: Jan 2, 2026 at 8:05am (UTC)
Duration: 1.80s, Total samples = 2.15s (119.39%)
Showing nodes accounting for 1.93s, 89.77% of 2.15s total
Dropped 65 nodes (cum <= 0.01s)
      flat  flat%   sum%        cum   cum%
     0.26s 12.09% 12.09%      0.53s 24.65%  runtime.mallocgc
     0.21s  9.77% 21.86%      0.40s 18.60%  runtime.selectgo
     0.14s  6.51% 28.37%      0.15s  6.98%  runtime.mapaccess1_fast64
     0.12s  5.58% 33.95%      0.12s  5.58%  runtime.nextFreeFast (inline)
     0.08s  3.72% 37.67%      0.30s 13.95%  runtime.growslice
     0.08s  3.72% 41.40%      0.15s  6.98%  runtime.mapassign_fast64
     0.07s  3.26% 44.65%      0.07s  3.26%  runtime.futex
     0.07s  3.26% 47.91%      0.07s  3.26%  sync.(*RWMutex).Lock
     0.06s  2.79% 50.70%      0.51s 23.72%  github.com/comalice/statechartx.(*Runtime).computeLCA
     0.06s  2.79% 53.49%      0.35s 16.28%  github.com/comalice/statechartx.(*Runtime).getAncestors (inline)
     0.06s  2.79% 56.28%      0.07s  3.26%  runtime.heapBitsSetType
     0.06s  2.79% 59.07%      0.06s  2.79%  runtime.memhash64
     0.06s  2.79% 61.86%      0.06s  2.79%  runtime.procyield
```

---

## Memory Profile Top Allocations

```
File: statechartx.test
Type: alloc_space
Time: Jan 2, 2026 at 8:05am (UTC)
Showing nodes accounting for 1893.12MB, 99.37% of 1905.12MB total
Dropped 18 nodes (cum <= 9.53MB)
      flat  flat%   sum%        cum   cum%
 1342.55MB 70.47% 70.47%  1342.55MB 70.47%  github.com/comalice/statechartx.NewRuntime (inline)
  199.52MB 10.47% 80.94%  1789.60MB 93.94%  github.com/comalice/statechartx.BenchmarkMemoryAllocation
   80.01MB  4.20% 85.14%   125.52MB  6.59%  github.com/comalice/statechartx.NewMachine
   67.02MB  3.52% 88.66%    73.52MB  3.86%  github.com/comalice/statechartx.(*Runtime).recordHistory
   45.51MB  2.39% 91.05%    45.51MB  2.39%  github.com/comalice/statechartx.NewMachine.func1
   38.50MB  2.02% 93.07%    38.50MB  2.02%  github.com/comalice/statechartx.(*Runtime).pickTransition
      37MB  1.94% 95.01%       37MB  1.94%  context.(*cancelCtx).Done
      33MB  1.73% 96.75%       33MB  1.73%  context.WithCancel
   15.50MB  0.81% 97.56%    48.50MB  2.55%  github.com/comalice/statechartx.(*Runtime).processMicrosteps
      11MB  0.58% 98.14%       46MB  2.41%  github.com/comalice/statechartx.(*Runtime).enterInitialState
   10.50MB  0.55% 98.69%    10.50MB  0.55%  github.com/comalice/statechartx.(*Runtime).getAncestors (inline)
       7MB  0.37% 99.06%   112.02MB  5.88%  github.com/comalice/statechartx.(*Runtime).processEvent
       6MB  0.31% 99.37%       85MB  4.46%  github.com/comalice/statechartx.(*Runtime).Start
         0     0% 99.37%       37MB  1.94%  github.com/comalice/statechartx.(*Runtime).SendEvent
```

---

## Benchmark Results

```
BenchmarkStateTransition-8          	 2354538	       518.2 ns/op	     248 B/op	      12 allocs/op
BenchmarkEventSending-8             	 5441018	       216.7 ns/op	      96 B/op	       2 allocs/op
BenchmarkLCAComputation-8           	31301630	        38.15 ns/op	       0 B/op	       0 allocs/op
BenchmarkLCAComputationDeep-8       	  250177	      4580 ns/op	    3219 B/op	       9 allocs/op
BenchmarkParallelRegionSpawn-8      	   61515	     19739 ns/op	   15332 B/op	     120 allocs/op
BenchmarkParallelRegionSpawn100-8   	    7880	    148525 ns/op	  113753 B/op	     961 allocs/op
BenchmarkEventRouting-8             	  384753	      3053 ns/op	    1583 B/op	      44 allocs/op
BenchmarkHistoryRestoration-8       	 2417528	       495.4 ns/op	     255 B/op	       7 allocs/op
BenchmarkStateCreation-8            	  139790	      8546 ns/op	   18344 B/op	     111 allocs/op
BenchmarkTransitionCreation-8       	  181093	      7904 ns/op	    9327 B/op	      99 allocs/op
BenchmarkComplexStatechart-8        	   52029	     22921 ns/op	   10054 B/op	     174 allocs/op
BenchmarkMemoryAllocation-8         	  385202	      3033 ns/op	    5388 B/op	      34 allocs/op
```

---

## Race Detection

```
✓ No data races detected
```

---

## Files Generated

total 52K
-rw-r--r-- 1 ubuntu ubuntu 4.2K Jan  2 08:05 SUMMARY.md
-rw-r--r-- 1 ubuntu ubuntu  221 Jan  2 08:05 alloc_report.txt
-rw-r--r-- 1 ubuntu ubuntu  189 Jan  2 08:05 bench_cpu.txt
-rw-r--r-- 1 ubuntu ubuntu  221 Jan  2 08:05 bench_mem.txt
-rw-r--r-- 1 ubuntu ubuntu 1.3K Jan  2 08:05 benchmarks_full.txt
-rw-r--r-- 1 ubuntu ubuntu 8.0K Jan  2 08:05 cpu.prof
-rw-r--r-- 1 ubuntu ubuntu 5.8K Jan  2 08:05 cpu_report.txt
-rw-r--r-- 1 ubuntu ubuntu 2.4K Jan  2 08:05 mem.prof
-rw-r--r-- 1 ubuntu ubuntu 2.1K Jan  2 08:05 mem_report.txt
-rw-r--r-- 1 ubuntu ubuntu   49 Jan  2 08:05 race_report.txt
