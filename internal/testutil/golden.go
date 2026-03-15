package testutil

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Update is a flag that, when set, causes golden file tests to regenerate their expected output.
// Usage: go test ./internal/vatxml/ -update
var Update = flag.Bool("update", false, "update golden files")

// AssertGolden compares actual output against a golden file.
// If the -update flag is set, the golden file is (re)written with the actual output.
// If the golden file does not exist and -update is not set, the test fails.
func AssertGolden(t *testing.T, goldenPath string, actual []byte) {
	t.Helper()

	if *Update {
		dir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("creating golden file directory %s: %v", dir, err)
		}
		if err := os.WriteFile(goldenPath, actual, 0o644); err != nil {
			t.Fatalf("writing golden file %s: %v", goldenPath, err)
		}
		t.Logf("updated golden file: %s", goldenPath)
		return
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("golden file %s not found (run with -update to create): %v", goldenPath, err)
	}

	if !bytes.Equal(expected, actual) {
		diff := unifiedDiff(string(expected), string(actual))
		t.Errorf("output differs from golden file %s:\n%s", goldenPath, diff)
	}
}

// unifiedDiff produces a simple line-by-line diff between two strings.
func unifiedDiff(expected, actual string) string {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	var b strings.Builder
	b.WriteString("--- expected\n+++ actual\n")

	maxLen := len(expectedLines)
	if len(actualLines) > maxLen {
		maxLen = len(actualLines)
	}

	for i := 0; i < maxLen; i++ {
		var expLine, actLine string
		if i < len(expectedLines) {
			expLine = expectedLines[i]
		}
		if i < len(actualLines) {
			actLine = actualLines[i]
		}

		if expLine != actLine {
			if i < len(expectedLines) {
				fmt.Fprintf(&b, "-%s\n", expLine)
			}
			if i < len(actualLines) {
				fmt.Fprintf(&b, "+%s\n", actLine)
			}
		}
	}

	return b.String()
}
