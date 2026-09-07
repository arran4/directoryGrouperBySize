package directoryGrouperBySize

import "fmt"

// Group partitions entries into disks, ensuring no disk exceeds maxSizeGB.
// Returns an error if any single entry exceeds maxSizeGB.
func Group(entries []Entry, maxSizeGB float64) ([][]Entry, error) {
	var disks [][]Entry
	var currentDisk []Entry
	var currentDiskSize float64

	for _, entry := range entries {
		if entry.SizeInGB > maxSizeGB {
			return nil, fmt.Errorf("entry exceeds capacity (max: %.2f GB): %s (%.2f GB)", maxSizeGB, entry.Name, entry.SizeInGB)
		}

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
