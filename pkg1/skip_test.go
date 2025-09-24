package pkg1

import (
	"runtime"
	"testing"
)

func TestSkippedTest(t *testing.T) {
	t.Skip("This test is intentionally skipped")
}

func TestSkippedConditional(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}
	// This will skip on Windows, pass on others
}

func TestSkippedWithReason(t *testing.T) {
	t.Log("Checking prerequisites...")
	t.Skip("Feature not implemented yet - TODO: implement feature X")
}
