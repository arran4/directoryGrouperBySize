package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func TestRun(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		stdin       io.Reader
		expectError bool
		errContains string
		outContains string
		setupFiles  func(string)
	}{
		{
			name:        "Missing maxsize",
			args:        []string{},
			stdin:       strings.NewReader(""),
			expectError: true,
			errContains: "please provide a valid -maxsize argument",
		},
		{
			name:        "Invalid maxsize format",
			args:        []string{"-maxsize", "xyz"},
			stdin:       strings.NewReader(""),
			expectError: true,
			errContains: "invalid size format",
		},
		{
			name:        "Conflicting flags",
			args:        []string{"-maxsize", "5G", "-f", "dummy.txt", "-scan", "."},
			stdin:       strings.NewReader(""),
			expectError: true,
			errContains: "cannot use both -f and -scan",
		},
		{
			name:        "Valid run with stdin",
			args:        []string{"-maxsize", "2G"},
			stdin:       strings.NewReader("1G folder1\n500M folder2\n"),
			expectError: false,
			outContains: "Disk 1",
		},
		{
			name:        "Oversized entry",
			args:        []string{"-maxsize", "2G"},
			stdin:       strings.NewReader("3G folder1\n"),
			expectError: true,
			errContains: "entry exceeds capacity",
		},
		{
			name:        "Version flag",
			args:        []string{"-version"},
			stdin:       strings.NewReader(""),
			expectError: false,
			outContains: "directoryGrouperBySize dev",
		},
		{
			name:        "Malformed listing",
			args:        []string{"-maxsize", "2G"},
			stdin:       strings.NewReader("invalid_listing_no_size"),
			expectError: true,
			errContains: "invalid input format",
		},
		{
			name:        "Unreadable input file",
			args:        []string{"-maxsize", "2G", "-f", "nonexistent_file.txt"},
			stdin:       strings.NewReader(""),
			expectError: true,
			errContains: "error opening file",
		},
		{
			name:        "Invalid -scan directory",
			args:        []string{"-maxsize", "2G", "-scan", "nonexistent_scan_dir"},
			stdin:       strings.NewReader(""),
			expectError: true,
			errContains: "error reading directory",
		},
		{
			name:  "Filename whitespace preservation",
			args:  []string{"-maxsize", "2G", "-scan", "mock_scan_dir_ws"},
			stdin: strings.NewReader(""),
			setupFiles: func(dir string) {
				os.MkdirAll(filepath.Join(dir, "mock_scan_dir_ws"), 0755)
				os.WriteFile(filepath.Join(dir, "mock_scan_dir_ws", " trailing space "), []byte("data"), 0644)
			},
			expectError: false,
			outContains: " trailing space \n",
		},
		{
			name:        "Stdin read error",
			args:        []string{"-maxsize", "2G"},
			stdin:       &errorReader{err: errors.New("simulated read error")},
			expectError: true,
			errContains: "error reading stdin: simulated read error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			// If setup files needed for scan, we run in a tmp dir
			var args []string
			args = append(args, tc.args...)
			if tc.setupFiles != nil {
				tmpDir, _ := os.MkdirTemp("", "grouper_test")
				defer os.RemoveAll(tmpDir)
				tc.setupFiles(tmpDir)

				// Fix args with tmpdir
				for i, a := range args {
					if a == "-scan" && i+1 < len(args) {
						args[i+1] = filepath.Join(tmpDir, args[i+1])
					}
				}
			}

			err := run(args, tc.stdin, &stdout, &stderr)

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
				t.Errorf("expected stdout to contain %q, got %q\nOut:\n%s", tc.outContains, stdout.String(), stdout.String())
			}
		})
	}
}

func TestRun_DefaultStrategy(t *testing.T) {
	stdin := strings.NewReader("6G A\n5G B\n4G C\n")
	var stdout, stderr bytes.Buffer
	args := []string{"-maxsize", "10G"}

	err := run(args, stdin, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := stdout.String()
	// FFD (6,5,4 capacity 10):
	// Disk 1: 6, 4
	// Disk 2: 5
	// Check for 2 disks
	if !strings.Contains(out, "Disk 2") || strings.Contains(out, "Disk 3") {
		t.Errorf("expected 2 disks, got: %s", out)
	}
	if !strings.Contains(out, "A\nC\n") && !strings.Contains(out, "C\nA\n") {
		t.Errorf("expected A and C to be in same disk, got: %s", out)
	}
}

func TestRun_NextFitStrategy(t *testing.T) {
	stdin := strings.NewReader("6G A\n5G B\n4G C\n")
	var stdout, stderr bytes.Buffer
	args := []string{"-maxsize", "10G", "-strategy", "next-fit"}

	err := run(args, stdin, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := stdout.String()
	// Next-fit (6,5,4 capacity 10):
	// Disk 1: 6
	// Disk 2: 5, 4
	if !strings.Contains(out, "Disk 2") || strings.Contains(out, "Disk 3") {
		t.Errorf("expected 2 disks, got: %s", out)
	}
	if !strings.Contains(out, "B\nC\n") {
		t.Errorf("expected B and C to be in same disk, got: %s", out)
	}
}

func TestRun_ExplicitFFDStrategy(t *testing.T) {
	stdin := strings.NewReader("6G A\n5G B\n4G C\n")
	var stdout, stderr bytes.Buffer
	args := []string{"-maxsize", "10G", "-strategy", "first-fit-decreasing"}

	err := run(args, stdin, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := stdout.String()
	// same as default
	if !strings.Contains(out, "Disk 2") || strings.Contains(out, "Disk 3") {
		t.Errorf("expected 2 disks, got: %s", out)
	}
}

func TestRun_UnknownStrategy(t *testing.T) {
	stdin := strings.NewReader("6G A\n")
	var stdout, stderr bytes.Buffer
	args := []string{"-maxsize", "10G", "-strategy", "bad-strategy"}

	err := run(args, stdin, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown strategy")
	}
	if !strings.Contains(err.Error(), "unknown strategy: bad-strategy") {
		t.Errorf("unexpected error message: %v", err)
	}
}
