package mock

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	"github.com/javaknight1/servicepro/backend/pkg/clients/storage"
)

func init() {
	storage.RegisterProvider(storage.ProviderMock, func(ctx context.Context, cfg *config.Config) (storage.Client, error) {
		mockCfg := &Config{
			Bucket: cfg.S3Compatible.Bucket,
		}
		if mockCfg.Bucket == "" {
			mockCfg.Bucket = "mock-bucket"
		}
		return NewClient(mockCfg), nil
	})
}

// Config holds configuration for the mock storage client
type Config struct {
	Bucket string
}

// storedObject represents an object stored in the mock
type storedObject struct {
	Key          string
	Data         []byte
	ContentType  string
	Metadata     map[string]string
	LastModified time.Time
	ETag         string
	StorageClass string
}

// Client implements the storage.Client interface for testing and development
type Client struct {
	config  *Config
	mu      sync.RWMutex
	objects map[string]*storedObject
}

// NewClient creates a new mock storage client
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		cfg = &Config{
			Bucket: "mock-bucket",
		}
	}
	return &Client{
		config:  cfg,
		objects: make(map[string]*storedObject),
	}
}

// Upload implements storage.Client
func (c *Client) Upload(ctx context.Context, input *storage.UploadInput) (*storage.UploadOutput, error) {
	if input.Body == nil {
		return nil, fmt.Errorf("body is required")
	}

	data, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	return c.uploadData(data, input)
}

// UploadBytes implements storage.Client
func (c *Client) UploadBytes(ctx context.Context, data []byte, input *storage.UploadInput) (*storage.UploadOutput, error) {
	return c.uploadData(data, input)
}

func (c *Client) uploadData(data []byte, input *storage.UploadInput) (*storage.UploadOutput, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := input.Key
	if key == "" {
		key = fmt.Sprintf("mock/%d/%s", time.Now().UnixNano(), input.FileName)
	}

	// Compute checksum
	hash := sha256.Sum256(data)
	checksum := hex.EncodeToString(hash[:])
	etag := fmt.Sprintf("\"%s\"", checksum[:32])

	// Store the object
	obj := &storedObject{
		Key:          key,
		Data:         data,
		ContentType:  input.ContentType,
		Metadata:     input.Metadata,
		LastModified: time.Now(),
		ETag:         etag,
		StorageClass: string(input.StorageClass),
	}
	if obj.ContentType == "" {
		obj.ContentType = "application/octet-stream"
	}
	if obj.Metadata == nil {
		obj.Metadata = make(map[string]string)
	}

	c.objects[key] = obj

	logging.Info(context.Background(), "[STORAGE-MOCK] uploaded", map[string]any{"key": key, "size": len(data)})

	return &storage.UploadOutput{
		Key:         key,
		ETag:        etag,
		URL:         fmt.Sprintf("mock://%s/%s", c.config.Bucket, key),
		Checksum:    checksum,
		Size:        int64(len(data)),
		ContentType: obj.ContentType,
		UploadedAt:  obj.LastModified,
	}, nil
}

// Download implements storage.Client
func (c *Client) Download(ctx context.Context, input *storage.DownloadInput) (*storage.DownloadOutput, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	obj, ok := c.objects[input.Key]
	if !ok {
		return nil, fmt.Errorf("object not found: %s", input.Key)
	}

	logging.Info(ctx, "[STORAGE-MOCK] downloaded", map[string]any{"key": input.Key, "size": len(obj.Data)})

	return &storage.DownloadOutput{
		Body:         io.NopCloser(bytes.NewReader(obj.Data)),
		Size:         int64(len(obj.Data)),
		ContentType:  obj.ContentType,
		ETag:         obj.ETag,
		LastModified: obj.LastModified,
		Metadata:     obj.Metadata,
	}, nil
}

// DownloadBytes implements storage.Client
func (c *Client) DownloadBytes(ctx context.Context, key string) ([]byte, error) {
	result, err := c.Download(ctx, &storage.DownloadInput{Key: key})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()

	return io.ReadAll(result.Body)
}

// GetPresignedURL implements storage.Client
func (c *Client) GetPresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	c.mu.RLock()
	_, ok := c.objects[key]
	c.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("object not found: %s", key)
	}

	// Return a mock presigned URL
	url := fmt.Sprintf("mock://%s/%s?presigned=true&expires=%d", c.config.Bucket, key, time.Now().Add(expiration).Unix())
	logging.Info(ctx, "[STORAGE-MOCK] generated presigned download URL", map[string]any{"key": key})
	return url, nil
}

// GetPresignedUploadURL implements storage.Client
func (c *Client) GetPresignedUploadURL(ctx context.Context, key, contentType string, expiration time.Duration) (string, error) {
	// Return a mock presigned upload URL
	url := fmt.Sprintf("mock://%s/%s?presigned=true&upload=true&expires=%d", c.config.Bucket, key, time.Now().Add(expiration).Unix())
	logging.Info(ctx, "[STORAGE-MOCK] generated presigned upload URL", map[string]any{"key": key})
	return url, nil
}

