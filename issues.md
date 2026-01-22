# Known Issues (as of 2026-01-18)

## Post-Alpha Improvements

1. **Realtime Test Coverage Enhancement**
   - ParallelStateHooks designed to allow realtime runtime to run same test suite as event-driven
   - Current: Event-driven has 175 SCXML tests, realtime only has 3 adapted tests
   - Action: Create test adapter using testutil/ pattern to run ALL SCXML tests against both runtimes
   - Impact: Should dramatically increase realtime coverage from 27.2%

2. **Documentation Reorganization**
   - docs/ folder well-organized, but some archive content could be better summarized
   - Action: Review docs/archive/ and add summary/index

3. **SCXML Conformance Gap**
   - 175/206 W3C tests implemented (85%)
   - Remaining 31 tests likely cover advanced features (invoke, datamodel, expressions)
   - Action: Crosswalk remaining tests with current implementation

4. **Example Enhancement**
   - examples/basic works but could be expanded
   - Action: Add more comprehensive examples

## Non-Blocking Issues

### Test Infrastructure

1. **Race conditions in test code** (NOT production code)
   - Multiple tests have data races when run with `-race` flag
   - Pattern: Tests access shared variables from action callbacks without synchronization
   - Examples: TestDoneEventWithData (line 306, 329), TestHistoryWithParallelStates, etc.
   - **Impact: Test infrastructure only** - production code passes race detection
   - Fix: Add sync.Mutex or atomic operations to test shared variables
   - Priority: Post-alpha (tests functionally work without -race flag)

2. **Memory calculation in stress tests** (statechart_stress_test.go:429)
   - Produces absurd values: 17592186044411.57 MB (17TB per machine)
   - Likely integer overflow or unit conversion bug in TEST CODE ONLY
   - Production code unaffected - not a blocker

3. **Stress test timeouts with -race**
   - TestMaxStates times out under race detector (>30s timeout)
   - Breaking point tests not designed for race detector overhead
   - Fix: Skip these tests or increase timeout when running with -race

### Resolved (Do Not Re-Open)
- ~~TestMemoryPressure~~ - PASSES (was stale info)
- ~~TestDeepVsShallowHistory~~ - PASSES
- ~~TestSCXML404_Realtime~~ - PASSES
- ~~Example vet warnings~~ - FIXED (removed redundant newlines)
- ~~DEBUG comments~~ - CLEANED UP (lines 1278, 1647, realtime/parallel.go:152)