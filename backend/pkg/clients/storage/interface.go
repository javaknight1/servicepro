package storage

import (
	"context"
	"time"
)

// Client defines the interface for storage operations.
// This interface provides a unified API for interacting with different
// storage backends (S3, R2, local filesystem, etc.)
type Client interface {
	// Upload uploads a file from an io.Reader
	Upload(ctx context.Context, input *UploadInput) (*UploadOutput, error)

	// UploadBytes uploads bytes directly
	UploadBytes(ctx context.Context, data []byte, input *UploadInput) (*UploadOutput, error)

	// Download downloads a file
	Download(ctx context.Context, input *DownloadInput) (*DownloadOutput, error)

	// DownloadBytes downloads a file as bytes
	DownloadBytes(ctx context.Context, key string) ([]byte, error)

	// GetPresignedURL generates a presigned URL for downloading
	GetPresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error)

	// GetPresignedUploadURL generates a presigned URL for uploading
	GetPresignedUploadURL(ctx context.Context, key, contentType string, expiration time.Duration) (string, error)

	// Delete deletes an object
	Delete(ctx context.Context, key string) error

	// Exists checks if an object exists
	Exists(ctx context.Context, key string) (bool, error)

	// GetObjectInfo gets object metadata without downloading the content
	GetObjectInfo(ctx context.Context, key string) (*ObjectInfo, error)

	// List lists objects with optional filtering
	List(ctx context.Context, input *ListInput) (*ListOutput, error)

	// Copy copies an object to a new location
	Copy(ctx context.Context, sourceKey, destKey string) (*CopyOutput, error)

	// Move moves an object to a new location (copy + delete)
	Move(ctx context.Context, sourceKey, destKey string) error

	// HealthCheck verifies the storage service is operational
	HealthCheck(ctx context.Context) error

	// Close releases any resources held by the client
	Close() error
}
