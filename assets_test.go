package main

import (
	"fmt"
	"testing"
)

func TestGetVideoAspectRatio(t *testing.T) {
	filePath := "/Users/qmtruong92/code/bootdev/learn-file-storage-s3-golang-starter/samples/boots-video-vertical.mp4"
	aspectRatio, err := getVideoAspectRatio(filePath)
	if err != nil {
		fmt.Errorf("Error testing aspect ratio: %w", err)
	}

	// Change the aspect ratio to a string
	// that describes orientation. Orientations can be 
	// "landscape" or "portrait" or "other"
	orientation := getAspectRatioOrientation(aspectRatio)
	fmt.Println(orientation)

	// Test cases
	testCases := [][]int{
		{1920, 1080},  // 16:9
		{1080, 1920},  // 9:16
		{608, 1080},   // Other
		{1000, 1000},  // Other (square)
	}

	for _, tc := range testCases {
		width, height := tc[0], tc[1]
		fmt.Printf("The aspect ratio %d:%d is classified as %s\n", width, height, closestAspectRatio(width, height))
	}
	
}