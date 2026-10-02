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

// Helper function to replace run() by testing generated command directly using os.Args mock
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	oldArgs := os.Args
	oldStdin := os.Stdin
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	defer func() {
		os.Args = oldArgs
		os.Stdin = oldStdin
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	os.Args = append([]string{"directoryGrouperBySize"}, args...)

	if stdin != nil {
		r, w, _ := os.Pipe()
		go func() {
			defer func() { _ = w.Close() }()
			_, _ = io.Copy(w, stdin)
		}()
		os.Stdin = r
	}

	var outPipeR, outPipeW *os.File
	if stdout != nil {
		outPipeR, outPipeW, _ = os.Pipe()
		os.Stdout = outPipeW
	}

	var errPipeR, errPipeW *os.File
	if stderr != nil {
		errPipeR, errPipeW, _ = os.Pipe()
		os.Stderr = errPipeW
	}

	outDone := make(chan struct{})
	if stdout != nil {
		go func() {
			_, _ = io.Copy(stdout, outPipeR)
			close(outDone)
		}()
	}

	errDone := make(chan struct{})
	if stderr != nil {
		go func() {
			_, _ = io.Copy(stderr, errPipeR)
			close(errDone)
		}()
	}

	root, err := NewRoot("directoryGrouperBySize", "dev", "none", "unknown")
	if err != nil {
		return err
	}

	execErr := root.Execute(args)

	if stdout != nil {
		_ = outPipeW.Close()
		<-outDone
	}
	if stderr != nil {
		_ = errPipeW.Close()
		<-errDone
	}

	return execErr
}

func TestErrorWrapping(t *testing.T) {
	root, err := NewRoot("directoryGrouperBySize", "dev", "none", "unknown")
	if err != nil {
		t.Fatalf("failed to create root cmd: %v", err)
	}

	err = root.Execute([]string{"-maxsize", "1GB", "-f", "nonexistent.txt"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var pathErr *os.PathError
	if !errors.As(err, &pathErr) {
		t.Errorf("expected error to wrap os.PathError, got: %T (%v)", err, err)
	}

	errMsg := err.Error()
	if strings.Contains(errMsg, "directorygrouperbysize failed: directorygrouperbysize failed:") {
		t.Errorf("expected error not to contain duplicated prefix, got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "directorygrouperbysize failed:") {
		t.Errorf("expected error to contain useful context prefix, got: %v", errMsg)
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		stdin       io.Reader
		expectError bool
		errContains string
		outContains string
		setupFiles  func(*testing.T, string)
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
			setupFiles: func(t *testing.T, dir string) {
				t.Helper()
				scanDir := filepath.Join(dir, "mock_scan_dir_ws")
				if err := os.MkdirAll(scanDir, 0755); err != nil {
					t.Fatalf("create scan dir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(scanDir, " trailing space "), []byte("data"), 0644); err != nil {
					t.Fatalf("write scan fixture: %v", err)
				}
			},
			expectError: false,
			outContains: " trailing space \n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			// If setup files needed for scan, we run in a tmp dir
			var args []string
			args = append(args, tc.args...)
			if tc.setupFiles != nil {
				tmpDir := t.TempDir()
				tc.setupFiles(t, tmpDir)

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

func TestHelpMessage(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := run([]string{"-h"}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("expected nil err when invoking -h since it is handled by the framework gracefully, got %v", err)
	}

	out := stderr.String()
	if !strings.Contains(out, "Directory to scan internally") {
		t.Errorf("expected help message to describe the internal scanner, got:\n%s", out)
	}
	if strings.Contains(out, "du -sh") {
		t.Errorf("expected help message not to mention du -sh, got:\n%s", out)
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

func TestValidRunFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	dir := t.TempDir()
	filepath := filepath.Join(dir, "valid_file.txt")
	_ = os.WriteFile(filepath, []byte("1G folder1\n500M folder2\n"), 0644)
	err := run([]string{"-maxsize", "2G", "-f", filepath}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Disk 1") {
		t.Errorf("expected Disk 1, got %s", stdout.String())
	}
}

func TestValidRunNull(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"-maxsize", "2G", "--null"}, strings.NewReader("1G\tfolder1\x00500M\tfolder2\x00"), &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Disk 1") {
		t.Errorf("expected Disk 1, got %s", stdout.String())
	}
}

func TestHelpMessageExtended(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := run([]string{"-h"}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("expected nil err when invoking -h since it is handled by the framework gracefully, got %v", err)
	}

	out := stderr.String()
	if strings.Contains(out, "please provide a valid -maxsize argument") {
		t.Errorf("expected help message not to contain validation diagnostic, got:\n%s", out)
	}
}
