package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
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


// Create a new (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) method:
// It should take a video database.Video as input and return a database.Video with the VideoURL field set to a presigned URL and an error (to be returned from the handler)
// It should first split the video.VideoURL on the comma to get the bucket and key
// Then it should use generatePresignedURL to get a presigned URL for the video
// Set the VideoURL field of the video to the presigned URL and return the updated video
func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {

	if video.VideoURL == nil {
		return database.Video{}, fmt.Errorf("video.VideoURL is nil or empty")
	}

	// Split video.VideoURL by comma to get bucket and key
	words := strings.Split(*video.VideoURL, ",")
	if len(words) != 2 {
		return database.Video{}, fmt.Errorf("invalid VideoURL format, expected 'bucket,key'")
	}
	bucket := words[0]
	key := words[1]

	presignedURL, err := generatePresignedURL(cfg.s3client, bucket, key, 3600 * time.Second)
	if err != nil {
		return database.Video{}, fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	video.VideoURL = &presignedURL
	
	return video, nil
}