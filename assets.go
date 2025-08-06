package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func isValidThumbnail(mediaType string) bool {
	return mediaType == "image/jpeg" || mediaType == "image/png"
}

func isValidVideo(mediaType string) bool {
	return mediaType == "video/mp4"
}

func getRandomAssetId() string {
	randBytes := make([]byte, 32)
	rand.Read(randBytes)
	return base64.RawURLEncoding.EncodeToString(randBytes)
}

func getAssetPath(assetId, mediaType string) string {
	ext := mediaTypeToExt(mediaType)
	return fmt.Sprintf("%s%s", assetId, ext)
}

func (cfg apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

func (cfg apiConfig) getAssetUrlLocal(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

func (cfg apiConfig) getAssetUrlS3(assetId string) string {
	// https://<bucket-name>.s3.<region>.amazonaws.com/<key>
	url := "https://%s.s3.%s.amazonaws.com/%s"
	return fmt.Sprintf(url, cfg.s3Bucket, cfg.s3Region, assetId)
}

func mediaTypeToExt(mediaType string) string {
	parts := strings.Split(mediaType, "/")
	if len(parts) != 2 {
		return ".bin"
	}
	return "." + parts[1]
}
