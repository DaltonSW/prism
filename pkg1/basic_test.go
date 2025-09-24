package pkg1

import (
	"testing"
	"time"
)

func TestPassing(t *testing.T) {
	// Simple passing test
}

func TestPassingWithOutput(t *testing.T) {
	t.Log("This is some log output")
	t.Log("Multiple lines of output")
	// This will pass
}

func TestQuickPass(t *testing.T) {
	// Very fast test
}

func TestSlowPass(t *testing.T) {
	time.Sleep(100 * time.Millisecond)
	// Slower test to show duration differences
}
