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

	// Defer closing and removing the temporary file.
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
	
	putObjectInput :=  &s3.PutObjectInput{
		Bucket: aws.String(cfg.s3Bucket),
		Key: aws.String(s3Key),
		Body: tempFile,
		ContentType: aws.String(mediaType), 
	}
	// Put file into S3
	_, err = cfg.s3client.PutObject(r.Context(), putObjectInput)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to upload to S3", err)
		return
	}

	// Update VideoURL with the URL of the video at S3
	awsRegion := cfg.s3Region
	awsBucketName := cfg.s3Bucket
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", awsBucketName, awsRegion, s3Key)
	video.VideoURL = &url

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	fmt.Println("uploading video ID: ", videoID, "\nBy user:", userID, "Saving to :", *video.ThumbnailURL + "\n")

	// Respond with video data in JSON format
	// marshalled by  database.Video
	respondWithJSON(w, http.StatusOK, database.Video(video))
}
