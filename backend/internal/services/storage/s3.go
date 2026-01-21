package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

// =============================================================================
// S3 Client
// =============================================================================

// S3Client provides S3 storage operations
type S3Client struct {
	client      *s3.Client
	uploader    *manager.Uploader
	downloader  *manager.Downloader
	presigner   *s3.PresignClient
	config      *Config
	uploadSem   chan struct{}
	downloadSem chan struct{}
	mu          sync.RWMutex
	stats       *ClientStats
	bufferPool  *sync.Pool
}

// ClientStats tracks client statistics
type ClientStats struct {
	TotalUploads    int64
	TotalDownloads  int64
	TotalBytes      int64
	FailedUploads   int64
	FailedDownloads int64
	mu              sync.RWMutex
}

// NewS3Client creates a new S3 client
func NewS3Client(ctx context.Context, cfg *Config) (*S3Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Build AWS config options
	var opts []func(*awsconfig.LoadOptions) error

	opts = append(opts, awsconfig.WithRegion(cfg.AWS.Region))

	// Use custom credentials if provided
	if cfg.AWS.AccessKeyID != "" && cfg.AWS.SecretAccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AWS.AccessKeyID,
				cfg.AWS.SecretAccessKey,
				cfg.AWS.SessionToken,
			),
		))
	}

	// Use profile if specified
	if cfg.AWS.Profile != "" {
		opts = append(opts, awsconfig.WithSharedConfigProfile(cfg.AWS.Profile))
	}

	// Load AWS configuration
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Build S3 client options
	s3Opts := []func(*s3.Options){}

	// Custom endpoint for S3-compatible services
	if cfg.AWS.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.AWS.Endpoint)
			o.UsePathStyle = cfg.AWS.UsePathStyle
		})
	}

	// Disable SSL for local testing
	if cfg.AWS.DisableSSL {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.EndpointOptions.DisableHTTPS = true
		})
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg, s3Opts...)

	// Create uploader with custom settings
	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = cfg.Upload.MultipartChunkSize
		u.Concurrency = cfg.Performance.MaxConcurrentUploads
	})

	// Create downloader with custom settings
	downloader := manager.NewDownloader(client, func(d *manager.Downloader) {
		d.PartSize = cfg.Upload.MultipartChunkSize
		d.Concurrency = cfg.Performance.MaxConcurrentDownloads
	})

	// Create presigner
	presigner := s3.NewPresignClient(client)

	// Create buffer pool for efficient memory usage
	bufferPool := &sync.Pool{
		New: func() interface{} {
			return make([]byte, 32*1024) // 32KB buffers
		},
	}

	return &S3Client{
		client:      client,
		uploader:    uploader,
		downloader:  downloader,
		presigner:   presigner,
		config:      cfg,
		uploadSem:   make(chan struct{}, cfg.Performance.MaxConcurrentUploads),
		downloadSem: make(chan struct{}, cfg.Performance.MaxConcurrentDownloads),
		stats:       &ClientStats{},
		bufferPool:  bufferPool,
	}, nil
}

// =============================================================================
// Upload Operations
// =============================================================================

// UploadInput contains parameters for uploading a file
type UploadInput struct {
	// File content
	Body io.Reader

	// File name (original name)
	FileName string

	// File size (for validation)
	Size int64

	// Content type (auto-detected if empty)
	ContentType string

	// Document type
	DocumentType DocumentType

	// Associated entity ID (customer_id, invoice_id, etc.)
	EntityID string

	// Associated entity type
	EntityType string

	// Custom key (overrides auto-generated key)
	Key string

	// Storage class
	StorageClass StorageClass

	// Custom metadata
	Metadata map[string]string

	// Tags
	Tags map[string]string

	// Enable encryption (overrides config)
	Encrypt *bool

	// Cache control header
	CacheControl string

	// Content disposition
	ContentDisposition string
}

// UploadOutput contains the result of an upload operation
type UploadOutput struct {
	// Object key
	Key string

	// Version ID (if versioning enabled)
	VersionID string

	// ETag
	ETag string

	// Object URL
	URL string

	// Checksum (SHA256)
	Checksum string

	// Size in bytes
	Size int64

	// Content type
	ContentType string

	// Upload timestamp
	UploadedAt time.Time
}

