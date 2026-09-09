package directoryGrouperBySize

import (
	"fmt"
	"sort"
	"strings"
)

// Group partitions entries into disks, ensuring no disk exceeds maxSizeBytes.
// Uses the best-fit strategy by default to optimize disk usage.
// Returns an error if any entry exceeds maxSizeBytes.
// If multiple entries are oversized, it collects and reports all of them.
func Group(entries []Entry, maxSizeBytes int64) ([][]Entry, error) {
	return GroupWithStrategy(entries, maxSizeBytes, "best-fit")
}

// GroupWithStrategy partitions entries according to the specified strategy.
// Supported strategies:
// - "next-fit": preserves input order, creates a new disk when current fills up.
// - "best-fit": alias for first-fit-decreasing, reorders entries to optimize space.
// - "first-fit-decreasing": sorts entries descending, places each in the first disk that fits it efficiently.
func GroupWithStrategy(entries []Entry, maxSizeBytes int64, strategy string) ([][]Entry, error) {
	var oversized []string
	for _, entry := range entries {
		if entry.SizeBytes > maxSizeBytes {
			oversized = append(oversized, fmt.Sprintf("%s (%s)", entry.Name, FormatBytesGB(entry.SizeBytes)))
		}
	}

	if len(oversized) > 0 {
		return nil, fmt.Errorf("entry exceeds capacity (max: %s): %s", FormatBytesGB(maxSizeBytes), strings.Join(oversized, ", "))
	}

	switch strategy {
	case "next-fit":
		return groupNextFit(entries, maxSizeBytes), nil
	case "best-fit", "first-fit-decreasing":
		return groupFirstFitDecreasing(entries, maxSizeBytes), nil
	default:
		return nil, fmt.Errorf("unknown strategy: %s", strategy)
	}
}

func groupNextFit(entries []Entry, maxSizeBytes int64) [][]Entry {
	var disks [][]Entry
	var currentDisk []Entry
	var currentDiskSize int64

	for _, entry := range entries {
		remaining := maxSizeBytes - currentDiskSize
		if entry.SizeBytes > remaining {
			disks = append(disks, currentDisk)
			currentDisk = []Entry{}
			currentDiskSize = 0
		}
		currentDisk = append(currentDisk, entry)
		currentDiskSize += entry.SizeBytes
	}

	if len(currentDisk) > 0 {
		disks = append(disks, currentDisk)
	}

	return disks
}

// groupFirstFitDecreasing implements a First-Fit Decreasing algorithm in O(n log n) time.
// It uses a segment tree to efficiently find the first disk with enough remaining space.
func groupFirstFitDecreasing(entries []Entry, maxSizeBytes int64) [][]Entry {
	if len(entries) == 0 {
		return nil
	}

	type entryWithIdx struct {
		entry Entry
		idx   int
	}

	sortedEntries := make([]entryWithIdx, len(entries))
	for i, e := range entries {
		sortedEntries[i] = entryWithIdx{entry: e, idx: i}
	}

	sort.Slice(sortedEntries, func(i, j int) bool {
		if sortedEntries[i].entry.SizeBytes == sortedEntries[j].entry.SizeBytes {
			return sortedEntries[i].idx < sortedEntries[j].idx
		}
		return sortedEntries[i].entry.SizeBytes > sortedEntries[j].entry.SizeBytes
	})

	n := len(entries)
	// Segment tree array. Size is 4*n to be safe.
	// Stores the maximum remaining space in any disk within the range.
	tree := make([]int64, 4*n)
	// Initialize segment tree: all disks have maxSizeBytes remaining initially.
	// Actually we only create disks as needed, but for the tree we can assume n disks exist,
	// each with maxSizeBytes remaining, because we will never need more than n disks.
	var build func(node, start, end int)
	build = func(node, start, end int) {
		if start == end {
			tree[node] = maxSizeBytes
			return
		}
		mid := (start + end) / 2
		build(2*node, start, mid)
		build(2*node+1, mid+1, end)
		tree[node] = maxSizeBytes
	}
	build(1, 0, n-1)

	// Update segment tree
	var update func(node, start, end, idx int, newSpace int64)
	update = func(node, start, end, idx int, newSpace int64) {
		if start == end {
			tree[node] = newSpace
			return
		}
		mid := (start + end) / 2
		if start <= idx && idx <= mid {
			update(2*node, start, mid, idx, newSpace)
		} else {
			update(2*node+1, mid+1, end, idx, newSpace)
		}
		if tree[2*node] > tree[2*node+1] {
			tree[node] = tree[2*node]
		} else {
			tree[node] = tree[2*node+1]
		}
	}

	// Query segment tree for the first disk index that has at least `required` space
	var query func(node, start, end int, required int64) int
	query = func(node, start, end int, required int64) int {
		if start == end {
			return start
		}
		mid := (start + end) / 2
		// If left child has enough space, go left (because we want the FIRST disk)
		if tree[2*node] >= required {
			return query(2*node, start, mid, required)
		}
		// Otherwise, go right
		return query(2*node+1, mid+1, end, required)
	}

	diskRemaining := make([]int64, n)
	for i := 0; i < n; i++ {
		diskRemaining[i] = maxSizeBytes
	}

	diskContents := make([][]Entry, n)
	maxDiskUsed := -1

	for _, se := range sortedEntries {
		entry := se.entry
		// Find first disk with enough space
		diskIdx := query(1, 0, n-1, entry.SizeBytes)

		// Update disk space
		diskRemaining[diskIdx] -= entry.SizeBytes
		update(1, 0, n-1, diskIdx, diskRemaining[diskIdx])

		// Add entry to disk
		diskContents[diskIdx] = append(diskContents[diskIdx], entry)

		if diskIdx > maxDiskUsed {
			maxDiskUsed = diskIdx
		}
	}

	return diskContents[:maxDiskUsed+1]
}
