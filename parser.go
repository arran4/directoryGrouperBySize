package directoryGrouperBySize

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Entry represents the size and name of one listing item.
type Entry struct {
	SizeInGB float64
	Name     string
}

var sizeRegexp = regexp.MustCompile(`(?i)^([0-9]+(?:\.[0-9]+)?)([tgmk]?b?)$`)

// SizeToGB converts a size string like "10G" or "500M" into gigabytes. The
// defaultUnit argument specifies which unit to assume when the string does not
// include one. Supported units are G/GB, M/MB, K/KB, T/TB and B. Matching is
// case-insensitive and an optional trailing "B" is allowed.
func SizeToGB(sizeStr string, defaultUnit string) (float64, error) {
	matches := sizeRegexp.FindStringSubmatch(strings.ToLower(strings.TrimSpace(sizeStr)))
	if matches == nil {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	sizeInFloat, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size value: %s", matches[1])
	}

	unit := strings.ToUpper(matches[2])
	if unit == "" {
		unit = strings.ToUpper(defaultUnit)
	}

	switch unit {
	case "G", "GB":
		return sizeInFloat, nil
	case "M", "MB":
		return sizeInFloat / 1024, nil
	case "K", "KB":
		return sizeInFloat / (1024 * 1024), nil
	case "T", "TB":
		return sizeInFloat * 1024, nil
	case "B":
		return sizeInFloat / (1024 * 1024 * 1024), nil
	default:
		return 0, fmt.Errorf("unknown size suffix: %s", unit)
	}
}

// ConvertToStructArray converts the list of strings to an array of Entry structs
func ConvertToStructArray(data []string) ([]Entry, error) {
	var result []Entry

	for _, line := range data {
		// skip purely blank lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Find the first whitespace to separate size and name
		idx := strings.IndexAny(line, " \t")
		if idx == -1 {
			return nil, fmt.Errorf("invalid input format: %s", line)
		}

		sizeStr := line[:idx]

		// consume exactly one separator character (the first space or tab)
		name := line[idx+1:]

		if sizeStr == "" || name == "" {
			return nil, fmt.Errorf("invalid input format: %s", line)
		}

		sizeInGB, err := SizeToGB(sizeStr, "B")
		if err != nil {
			return nil, err
		}

		result = append(result, Entry{SizeInGB: sizeInGB, Name: name})
	}

	return result, nil
}
