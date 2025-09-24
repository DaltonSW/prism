package pkg1

import (
	"fmt"
	"testing"
)

func TestSimpleFail(t *testing.T) {
	t.Error("This test intentionally fails")
}

func TestFailWithOutput(t *testing.T) {
	t.Log("Setting up test...")
	t.Log("Performing operation...")

	expected := 42
	actual := 24

	t.Logf("Expected: %d", expected)
	t.Logf("Actual: %d", actual)
	t.Errorf("Values don't match: expected %d, got %d", expected, actual)
}

func TestFailWithPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic recovered: %v", r)
		}
	}()

	t.Log("About to panic...")
	panic("intentional panic for testing")
}

func TestMultipleErrors(t *testing.T) {
	t.Log("Running multiple assertions...")

	if 1+1 != 3 {
		t.Error("Math is broken: 1+1 should equal 3")
	}

	if "hello" != "world" {
		t.Error("String comparison failed")
	}

	fmt.Println("This goes to stdout")
	t.Log("This goes to test output")

	t.Fatal("This is a fatal error")
	t.Log("This won't be reached")
}
