package directoryGrouperBySize

import (
	"fmt"
	"strings"
)

// Group partitions entries into disks, ensuring no disk exceeds maxSizeGB.
// Returns an error if any entry exceeds maxSizeGB.
// If multiple entries are oversized, it collects and reports all of them.
func Group(entries []Entry, maxSizeGB float64) ([][]Entry, error) {
	var oversized []string
	for _, entry := range entries {
		if entry.SizeInGB > maxSizeGB {
			oversized = append(oversized, fmt.Sprintf("%s (%.2f GB)", entry.Name, entry.SizeInGB))
		}
	}

	if len(oversized) > 0 {
		return nil, fmt.Errorf("entry exceeds capacity (max: %.2f GB): %s", maxSizeGB, strings.Join(oversized, ", "))
	}

	var disks [][]Entry
	var currentDisk []Entry
	var currentDiskSize float64

	for _, entry := range entries {
		if currentDiskSize+entry.SizeInGB > maxSizeGB {
			disks = append(disks, currentDisk)
			currentDisk = []Entry{}
			currentDiskSize = 0
		}
		currentDisk = append(currentDisk, entry)
		currentDiskSize += entry.SizeInGB
	}

	if len(currentDisk) > 0 {
		disks = append(disks, currentDisk)
	}

	return disks, nil
}
