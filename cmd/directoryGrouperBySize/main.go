package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/arran4/directoryGrouperBySize"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("directoryGrouperBySize", flag.ContinueOnError)
	fs.SetOutput(stderr)

	versionFlag := fs.Bool("version", false, "Print version information and exit")
	fileFlag := fs.String("f", "", "File to read data from")
	scanFlag := fs.String("scan", "", "Directory to scan with du -sh")
	strategyFlag := fs.String("strategy", "first-fit-decreasing", "Grouping algorithm strategy to use: first-fit-decreasing (default, reorders inputs for efficient bin packing) or next-fit (preserves original input order)")
	nulFlag0 := fs.Bool("0", false, "Read NUL-delimited records instead of newline-delimited (for filenames with newlines)")
	nulFlagNull := fs.Bool("null", false, "Read NUL-delimited records instead of newline-delimited (for filenames with newlines)")

	var maxSizeBytes int64
	fs.Func("maxsize", "Maximum size per disk with optional unit suffix (default GB)", func(s string) error {
		val, err := directoryGrouperBySize.ParseSize(s, "GB")
		if err != nil {
			return err
		}
		maxSizeBytes = val
		return nil
	})

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *versionFlag {
		fmt.Fprintf(stdout, "directoryGrouperBySize %s\nCommit: %s\nDate: %s\n", version, commit, date)
		return nil
	}

	if maxSizeBytes <= 0 {
		return fmt.Errorf("please provide a valid -maxsize argument")
	}

	if *fileFlag != "" && *scanFlag != "" {
		return fmt.Errorf("cannot use both -f and -scan flags together")
	}

	var entries []directoryGrouperBySize.Entry

	if *scanFlag != "" {
		var err error
		entries, err = directoryGrouperBySize.ScanDirectory(context.Background(), *scanFlag)
		if err != nil {
			return fmt.Errorf("error scanning directory: %v", err)
		}
	} else {
		nulMode := *nulFlag0 || *nulFlagNull

		var data []string
		var reader io.Reader
		if *fileFlag != "" {
			file, err := os.Open(*fileFlag)
			if err != nil {
				return fmt.Errorf("error opening file: %v", err)
			}
			defer file.Close()
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
					return fmt.Errorf("error reading input: %v", err)
				}
				data = append(data, record[:len(record)-1]) // strip the NUL byte
			}
		} else {
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				data = append(data, scanner.Text())
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("error reading input: %v", err)
			}
		}

		var err error
		entries, err = directoryGrouperBySize.ConvertToStructArray(data, nulMode)
		if err != nil {
			return fmt.Errorf("error converting input: %v", err)
		}
	}

	disks, err := directoryGrouperBySize.GroupWithStrategy(entries, maxSizeBytes, *strategyFlag)
	if err != nil {
		return fmt.Errorf("grouping failed: %v", err)
	}

	for i, disk := range disks {
		var diskSizeBytes int64
		for _, entry := range disk {
			diskSizeBytes += entry.SizeBytes
		}

		usedGB := float64(diskSizeBytes) / (1024 * 1024 * 1024)
		freeGB := float64(maxSizeBytes-diskSizeBytes) / (1024 * 1024 * 1024)
		fmt.Fprintf(stdout, "## Disk %d (%.2f GB used, %.2f GB free)\n", i+1, usedGB, freeGB)
		for _, entry := range disk {
			fmt.Fprintf(stdout, "%s\n", entry.Name)
		}
		fmt.Fprintln(stdout)
	}

	return nil
}
