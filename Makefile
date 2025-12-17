# Helper Makefile for Go projects
# Aligned with the golang-development Claude Code skill best practices

.PHONY: help format lint vet staticcheck test test-race bench fuzz coverage clean

# Default target
help:
	@echo "Go Development Helpers (aligned with golang-development skill)"
	@echo ""
	@echo "format       - Run gofmt/goimports on all .go files"
	@echo "lint         - Run revive (modern golint replacement)"
	@echo "vet          - Run go vet"
	@echo "staticcheck  - Run staticcheck (honorable linter)"
	@echo "test         - Run tests"
	@echo "test-race    - Run tests with race detector"
	@echo "bench        - Run benchmarks (with memory stats)"
	@echo "bench-cmp    - Compare benchmarks (requires old.txt/new.txt)"
	@echo "fuzz         - Run all fuzz tests for 30s each"
	@echo "coverage     - Generate test coverage report"
	@echo "clean        - Remove build artifacts and coverage files"
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
	go test -race -count=1 ./...

# Benchmarking
bench:
	go test -bench=. -benchmem -run=^$$ ./...

bench-cmp:
	@if [ ! -f old.txt ] || [ ! -f new.txt ]; then \
		echo "Error: Need old.txt and new.txt for comparison"; \
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
