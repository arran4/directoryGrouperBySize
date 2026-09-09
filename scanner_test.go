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
			// Expected logical size is precisely the sum of non-directory files inside.
			if entry.SizeBytes != 40 {
				t.Errorf("expected exactly 40 bytes for sub1, got %d", entry.SizeBytes)
			}
		} else {
			t.Errorf("unexpected entry %q", entry.Name)
		}
	}

	// 4. Symlink semantics (Immediate and Nested)
	symlinkDir := filepath.Join(tmpDir, "symlinks")
	os.Mkdir(symlinkDir, 0755)

	// Create a target directory and file
	targetDir := filepath.Join(symlinkDir, "target_dir")
	os.Mkdir(targetDir, 0755)
	os.WriteFile(filepath.Join(targetDir, "huge.txt"), make([]byte, 500), 0644)

	targetFile := filepath.Join(symlinkDir, "target.txt")
	os.WriteFile(targetFile, make([]byte, 100), 0644)

	// Create symlinks
	linkPath := filepath.Join(symlinkDir, "link.txt")
	dirLinkPath := filepath.Join(symlinkDir, "dir_link")

	errFileSym := os.Symlink("target.txt", linkPath)
	errDirSym := os.Symlink("target_dir", dirLinkPath)

	if errFileSym == nil && errDirSym == nil {
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
			if entry.Name == "dir_link" {
				if entry.SizeBytes >= 500 {
					t.Errorf("symlink to dir should NOT traverse target. Got %d bytes (expected tiny link size)", entry.SizeBytes)
				}
			}
		}
	} else {
		t.Logf("skipping symlink test as symlink creation failed (common on Windows without admin): file=%v dir=%v", errFileSym, errDirSym)
	}

	// 5. Unreadable directory (permission error)
	unreadableDir := filepath.Join(tmpDir, "unreadable")
	os.Mkdir(unreadableDir, 0755)
	subUnreadable := filepath.Join(unreadableDir, "sub")
	os.Mkdir(subUnreadable, 0000) // remove permissions

	// Verify if environment actually honors 0000
	_, verifyErr := os.ReadDir(subUnreadable)

	if verifyErr != nil {
		_, err = ScanDirectory(ctx, unreadableDir)
		if err == nil {
			t.Errorf("expected error scanning unreadable directory, got none")
		}
	} else {
		t.Logf("Skipping strict permission error test. Environment (e.g. Windows/Root) does not restrict reading dir with 0000 perms.")
	}

	// restore permissions so os.RemoveAll can clean it up later
	os.Chmod(subUnreadable, 0755)

	// 6. Context cancellation
	cancelDir := filepath.Join(tmpDir, "cancel")
	os.Mkdir(cancelDir, 0755)
	os.WriteFile(filepath.Join(cancelDir, "a.txt"), make([]byte, 10), 0644)

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err = ScanDirectory(cancelCtx, cancelDir)
	if err == nil || err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}

	// 7. Large file
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
