package directoryGrouperBySize

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
)

//go:generate ./generate.sh

// DirectoryGrouperBySize is a subcommand `directoryGrouperBySize` that groups entries into virtual disks
//
// Flags:
//
//	maxsize: (-maxsize; default: "") Maximum size per disk with optional unit suffix (default GB)
//	f: (-f; default: "") File to read data from
//	scan: (-scan; default: "") Directory to scan internally (logical/apparent byte sizes)
//	strategy: (-strategy; default: "first-fit-decreasing") Grouping algorithm strategy to use
//	null: (-0; --null; default: false) Use zero bytes to separate records
//	versionFlag: (-version; default: false) Print version information and exit
//
// Supported strategy values:
//
// first-fit-decreasing — reorder inputs for efficient bin packing.
// next-fit — preserve the original input order.
func DirectoryGrouperBySize(
	maxsize string,
	f string,
	scan string,
	strategy string,
	null bool,
	version bool,
) error {
	return runImpl(maxsize, f, scan, strategy, null, version, os.Stdin, os.Stdout)
}

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func runImpl(
	maxsize string,
	f string,
	scan string,
	strategy string,
	null bool,
	versionFlag bool,
	stdin io.Reader,
	stdout io.Writer,
) (retErr error) {
	if versionFlag {
		if _, err := fmt.Fprintf(stdout, "directoryGrouperBySize %s\nCommit: %s\nDate: %s\n", version, commit, date); err != nil {
			return fmt.Errorf("error writing version output: %w", err)
		}
		return nil
	}

	var maxSizeBytes int64
	if maxsize != "" {
		val, err := ParseSize(maxsize, "GB")
		if err != nil {
			return err
		}
		maxSizeBytes = val
	}

	if maxSizeBytes <= 0 {
		return fmt.Errorf("please provide a valid -maxsize argument")
	}

	if f != "" && scan != "" {
		return fmt.Errorf("cannot use both -f and -scan flags together")
	}

	var entries []Entry

	if scan != "" {
		var err error
		entries, err = ScanDirectory(context.Background(), scan)
		if err != nil {
			return fmt.Errorf("error scanning directory: %w", err)
		}
	} else {
		nulMode := null

		var data []string
		var reader io.Reader
		if f != "" {
			file, err := os.Open(f)
			if err != nil {
				return fmt.Errorf("error opening file: %w", err)
			}
			defer func() {
				if err := file.Close(); err != nil && retErr == nil {
					retErr = fmt.Errorf("error closing input file: %w", err)
				}
			}()
			reader = file
		} else {
			reader = stdin
		}

		if nulMode {
			bufReader := bufio.NewReader(reader)
			for {
				record, err := bufReader.ReadString('\x00')
				if err != nil {
					if err == io.EOF {
						if len(record) > 0 {
							return fmt.Errorf("malformed input: missing NUL terminator at end of file")
						}
						break
					}
					return fmt.Errorf("error reading input: %w", err)
				}
				data = append(data, record[:len(record)-1]) // strip the NUL byte
			}
		} else {
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				data = append(data, scanner.Text())
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("error reading input: %w", err)
			}
		}

		var err error
		if nulMode {
			entries, err = ConvertToStructArrayNULMode(data)
		} else {
			entries, err = ConvertToStructArray(data)
		}
		if err != nil {
			return fmt.Errorf("error converting input: %w", err)
		}
	}

	disks, err := GroupWithStrategy(entries, maxSizeBytes, strategy)
	if err != nil {
		return fmt.Errorf("grouping failed: %w", err)
	}

	for i, disk := range disks {
		var diskSizeBytes int64
		for _, entry := range disk {
			diskSizeBytes += entry.SizeBytes
		}

		usedGB := float64(diskSizeBytes) / (1024 * 1024 * 1024)
		freeGB := float64(maxSizeBytes-diskSizeBytes) / (1024 * 1024 * 1024)
		if _, err := fmt.Fprintf(stdout, "## Disk %d (%.2f GB used, %.2f GB free)\n", i+1, usedGB, freeGB); err != nil {
			return fmt.Errorf("error writing disk header: %w", err)
		}
		for _, entry := range disk {
			if _, err := fmt.Fprintf(stdout, "%s\n", entry.Name); err != nil {
				return fmt.Errorf("error writing entry: %w", err)
			}
		}
		if _, err := fmt.Fprintln(stdout); err != nil {
			return fmt.Errorf("error writing disk separator: %w", err)
		}
	}

	return nil
}

// SetIOMocks for testing
var RunImpl = runImpl
