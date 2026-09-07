package directoryGrouperBySize

import "testing"

func TestConvertToStructArray(t *testing.T) {
	input := []string{
		"1G foo",
		"500M bar",
		"2gb baz",
		"300mb qux",
		"1G   repeated  spaces  ",
		"1G\t tabs\tinside  ",
		"",
		"  \t  ",
	}
	got, err := ConvertToStructArray(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 6 {
		t.Fatalf("expected 6 items, got %d", len(got))
	}
	if got[0].Name != "foo" || got[0].SizeInGB != 1 {
		t.Errorf("unexpected first item: %+v", got[0])
	}
	expectedSize := 500.0 / 1024.0
	if got[1].Name != "bar" || got[1].SizeInGB != expectedSize {
		t.Errorf("unexpected second item: %+v", got[1])
	}
	if got[2].Name != "baz" || got[2].SizeInGB != 2 {
		t.Errorf("unexpected third item: %+v", got[2])
	}
	expectedSize2 := 300.0 / 1024.0
	if got[3].Name != "qux" || got[3].SizeInGB != expectedSize2 {
		t.Errorf("unexpected fourth item: %+v", got[3])
	}
	if got[4].Name != "repeated  spaces  " || got[4].SizeInGB != 1 {
		t.Errorf("unexpected fifth item: %+v", got[4])
	}
	if got[5].Name != "tabs\tinside  " || got[5].SizeInGB != 1 {
		t.Errorf("unexpected sixth item: %+v", got[5])
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

func TestSizeToGB(t *testing.T) {
	tests := []struct {
		in     string
		def    string
		expect float64
	}{
		{"55", "GB", 55},
		{"10M", "GB", 10.0 / 1024.0},
		{"1024", "B", 1024.0 / (1024 * 1024 * 1024)},
	}

	for _, tt := range tests {
		got, err := SizeToGB(tt.in, tt.def)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", tt.in, err)
		}
		if got != tt.expect {
			t.Errorf("SizeToGB(%q) = %v, want %v", tt.in, got, tt.expect)
		}
	}
}
