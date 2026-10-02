package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestGeneratedVersionCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"-maxsize", "1G", "version"}, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error running version command: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Version: dev") || !strings.Contains(out, "Commit: none") {
		t.Errorf("expected version output to contain dev and none, got: %s", out)
	}
}

func TestLegacyVersionFlag(t *testing.T) {
    var stdout, stderr bytes.Buffer
    // `-version` triggers the application logic.
    // The application logic reads its own `version` variable.
    err := run([]string{"-version"}, nil, &stdout, &stderr)
    if err != nil {
        t.Fatalf("unexpected error running -version flag: %v", err)
    }

    out := stdout.String()
    if !strings.Contains(out, "directoryGrouperBySize dev") {
        t.Errorf("expected -version output to contain 'directoryGrouperBySize dev', got: %s", out)
    }
}
