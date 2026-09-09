package directoryGrouperBySize

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScanDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "scanner_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// 1. Empty root
	emptyDir := filepath.Join(tmpDir, "empty")
	os.Mkdir(emptyDir, 0755)
	entries, err := ScanDirectory(ctx, emptyDir)
	if err != nil {
		t.Errorf("expected no error for empty dir, got %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty dir, got %d", len(entries))
	}

	// 2. Immediate files, unusual characters
	filesDir := filepath.Join(tmpDir, "files")
	os.Mkdir(filesDir, 0755)
	fileNames := []string{"a.txt", " spaces and \n newlines.txt", "c.dat"}
	fileSizes := []int64{10, 20, 30}
	for i, name := range fileNames {
		path := filepath.Join(filesDir, name)
		data := make([]byte, fileSizes[i])
		os.WriteFile(path, data, 0644)
	}

	entries, err = ScanDirectory(ctx, filesDir)
	if err != nil {
		t.Errorf("expected no error for files dir, got %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}

	// Entries should be sorted alphabetically by name
	expectedNames := []string{" spaces and \n newlines.txt", "a.txt", "c.dat"}
	expectedSizes := []int64{20, 10, 30}
	for i, entry := range entries {
		if entry.Name != expectedNames[i] {
			t.Errorf("expected name %q at index %d, got %q", expectedNames[i], i, entry.Name)
		}
		if entry.SizeBytes != expectedSizes[i] {
			t.Errorf("expected size %d at index %d, got %d", expectedSizes[i], i, entry.SizeBytes)
		}
	}

	// 3. Nested directories
	nestedDir := filepath.Join(tmpDir, "nested")
	os.MkdirAll(filepath.Join(nestedDir, "sub1", "sub2"), 0755)

	// Add file in sub1: 15 bytes
	os.WriteFile(filepath.Join(nestedDir, "sub1", "file1.txt"), make([]byte, 15), 0644)
	// Add file in sub2: 25 bytes
	os.WriteFile(filepath.Join(nestedDir, "sub1", "sub2", "file2.txt"), make([]byte, 25), 0644)
	// Add file in nestedDir (immediate): 5 bytes
	os.WriteFile(filepath.Join(nestedDir, "file3.txt"), make([]byte, 5), 0644)

	entries, err = ScanDirectory(ctx, nestedDir)
	if err != nil {
		t.Errorf("expected no error for nested dir, got %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries (file3.txt and sub1), got %d", len(entries))
	}

	for _, entry := range entries {
		if entry.Name == "file3.txt" {
			if entry.SizeBytes != 5 {
				t.Errorf("expected 5 bytes for file3.txt, got %d", entry.SizeBytes)
			}
		} else if entry.Name == "sub1" {
			// directory apparent size usually includes 4096 bytes on Linux for the dir itself.
			// Let's just check that it's at least 15+25 = 40.
			if entry.SizeBytes < 40 {
				t.Errorf("expected at least 40 bytes for sub1, got %d", entry.SizeBytes)
			}
		} else {
			t.Errorf("unexpected entry %q", entry.Name)
		}
	}

	// 4. Symlink semantics
	symlinkDir := filepath.Join(tmpDir, "symlinks")
	os.Mkdir(symlinkDir, 0755)
	targetFile := filepath.Join(symlinkDir, "target.txt")
	os.WriteFile(targetFile, make([]byte, 100), 0644)
	linkPath := filepath.Join(symlinkDir, "link.txt")
	err = os.Symlink("target.txt", linkPath)
	if err == nil {
		entries, err = ScanDirectory(ctx, symlinkDir)
		if err != nil {
			t.Errorf("expected no error for symlink dir, got %v", err)
		}

		for _, entry := range entries {
			if entry.Name == "link.txt" {
				if entry.SizeBytes >= 100 {
					t.Errorf("symlink should have size of link itself, not target. Got %d bytes", entry.SizeBytes)
				}
			}
		}
	} else {
		t.Logf("skipping symlink test as symlink creation failed (common on Windows without admin): %v", err)
	}

	// 5. Unreadable directory (permission error)
	// This test might not work as expected on Windows or if running as root
	unreadableDir := filepath.Join(tmpDir, "unreadable")
	os.Mkdir(unreadableDir, 0755)
	subUnreadable := filepath.Join(unreadableDir, "sub")
	os.Mkdir(subUnreadable, 0000) // no permissions

	_, err = ScanDirectory(ctx, unreadableDir)
	if err == nil {
		t.Logf("Expected error scanning unreadable directory, got none. This can happen on Windows or when running tests as root.")
	} else {
		t.Logf("Got expected error scanning unreadable directory: %v", err)
	}

	// restore permissions so os.RemoveAll can clean it up later
	os.Chmod(subUnreadable, 0755)

	// 6. Large file
	largeDir := filepath.Join(tmpDir, "large")
	os.Mkdir(largeDir, 0755)
	largeFile := filepath.Join(largeDir, "large.dat")

	// create a sparse file (or just write a few bytes at a large offset)
	f, err := os.Create(largeFile)
	if err != nil {
		t.Fatalf("failed to create large file: %v", err)
	}
	// 5GB size
	_, err = f.Seek(5*1024*1024*1024-1, 0)
	if err != nil {
		t.Fatalf("failed to seek: %v", err)
	}
	f.Write([]byte{1})
	f.Close()

	entries, err = ScanDirectory(ctx, largeDir)
	if err != nil {
		t.Errorf("expected no error for large file, got %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].SizeBytes != 5*1024*1024*1024 {
		t.Errorf("expected 5GB exactly, got %d", entries[0].SizeBytes)
	}
}
