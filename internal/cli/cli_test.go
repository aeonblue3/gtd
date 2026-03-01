package cli

import "testing"

func TestPrintCommandUsageKnown(t *testing.T) {
	if err := printCommandUsage("add"); err != nil {
		t.Fatalf("expected known help topic to succeed: %v", err)
	}
}

func TestPrintCommandUsageUnknown(t *testing.T) {
	if err := printCommandUsage("missing-topic"); err == nil {
		t.Fatal("expected unknown help topic to fail")
	}
}
