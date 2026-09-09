package directoryGrouperBySize

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ScanDirectory enumerates immediate children of the requested directory and
// computes the apparent (logical) size of each child in exact bytes.
// It returns entries directly in the repository's canonical exact size representation,
// avoiding human-readable string round-tripping.
//
// Size calculation semantics:
//   - Uses logical/apparent file sizes (bytes), not allocated filesystem block usage.
//     Regular files contribute their exact logical byte length.
//   - Directory sizes are calculated as the recursive sum of non-directory entries,
//     excluding directory metadata itself.
//   - Symlinks are not followed either inside or outside the hierarchy. They contribute
//     their platform/filesystem-dependent link metadata size.
//   - Hard links are counted individually per directory entry encountered.
//   - Fails cleanly and immediately on permission errors (does not silently omit capacities).
//   - Preserves exact filenames as provided by the filesystem (including spaces and newlines).
//   - Results are deterministic for a given filesystem. Exact byte-for-byte cross-platform
//     equality is guaranteed for directory trees containing purely regular files and directories,
//     but may vary when symlinks or other non-regular entries are present.
func ScanDirectory(ctx context.Context, root string) ([]Entry, error) {
	dirEntries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("error reading directory %q: %w", root, err)
	}

	var results []Entry

	for _, de := range dirEntries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		childPath := filepath.Join(root, de.Name())

		info, err := de.Info()
		if err != nil {
			return nil, fmt.Errorf("error getting info for %q: %w", childPath, err)
		}

		var totalSize int64

		if info.IsDir() {
			// Calculate size of directory recursively
			err := filepath.WalkDir(childPath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return fmt.Errorf("error accessing path %q: %w", path, err)
				}

				if err := ctx.Err(); err != nil {
					return err
				}

				if !d.IsDir() {
					dInfo, err := d.Info()
					if err != nil {
						return fmt.Errorf("error getting info for %q: %w", path, err)
					}

					totalSize += dInfo.Size()
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			totalSize = info.Size()
		}

		results = append(results, Entry{
			Name:      de.Name(),
			SizeBytes: totalSize,
		})
	}

	// os.ReadDir returns entries sorted by filename, ensuring a deterministic order.
	return results, nil
}
