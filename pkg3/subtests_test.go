package main

import (
	"testing"
)

func TestSubtests(t *testing.T) {
	testCases := []struct {
		name       string
		input      int
		expected   int
		shouldFail bool
	}{
		{"positive", 5, 25, false},
		{"zero", 0, 0, false},
		{"negative", -3, 9, false},
		{"failing_case", 4, 15, true}, // This will fail: 4*4 != 15
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing input: %d", tc.input)

			result := tc.input * tc.input
			t.Logf("Result: %d", result)

			if result != tc.expected {
				if !tc.shouldFail {
					t.Errorf("Expected %d, got %d", tc.expected, result)
				}
			}
		})
	}
}

func TestNestedSubtests(t *testing.T) {
	t.Run("group1", func(t *testing.T) {
		t.Run("pass", func(t *testing.T) {
			t.Log("Nested test that passes")
		})

		t.Run("fail", func(t *testing.T) {
			t.Log("Nested test that fails")
			t.Error("This nested test fails")
		})
	})

	t.Run("group2", func(t *testing.T) {
		t.Run("skip", func(t *testing.T) {
			t.Skip("Nested test that skips")
		})
	})
}
