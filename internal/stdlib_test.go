package stdlib_test

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

func TestStdlibOnlyCore(t *testing.T) {
	goModPath := "go.mod"
	f, err := os.Open(goModPath)
	if err != nil {
		t.Fatalf("Failed to open go.mod: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	hasNonStdlib := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "require ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				mod := parts[1]
				// Any require directive indicates non-stdlib dependency
				// (stdlib modules are not listed in go.mod requires)
				t.Errorf("Non-stdlib dependency found in go.mod: %s", mod)
				hasNonStdlib = true
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading go.mod: %v", err)
	}
	if hasNonStdlib {
		t.Error("Core engine must be stdlib-only (no require directives in go.mod)")
	}
}