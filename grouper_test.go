package directoryGrouperBySize

import (
	"strings"
	"testing"
)

func TestGroup_Valid(t *testing.T) {
	entries := []Entry{
		{SizeInGB: 1, Name: "A"},
		{SizeInGB: 2, Name: "B"},
		{SizeInGB: 3, Name: "C"}, // Should trigger a new disk if max is 3
	}

	disks, err := Group(entries, 3.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(disks) != 2 {
		t.Fatalf("expected 2 disks, got %d", len(disks))
	}
	if len(disks[0]) != 2 || disks[0][0].Name != "A" || disks[0][1].Name != "B" {
		t.Errorf("unexpected first disk: %+v", disks[0])
	}
	if len(disks[1]) != 1 || disks[1][0].Name != "C" {
		t.Errorf("unexpected second disk: %+v", disks[1])
	}
}

func TestGroup_ExactFit(t *testing.T) {
	entries := []Entry{
		{SizeInGB: 3, Name: "A"},
	}

	disks, err := Group(entries, 3.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(disks) != 1 {
		t.Fatalf("expected 1 disk, got %d", len(disks))
	}
}

func TestGroup_OversizedFirst(t *testing.T) {
	entries := []Entry{
		{SizeInGB: 4, Name: "A"},
	}

	_, err := Group(entries, 3.0)
	if err == nil {
		t.Fatal("expected error for oversized entry")
	}
	if !strings.Contains(err.Error(), "A (4.00 GB)") {
		t.Errorf("expected error to contain A (4.00 GB), got %v", err)
	}
}

func TestGroup_OversizedLater(t *testing.T) {
	entries := []Entry{
		{SizeInGB: 1, Name: "A"},
		{SizeInGB: 4, Name: "B"},
	}

	_, err := Group(entries, 3.0)
	if err == nil {
		t.Fatal("expected error for oversized entry")
	}
	if !strings.Contains(err.Error(), "B (4.00 GB)") {
		t.Errorf("expected error to contain B (4.00 GB), got %v", err)
	}
}

func TestGroup_MultipleOversized(t *testing.T) {
	entries := []Entry{
		{SizeInGB: 1, Name: "A"},
		{SizeInGB: 4, Name: "B"},
		{SizeInGB: 2, Name: "C"},
		{SizeInGB: 5, Name: "D"},
	}

	_, err := Group(entries, 3.0)
	if err == nil {
		t.Fatal("expected error for oversized entries")
	}
	if !strings.Contains(err.Error(), "B (4.00 GB), D (5.00 GB)") {
		t.Errorf("expected error to contain both B and D, got %v", err)
	}
}
