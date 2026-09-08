package directoryGrouperBySize

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

// Entry represents the size and name of one listing item.
type Entry struct {
	SizeBytes int64
	Name      string
}

// sizeRegexp matches a size (e.g., 10, 1.5) followed by an optional unit.
var sizeRegexp = regexp.MustCompile(`(?i)^([0-9]+(?:\.[0-9]+)?)([a-z]*)$`)

// ParseSize converts a size string like "10G" or "500M" into exact bytes.
// The defaultUnit argument specifies which unit to assume when the string does not
// include one. Supported units include B, K/KB/KiB, M/MB/MiB, G/GB/GiB, T/TB/TiB.
// Note: for compatibility with existing behaviour and `du -sh`, legacy suffixes
// (K, KB, M, MB, G, GB, T, TB) are treated as binary IEC units (powers of 1024),
// exactly like their IEC counterparts (KiB, MiB, GiB, TiB).
func ParseSize(sizeStr string, defaultUnit string) (int64, error) {
	matches := sizeRegexp.FindStringSubmatch(strings.TrimSpace(sizeStr))
	if matches == nil {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	sizeValStr := matches[1]
	unit := strings.ToUpper(matches[2])
	if unit == "" {
		unit = strings.ToUpper(defaultUnit)
	}

	var multiplier int64

	switch unit {
	case "B":
		multiplier = 1
	case "K", "KB", "KIB":
		multiplier = 1024
	case "M", "MB", "MIB":
		multiplier = 1024 * 1024
	case "G", "GB", "GIB":
		multiplier = 1024 * 1024 * 1024
	case "T", "TB", "TIB":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown size suffix: %s", unit)
	}

	// Use math/big to handle large sizes exactly without float64 precision loss.
	// Parse as a big.Rat to cleanly handle fractional inputs.
	rat, ok := new(big.Rat).SetString(sizeValStr)
	if !ok {
		return 0, fmt.Errorf("invalid size value: %s", sizeValStr)
	}

	// Multiply by unit
	rat.Mul(rat, new(big.Rat).SetInt64(multiplier))

	// Round to nearest integer (half away from zero)
	// Add 1/2 to round half up.
	half := big.NewRat(1, 2)
	rat.Add(rat, half)

	// Truncate to int
	intVal := new(big.Int)
	intVal.Div(rat.Num(), rat.Denom())

	// Check if it fits in int64
	if !intVal.IsInt64() {
		return 0, fmt.Errorf("size out of range for int64: %s", sizeStr)
	}

	return intVal.Int64(), nil
}

// FormatBytesGB formats a byte count into a GB string for display, using 1024-based GB.
func FormatBytesGB(bytes int64) string {
	return fmt.Sprintf("%.2f GB", float64(bytes)/(1024*1024*1024))
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

		remainder := line[idx:]
		sepLen := 0

		// If the first delimiter character is a tab, consume exactly one tab.
		// Otherwise (it's a space), consume the contiguous block of spaces.
		if remainder[0] == '\t' {
			sepLen = 1
		} else {
			for i, r := range remainder {
				if r != ' ' {
					sepLen = i
					break
				}
			}
			if sepLen == 0 {
				sepLen = len(remainder)
			}
		}

		name := remainder[sepLen:]

		if sizeStr == "" || name == "" {
			return nil, fmt.Errorf("invalid input format: %s", line)
		}

		sizeBytes, err := ParseSize(sizeStr, "B")
		if err != nil {
			return nil, err
		}

		result = append(result, Entry{SizeBytes: sizeBytes, Name: name})
	}

	return result, nil
}
