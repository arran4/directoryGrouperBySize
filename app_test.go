package directoryGrouperBySize

import (
	"bytes"
	"fmt"
	"testing"
)

type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func TestRunImpl_StdinReadError(t *testing.T) {
	var stdout bytes.Buffer
	err := runImpl("2G", "", "", "first-fit-decreasing", false, false, &errorReader{fmt.Errorf("simulated read error")}, &stdout)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("simulated read error")) {
		t.Fatalf("expected error to contain 'simulated read error', got %v", err)
	}
}
