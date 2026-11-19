package main

import (
	"bytes"
	"testing"
)

func TestRun(t *testing.T) {
	var buf bytes.Buffer
	if err := run(&buf); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if got, want := buf.String(), "payments-ledger\n"; got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}