// Upload uploads a file to S3
func (c *S3Client) Upload(ctx context.Context, input *UploadInput) (*UploadOutput, error) {
	// Acquire semaphore
	select {
	case c.uploadSem <- struct{}{}:
		defer func() { <-c.uploadSem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Validate input
	if err := c.validateUpload(input); err != nil {
		c.stats.mu.Lock()
		c.stats.FailedUploads++
		c.stats.mu.Unlock()
		return nil, err
	}

	// Generate key if not provided
	key := input.Key
	if key == "" {
		key = c.generateKey(input)
	}

	// Detect content type
	contentType := input.ContentType
	if contentType == "" {
		contentType = c.detectContentType(input.FileName, input.Body)
	}

	// Read body into buffer for checksum calculation
	var body io.Reader = input.Body
	var checksum string

	if c.config.Upload.VerifyChecksum {
		data, err := io.ReadAll(input.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read body: %w", err)
		}

		hash := sha256.Sum256(data)
		checksum = hex.EncodeToString(hash[:])
		body = bytes.NewReader(data)
	}

	// Build upload input
	uploadInput := &s3.PutObjectInput{
		Bucket:      aws.String(c.config.Buckets.Primary),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	}

	// Set storage class
	if input.StorageClass != "" {
		uploadInput.StorageClass = types.StorageClass(input.StorageClass)
	}

	// Set encryption
	encrypt := c.config.Upload.EnableEncryption
	if input.Encrypt != nil {
		encrypt = *input.Encrypt
	}
	if encrypt {
		if c.config.Security.ServerSideEncryption == "aws:kms" {
			uploadInput.ServerSideEncryption = types.ServerSideEncryptionAwsKms
			if c.config.Security.KMSKeyID != "" {
				uploadInput.SSEKMSKeyId = aws.String(c.config.Security.KMSKeyID)
			}
		} else {
			uploadInput.ServerSideEncryption = types.ServerSideEncryptionAes256
		}
	}

	// Set metadata
	if len(input.Metadata) > 0 {
		uploadInput.Metadata = input.Metadata
	}

	// Add standard metadata
	if uploadInput.Metadata == nil {
		uploadInput.Metadata = make(map[string]string)
	}
	uploadInput.Metadata["document-type"] = string(input.DocumentType)
	uploadInput.Metadata["original-filename"] = input.FileName
	uploadInput.Metadata["uploaded-at"] = time.Now().UTC().Format(time.RFC3339)
	if input.EntityID != "" {
		uploadInput.Metadata["entity-id"] = input.EntityID
	}
	if input.EntityType != "" {
		uploadInput.Metadata["entity-type"] = input.EntityType
	}
	if checksum != "" {
		uploadInput.Metadata["checksum-sha256"] = checksum
	}

	// Set cache control
	if input.CacheControl != "" {
		uploadInput.CacheControl = aws.String(input.CacheControl)
	}

	// Set content disposition
	if input.ContentDisposition != "" {
		uploadInput.ContentDisposition = aws.String(input.ContentDisposition)
	}

	// Set tags
	if len(input.Tags) > 0 {
		var tagPairs []string
		for k, v := range input.Tags {
			tagPairs = append(tagPairs, fmt.Sprintf("%s=%s", k, v))
		}
		uploadInput.Tagging = aws.String(strings.Join(tagPairs, "&"))
	}

	// Perform upload
	result, err := c.uploader.Upload(ctx, uploadInput)
	if err != nil {
		c.stats.mu.Lock()
		c.stats.FailedUploads++
		c.stats.mu.Unlock()
		return nil, fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}

	// Update stats
	c.stats.mu.Lock()
	c.stats.TotalUploads++
	c.stats.TotalBytes += input.Size
	c.stats.mu.Unlock()

	// Build output
	output := &UploadOutput{
		Key:         key,
		ETag:        aws.ToString(result.ETag),
		URL:         result.Location,
		Checksum:    checksum,
		Size:        input.Size,
		ContentType: contentType,
		UploadedAt:  time.Now().UTC(),
	}

	if result.VersionID != nil {
		output.VersionID = *result.VersionID
	}

	return output, nil
}

// UploadBytes uploads bytes to S3
func (c *S3Client) UploadBytes(ctx context.Context, data []byte, input *UploadInput) (*UploadOutput, error) {
	input.Body = bytes.NewReader(data)
	input.Size = int64(len(data))
	return c.Upload(ctx, input)
}

// =============================================================================
// Download Operations
// =============================================================================

// DownloadInput contains parameters for downloading a file
type DownloadInput struct {
	// Object key
	Key string

	// Version ID (optional)
	VersionID string

	// Byte range (optional)
	Range string

	// If-Modified-Since (optional)
	IfModifiedSince *time.Time

	// If-None-Match (optional, ETag)
	IfNoneMatch string
}

// DownloadOutput contains the result of a download operation
type DownloadOutput struct {
	// File content
	Body io.ReadCloser

	// File size
	Size int64

	// Content type
	ContentType string

	// ETag
	ETag string

	// Last modified
	LastModified time.Time

	// Version ID
	VersionID string

	// Metadata
	Metadata map[string]string

	// Cache control
	CacheControl string

	// Content disposition
	ContentDisposition string
}

// Download downloads a file from S3
func (c *S3Client) Download(ctx context.Context, input *DownloadInput) (*DownloadOutput, error) {
	// Acquire semaphore
	select {
	case c.downloadSem <- struct{}{}:
		defer func() { <-c.downloadSem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Build get object input
	getInput := &s3.GetObjectInput{
		Bucket: aws.String(c.config.Buckets.Primary),
		Key:    aws.String(input.Key),
	}

	if input.VersionID != "" {
		getInput.VersionId = aws.String(input.VersionID)
	}

	if input.Range != "" {
		getInput.Range = aws.String(input.Range)
	}

	if input.IfModifiedSince != nil {
		getInput.IfModifiedSince = input.IfModifiedSince
	}

	if input.IfNoneMatch != "" {
		getInput.IfNoneMatch = aws.String(input.IfNoneMatch)
	}

	// Get object
	result, err := c.client.GetObject(ctx, getInput)
	if err != nil {
		c.stats.mu.Lock()
		c.stats.FailedDownloads++
		c.stats.mu.Unlock()
		return nil, fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}

	// Update stats
	c.stats.mu.Lock()
	c.stats.TotalDownloads++
	if result.ContentLength != nil {
		c.stats.TotalBytes += *result.ContentLength
	}
	c.stats.mu.Unlock()

	// Build output
	output := &DownloadOutput{
		Body:        result.Body,
		ContentType: aws.ToString(result.ContentType),
		ETag:        aws.ToString(result.ETag),
		Metadata:    result.Metadata,
	}

	if result.ContentLength != nil {
		output.Size = *result.ContentLength
	}

	if result.LastModified != nil {
		output.LastModified = *result.LastModified
	}

	if result.VersionId != nil {
		output.VersionID = *result.VersionId
	}

	if result.CacheControl != nil {
		output.CacheControl = *result.CacheControl
	}

	if result.ContentDisposition != nil {
		output.ContentDisposition = *result.ContentDisposition
	}

	return output, nil
}

// DownloadToWriter downloads a file to an io.Writer
func (c *S3Client) DownloadToWriter(ctx context.Context, w io.WriterAt, input *DownloadInput) (int64, error) {
	downloadInput := &s3.GetObjectInput{
		Bucket: aws.String(c.config.Buckets.Primary),
		Key:    aws.String(input.Key),
	}

	if input.VersionID != "" {
		downloadInput.VersionId = aws.String(input.VersionID)
	}

	n, err := c.downloader.Download(ctx, w, downloadInput)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}

	return n, nil
}

// DownloadBytes downloads a file as bytes
func (c *S3Client) DownloadBytes(ctx context.Context, key string) ([]byte, error) {
	result, err := c.Download(ctx, &DownloadInput{Key: key})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()

	return io.ReadAll(result.Body)
}

// =============================================================================
// Presigned URLs
// =============================================================================

// GetPresignedURL generates a presigned URL for downloading
func (c *S3Client) GetPresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	if expiration <= 0 {
		expiration = c.config.Security.PresignedURLExpiration
	}

	presignResult, err := c.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.config.Buckets.Primary),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiration))

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignResult.URL, nil
}

