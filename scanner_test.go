package directoryGrouperBySize

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"
)

func mustMkdir(t *testing.T, path string, perm os.FileMode) {
	t.Helper()
	if err := os.Mkdir(path, perm); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path string, data []byte, perm os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, data, perm); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestScanFS(t *testing.T) {
	ctx := context.Background()

	// 1. Empty root
	emptyFS := fstest.MapFS{}
	entries, err := scanFS(ctx, emptyFS, "empty")
	if err != nil {
		t.Errorf("expected no error for empty dir, got %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty dir, got %d", len(entries))
	}

	// 2. Immediate files, unusual characters
	filesFS := fstest.MapFS{
		"a.txt":                       &fstest.MapFile{Data: make([]byte, 10)},
		" spaces and \n newlines.txt": &fstest.MapFile{Data: make([]byte, 20)},
		"c.dat":                       &fstest.MapFile{Data: make([]byte, 30)},
	}
	entries, err = scanFS(ctx, filesFS, "files")
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
	nestedFS := fstest.MapFS{
		"sub1/file1.txt":      &fstest.MapFile{Data: make([]byte, 15)},
		"sub1/sub2/file2.txt": &fstest.MapFile{Data: make([]byte, 25)},
		"file3.txt":           &fstest.MapFile{Data: make([]byte, 5)},
	}
	entries, err = scanFS(ctx, nestedFS, "nested")
	if err != nil {
		t.Errorf("expected no error for nested dir, got %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries (file3.txt and sub1), got %d", len(entries))
	}

	for _, entry := range entries {
		switch entry.Name {
		case "file3.txt":
			if entry.SizeBytes != 5 {
				t.Errorf("expected 5 bytes for file3.txt, got %d", entry.SizeBytes)
			}
		case "sub1":
			// Expected logical size is precisely the sum of non-directory files inside.
			if entry.SizeBytes != 40 {
				t.Errorf("expected exactly 40 bytes for sub1, got %d", entry.SizeBytes)
			}
		default:
			t.Errorf("unexpected entry %q", entry.Name)
		}
	}

	// 4. Context cancellation
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	_, err = scanFS(cancelCtx, filesFS, "cancel")
	if err == nil || err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}

	// 5. Large apparent file sizes
	largeFS := largeVirtualFS{}
	entries, err = scanFS(ctx, largeFS, "large")
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

// Custom FS implementation to simulate a 5GB file without memory allocation
type largeVirtualFS struct{}

func (largeVirtualFS) Open(name string) (fs.File, error) {
	if name == "." {
		return dirFile{}, nil
	}
	if name == "large.dat" {
		return largeFile{}, nil
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}
func (largeVirtualFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == "." {
		return []fs.DirEntry{fs.FileInfoToDirEntry(largeInfo{})}, nil
	}
	return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
}

type dirFile struct{}

func (dirFile) Stat() (fs.FileInfo, error) { return dirInfo{}, nil }
func (dirFile) Read([]byte) (int, error)   { return 0, os.ErrInvalid }
func (dirFile) Close() error               { return nil }

type dirInfo struct{}

func (dirInfo) Name() string       { return "." }
func (dirInfo) Size() int64        { return 0 }
func (dirInfo) Mode() os.FileMode  { return fs.ModeDir | 0755 }
func (dirInfo) ModTime() time.Time { return time.Time{} }
func (dirInfo) IsDir() bool        { return true }
func (dirInfo) Sys() any           { return nil }

type largeFile struct{}

func (largeFile) Stat() (fs.FileInfo, error) { return largeInfo{}, nil }
func (largeFile) Read([]byte) (int, error)   { return 0, nil }
func (largeFile) Close() error               { return nil }

type largeInfo struct{}

func (largeInfo) Name() string       { return "large.dat" }
func (largeInfo) Size() int64        { return 5 * 1024 * 1024 * 1024 }
func (largeInfo) Mode() os.FileMode  { return 0644 }
func (largeInfo) ModTime() time.Time { return time.Time{} }
func (largeInfo) IsDir() bool        { return false }
func (largeInfo) Sys() any           { return nil }

func TestScanDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	var err error

	ctx := context.Background()

	// Symlink semantics (Immediate and Nested)
	symlinkDir := filepath.Join(tmpDir, "symlinks")
	mustMkdir(t, symlinkDir, 0755)

	// Create a target directory and file outside of the scanned child directory
	targetDir := filepath.Join(symlinkDir, "target_dir")
	mustMkdir(t, targetDir, 0755)
	mustWriteFile(t, filepath.Join(targetDir, "huge.txt"), make([]byte, 500), 0644)

	targetFile := filepath.Join(symlinkDir, "target.txt")
	mustWriteFile(t, targetFile, make([]byte, 100), 0644)

	// Create an immediate child directory that contains symlinks
	nestedChildDir := filepath.Join(symlinkDir, "nested_child")
	mustMkdir(t, nestedChildDir, 0755)

	// Create symlinks inside the nested child to trigger WalkDir behavior
	linkPath := filepath.Join(nestedChildDir, "link.txt")
	dirLinkPath := filepath.Join(nestedChildDir, "dir_link")

	// Target files are in parent dir relative to symlinks
	errFileSym := os.Symlink("../target.txt", linkPath)
	errDirSym := os.Symlink("../target_dir", dirLinkPath)

	if errFileSym == nil && errDirSym == nil {
		entries, err := ScanDirectory(ctx, symlinkDir)
		if err != nil {
			t.Errorf("expected no error for symlink dir, got %v", err)
		}

		for _, entry := range entries {
			if entry.Name == "nested_child" {
				if entry.SizeBytes >= 500 {
					t.Errorf("symlink to dir inside nested_child should NOT traverse target recursively. Got %d bytes (expected tiny combined link sizes)", entry.SizeBytes)
				}
			}
		}
	} else {
		t.Logf("skipping symlink test as symlink creation failed (common on Windows without admin): file=%v dir=%v", errFileSym, errDirSym)
	}

	// Unreadable directory (permission error)
	unreadableDir := filepath.Join(tmpDir, "unreadable")
	mustMkdir(t, unreadableDir, 0755)
	subUnreadable := filepath.Join(unreadableDir, "sub")
	mustMkdir(t, subUnreadable, 0000) // remove permissions

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

	// Restore permissions so the temporary directory cleanup can remove it later.
	if err := os.Chmod(subUnreadable, 0755); err != nil {
		t.Fatalf("restore permissions on unreadable directory: %v", err)
	}

	// Large file
	largeDir := filepath.Join(tmpDir, "large")
	mustMkdir(t, largeDir, 0755)
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
	if _, err := f.Write([]byte{1}); err != nil {
		closeErr := f.Close()
		if closeErr != nil {
			t.Errorf("close large file after write failure: %v", closeErr)
		}
		t.Fatalf("failed to write sparse file tail byte: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close large file: %v", err)
	}

	entries, err := ScanDirectory(ctx, largeDir)
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
