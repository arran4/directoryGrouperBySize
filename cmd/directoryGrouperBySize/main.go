package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/arran4/directoryGrouperBySize"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, execCommand); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// execCommand is a variable to allow injection of a mock for testing du.
var execCommand = func(name string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)
	return cmd.Output()
}

type execCmdFunc func(string, ...string) ([]byte, error)

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cmdRunner execCmdFunc) error {
	fs := flag.NewFlagSet("directoryGrouperBySize", flag.ContinueOnError)
	fs.SetOutput(stderr)

	versionFlag := fs.Bool("version", false, "Print version information and exit")
	fileFlag := fs.String("f", "", "File to read data from")
	scanFlag := fs.String("scan", "", "Directory to scan with du -sh")

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

	var data []string

	switch {
	case *scanFlag != "":
		entries, err := os.ReadDir(*scanFlag)
		if err != nil {
			return fmt.Errorf("error reading directory: %v", err)
		}
		for _, e := range entries {
			path := filepath.Join(*scanFlag, e.Name())
			out, err := cmdRunner("du", "-sh", path)
			if err != nil {
				return fmt.Errorf("error running du: %v", err)
			}
			// Only trim newlines, preserving any trailing spaces in the filename
			trimmedOut := strings.TrimRight(string(out), "\r\n")
			data = append(data, trimmedOut)
		}
	case *fileFlag != "":
		file, err := os.Open(*fileFlag)
		if err != nil {
			return fmt.Errorf("error opening file: %v", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			data = append(data, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error reading file: %v", err)
		}
	default:
		// Stdin
		scanner := bufio.NewScanner(stdin)
		for scanner.Scan() {
			data = append(data, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error reading stdin: %v", err)
		}
	}

	entries, err := directoryGrouperBySize.ConvertToStructArray(data)
	if err != nil {
		return fmt.Errorf("error converting input: %v", err)
	}

	disks, err := directoryGrouperBySize.Group(entries, maxSizeBytes)
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
