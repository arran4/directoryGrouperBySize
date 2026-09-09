# directoryGrouperBySize

`directoryGrouperBySize` helps plan how data will fit onto fixed-size storage.
It reads a `du -sh` style listing and groups entries into virtual disks up to a
specified size.

## Installation

Pre-built binaries are available on the
[releases page](https://github.com/arran4/directoryGrouperBySize/releases).
To build from source you will need Go 1.22 or newer:

```bash
go install github.com/arran4/directoryGrouperBySize/cmd/directoryGrouperBySize@latest
```

You can also clone the repository and build it manually:

```bash
git clone https://github.com/arran4/directoryGrouperBySize.git
cd directoryGrouperBySize
go build ./cmd/directoryGrouperBySize
```

## Usage

```bash
directoryGrouperBySize -maxsize 55G -f input.txt
```

Or let the tool run `du` for you:

```bash
directoryGrouperBySize -maxsize 55G -scan /media
```

You can also pipe data directly from `du`:

```bash
du -sh * | directoryGrouperBySize -maxsize 55G
```

**Options**

| Flag        | Description                                       |
|-------------|---------------------------------------------------|
| `-maxsize`  | Maximum size for each group. Accepts G/GB/GiB, M/MB/MiB etc. Without a suffix GB is assumed. (required)|
| `-f`        | Path to input file. If omitted, data is read from stdin |
| `-scan`     | Run `du -sh` on this directory instead of reading input |
| `-strategy` | Grouping algorithm strategy to use: `first-fit-decreasing` (default) or `next-fit`. |

**Grouping Strategies:**
By default, the tool uses a **first-fit-decreasing** strategy (an optimized First-Fit Decreasing algorithm implemented via a segment tree for deterministic O(n log n) total time). This reorders your input entries from largest to smallest to minimize the total number of disks and reduce wasted capacity efficiently even on very large directory listings. It is a heuristic and does not guarantee the absolute optimal disk count, but performs well. If preserving the original input order is critical to your use case, use `-strategy next-fit`, which fills disks sequentially in O(n) time. The alias `best-fit` is supported as a legacy compatibility alias for `first-fit-decreasing`.

Input should match the output of `du -sh`. Units are case-insensitive and may include an optional `B` or `iB`.

**Note on unit semantics and API changes:**
To provide exact boundary guarantees when grouping, sizes are stored and compared exactly using integers (`int64` bytes). Because standard `du -sh` output traditionally reports sizes using binary units (powers of 1024) but labels them without the "i" (e.g. `1G` or `1GB`), `directoryGrouperBySize` continues to interpret K, KB, M, MB, G, GB, T, and TB as 1024-based binary sizes to maintain compatibility with scripts and `du` itself. Unambiguous IEC suffixes (KiB, MiB, GiB, TiB) are also accepted and processed exactly the same way. The exported API (`Entry.SizeInGB` and `SizeToGB`) has been updated to use explicit `int64` bytes (`Entry.SizeBytes` and `ParseSize`) to support this exact bounding logic.

For example:

```text
25G    Movies
18G    TVShows
8G     Music
5G     Documents
3.6G   FileFolder1
1.6G   FileFolder2
27G    FileFolder3
300M   Temp
```

### Example output

```
## Disk 1 (51.00 GB used, 4.00 GB free)
Movies
TVShows
Music

## Disk 2 (37.49 GB used, 17.51 GB free)
Documents
FileFolder1
FileFolder2
FileFolder3
Temp
```

Larger listings will be divided across multiple disks:

```
## Disk 1 (55.00 GB used, 0.00 GB free)
Video1
Video2

## Disk 2 (40.00 GB used, 15.00 GB free)
Video3
```


Release packages include a manual page installable via `man directoryGrouperBySize`.
If you build from source, generate the man page with:

```bash
go install github.com/cpuguy83/go-md2man/v2@latest
go-md2man -in man/directoryGrouperBySize.md -out directoryGrouperBySize.1
sudo mv directoryGrouperBySize.1 /usr/share/man/man1/
```

## Development

Run tests with:

```bash
go test ./...
```

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

### Limitations
- **Filename Line-Oriented Limitations**: `directoryGrouperBySize` relies on line-oriented input, similar to standard utilities like `du`. Therefore, filenames containing newline characters (`\n`) cannot be correctly parsed when using piped input or reading from a file (`-f`).
- **Input Sources**: The `-f` and `-scan` flags are mutually exclusive. Choose only one input source.
