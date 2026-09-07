package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		stdin       string
		expectError bool
		errContains string
		outContains string
	}{
		{
			name:        "Missing maxsize",
			args:        []string{},
			expectError: true,
			errContains: "please provide a valid -maxsize argument",
		},
		{
			name:        "Invalid maxsize format",
			args:        []string{"-maxsize", "xyz"},
			expectError: true,
			errContains: "invalid size format",
		},
		{
			name:        "Conflicting flags",
			args:        []string{"-maxsize", "5G", "-f", "dummy.txt", "-scan", "."},
			expectError: true,
			errContains: "cannot use both -f and -scan",
		},
		{
			name:        "Valid run with stdin",
			args:        []string{"-maxsize", "2G"},
			stdin:       "1G folder1\n500M folder2\n",
			expectError: false,
			outContains: "Disk 1",
		},
		{
			name:        "Oversized entry",
			args:        []string{"-maxsize", "2G"},
			stdin:       "3G folder1\n",
			expectError: true,
			errContains: "entry exceeds capacity",
		},
		{
			name:        "Version flag",
			args:        []string{"-version"},
			expectError: false,
			outContains: "directoryGrouperBySize dev",
		},
		{
			name:        "Malformed listing",
			args:        []string{"-maxsize", "2G"},
			stdin:       "invalid_listing_no_size",
			expectError: true,
			errContains: "invalid input format",
		},
		{
			name:        "Unreadable input file",
			args:        []string{"-maxsize", "2G", "-f", "nonexistent_file.txt"},
			expectError: true,
			errContains: "error opening file",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			stdin := strings.NewReader(tc.stdin)

			err := run(tc.args, stdin, &stdout, &stderr)

			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("expected error to contain %q, got %q", tc.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			if tc.outContains != "" && !strings.Contains(stdout.String(), tc.outContains) {
				t.Errorf("expected stdout to contain %q, got %q", tc.outContains, stdout.String())
			}
		})
	}
}
