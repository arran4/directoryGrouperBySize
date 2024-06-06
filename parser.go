package directoryGrouperBySize

import (
	"fmt"
	"strconv"
	"strings"
)

// Anime represents the structure of each item
type Anime struct {
	SizeInGB float64
	Name     string
}

// ConvertToStructArray converts the list of strings to an array of Anime structs
func ConvertToStructArray(data []string) ([]Anime, error) {
	var result []Anime

	for _, line := range data {
		// Split the line into size and name parts
		parts := strings.Fields(line)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid input format: %s", line)
		}

		// Join the remaining parts as the name
		name := strings.Join(parts[1:], " ")

		// Extract the size string and the suffix
		sizeStr := parts[0]
		sizeValue, sizeSuffix := sizeStr[:len(sizeStr)-1], sizeStr[len(sizeStr)-1]

		// Parse the size value
		sizeInFloat, err := strconv.ParseFloat(sizeValue, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid size format: %s", sizeStr)
		}

		// Convert the size to GB based on the suffix
		var sizeInGB float64
		switch sizeSuffix {
		case 'G':
			sizeInGB = sizeInFloat
		case 'M':
			sizeInGB = sizeInFloat / 1024 // Convert MB to GB
		case 'K':
			sizeInGB = sizeInFloat / (1024 * 1024) // Convert KB to GB
		case 'T':
			sizeInGB = sizeInFloat * 1024 // Convert TB to GB
		default:
			return nil, fmt.Errorf("unknown size suffix: %s", sizeSuffix)
		}

		// Create an Anime struct and add it to the result
		result = append(result, Anime{SizeInGB: sizeInGB, Name: name})
	}

	return result, nil
}
