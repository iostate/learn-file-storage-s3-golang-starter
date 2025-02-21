package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {

	// AUTH
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	// AUTH - userID
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
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
	
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)
	file, header,  err := r.FormFile("video"); 
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close();

	// get media type 
	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Trouble extracting media type from header", err)
		return
	}
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Uploaded file is not an MP4", err)
		return
	}

	// get the extension type
	ext, err := cfg.getExtensionType(mediaType)
	if err != nil {
		log.Print(err)
	}

	// load 32 random bytes used for the s3 key name
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error generating random bytes", err)
		return
	}
	
	// Create name with random bytes for cache busting, add extension as well
	s3Key := hex.EncodeToString(randomBytes) + ext

	// Create temporary file to store video
	tempFile, err := os.CreateTemp("", "*." + ext)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create temporary file", err)
		return
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()

	// Copy the request body into the temporary file.
	if _, err := io.Copy(tempFile, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create copy video into temporary file", err)
		return
	}
	// Set pointer of file back to start
	if _, err := tempFile.Seek(0, io.SeekStart); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to reset file pointer", err)
		return
	}

	// Create processed video using the temp file
	processedVideoPath, err := processVideoForFastStart(tempFile.Name()) 
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to process video", err)
		return
	}

	// Open file for processing
	processedVideoFile, err := os.Open(processedVideoPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to open processed video for upload", err)
		return
	}
	defer processedVideoFile.Close()

	// Check that video is optimized for streaming
	_, err = checkFileContainsString(processedVideoFile.Name(), "moov")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	
	// append aspect ratio orientation to end of string
	aspectRatio, err := getVideoAspectRatio(processedVideoFile.Name())
	if err != nil {
		fmt.Printf("Error getting aspect ratio of file %s: %v", processedVideoFile.Name(), err)
	}

	// Create directory bucket string using aspect ratio
	aspectRatioOrientation := getAspectRatioOrientation(aspectRatio)
	s3KeyWithAspectRatioOrientation := createDirectoryBucketPrefix(aspectRatioOrientation) + s3Key

	// Uploaded processed video to s3
	putObjectInput :=  &s3.PutObjectInput{
		Bucket: aws.String(cfg.s3Bucket),
		Key: aws.String(s3KeyWithAspectRatioOrientation),
		Body: processedVideoFile,
		ContentType: aws.String(mediaType), 
	}

	_, err = cfg.s3client.PutObject(r.Context(), putObjectInput)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to upload to S3", err)
		return
	}

	// Use URL path to our CDN, CloudFront
	// Grabbing CloudFront distribution from env
	url := strings.Join([]string{cfg.s3CfDistribution, s3KeyWithAspectRatioOrientation}, "/")
	fmt.Printf("video.VideoURL = %s\n", url)
	video.VideoURL = &url

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}
	
	fmt.Println("uploading video ID: ", videoID, "\nBy user:", userID, "\nSaving to :", *video.VideoURL + "\n")
	
	os.Remove(processedVideoPath)

	respondWithJSON(w, http.StatusOK, database.Video(video))

}
