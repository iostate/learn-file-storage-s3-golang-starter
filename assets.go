package main

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	// "github.com/google/uuid"
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