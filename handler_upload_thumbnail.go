package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	// AUTH
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	// AUTH
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	// Allow 10 MB for form parsing (including file uploads)
	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
        http.Error(w, "Failed to parse form", http.StatusBadRequest)
        return
    }

	// return first file with form key "thumbnail"
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	// read image data
	imageData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to read file", err)
		return
	}

	// used to get extension type
	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", err)
		return
	}

	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", err)
		return
	}

	// BEGIN
	// Create the /assets disk path
	// Create new file path using crypto/rand.Read 
	// to fill a 32-byte slice with random bytes

	// TODO: Separate this into its own function for use elsewhere
	// Used for cache busting
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error generating random bytes", err)
		return
	}

	// create asset path 
	assetPath, err := cfg.getAssetPath(base64.RawURLEncoding.EncodeToString(randomBytes), mediaType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong when getting the asset path", err)
	}
	assetDiskPath := cfg.getAssetDiskPath(assetPath)
	
	// Create the file where we are going to copy image data
	imageFileCreate, err := os.Create(assetDiskPath)
	defer imageFileCreate.Close()

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Something went wrong when trying to create the file at path %s", assetDiskPath), err)
		return
	}

	// Copy image data to newly created file
	err = os.WriteFile(assetDiskPath, imageData, 0644)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Something went wrong when trying to copy raw image bytes to path %s", assetDiskPath), err)
		return
	}

	// Get video for updating metadata
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Video not found", err)
		return
	}

	// Check to see if this user is owner of this video
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User does not own this video", err)
		return
	}

	// create URL
	thumbnailURL := cfg.getAssetURL(assetPath)
	video.ThumbnailURL = &thumbnailURL

	// Update the video in the database if everything is 
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't updarte video", err)
		return
	}

	fmt.Printf("Uploading thumbnail for video:%s\nBy user: %s\nSaving to: %s\n", videoID, userID, assetDiskPath)

	// Respond with video data in JSON format
	// marshalled by  database.Video
	respondWithJSON(w, http.StatusOK, database.Video(video))
}
