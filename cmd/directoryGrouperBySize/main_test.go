package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun_DefaultStrategy(t *testing.T) {
	stdin := strings.NewReader("6G A\n5G B\n4G C\n")
	var stdout, stderr bytes.Buffer
	args := []string{"-maxsize", "10G"}

	err := run(args, stdin, &stdout, &stderr, nil)
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

	err := run(args, stdin, &stdout, &stderr, nil)
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

	err := run(args, stdin, &stdout, &stderr, nil)
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

	err := run(args, stdin, &stdout, &stderr, nil)
	if err == nil {
		t.Fatal("expected error for unknown strategy")
	}
	if !strings.Contains(err.Error(), "unknown strategy: bad-strategy") {
		t.Errorf("unexpected error message: %v", err)
	}
}
