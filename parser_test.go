package directoryGrouperBySize

import (
	"math"
	"strings"
	"testing"
)

func TestConvertToStructArray(t *testing.T) {
	input := []string{
		"1G foo",
		"500M bar",
		"2gb baz",
		"300mb qux",
		"1G    Photos  2025  ",
		"1G\ttabs\tinside  ",
		"1G\t  leading whitespace in name",
		"",
		"  \t  ",
	}
	got, err := ConvertToStructArray(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 7 {
		t.Fatalf("expected 7 items, got %d", len(got))
	}
	if got[0].Name != "foo" || got[0].SizeBytes != 1024*1024*1024 {
		t.Errorf("unexpected first item: %+v", got[0])
	}
	expectedSize := int64(500 * 1024 * 1024)
	if got[1].Name != "bar" || got[1].SizeBytes != expectedSize {
		t.Errorf("unexpected second item: %+v", got[1])
	}
	if got[2].Name != "baz" || got[2].SizeBytes != 2*1024*1024*1024 {
		t.Errorf("unexpected third item: %+v", got[2])
	}
	expectedSize2 := int64(300 * 1024 * 1024)
	if got[3].Name != "qux" || got[3].SizeBytes != expectedSize2 {
		t.Errorf("unexpected fourth item: %+v", got[3])
	}
	if got[4].Name != "Photos  2025  " || got[4].SizeBytes != 1024*1024*1024 {
		t.Errorf("unexpected fifth item: %+v", got[4])
	}
	if got[5].Name != "tabs\tinside  " || got[5].SizeBytes != 1024*1024*1024 {
		t.Errorf("unexpected sixth item: %+v", got[5])
	}
	if got[6].Name != "  leading whitespace in name" || got[6].SizeBytes != 1024*1024*1024 {
		t.Errorf("unexpected seventh item: %+v", got[6])
	}
}

func TestConvertToStructArrayInvalid(t *testing.T) {
	tests := []string{
		"invalidline",
		"1G",
		" 1G",
	}
	for _, tc := range tests {
		_, err := ConvertToStructArray([]string{tc})
		if err == nil {
			t.Errorf("expected error for invalid input: %q", tc)
		}
	}
}

func TestConvertToStructArray_NulMode(t *testing.T) {
	input := []string{
		"1G\tfoo",
		"500M\tbar\nwith\nnewlines",
		"1G\t  leading spaces preserved",
		"1G\t\ttabs inside  ", // Note: size\t\tname -> name="\ttabs inside  "
	}
	got, err := ConvertToStructArrayNULMode(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("expected 4 items, got %d", len(got))
	}
	if got[0].Name != "foo" || got[0].SizeBytes != 1024*1024*1024 {
		t.Errorf("unexpected first item: %+v", got[0])
	}
	expectedSize := int64(500 * 1024 * 1024)
	if got[1].Name != "bar\nwith\nnewlines" || got[1].SizeBytes != expectedSize {
		t.Errorf("unexpected second item: %+v", got[1])
	}
	if got[2].Name != "  leading spaces preserved" || got[2].SizeBytes != 1024*1024*1024 {
		t.Errorf("unexpected third item: %+v", got[2])
	}
	if got[3].Name != "\ttabs inside  " || got[3].SizeBytes != 1024*1024*1024 {
		t.Errorf("unexpected fourth item: %+v", got[3])
	}
}

func TestConvertToStructArray_NulModeInvalid(t *testing.T) {
	tests := []string{
		"invalidline",
		"1G",
		"",
		"2gb baz", // space as separator should fail in NUL mode
		"1G  space instead of tab",
	}
	for _, tc := range tests {
		_, err := ConvertToStructArrayNULMode([]string{tc})
		if err == nil {
			t.Errorf("expected error for invalid nul input: %q", tc)
		}
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		in     string
		def    string
		expect int64
	}{
		// Default units
		{"55", "GB", 55 * 1024 * 1024 * 1024},
		{"1024", "B", 1024},
		// Case insensitivity
		{"1g", "GB", 1 * 1024 * 1024 * 1024},
		{"1G", "GB", 1 * 1024 * 1024 * 1024},
		{"1gB", "GB", 1 * 1024 * 1024 * 1024},
		// Legacy / IEC suffixes
		{"10M", "GB", 10 * 1024 * 1024},
		{"10MiB", "GB", 10 * 1024 * 1024},
		{"10MB", "GB", 10 * 1024 * 1024},
		{"5K", "GB", 5 * 1024},
		{"5KiB", "GB", 5 * 1024},
		{"2T", "GB", 2 * 1024 * 1024 * 1024 * 1024},
		{"2TiB", "GB", 2 * 1024 * 1024 * 1024 * 1024},
		// Deterministic fractional inputs
		{"1.5G", "B", int64(1.5 * 1024 * 1024 * 1024)},
		{"1.5GB", "B", int64(1.5 * 1024 * 1024 * 1024)},
		{"1.5GiB", "B", int64(1.5 * 1024 * 1024 * 1024)},

		// Large inputs past float64 mantissa exact representability
		{"9007199254740993B", "GB", 9007199254740993}, // 2^53 + 1
		{"9223372036854775807B", "GB", math.MaxInt64}, // MaxInt64

		// Rounding check
		{"1.5B", "GB", 2},
		{"1.4B", "GB", 1},
	}

	for _, tt := range tests {
		got, err := ParseSize(tt.in, tt.def)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", tt.in, err)
		}
		if got != tt.expect {
			t.Errorf("ParseSize(%q, %q) = %v, want %v", tt.in, tt.def, got, tt.expect)
		}
	}
}

func TestParseSize_Overflow(t *testing.T) {
	tests := []struct {
		in  string
		def string
	}{
		{"9223372036854775808B", "GB"}, // math.MaxInt64 + 1
		{"10000000000000000000B", "GB"},
		{"9223372036854775807.5B", "GB"}, // Rounds up to math.MaxInt64 + 1
	}

	for _, tt := range tests {
		_, err := ParseSize(tt.in, tt.def)
		if err == nil {
			t.Errorf("expected error for overflow size %s, got nil", tt.in)
		}
	}
}

func TestConvertToStructArray_EmptyFilename(t *testing.T) {
	input := []string{
		"1G\t",
	}
	_, err := ConvertToStructArrayNULMode(input)
	if err == nil {
		t.Fatalf("expected error for empty filename in NUL mode, got nil")
	}
	if !strings.Contains(err.Error(), "invalid input format") {
		t.Errorf("expected 'invalid input format' error, got %v", err)
	}

	inputLineOriented := []string{
		"1G\t",
	}
	_, err = ConvertToStructArray(inputLineOriented)
	if err == nil {
		t.Fatalf("expected error for empty filename in line-oriented mode, got nil")
	}
	if !strings.Contains(err.Error(), "invalid input format") {
		t.Errorf("expected 'invalid input format' error, got %v", err)
	}
}
