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
// - "best-fit": sorts entries descending, places each in the disk leaving the least free space.
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
	case "best-fit":
		return groupBestFit(entries, maxSizeBytes), nil
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

func groupBestFit(entries []Entry, maxSizeBytes int64) [][]Entry {
	if len(entries) == 0 {
		return nil
	}

	// We need to keep original index to ensure deterministic sorting for equal sizes
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

	type diskInfo struct {
		entries []Entry
		size    int64
	}

	var disks []*diskInfo

	for _, se := range sortedEntries {
		entry := se.entry
		bestDiskIdx := -1
		var minSpaceRemaining int64 = maxSizeBytes + 1

		for i, disk := range disks {
			remaining := maxSizeBytes - disk.size
			if entry.SizeBytes <= remaining && remaining-entry.SizeBytes < minSpaceRemaining {
				bestDiskIdx = i
				minSpaceRemaining = remaining - entry.SizeBytes
			}
		}

		if bestDiskIdx != -1 {
			disks[bestDiskIdx].entries = append(disks[bestDiskIdx].entries, entry)
			disks[bestDiskIdx].size += entry.SizeBytes
		} else {
			disks = append(disks, &diskInfo{
				entries: []Entry{entry},
				size:    entry.SizeBytes,
			})
		}
	}

	result := make([][]Entry, len(disks))
	for i, disk := range disks {
		result[i] = disk.entries
	}

	return result
}
