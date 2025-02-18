package main

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"
)



func GCD(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// Define acceptable aspect ratios
var aspectRatios = map[string]float64{
	"16:9": 16.0 / 9.0,
	"9:16": 9.0 / 16.0,
}

// Function to check closest aspect ratio (only 16:9 and 9:16)
func closestAspectRatio(width, height int) string {
	givenRatio := float64(width) / float64(height)
	closest := "other" // Default to "Other"
	minDiff := 0.05    // Tolerance threshold

	for ratioName, ratioValue := range aspectRatios {
		diff := math.Abs(givenRatio - ratioValue)
		if diff < minDiff {
			return ratioName
		}
	}

	return closest
}

func createDirectoryBucketPrefix(directory string) string {
	return directory + "/"
}

func getAspectRatioOrientation(aspectRatio string ) (orientation string) {
	switch (aspectRatio) {
		case "16:9": 
			return "landscape";
		case "9:16": 
			return "portrait";
		}

	return aspectRatio
}

// processVideoForFastStart processes the video and moves the `moov` atom to the start
func processVideoForFastStart(filePath string) (string, error) {
	outputFilePath := filePath + ".processing"

	// Create the ffmpeg command
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputFilePath)


	// Capture both stdout and stderr
	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b
	// Run the command
	if err := cmd.Run(); err != nil {	
		return "", fmt.Errorf("Error creating fast start video: %w\n%s", err, b.String())
	}
	// Print output of ffmpeg in server stdout
	// fmt.Println(b.String()) 

	return outputFilePath, nil
}

func checkFileContainsString(filePath string, searchString string) (bool, error) {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read file content
	data := make([]byte, 1024*1024) // Read first 1MB (adjust size if needed)
	n, err := file.Read(data)
	if err != nil && err.Error() != "EOF" {
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	// Convert to string and check for "moov"
	return strings.Contains(string(data[:n]), searchString), nil
}

// Check to see if moov atom got moved to start of a file
// func hasMoovAtomAtBeginning(videoFilePath string) (bool, err) {
// 	cmd := exec.Command("ffpmeg", "-v", "-error")
// }