package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestEmptyTest(t *testing.T) {
	// Test with no output
}

func TestUnicodeOutput(t *testing.T) {
	t.Log("Testing with unicode: 🚀 ✓ ✗ ⊝")
	t.Log("Chinese: 你好世界")
	t.Log("Emojis: 🎉 🔥 💯")

	text := "Hello, 世界! 🌍"
	if !utf8.ValidString(text) {
		t.Error("String should be valid UTF-8")
	}
}

func TestLongOutput(t *testing.T) {
	longString := strings.Repeat("A", 1000)
	t.Logf("Long string length: %d", len(longString))
	t.Log("First part: " + longString[:50])
	t.Log("Last part: " + longString[len(longString)-50:])
}

func TestMultilineOutput(t *testing.T) {
	output := `This is a multiline
output that spans
several lines and should
be handled correctly
by the formatter`

	t.Log(output)

	lines := strings.Split(output, "\n")
	if len(lines) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(lines))
	}
}

func TestSpecialCharacters(t *testing.T) {
	t.Log("Testing special chars: !@#$%^&*()_+-=[]{}|;:,.<>?")
	t.Log("Quotes: \"single\" and 'double'")
	t.Log("Backslashes: \\n \\t \\r \\\\")

	// This will fail to show special characters in error output
	t.Error("Error with special chars: \n\t<>&\"'\\")
}
