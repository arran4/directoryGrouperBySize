package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type mockScannerReader struct {
	content string
	err     error
}

func (m *mockScannerReader) Read(p []byte) (n int, err error) {
	if m.err != nil {
		return 0, m.err
	}
	n = copy(p, []byte(m.content))
	m.content = m.content[n:]
	if len(m.content) == 0 {
		return n, fmt.Errorf("EOF")
	}
	return n, nil
}

func TestRun(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		stdin       string
		expectError bool
		errContains string
		outContains string
		cmdRunner   execCmdFunc
		setupFiles  func(string)
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
		{
			name:        "Invalid -scan directory",
			args:        []string{"-maxsize", "2G", "-scan", "nonexistent_scan_dir"},
			expectError: true,
			errContains: "error reading directory",
		},
		{
			name: "Failing du invocation",
			args: []string{"-maxsize", "2G", "-scan", "mock_scan_dir"},
			setupFiles: func(dir string) {
				os.MkdirAll(filepath.Join(dir, "mock_scan_dir"), 0755)
				os.WriteFile(filepath.Join(dir, "mock_scan_dir", "a.txt"), []byte("data"), 0644)
			},
			cmdRunner: func(name string, arg ...string) ([]byte, error) {
				return nil, fmt.Errorf("mock du failed")
			},
			expectError: true,
			errContains: "error running du: mock du failed",
		},
		{
			name: "du filename whitespace preservation",
			args: []string{"-maxsize", "2G", "-scan", "mock_scan_dir_ws"},
			setupFiles: func(dir string) {
				os.MkdirAll(filepath.Join(dir, "mock_scan_dir_ws"), 0755)
				os.WriteFile(filepath.Join(dir, "mock_scan_dir_ws", " trailing space "), []byte("data"), 0644)
			},
			cmdRunner: func(name string, arg ...string) ([]byte, error) {
				// Mock returning a size and path with trailing spaces as du does, with newline
				return []byte("1M\t" + arg[len(arg)-1] + "\n"), nil
			},
			expectError: false,
			outContains: " trailing space \n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			stdin := strings.NewReader(tc.stdin)

			runner := execCommand
			if tc.cmdRunner != nil {
				runner = tc.cmdRunner
			}

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

			err := run(args, stdin, &stdout, &stderr, runner)

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
