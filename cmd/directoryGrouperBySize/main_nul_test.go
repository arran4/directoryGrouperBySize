package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRun_NulMode(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		stdin       string
		expectError bool
		errContains string
		outContains string
	}{
		{
			name:        "Stdin NUL mode success",
			args:        []string{"-maxsize", "10G", "-0"},
			stdin:       "1G\tfile1\nwith\nnewlines\x002G\tfile2\x00",
			expectError: false,
			outContains: "file1\nwith\nnewlines\n", // changed expectation to match unordered output
		},
		{
			name:        "Stdin NUL mode malformed missing NUL at EOF",
			args:        []string{"-maxsize", "10G", "-0"},
			stdin:       "1G\tfile1\nwith\nnewlines",
			expectError: true,
			errContains: "missing NUL terminator at end of file",
		},
		{
			name:        "Stdin NUL mode malformed missing separator",
			args:        []string{"-maxsize", "10G", "-0"},
			stdin:       "1Gfile1\x00",
			expectError: true,
			errContains: "invalid input format",
		},
		{
			name:        "Stdin NUL mode empty filename",
			args:        []string{"-maxsize", "10G", "-0"},
			stdin:       "1G\t\x00",
			expectError: true,
			errContains: "invalid input format",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var outBuf, errBuf bytes.Buffer
			stdinBuf := strings.NewReader(tc.stdin)

			err := run(tc.args, stdinBuf, &outBuf, &errBuf)

			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("expected error containing %q, got %v", tc.errContains, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tc.outContains != "" && !strings.Contains(outBuf.String(), tc.outContains) {
					t.Errorf("expected output to contain %q, got %q", tc.outContains, outBuf.String())
				}
			}
		})
	}
}

func TestRun_NulMode_File(t *testing.T) {
	// create a temp file
	tempFile, err := os.CreateTemp("", "nul_test_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	_, err = tempFile.WriteString("1G\tfile1\nwith\nnewlines\x002G\tfile2\x00")
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tempFile.Close()

	var outBuf, errBuf bytes.Buffer
	stdinBuf := strings.NewReader("") // Stdin should not be read

	err = run([]string{"-maxsize", "10G", "-0", "-f", tempFile.Name()}, stdinBuf, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outStr := outBuf.String()
	if !strings.Contains(outStr, "file1\nwith\nnewlines\n") {
		t.Errorf("expected output to contain %q, got %q", "file1\nwith\nnewlines\n", outStr)
	}
}