// GetPresignedUploadURL generates a presigned URL for uploading
func (c *S3Client) GetPresignedUploadURL(ctx context.Context, key, contentType string, expiration time.Duration) (string, error) {
	if expiration <= 0 {
		expiration = c.config.Security.PresignedURLExpiration
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(c.config.Buckets.Primary),
		Key:    aws.String(key),
	}

	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}

	presignResult, err := c.presigner.PresignPutObject(ctx, input, s3.WithPresignExpires(expiration))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload URL: %w", err)
	}

	return presignResult.URL, nil
}

// =============================================================================
// Object Management
// =============================================================================

// Delete deletes an object from S3
func (c *S3Client) Delete(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.config.Buckets.Primary),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeleteFailed, err)
	}

	return nil
}

// DeleteVersion deletes a specific version of an object
func (c *S3Client) DeleteVersion(ctx context.Context, key, versionID string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket:    aws.String(c.config.Buckets.Primary),
		Key:       aws.String(key),
		VersionId: aws.String(versionID),
	})

	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeleteFailed, err)
	}

	return nil
}

// DeleteBatch deletes multiple objects
func (c *S3Client) DeleteBatch(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	objects := make([]types.ObjectIdentifier, len(keys))
	for i, key := range keys {
		objects[i] = types.ObjectIdentifier{
			Key: aws.String(key),
		}
	}

	_, err := c.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(c.config.Buckets.Primary),
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(true),
		},
	})

	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeleteFailed, err)
	}

	return nil
}

