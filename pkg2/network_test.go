package main

import (
	"net/http"
	"testing"
	"time"
)

func TestHTTPRequest(t *testing.T) {
	t.Log("Testing HTTP functionality")

	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get("https://httpbin.org/status/200")
	if err != nil {
		t.Skip("Network unavailable, skipping HTTP test")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestNetworkFailure(t *testing.T) {
	t.Log("Testing network failure handling")

	client := &http.Client{Timeout: 1 * time.Millisecond}
	_, err := client.Get("https://httpbin.org/delay/1")

	if err == nil {
		t.Error("Expected timeout error, but request succeeded")
	}

	t.Logf("Got expected error: %v", err)
}

func TestLongRunning(t *testing.T) {
	t.Log("Starting long-running test...")

	for i := 0; i < 5; i++ {
		time.Sleep(50 * time.Millisecond)
		t.Logf("Progress: %d/5", i+1)
	}

	t.Log("Long-running test completed")
}
