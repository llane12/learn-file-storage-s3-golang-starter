package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"tubely/internal/auth"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	// Limit size of uploaded files
	const maxMemory = 1 << 30 // 1GB
	http.MaxBytesReader(w, r.Body, maxMemory)

	// Get video ID from URL parameter
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid video ID", err)
		return
	}

	// Authenticate user
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unable to find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unable to validate JWT", err)
		return
	}

	fmt.Println("uploading video file for video", videoID, "by user", userID)

	// Get video record from database
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to locate video record", err)
		return
	}

	// Verify user is video owner
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	// Parse video file from form data
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing multipart form", err)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	// Check content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		respondWithError(w, http.StatusBadRequest, "Missing Content-Type header", nil)
		return
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type header", err)
		return
	}
	if !isValidVideo(mediaType) {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", nil)
		return
	}

	// Save uploaded file to temporary file on disk
	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error saving file to disk", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close() // defer is LIFO, so close happens before remove

	written, err := io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error saving file to disk", err)
		return
	}
	if written == 0 {
		respondWithError(w, http.StatusInternalServerError, "Error saving file to disk", nil)
		return
	}

	// Reset temporary file's file pointer
	offset, err := tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error reading temporary file", err)
		return
	}
	if offset != 0 {
		respondWithError(w, http.StatusInternalServerError, "Error reading temporary file", nil)
		return
	}

	// Process video file for fast start
	processedFilePath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error processing video file for faststart", err)
		return
	}

	processedFile, err := os.Open(processedFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error reading processed video file", err)
		return
	}

	// Get aspect ratio of video file
	ratio, err := getVideoAspectRatio(processedFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error calculating aspect ratio", err)
		return
	}
	prefix := "other"
	switch ratio {
	case "16:9":
		prefix = "landscape"
	case "9:16":
		prefix = "portrait"
	}

	// Upload temp file to S3
	assetId := getRandomAssetId()
	assetId = fmt.Sprintf("%s/%s", prefix, assetId)
	assetPath := getAssetPath(assetId, mediaType)

	putObjectParams := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &assetPath,
		Body:        processedFile,
		ContentType: &mediaType,
	}
	_, err = cfg.s3Client.PutObject(r.Context(), &putObjectParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error uploading video file", err)
		return
	}

	// Update video URL
	videoUrl := cfg.getAssetUrlS3(assetPath)
	video.VideoURL = &videoUrl

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating video record", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
