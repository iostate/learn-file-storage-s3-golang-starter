package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func (cfg apiConfig) getAssetPath(videoID string, mediaType string) (string, error) {
	ext, err := cfg.getExtensionType(mediaType)
	if err != nil {
		fmt.Printf("%s", err)
	}
	return fmt.Sprintf("%s%s", videoID, ext), nil
}

func (cfg apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

func (cfg apiConfig) mediaTypeToExt(mediaType string) string {
	return mime.TypeByExtension(mediaType)
}

func (cfg apiConfig) getAssetURL(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

func (cfg apiConfig) getExtensionType(mediaType string) (string, error) {
	candidates, err := mime.ExtensionsByType(mediaType); 
	// ExtensionsByType will return a slice of all possible extensions
	// for a filetype, if one of them is .mp4, for videos, make it return that
	for _, ext := range candidates {
		if ext == ".mp4" {
			return ext, nil
		}
	}
	if err != nil || len(candidates) == 0 {
		return "", fmt.Errorf("Extension not found")
	}
	return candidates[0], nil
}

type FFProbeOutput struct {
	Streams []struct {
		Width	int		`json:"width`
		Height	int		`json:"height`
		Codec 	string	`json:"codec"`
	}`json:"streams"`
}

// Placing this here since we are passing it an asset disk path
func getVideoAspectRatio(filePath string) (string, error) {
	// run exec.Command to run ffprobe -v error -print_format json -show_streams PATH_TO_VIDEO
	cmd := exec.Command("ffprobe",  "-v", "error", "-print_format", "json", "-show_streams", filePath)
	// write to buffer
	var b bytes.Buffer
	cmd.Stdout = &b

	// Run command
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("Failed to run ffprobe: %w", err)
	}

	// Unmarshal JSON output
	var output FFProbeOutput
	if err := json.Unmarshal(b.Bytes(), &output); err != nil {
		return "", fmt.Errorf("Failed to unmarshal ffprobe out: %w", err)
	}

	if len(output.Streams) > 0 {
		width := output.Streams[0].Width
		height := output.Streams[0].Height

		if width > 0 && height > 0 {
			gcd := GCD(int(width), int(height))
			aspectRatio := closestAspectRatio(width/gcd,height/gcd)
			return fmt.Sprintf("%s", aspectRatio), nil
		}
	}

	return "", fmt.Errorf("Could not determine aspect ratio")
}