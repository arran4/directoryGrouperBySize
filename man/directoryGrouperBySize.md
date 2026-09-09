% directoryGrouperBySize(1)
# NAME

directoryGrouperBySize - group directory listings into disks of a target size

# SYNOPSIS

`directoryGrouperBySize -maxsize <size> [-f file] [-scan dir]`

# DESCRIPTION

`directoryGrouperBySize` reads a list of directory sizes, typically from `du -sh`, and groups entries into virtual disks up to the specified size exactly. To maintain compatibility with `du`, suffixes like G and M are processed using binary 1024-based multipliers. IEC standard suffixes like GiB and MiB are also supported.

# OPTIONS

`-maxsize`  Maximum size for each disk. Accepts suffixes (G/GB/GiB, M/MB/MiB, K/KB/KiB, T/TB/TiB) and defaults to gigabytes when omitted. (required).

`-f`  Path to file to read listing from. If omitted, standard input is used.

`-scan`  Run an internal directory scan on the specified directory instead of reading input.

`-strategy`  Grouping algorithm strategy to use. `first-fit-decreasing` (default) reorders input from largest to smallest to efficiently minimize disks using an O(n log n) segment-tree heuristic. `next-fit` preserves the original input sequence sequentially in O(n) time. `best-fit` is supported as a legacy alias for `first-fit-decreasing`.

`-0`, `--null`  Read NUL-delimited records instead of newline-delimited, safely preserving filenames containing newlines.

# EXAMPLE

`directoryGrouperBySize -maxsize 55G -f dirs.txt`

`directoryGrouperBySize -maxsize 55G -scan /media`

`find . -maxdepth 1 -print0 | xargs -0 du -sh -0 | directoryGrouperBySize -maxsize 55G -0`

# SEE ALSO

`du(1)`

### Using NUL-Delimited Input
The existing line-oriented mode (without `-0` or `--null`) separates entries with newlines. This limits the safe representation of filenames that contain embedded newline characters.
To parse filenames with embedded newlines, use a NUL-delimited format and pass the `-0` or `--null` flag, like the `du -0` example above.

### Limitations

- **Filename Line-Oriented Limitations**: When using piped input or reading from a file (`-f`) without `-0` / `--null`, `directoryGrouperBySize` relies on line-oriented input parsing. Therefore, filenames containing newline characters (`\n`) cannot be reliably parsed. Using the `-0` / `--null` flag or the `-scan` flag avoids this limitation.

# SCAN SEMANTICS

The `-scan` flag uses an internal Go scanning backend instead of an external `du` command. This native-scanning is a separate follow-up tracked in #11. Its deterministic logical-size contract is explicitly defined as:

*   **Size calculation:** Uses logical/apparent file sizes (bytes), not allocated filesystem block usage. Regular files contribute their exact logical byte length.
*   **Directories:** Directory metadata sizes are excluded; a directory's size is the exact recursive sum of its non-directory entries.
*   **Symlinks:** Symlinks are not followed either inside or outside the hierarchy. They contribute their platform/filesystem-dependent link metadata size.
*   **Cross-platform determinism:** Results are deterministic for a given filesystem. Exact byte-for-byte cross-platform equality is guaranteed for directory trees containing purely regular files and directories, but may vary when symlinks or other non-regular entries report platform-dependent metadata sizes.
*   **Hard links:** Hard links are counted individually per directory entry encountered.
*   **Permission errors:** The scan fails cleanly and immediately if it encounters unreadable files or directories.
*   **Names:** Preserves exact filenames as provided by the filesystem (including spaces and newlines).
