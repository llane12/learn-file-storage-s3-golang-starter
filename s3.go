package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"tubely/internal/database"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video, ctx context.Context) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}

	parts := strings.Split(*video.VideoURL, ",")
	if len(parts) != 2 {
		return database.Video{}, errors.New("invalid VideoURL")
	}

	presignedUrl, err := generatePresignedURL(cfg.s3Client, parts[0], parts[1], 5*time.Minute, ctx)
	if err != nil {
		return database.Video{}, err
	}

	video.VideoURL = &presignedUrl
	return video, nil
}

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration, ctx context.Context) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)

	params := s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}
	presignReq, err := presignClient.PresignGetObject(ctx, &params, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}

	return presignReq.URL, nil
}
