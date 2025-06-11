package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/arran4/directoryGrouperBySize"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	// Define the flags
	fileFlag := flag.String("f", "", "File to read data from")
	scanFlag := flag.String("scan", "", "Directory to scan with du -sh")

	var maxSizeGB float64
	flag.Func("maxsize", "Maximum size per disk with optional unit suffix (default GB)", func(s string) error {
		val, err := directoryGrouperBySize.SizeToGB(s, "GB")
		if err != nil {
			return err
		}
		maxSizeGB = val
		return nil
	})
	flag.Parse()

	if maxSizeGB <= 0 {
		fmt.Println("Please provide a valid -maxsize argument.")
		return
	}

	var data []string

	switch {
	case *scanFlag != "":
		entries, err := os.ReadDir(*scanFlag)
		if err != nil {
			fmt.Printf("Error reading directory: %v\n", err)
			return
		}
		for _, e := range entries {
			path := filepath.Join(*scanFlag, e.Name())
			cmd := exec.Command("du", "-sh", path)
			out, err := cmd.Output()
			if err != nil {
				fmt.Printf("Error running du: %v\n", err)
				return
			}
			data = append(data, strings.TrimSpace(string(out)))
		}
	case *fileFlag != "":
		file, err := os.Open(*fileFlag)
		if err != nil {
			fmt.Printf("Error opening file: %v\n", err)
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			data = append(data, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			return
		}
	default:
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println("Enter data (CTRL+D to end):")
		for scanner.Scan() {
			data = append(data, scanner.Text())
		}
	}

	entries, err := directoryGrouperBySize.ConvertToStructArray(data)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	var disks [][]directoryGrouperBySize.Entry
	var currentDisk []directoryGrouperBySize.Entry
	var currentDiskSize float64

	for _, entry := range entries {
		if currentDiskSize+entry.SizeInGB > maxSizeGB {
			disks = append(disks, currentDisk)
			currentDisk = []directoryGrouperBySize.Entry{}
			currentDiskSize = 0
		}
		currentDisk = append(currentDisk, entry)
		currentDiskSize += entry.SizeInGB
	}

	if len(currentDisk) > 0 {
		disks = append(disks, currentDisk)
	}

	for i, disk := range disks {
		var diskSize float64
		for _, entry := range disk {
			diskSize += entry.SizeInGB
		}
		fmt.Printf("## Disk %d (%.2f GB used, %.2f GB free)\n", i+1, diskSize, maxSizeGB-diskSize)
		for _, entry := range disk {
			fmt.Printf("%s\n", entry.Name)
		}
		fmt.Println()
	}
}