// Exists checks if an object exists
func (c *S3Client) Exists(ctx context.Context, key string) (bool, error) {
	_, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.config.Buckets.Primary),
		Key:    aws.String(key),
	})

	if err != nil {
		// Check if error is "not found"
		var notFound *types.NotFound
		if ok := errors.As(err, &notFound); ok {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// GetObjectInfo gets object metadata without downloading
func (c *S3Client) GetObjectInfo(ctx context.Context, key string) (*ObjectInfo, error) {
	result, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.config.Buckets.Primary),
		Key:    aws.String(key),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	info := &ObjectInfo{
		Key:          key,
		ContentType:  aws.ToString(result.ContentType),
		ETag:         aws.ToString(result.ETag),
		StorageClass: string(result.StorageClass),
		Metadata:     result.Metadata,
	}

	if result.ContentLength != nil {
		info.Size = *result.ContentLength
	}

	if result.LastModified != nil {
		info.LastModified = *result.LastModified
	}

	if result.VersionId != nil {
		info.VersionID = *result.VersionId
	}

	return info, nil
}

// ObjectInfo contains object metadata
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
	VersionID    string
	StorageClass string
	Metadata     map[string]string
}

// Copy copies an object to a new location
func (c *S3Client) Copy(ctx context.Context, sourceKey, destKey string) (*CopyOutput, error) {
	result, err := c.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(c.config.Buckets.Primary),
		CopySource: aws.String(fmt.Sprintf("%s/%s", c.config.Buckets.Primary, sourceKey)),
		Key:        aws.String(destKey),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to copy object: %w", err)
	}

	return &CopyOutput{
		ETag:         aws.ToString(result.CopyObjectResult.ETag),
		LastModified: aws.ToTime(result.CopyObjectResult.LastModified),
	}, nil
}

// CopyOutput contains the result of a copy operation
type CopyOutput struct {
	ETag         string
	LastModified time.Time
	VersionID    string
}

// Move moves an object to a new location
func (c *S3Client) Move(ctx context.Context, sourceKey, destKey string) error {
	// Copy first
	_, err := c.Copy(ctx, sourceKey, destKey)
	if err != nil {
		return err
	}

	// Then delete source
	return c.Delete(ctx, sourceKey)
}

// =============================================================================
// Listing Operations
// =============================================================================

// ListInput contains parameters for listing objects
type ListInput struct {
	Prefix            string
	Delimiter         string
	MaxKeys           int32
	ContinuationToken string
	StartAfter        string
}

// ListOutput contains the result of a list operation
type ListOutput struct {
	Objects           []ObjectInfo
	CommonPrefixes    []string
	ContinuationToken string
	IsTruncated       bool
}

// List lists objects in the bucket
func (c *S3Client) List(ctx context.Context, input *ListInput) (*ListOutput, error) {
	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(c.config.Buckets.Primary),
	}

	if input.Prefix != "" {
		listInput.Prefix = aws.String(input.Prefix)
	}

	if input.Delimiter != "" {
		listInput.Delimiter = aws.String(input.Delimiter)
	}

	if input.MaxKeys > 0 {
		listInput.MaxKeys = aws.Int32(input.MaxKeys)
	}

	if input.ContinuationToken != "" {
		listInput.ContinuationToken = aws.String(input.ContinuationToken)
	}

	if input.StartAfter != "" {
		listInput.StartAfter = aws.String(input.StartAfter)
	}

	result, err := c.client.ListObjectsV2(ctx, listInput)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	output := &ListOutput{
		Objects:     make([]ObjectInfo, len(result.Contents)),
		IsTruncated: aws.ToBool(result.IsTruncated),
	}

	for i, obj := range result.Contents {
		output.Objects[i] = ObjectInfo{
			Key:          aws.ToString(obj.Key),
			Size:         aws.ToInt64(obj.Size),
			ETag:         aws.ToString(obj.ETag),
			LastModified: aws.ToTime(obj.LastModified),
			StorageClass: string(obj.StorageClass),
		}
	}

	for _, prefix := range result.CommonPrefixes {
		output.CommonPrefixes = append(output.CommonPrefixes, aws.ToString(prefix.Prefix))
	}

	if result.NextContinuationToken != nil {
		output.ContinuationToken = *result.NextContinuationToken
	}

	return output, nil
}

