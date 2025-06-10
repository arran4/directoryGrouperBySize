package directoryGrouperBySize

import "testing"

func TestConvertToStructArray(t *testing.T) {
	input := []string{"1G foo", "500M bar"}
	got, err := ConvertToStructArray(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0].Name != "foo" || got[0].SizeInGB != 1 {
		t.Errorf("unexpected first item: %+v", got[0])
	}
	expectedSize := 500.0 / 1024.0
	if got[1].Name != "bar" || got[1].SizeInGB != expectedSize {
		t.Errorf("unexpected second item: %+v", got[1])
	}
}

func TestConvertToStructArrayInvalid(t *testing.T) {
	_, err := ConvertToStructArray([]string{"invalidline"})
	if err == nil {
		t.Fatal("expected error for invalid input")
	}
}