// Delete implements storage.Client
func (c *Client) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.objects[key]; !ok {
		return fmt.Errorf("object not found: %s", key)
	}

	delete(c.objects, key)
	logging.Info(ctx, "[STORAGE-MOCK] deleted", map[string]any{"key": key})
	return nil
}

// Exists implements storage.Client
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, ok := c.objects[key]
	return ok, nil
}

// GetObjectInfo implements storage.Client
func (c *Client) GetObjectInfo(ctx context.Context, key string) (*storage.ObjectInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	obj, ok := c.objects[key]
	if !ok {
		return nil, fmt.Errorf("object not found: %s", key)
	}

	return &storage.ObjectInfo{
		Key:          obj.Key,
		Size:         int64(len(obj.Data)),
		ContentType:  obj.ContentType,
		ETag:         obj.ETag,
		LastModified: obj.LastModified,
		StorageClass: obj.StorageClass,
		Metadata:     obj.Metadata,
	}, nil
}

// List implements storage.Client
func (c *Client) List(ctx context.Context, input *storage.ListInput) (*storage.ListOutput, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var objects []storage.ObjectInfo
	commonPrefixes := make(map[string]bool)

	for key, obj := range c.objects {
		// Apply prefix filter
		if input.Prefix != "" && !strings.HasPrefix(key, input.Prefix) {
			continue
		}

		// Apply start after filter
		if input.StartAfter != "" && key <= input.StartAfter {
			continue
		}

		// Handle delimiter for directory-like listing
		if input.Delimiter != "" {
			// Get the part after the prefix
			relativePath := strings.TrimPrefix(key, input.Prefix)
			delimIndex := strings.Index(relativePath, input.Delimiter)
			if delimIndex >= 0 {
				// This is a "directory" - add to common prefixes
				prefix := input.Prefix + relativePath[:delimIndex+1]
				commonPrefixes[prefix] = true
				continue
			}
		}

		objects = append(objects, storage.ObjectInfo{
			Key:          obj.Key,
			Size:         int64(len(obj.Data)),
			ContentType:  obj.ContentType,
			ETag:         obj.ETag,
			LastModified: obj.LastModified,
			StorageClass: obj.StorageClass,
			Metadata:     obj.Metadata,
		})
	}

	// Sort objects by key
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].Key < objects[j].Key
	})

	// Apply max keys limit
	isTruncated := false
	if input.MaxKeys > 0 && int32(len(objects)) > input.MaxKeys {
		objects = objects[:input.MaxKeys]
		isTruncated = true
	}

	// Convert common prefixes map to slice
	prefixList := make([]string, 0, len(commonPrefixes))
	for prefix := range commonPrefixes {
		prefixList = append(prefixList, prefix)
	}
	sort.Strings(prefixList)

	return &storage.ListOutput{
		Objects:        objects,
		CommonPrefixes: prefixList,
		IsTruncated:    isTruncated,
	}, nil
}

// Copy implements storage.Client
func (c *Client) Copy(ctx context.Context, sourceKey, destKey string) (*storage.CopyOutput, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	source, ok := c.objects[sourceKey]
	if !ok {
		return nil, fmt.Errorf("source object not found: %s", sourceKey)
	}

	// Create a copy
	dest := &storedObject{
		Key:          destKey,
		Data:         make([]byte, len(source.Data)),
		ContentType:  source.ContentType,
		Metadata:     make(map[string]string),
		LastModified: time.Now(),
		ETag:         source.ETag,
		StorageClass: source.StorageClass,
	}
	copy(dest.Data, source.Data)
	for k, v := range source.Metadata {
		dest.Metadata[k] = v
	}

	c.objects[destKey] = dest

	logging.Info(ctx, "[STORAGE-MOCK] copied", map[string]any{"source_key": sourceKey, "dest_key": destKey})

	return &storage.CopyOutput{
		ETag:         dest.ETag,
		LastModified: dest.LastModified,
	}, nil
}

// Move implements storage.Client
func (c *Client) Move(ctx context.Context, sourceKey, destKey string) error {
	// Copy first
	_, err := c.Copy(ctx, sourceKey, destKey)
	if err != nil {
		return err
	}

	// Then delete source
	return c.Delete(ctx, sourceKey)
}

// HealthCheck implements storage.Client
func (c *Client) HealthCheck(ctx context.Context) error {
	logging.Info(ctx, "[STORAGE-MOCK] health check passed", nil)
	return nil
}

// Close implements storage.Client
func (c *Client) Close() error {
	return nil
}

// GetAllObjects returns all stored objects (for testing)
func (c *Client) GetAllObjects() map[string][]byte {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string][]byte)
	for key, obj := range c.objects {
		result[key] = obj.Data
	}
	return result
}

// ClearAllObjects removes all stored objects (for testing)
func (c *Client) ClearAllObjects() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.objects = make(map[string]*storedObject)
}

// Ensure Client implements storage.Client
var _ storage.Client = (*Client)(nil)