// ListAll lists all objects with the given prefix
func (c *S3Client) ListAll(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	var allObjects []ObjectInfo
	var continuationToken string

	for {
		result, err := c.List(ctx, &ListInput{
			Prefix:            prefix,
			MaxKeys:           1000,
			ContinuationToken: continuationToken,
		})

		if err != nil {
			return nil, err
		}

		allObjects = append(allObjects, result.Objects...)

		if !result.IsTruncated {
			break
		}

		continuationToken = result.ContinuationToken
	}

	return allObjects, nil
}

// =============================================================================
// Helper Methods
// =============================================================================

// validateUpload validates upload input
func (c *S3Client) validateUpload(input *UploadInput) error {
	if input.Body == nil {
		return fmt.Errorf("%w: body is required", ErrInvalidContent)
	}

	if input.Size > c.config.Upload.MaxFileSize {
		return fmt.Errorf("%w: file size %d exceeds maximum %d",
			ErrFileTooLarge, input.Size, c.config.Upload.MaxFileSize)
	}

	// Check MIME type
	if len(c.config.Upload.AllowedMimeTypes) > 0 && input.ContentType != "" {
		allowed := false
		for _, mimeType := range c.config.Upload.AllowedMimeTypes {
			if strings.EqualFold(input.ContentType, mimeType) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("%w: %s", ErrInvalidFileType, input.ContentType)
		}
	}

	// Check blocked MIME types
	for _, blocked := range c.config.Upload.BlockedMimeTypes {
		if strings.EqualFold(input.ContentType, blocked) {
			return fmt.Errorf("%w: %s is blocked", ErrInvalidFileType, input.ContentType)
		}
	}

	// Check file extension
	if len(c.config.Upload.AllowedExtensions) > 0 && input.FileName != "" {
		ext := strings.ToLower(filepath.Ext(input.FileName))
		allowed := false
		for _, allowedExt := range c.config.Upload.AllowedExtensions {
			if ext == strings.ToLower(allowedExt) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("%w: extension %s not allowed", ErrInvalidFileType, ext)
		}
	}

	return nil
}

// generateKey generates a unique object key
func (c *S3Client) generateKey(input *UploadInput) string {
	// Format: {document_type}/{year}/{month}/{uuid}_{filename}
	now := time.Now().UTC()
	id := uuid.New().String()

	ext := filepath.Ext(input.FileName)
	baseName := strings.TrimSuffix(input.FileName, ext)
	safeName := sanitizeFileName(baseName)

	key := fmt.Sprintf("%s/%d/%02d/%s_%s%s",
		input.DocumentType,
		now.Year(),
		now.Month(),
		id[:8],
		safeName,
		ext,
	)

	return key
}

// detectContentType detects the content type of a file
func (c *S3Client) detectContentType(fileName string, body io.Reader) string {
	// Try to get from extension first
	ext := filepath.Ext(fileName)
	if ext != "" {
		mimeType := mime.TypeByExtension(ext)
		if mimeType != "" {
			return mimeType
		}
	}

	// Try to detect from content
	if seeker, ok := body.(io.Seeker); ok {
		buffer := make([]byte, 512)
		n, err := body.Read(buffer)
		if err == nil && n > 0 {
			contentType := http.DetectContentType(buffer[:n])
			seeker.Seek(0, io.SeekStart)
			return contentType
		}
		seeker.Seek(0, io.SeekStart)
	}

	return "application/octet-stream"
}

// sanitizeFileName removes unsafe characters from filename
func sanitizeFileName(name string) string {
	// Replace unsafe characters
	replacer := strings.NewReplacer(
		" ", "_",
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	name = replacer.Replace(name)

	// Limit length
	if len(name) > 100 {
		name = name[:100]
	}

	return name
}

// calculateMD5 calculates MD5 hash for content integrity
func calculateMD5(data []byte) string {
	hash := md5.Sum(data)
	return base64.StdEncoding.EncodeToString(hash[:])
}

// GetStats returns client statistics
func (c *S3Client) GetStats() ClientStats {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()

	return ClientStats{
		TotalUploads:    c.stats.TotalUploads,
		TotalDownloads:  c.stats.TotalDownloads,
		TotalBytes:      c.stats.TotalBytes,
		FailedUploads:   c.stats.FailedUploads,
		FailedDownloads: c.stats.FailedDownloads,
	}
}

// GetClient returns the underlying S3 client
func (c *S3Client) GetClient() *s3.Client {
	return c.client
}

// GetConfig returns the configuration
func (c *S3Client) GetConfig() *Config {
	return c.config
}
