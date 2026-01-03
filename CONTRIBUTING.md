# Contributing to StatechartX

Thank you for your interest in contributing to StatechartX! This document provides guidelines and instructions for contributing to the project.

## Code Style

### Go Standards

- Follow standard Go conventions and idioms
- Run `gofmt` and `goimports` on all code (use `make format`)
- Keep functions focused and modular
- Use meaningful variable and function names
- Add comments for exported types and functions (godoc format)

### Project-Specific Guidelines

- State IDs are numeric (`StateID` and `EventID` are `int` types)
- Prefer explicit error handling over panics
- Keep core `statechart.go` minimal and focused
- Use subpackages for additional functionality (e.g., `realtime/`)

## Testing Requirements

### Mandatory Tests

All contributions must include appropriate tests:

```bash
# Run all tests
make test

# CRITICAL: Run with race detector for parallel state work
make test-race

# Run specific test
go test -v -run TestName
```

### Test Coverage

- Maintain or improve test coverage
- Add tests for new features
- Include both positive and negative test cases
- Test edge cases and error conditions

### Parallel State Testing

**CRITICAL**: Always run `make test-race` when working with parallel states. Race conditions can be subtle and only appear under concurrent load.

```bash
# This is MANDATORY for parallel state changes
make test-race
```

## Development Workflow

### 1. Set Up Your Environment

```bash
# Clone the repository
git clone https://github.com/comalice/statechartx.git
cd statechartx

# Build the project
go build ./...

# Run tests
make test
```

### 2. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
```

### 3. Make Your Changes

- Write clear, focused commits
- Follow the code style guidelines
- Add tests for new functionality
- Update documentation as needed

### 4. Run the Full Test Suite

```bash
# Run all validation checks
make check  # format, vet, staticcheck, lint, test-race

# Run benchmarks to check for regressions
make bench

# Run stress tests (optional, for significant changes)
go test -v -run "TestMillion|TestMassive"
```

### 5. Submit a Pull Request

- Provide a clear description of the changes
- Reference any related issues
- Ensure all CI checks pass
- Be responsive to code review feedback

## Pull Request Guidelines

### PR Description Template

```markdown
## Description
Brief description of what this PR does

## Motivation
Why is this change needed? What problem does it solve?

## Changes
- List key changes
- Include any breaking changes

## Testing
- What tests were added/modified?
- How was this tested?

## Checklist
- [ ] Tests pass (`make test`)
- [ ] Race detector passes (`make test-race`)
- [ ] Benchmarks show no regression
- [ ] Documentation updated
- [ ] Code follows style guidelines
```

### Review Process

1. Automated CI checks must pass
2. At least one maintainer review required
3. All comments must be addressed
4. Squash commits before merging (if requested)

## Performance Considerations

### Benchmarking

Run benchmarks to ensure no performance regressions:

```bash
make bench

# Compare before/after
go test -bench=. -benchmem -benchtime=3s > new.txt
# Compare with baseline
```

### Performance Targets

Maintain these performance targets:

- State transition: < 1 μs
- Event sending: < 500 ns
- LCA computation: < 100 ns
- Event throughput: > 10K/sec

## Documentation

### Required Documentation

- Update CLAUDE.md for significant architectural changes
- Update README.md for new features
- Add/update package-level documentation (doc.go)
- Include code examples for new features
- Update relevant docs in docs/ directory

### Documentation Style

- Use clear, concise language
- Include code examples
- Document edge cases and limitations
- Explain *why*, not just *what*

## Issue Reporting

### Bug Reports

Include:

- Go version
- Operating system
- Minimal reproduction case
- Expected vs actual behavior
- Stack trace (if applicable)

### Feature Requests

Include:

- Use case description
- Proposed API (if applicable)
- Examples of how it would be used
- Alternative approaches considered

## Code Review Guidelines

### For Contributors

- Be open to feedback
- Respond to comments promptly
- Ask questions if anything is unclear
- Be patient - maintainers are volunteers

### For Reviewers

- Be respectful and constructive
- Focus on the code, not the person
- Explain the "why" behind suggestions
- Approve when ready, request changes if needed

## Community Guidelines

- Be respectful and inclusive
- Help others learn and grow
- Share knowledge and best practices
- Follow the project's Code of Conduct

## Getting Help

- Check existing documentation
- Search existing issues
- Ask questions in issue discussions
- Join community discussions

## Development Commands Reference

```bash
# Building
go build ./...

# Testing
make test           # Run all tests
make test-race      # Run with race detector
make coverage       # Generate coverage report

# Code Quality
make lint           # Run revive linter
make vet            # Run go vet
make staticcheck    # Run staticcheck
make format         # Format code
make check          # All-in-one validation

# Performance
make bench          # Run benchmarks
make fuzz           # Run fuzz tests

# Profiling
scripts/profile_quick.sh   # Quick profiling
scripts/profile_all.sh     # Comprehensive profiling
```

## License

By contributing to StatechartX, you agree that your contributions will be licensed under the MIT License.
