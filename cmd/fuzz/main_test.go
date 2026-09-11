package main

import (
	"os"
	"testing"
)

func TestLiteIntegration(t *testing.T) {
	bin := os.Getenv("GOPIE_INTEGRATION_BIN")
	if bin == "" {
		t.Skip("set GOPIE_INTEGRATION_BIN to run the external fuzz integration test")
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("GOPIE_INTEGRATION_BIN: %v", err)
	}
	Lite(bin, "", "debug", 5, 50, 0, 2, "goroutine")
}
