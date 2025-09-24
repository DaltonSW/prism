package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestWithBenchmark(t *testing.T) {
	// This test passes but shows we can handle different test types
	result := strings.Repeat("a", 1000)
	if len(result) != 1000 {
		t.Errorf("Expected length 1000, got %d", len(result))
	}
}

func BenchmarkStringConcat(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("test%d", i)
	}
}

func BenchmarkStringBuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var sb strings.Builder
		sb.WriteString("test")
		sb.WriteString(fmt.Sprintf("%d", i))
		_ = sb.String()
	}
}
