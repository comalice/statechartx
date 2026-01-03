# SCXML Conformance Testing

StatechartX includes extensive conformance testing against the W3C SCXML (State Chart XML) IRP test suite to ensure standards compliance and correctness.

## Test Suite Overview

The project uses the official W3C SCXML Interpretation and Reporting Protocol (IRP) test suite to validate state machine semantics against the SCXML standard.

### Test Suite Location

```
test/scxml/w3c_test_suite/
├── manifest.xml          - Test suite manifest
├── 144/                  - Test case directories (by number)
├── 147/
├── 148/
...
└── 580/
```

**Total Test Cases**: 212 SCXML test files

## Downloading the Test Suite

Use the included downloader tool:

```bash
# Download to default location (test/scxml/w3c_test_suite/)
go run cmd/scxml_downloader/main.go

# Force re-download
go run cmd/scxml_downloader/main.go -f
```

The downloader:
- Fetches tests from the official W3C repository
- Includes automatic retry with exponential backoff
- Validates downloads and creates manifest
- Organizes tests by test number

## Test Organization

SCXML conformance tests are translated to Go and organized in root-level test files grouped by test number ranges:

```
statechart_scxml_100-199_test.go  - Tests 100-199
statechart_scxml_200-299_test.go  - Tests 200-299
statechart_scxml_300-399_test.go  - Tests 300-399
statechart_scxml_400-499_test.go  - Tests 400-499
statechart_scxml_500-599_test.go  - Tests 500-599
```

This organization:
- Keeps individual files manageable (< 2000 lines)
- Enables selective test execution
- Provides clear test categorization
- Simplifies maintenance and debugging

## SCXML to Go Translation

The project includes a custom Claude Code skill (`scxml-translator`) that automates translation of SCXML test cases to Go unit tests.

### Translation Mapping

| SCXML Element | StatechartX Equivalent |
|---------------|------------------------|
| `<state id="s1">` | `&State{ID: StateID(hashString("s1"))}` |
| `<transition event="e" target="s2"/>` | `Transitions: []*Transition{{Event: EventID(hashString("e")), Target: StateID(hashString("s2"))}}` |
| `<onentry><raise event="foo"/></onentry>` | `EntryAction: func(ctx, evt, from, to) error { return rt.SendEvent(ctx, Event{ID: EventID(hashString("foo"))}) }` |
| `initial="s0"` attribute | `Initial: StateID(hashString("s0"))` |
| `<history type="shallow"/>` | `IsHistoryState: true, HistoryType: HistoryShallow` |
| `<parallel>` | `IsParallel: true` |
| `conf:pass` final state | Assert `rt.IsInState(StateID(hashString("pass")))` |

### State ID Hashing

SCXML uses string-based state IDs, while StatechartX uses numeric IDs. Translation uses a hash function:

```go
func hashString(s string) int {
    h := fnv.New32a()
    h.Write([]byte(s))
    return int(h.Sum32())
}
```

This provides:
- Deterministic ID generation
- Collision resistance for test-sized state spaces
- Simple mapping from SCXML strings to Go ints

## Running Conformance Tests

### Run All SCXML Tests

```bash
go test -v -run "SCXML"
```

### Run Specific Test Range

```bash
go test -v -run "SCXML_1"     # Tests 100-199
go test -v -run "SCXML_2"     # Tests 200-299
go test -v -run "SCXML_3"     # Tests 300-399
go test -v -run "SCXML_4"     # Tests 400-499
go test -v -run "SCXML_5"     # Tests 500-599
```

### Run Individual Test

```bash
go test -v -run "Test144"     # Run specific test by number
```

## Test Structure

Each translated test follows this pattern:

```go
func Test144(t *testing.T) {
    // 1. Build state tree from SCXML
    root := &State{ID: StateID(hashString("ScxmlRoot"))}
    s1 := &State{ID: StateID(hashString("s1")), Parent: root}
    pass := &State{ID: StateID(hashString("pass")), Parent: root, IsFinal: true}
    fail := &State{ID: StateID(hashString("fail")), Parent: root, IsFinal: true}

    // 2. Set up transitions
    s1.Transitions = []*Transition{
        {Event: EventID(hashString("e")), Target: StateID(hashString("pass"))},
        // ...
    }

    // 3. Create and start runtime
    machine, err := NewMachine(root)
    if err != nil {
        t.Fatal(err)
    }
    rt := NewRuntime(machine, nil)
    ctx := context.Background()
    rt.Start(ctx)
    defer rt.Stop()

    // 4. Send test events
    rt.SendEvent(ctx, Event{ID: EventID(hashString("e"))})

    // 5. Assert final state
    time.Sleep(10 * time.Millisecond)  // Allow processing
    if !rt.IsInState(StateID(hashString("pass"))) {
        t.Fatal("Expected to be in 'pass' state")
    }
}
```

## Translation Limitations

The SCXML test translation has some known limitations:

### Not Supported

1. **Datamodel/ECMAScript Expressions**
   - SCXML supports `<datamodel>` with ECMAScript expressions
   - StatechartX uses Go functions for guards/actions
   - Guards may be stubbed where datamodel is required

2. **External Communication**
   - `<invoke>` - External service invocation
   - `<send>` - External event sending
   - Only internal event raising (`<raise>`) is supported

3. **Advanced SCXML Features**
   - `<foreach>` - Iteration over data
   - `<assign>` - Variable assignment
   - `<script>` - Embedded scripting

### Workarounds

- **Guards**: Use Go functions instead of SCXML conditional expressions
- **Actions**: Implement in Go instead of SCXML `<script>` blocks
- **Data**: Pass context objects instead of SCXML `<datamodel>`

## Custom Skill: scxml-translator

The `.claude/skills/scxml-translator/` skill automates SCXML test translation.

### Usage

The skill is automatically available in Claude Code:

1. Provide an SCXML test file path
2. Skill analyzes the test structure
3. Generates equivalent Go test code
4. Places in appropriate `statechart_scxml_*_test.go` file

### Skill Capabilities

- Parses SCXML XML structure
- Maps states, transitions, and attributes to Go
- Generates table-driven test functions
- Handles history states and parallel regions
- Creates proper assertions for final states

See `.claude/skills/scxml-translator/SKILL.md` for details.

## Conformance Coverage

StatechartX implements the core SCXML semantics:

✅ **Supported**:
- Hierarchical states
- Compound states with initial states
- History states (shallow and deep)
- Parallel states
- Guarded transitions
- Entry/exit actions
- Done events
- Eventless (automatic) transitions
- Internal transitions
- LCA-based transition semantics

❌ **Not Implemented**:
- Full datamodel (use Go context instead)
- External communications (`<invoke>`, `<send>`)
- Scripting (`<script>`)
- Data manipulation (`<assign>`, `<foreach>`)

## Contributing Test Translations

When adding new SCXML test translations:

1. Use the `scxml-translator` skill when available
2. Follow the established test structure pattern
3. Group tests in the appropriate `statechart_scxml_*_test.go` file
4. Add hash function usage for string-to-ID conversion
5. Include proper assertions for final states
6. Document any translation workarounds

## References

- [W3C SCXML Specification](https://www.w3.org/TR/scxml/)
- [SCXML IRP Test Suite](https://www.w3.org/Voice/2013/scxml-irp/)
- Custom Skill: `.claude/skills/scxml-translator/SKILL.md`
- Test Suite Location: `test/scxml/w3c_test_suite/`
