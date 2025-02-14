package main

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func (cfg apiConfig) getAssetPath(videoID uuid.UUID, mediaType string) (string, error) {
	exts, err := mime.ExtensionsByType(mediaType); 
	if err != nil || len(exts) == 0 {
		return "", fmt.Errorf("Extension not found")
	}
	return fmt.Sprintf("%s%s", videoID, exts[0]), nil
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