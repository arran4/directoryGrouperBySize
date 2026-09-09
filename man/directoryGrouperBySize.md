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

# EXAMPLE

`directoryGrouperBySize -maxsize 55G -f dirs.txt`

`directoryGrouperBySize -maxsize 55G -scan /media`

# SEE ALSO

`du(1)`

### Limitations

- **Filename Line-Oriented Limitations**: When using piped input or reading from a file (`-f`), `directoryGrouperBySize` relies on line-oriented input parsing. Therefore, filenames containing newline characters (`\n`) cannot be reliably parsed. Using the `-scan` flag avoids this limitation.

# SCAN SEMANTICS

The `-scan` flag uses an internal Go scanning backend instead of an external `du` command to guarantee exact cross-platform consistency.

*   **Size calculation:** Uses logical/apparent file sizes (bytes), not allocated filesystem block usage.
*   **Symlinks:** Symlinks are not followed either inside or outside the hierarchy. A symlink's size is counted only as its own link length.
*   **Hard links:** Hard links are counted individually per directory entry encountered.
*   **Permission errors:** The scan fails cleanly and immediately if it encounters unreadable files or directories.
*   **Names:** Preserves exact filenames as provided by the filesystem (including spaces and newlines).
