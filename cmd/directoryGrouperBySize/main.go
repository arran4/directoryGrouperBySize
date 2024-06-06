package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/arran4/directoryGrouperBySize"
	"os"
)

func main() {
	// Define the flags
	fileFlag := flag.String("f", "", "File to read data from")
	maxSizeFlag := flag.Float64("maxsize", 0, "Maximum size in GB for each disk")
	flag.Parse()

	if *maxSizeFlag <= 0 {
		fmt.Println("Please provide a valid -maxsize argument.")
		return
	}

	var data []string

	// Read from the specified file or stdin
	if *fileFlag != "" {
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
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println("Enter data (CTRL+D to end):")
		for scanner.Scan() {
			data = append(data, scanner.Text())
		}
	}

	animes, err := directoryGrouperBySize.ConvertToStructArray(data)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	var disks [][]directoryGrouperBySize.Anime
	var currentDisk []directoryGrouperBySize.Anime
	var currentDiskSize float64

	for _, anime := range animes {
		if currentDiskSize+anime.SizeInGB > *maxSizeFlag {
			disks = append(disks, currentDisk)
			currentDisk = []directoryGrouperBySize.Anime{}
			currentDiskSize = 0
		}
		currentDisk = append(currentDisk, anime)
		currentDiskSize += anime.SizeInGB
	}

	if len(currentDisk) > 0 {
		disks = append(disks, currentDisk)
	}

	for i, disk := range disks {
		var diskSize float64
		for _, anime := range disk {
			diskSize += anime.SizeInGB
		}
		fmt.Printf("## Disk %d (%.2f GB used, %.2f GB free)\n", i+1, diskSize, *maxSizeFlag-diskSize)
		for _, anime := range disk {
			fmt.Printf("%s\n", anime.Name)
		}
		fmt.Println()
	}
}
