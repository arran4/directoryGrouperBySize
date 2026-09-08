package directoryGrouperBySize

import (
	"strings"
	"testing"
)

func TestGroup_Valid(t *testing.T) {
	entries := []Entry{
		{SizeBytes: 1 * 1024 * 1024 * 1024, Name: "A"},
		{SizeBytes: 2 * 1024 * 1024 * 1024, Name: "B"},
		{SizeBytes: 3 * 1024 * 1024 * 1024, Name: "C"}, // Should trigger a new disk if max is 3
	}

	disks, err := Group(entries, 3*1024*1024*1024)
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
		{SizeBytes: 3 * 1024 * 1024 * 1024, Name: "A"},
	}

	disks, err := Group(entries, 3*1024*1024*1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(disks) != 1 {
		t.Fatalf("expected 1 disk, got %d", len(disks))
	}
}

func TestGroup_OneByteOverCapacity(t *testing.T) {
	entries := []Entry{
		{SizeBytes: 3*1024*1024*1024 + 1, Name: "A"}, // exactly one byte over
	}

	_, err := Group(entries, 3*1024*1024*1024)
	if err == nil {
		t.Fatal("expected error for one-byte oversized entry")
	}
	// "3.00 GB" formatting is slightly imprecise but the error string is what we test for now
	if !strings.Contains(err.Error(), "A (3.00 GB)") {
		t.Errorf("expected error to contain A (3.00 GB), got %v", err)
	}
}

func TestGroup_OversizedFirst(t *testing.T) {
	entries := []Entry{
		{SizeBytes: 4 * 1024 * 1024 * 1024, Name: "A"},
	}

	_, err := Group(entries, 3*1024*1024*1024)
	if err == nil {
		t.Fatal("expected error for oversized entry")
	}
	if !strings.Contains(err.Error(), "A (4.00 GB)") {
		t.Errorf("expected error to contain A (4.00 GB), got %v", err)
	}
}

func TestGroup_OversizedLater(t *testing.T) {
	entries := []Entry{
		{SizeBytes: 1 * 1024 * 1024 * 1024, Name: "A"},
		{SizeBytes: 4 * 1024 * 1024 * 1024, Name: "B"},
	}

	_, err := Group(entries, 3*1024*1024*1024)
	if err == nil {
		t.Fatal("expected error for oversized entry")
	}
	if !strings.Contains(err.Error(), "B (4.00 GB)") {
		t.Errorf("expected error to contain B (4.00 GB), got %v", err)
	}
}

func TestGroup_MultipleOversized(t *testing.T) {
	entries := []Entry{
		{SizeBytes: 1 * 1024 * 1024 * 1024, Name: "A"},
		{SizeBytes: 4 * 1024 * 1024 * 1024, Name: "B"},
		{SizeBytes: 2 * 1024 * 1024 * 1024, Name: "C"},
		{SizeBytes: 5 * 1024 * 1024 * 1024, Name: "D"},
	}

	_, err := Group(entries, 3*1024*1024*1024)
	if err == nil {
		t.Fatal("expected error for oversized entries")
	}
	if !strings.Contains(err.Error(), "B (4.00 GB), D (5.00 GB)") {
		t.Errorf("expected error to contain both B and D, got %v", err)
	}
}

func TestGroup_AccumulatedRounding(t *testing.T) {
	// A small file that doesn't cleanly represent well as a float in GB without rounding
	smallSize := int64(1 * 1024 * 1024) // 1 MB

	// Create enough of them to exactly hit max capacity
	count := 3000
	maxSizeBytes := smallSize * int64(count)

	var entries []Entry
	for i := 0; i < count; i++ {
		entries = append(entries, Entry{SizeBytes: smallSize, Name: "Small"})
	}

	disks, err := Group(entries, maxSizeBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should exactly fit into 1 disk due to precise integer math
	if len(disks) != 1 {
		t.Fatalf("expected exactly 1 disk, got %d", len(disks))
	}
	if len(disks[0]) != count {
		t.Fatalf("expected %d entries on the disk, got %d", count, len(disks[0]))
	}
}

func TestGroup_OverflowSafe(t *testing.T) {
	// math.MaxInt64 is 9223372036854775807
	maxSizeBytes := int64(9223372036854775807)
	entries := []Entry{
		{SizeBytes: maxSizeBytes - 1, Name: "A"},
		{SizeBytes: 2, Name: "B"},
	}

	// This should group them into two separate disks because (maxSizeBytes - 1) + 2 overflows maxSizeBytes.
	// We pass maxSizeBytes as the capacity. Both individual entries are <= maxSizeBytes.
	disks, err := Group(entries, maxSizeBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(disks) != 2 {
		t.Fatalf("expected 2 disks, got %d", len(disks))
	}
	if len(disks[0]) != 1 || disks[0][0].Name != "A" {
		t.Errorf("unexpected first disk: %+v", disks[0])
	}
	if len(disks[1]) != 1 || disks[1][0].Name != "B" {
		t.Errorf("unexpected second disk: %+v", disks[1])
	}
}
