package downloader

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ObjectStore uploads job results to an S3-compatible bucket (DigitalOcean Spaces).
type ObjectStore struct {
	client   *minio.Client
	endpoint string
	bucket   string
}

// newObjectStoreFromEnv builds the store from environment variables:
// S3_ENDPOINT (e.g. nyc3.digitaloceanspaces.com), S3_BUCKET, S3_ACCESS_KEY, S3_SECRET_KEY.
func newObjectStoreFromEnv() (*ObjectStore, error) {
	endpoint := os.Getenv("S3_ENDPOINT")
	bucket := os.Getenv("S3_BUCKET")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")

	for name, value := range map[string]string{
		"S3_ENDPOINT":   endpoint,
		"S3_BUCKET":     bucket,
		"S3_ACCESS_KEY": accessKey,
		"S3_SECRET_KEY": secretKey,
	} {
		if value == "" {
			return nil, fmt.Errorf("environment variable %s is not set", name)
		}
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: true,
	})
	if err != nil {
		return nil, fmt.Errorf("creating s3 client: %w", err)
	}

	return &ObjectStore{client: client, endpoint: endpoint, bucket: bucket}, nil
}

// UploadCSV stores data under key and returns the object's public URL.
func (s *ObjectStore) UploadCSV(ctx context.Context, key string, data []byte) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, key,
		bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: "text/csv"},
	)
	if err != nil {
		return "", fmt.Errorf("uploading %s: %w", key, err)
	}
	return fmt.Sprintf("https://%s.%s/%s", s.bucket, s.endpoint, key), nil
}
