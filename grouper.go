package directoryGrouperBySize

import (
	"fmt"
	"strings"
)

// Group partitions entries into disks, ensuring no disk exceeds maxSizeBytes.
// Returns an error if any entry exceeds maxSizeBytes.
// If multiple entries are oversized, it collects and reports all of them.
func Group(entries []Entry, maxSizeBytes int64) ([][]Entry, error) {
	var oversized []string
	for _, entry := range entries {
		if entry.SizeBytes > maxSizeBytes {
			oversized = append(oversized, fmt.Sprintf("%s (%s)", entry.Name, FormatBytesGB(entry.SizeBytes)))
		}
	}

	if len(oversized) > 0 {
		return nil, fmt.Errorf("entry exceeds capacity (max: %s): %s", FormatBytesGB(maxSizeBytes), strings.Join(oversized, ", "))
	}

	var disks [][]Entry
	var currentDisk []Entry
	var currentDiskSize int64

	for _, entry := range entries {
		// Overflow-safe check: check remaining capacity instead of adding
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

	return disks, nil
}
