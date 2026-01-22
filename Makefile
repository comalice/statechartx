# Helper Makefile for Go projects
# Aligned with the golang-development Claude Code skill best practices

.PHONY: help format lint vet staticcheck test test-race bench fuzz coverage clean

# Default target
help:
	@echo "Go Development Helpers (aligned with golang-development skill)"
	@echo ""
	@echo "format              - Run gofmt/goimports on all .go files"
	@echo "lint                - Run revive (modern golint replacement)"
	@echo "vet                 - Run go vet"
	@echo "staticcheck         - Run staticcheck (honorable linter)"
	@echo "test                - Run tests"
	@echo "test-race           - Run tests with race detector"
	@echo "bench               - Run benchmarks (with memory stats)"
	@echo "install-benchstat   - Install benchstat tool for comparisons"
	@echo "bench-baseline      - Capture baseline benchmark results"
	@echo "bench-vs-baseline   - Compare current benchmarks against baseline"
	@echo "bench-snapshot      - Capture dated snapshot for documentation"
	@echo "bench-cmp           - Compare old.txt vs new.txt manually"
	@echo "fuzz                - Run all fuzz tests for 30s each"
	@echo "coverage            - Generate test coverage report"
	@echo "clean               - Remove build artifacts and coverage files"
	@echo ""

# Formatting
format:
	go fmt ./...
	goimports -w .

# Linting & static analysis
lint:
	revive -config .revive.toml -formatter friendly ./... || true

vet:
	go vet ./...

staticcheck:
	staticcheck ./...

# Testing
test:
	go test -short ./...

test-race:
	go test -race -count=1 -timeout=30s ./...

# Benchmarking
bench:
	go test -bench=. -benchmem -run=^$$ ./...

# Install benchstat if not present
install-benchstat:
	@which benchstat > /dev/null || (echo "Installing benchstat..." && go install golang.org/x/perf/cmd/benchstat@latest)

# Capture baseline benchmark results
bench-baseline:
	@echo "Capturing baseline benchmark results..."
	go test -bench=. -benchmem -benchtime=1s ./benchmarks ./internal/core > benchmarks/results/baseline.txt 2>&1
	@echo "Baseline saved to benchmarks/results/baseline.txt"

# Compare current benchmarks against baseline
bench-vs-baseline: install-benchstat
	@echo "Running current benchmarks..."
	@go test -bench=. -benchmem -benchtime=1s ./benchmarks ./internal/core > new.txt 2>&1
	@if [ ! -f benchmarks/results/baseline.txt ]; then \
		echo "Error: No baseline found. Run 'make bench-baseline' first."; \
		exit 1; \
	fi
	@echo "Comparing against baseline..."
	@benchstat benchmarks/results/baseline.txt new.txt

# Capture dated snapshot (for PR/issue discussion)
bench-snapshot:
	@DATE=$$(date +%Y-%m-%d); \
	echo "Capturing snapshot for $$DATE..."; \
	go test -bench=. -benchmem -benchtime=1s ./benchmarks ./internal/core > benchmarks/results/$$DATE.txt 2>&1; \
	echo "Snapshot saved to benchmarks/results/$$DATE.txt"

# Compare old.txt vs new.txt
bench-cmp: install-benchstat
	@if [ ! -f old.txt ] || [ ! -f new.txt ]; then \
		echo "Error: Need old.txt and new.txt for comparison"; \
		echo "Tip: Use 'make bench-vs-baseline' to compare against baseline"; \
		exit 1; \
	fi
	benchstat old.txt new.txt

# Fuzzing
fuzz:
	go test -fuzz=. -fuzztime=30s ./...

# Coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Clean
clean:
	rm -f coverage.out coverage.html *.prof *.txt
	go clean

# All-in-one quality gate (recommended for CI or pre-commit)
check: format vet staticcheck lint test-race
	@echo "All checks passed!"
