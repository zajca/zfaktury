package service

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/zajca/zfaktury/internal/config"
)

// S3Storage stores backup archives in an S3-compatible bucket.
type S3Storage struct {
	client *minio.Client
	bucket string
}

// NewS3Storage creates an S3Storage, verifies the bucket exists, and returns it.
func NewS3Storage(cfg config.S3Config) (*S3Storage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("creating S3 client: %w", err)
	}

	exists, err := client.BucketExists(context.Background(), cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("checking S3 bucket existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("S3 bucket %q does not exist", cfg.Bucket)
	}

	return &S3Storage{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

// Upload puts the local archive file into S3.
func (s *S3Storage) Upload(ctx context.Context, localPath, filename string) error {
	_, err := s.client.FPutObject(ctx, s.bucket, filename, localPath, minio.PutObjectOptions{
		ContentType: "application/gzip",
	})
	if err != nil {
		return fmt.Errorf("uploading to S3: %w", err)
	}
	return nil
}

// Download returns a reader for the S3 object and its size.
func (s *S3Storage) Download(ctx context.Context, filename string) (io.ReadCloser, int64, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, filename, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("getting S3 object: %w", err)
	}

	info, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		return nil, 0, fmt.Errorf("stat S3 object: %w", err)
	}

	return obj, info.Size, nil
}

// Delete removes the object from S3. Missing objects are silently ignored.
func (s *S3Storage) Delete(ctx context.Context, filename string) error {
	err := s.client.RemoveObject(ctx, s.bucket, filename, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("deleting S3 object: %w", err)
	}
	return nil
}
