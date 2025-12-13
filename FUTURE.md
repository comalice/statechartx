# Future Development Plan

## 1. Commit Benchmarks Changes - DONE
```
git add benchmarks/*
git commit -m \"bench: add helpers + update memory/throughput/transition benches

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude &lt;noreply@anthropic.com&gt;\"
```

## 2. Add YAML Persister Integration Tests
- Create `internal/production/persister_test.go`
- Test serialize → hydrate → Send events → verify state/Current()
- Use examples configs for hierarchical/parallel/history.

## 3. Enhance Visualizer CLI
- Update `cmd/demo/main.go` or new `cmd/viz/`
- Load YAML config → generate DOT/SVG via existing Visualizer.
- Usage: `statechart viz config.yaml -o out.svg`

## 4. Run & Profile Benchmarks
- `go test -bench=. ./benchmarks/...`
- Profile CPU/memory for deep/wide cases if slow.
- Add badges to README (e.g., throughput events/sec).

## 5. Update Docs
- README.md: Add features list, benchmark results table, quickstart.
- Expand docs/ARCHITECTURE.md with persistence flow diagram.
