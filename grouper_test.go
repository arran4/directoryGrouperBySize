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

	// For standard valid test, let's use next-fit to match old behavior
	disks, err := GroupWithStrategy(entries, 3*1024*1024*1024, "next-fit")
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

	disks, err := GroupWithStrategy(entries, maxSizeBytes, "next-fit")
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
	disks, err := GroupWithStrategy(entries, maxSizeBytes, "next-fit")
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

func TestGroup_NextFitVsBestFit(t *testing.T) {
	// A scenario where next-fit performs worse than best-fit (now first-fit-decreasing).
	// Sizes: 6, 5, 4, 3, 2, capacity: 10
	entries := []Entry{
		{SizeBytes: 6, Name: "A"},
		{SizeBytes: 5, Name: "B"},
		{SizeBytes: 4, Name: "C"},
		{SizeBytes: 3, Name: "D"},
		{SizeBytes: 2, Name: "E"},
	}

	// Next-fit:
	// Disk 1: 6 (remaining 4) - B (5) doesn't fit -> Disk 1: A
	// Disk 2: 5 + 4 = 9 (remaining 1) - D (3) doesn't fit -> Disk 2: B, C
	// Disk 3: 3 + 2 = 5 -> Disk 3: D, E
	// Total 3 disks
	nextFitDisks, err := GroupWithStrategy(entries, 10, "next-fit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nextFitDisks) != 3 {
		t.Fatalf("expected 3 disks for next-fit, got %d", len(nextFitDisks))
	}

	// First-fit-decreasing:
	// Sorted: 6, 5, 4, 3, 2
	// A(6) -> Disk 1 (rem 4)
	// B(5) -> Disk 2 (rem 5)
	// C(4) -> fits in Disk 1 (rem 4) perfectly -> Disk 1 (rem 0)
	// D(3) -> fits in Disk 2 (rem 5) -> Disk 2 (rem 2)
	// E(2) -> fits in Disk 2 (rem 2) perfectly -> Disk 2 (rem 0)
	// Total 2 disks
	bestFitDisks, err := GroupWithStrategy(entries, 10, "first-fit-decreasing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bestFitDisks) != 2 {
		t.Fatalf("expected 2 disks for first-fit-decreasing, got %d", len(bestFitDisks))
	}

	// Also test Group default behavior
	defaultDisks, err := Group(entries, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defaultDisks) != 2 {
		t.Fatalf("expected default Group to use first-fit-decreasing and return 2 disks, got %d", len(defaultDisks))
	}
}

func TestGroup_FFDDeterministicEqualSizes(t *testing.T) {
	// Duplicated equal-sized entries should be placed deterministically.
	entries := []Entry{
		{SizeBytes: 5, Name: "A1"},
		{SizeBytes: 5, Name: "A2"},
		{SizeBytes: 5, Name: "A3"},
	}

	disks, err := GroupWithStrategy(entries, 6, "first-fit-decreasing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(disks) != 3 {
		t.Fatalf("expected 3 disks, got %d", len(disks))
	}

	// Should be A1, then A2, then A3 due to stable sorting criteria based on original index.
	if disks[0][0].Name != "A1" || disks[1][0].Name != "A2" || disks[2][0].Name != "A3" {
		t.Errorf("expected deterministic ordering A1, A2, A3; got %s, %s, %s", disks[0][0].Name, disks[1][0].Name, disks[2][0].Name)
	}
}

func TestGroup_ZeroAndSmallSizes(t *testing.T) {
	entries := []Entry{
		{SizeBytes: 0, Name: "Zero1"},
		{SizeBytes: 1, Name: "Small"},
		{SizeBytes: 0, Name: "Zero2"},
	}

	disks, err := GroupWithStrategy(entries, 2, "first-fit-decreasing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// FFD sorts descending, so 1 comes first, then the zeros.
	if len(disks) != 1 {
		t.Fatalf("expected 1 disk, got %d", len(disks))
	}
	if len(disks[0]) != 3 {
		t.Fatalf("expected 3 entries in disk, got %d", len(disks[0]))
	}
	// Small(1) is largest so it's first
	if disks[0][0].Name != "Small" {
		t.Errorf("expected Small to be first, got %s", disks[0][0].Name)
	}
}

func TestGroup_FFDExactAndNearCapacity(t *testing.T) {
	entries := []Entry{
		{SizeBytes: 5, Name: "Half"},
		{SizeBytes: 4, Name: "NearHalf"},
		{SizeBytes: 1, Name: "One"},
	}
	// Capacity 5
	// FFD sorted: Half(5), NearHalf(4), One(1)
	// Half -> Disk 1 (rem 0)
	// NearHalf -> Disk 2 (rem 1)
	// One -> fits in Disk 2 (rem 0)
	disks, err := GroupWithStrategy(entries, 5, "first-fit-decreasing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(disks) != 2 {
		t.Fatalf("expected 2 disks, got %d", len(disks))
	}
	if len(disks[0]) != 1 || disks[0][0].Name != "Half" {
		t.Errorf("Disk 1 incorrect")
	}
	if len(disks[1]) != 2 || disks[1][0].Name != "NearHalf" || disks[1][1].Name != "One" {
		t.Errorf("Disk 2 incorrect")
	}
}

func TestGroup_FFD_LargeSynthetic(t *testing.T) {
	// Let's create a large synthetic test to ensure it performs well and correctly.
	const numEntries = 100000
	const maxSizeBytes = 100

	entries := make([]Entry, numEntries)
	for i := 0; i < numEntries; i++ {
		// Values from 1 to 99
		entries[i] = Entry{SizeBytes: int64(1 + (i % 99)), Name: "Synthetic"}
	}

	disks, err := GroupWithStrategy(entries, maxSizeBytes, "first-fit-decreasing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(disks) == 0 {
		t.Fatal("expected disks, got 0")
	}

	// verify all disks are within capacity
	for i, disk := range disks {
		var sum int64
		for _, e := range disk {
			sum += e.SizeBytes
		}
		if sum > maxSizeBytes {
			t.Fatalf("disk %d exceeded max capacity: %d > %d", i, sum, maxSizeBytes)
		}
	}
}
