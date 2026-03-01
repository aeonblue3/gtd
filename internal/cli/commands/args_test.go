package commands

import "testing"

func TestSplitLeadingPositional(t *testing.T) {
	pos, rest := SplitLeadingPositional([]string{"abc", "--flag", "x"})
	if pos != "abc" {
		t.Fatalf("expected positional 'abc', got %q", pos)
	}
	if len(rest) != 2 || rest[0] != "--flag" {
		t.Fatalf("unexpected remainder: %#v", rest)
	}
}

func TestSplitLeadingPositionalWithFlagFirst(t *testing.T) {
	pos, rest := SplitLeadingPositional([]string{"--json", "abc"})
	if pos != "" {
		t.Fatalf("expected empty positional, got %q", pos)
	}
	if len(rest) != 2 || rest[0] != "--json" {
		t.Fatalf("unexpected remainder: %#v", rest)
	}
}
