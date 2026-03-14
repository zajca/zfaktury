package service

import (
	"context"
	"io"
)

// BackupStorage abstracts backup archive storage (local filesystem or S3-compatible).
type BackupStorage interface {
	// Upload copies the archive from localPath into storage under filename.
	Upload(ctx context.Context, localPath, filename string) error
	// Download returns a reader for the stored archive and its size.
	Download(ctx context.Context, filename string) (io.ReadCloser, int64, error)
	// Delete removes the archive from storage. Missing files are silently ignored.
	Delete(ctx context.Context, filename string) error
}
