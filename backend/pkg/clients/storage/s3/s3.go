package s3

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	"github.com/javaknight1/servicepro/backend/pkg/clients/storage"
)

func init() {
	storage.RegisterProvider(storage.ProviderS3, func(ctx context.Context, cfg *config.Config) (storage.Client, error) {
		s3Cfg := &Config{
			Endpoint:        cfg.S3Compatible.Endpoint,
			Bucket:          cfg.S3Compatible.Bucket,
			Region:          cfg.S3Compatible.Region,
			AccessKeyID:     cfg.S3Compatible.AccessKeyID,
			SecretAccessKey: cfg.S3Compatible.SecretAccessKey,
			UsePathStyle:    cfg.S3Compatible.UsePathStyle,
			DisableSSL:      cfg.S3Compatible.DisableSSL,
			PublicURL:       cfg.S3Compatible.PublicURL,
		}
		return NewClient(ctx, s3Cfg)
	})
}

// Config holds configuration for the S3-compatible storage client
// Works with any S3-compatible service (AWS S3, Cloudflare R2, MinIO, etc.)
type Config struct {
	Endpoint        string // Custom endpoint for S3-compatible services (leave empty for AWS)
	Bucket          string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool   // Required for MinIO and some S3-compatible services
	DisableSSL      bool   // For local development (e.g., MinIO without TLS)
	PublicURL       string // Optional: base URL for public access (e.g., CDN or MinIO public URL)
}

// Client implements the storage.Client interface using S3-compatible storage
type Client struct {
	client    *s3.Client
	uploader  *manager.Uploader
	presigner *s3.PresignClient
	config    *Config
}

// NewClient creates a new S3 storage client
func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	if cfg.Region == "" {
		return nil, fmt.Errorf("region is required")
	}

	if cfg.Bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}

	// Build S3 SDK config options
	var opts []func(*awsconfig.LoadOptions) error
	opts = append(opts, awsconfig.WithRegion(cfg.Region))

	// Use custom credentials if provided
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			),
		))
	}

	// Load S3 SDK configuration
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load S3 SDK config: %w", err)
	}

	// Build S3 client options
	s3Opts := []func(*s3.Options){}

	// Custom endpoint for S3-compatible services
	if cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = cfg.UsePathStyle
		})
	}

	// Disable SSL for local testing
	if cfg.DisableSSL {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.EndpointOptions.DisableHTTPS = true
		})
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg, s3Opts...)

	// Create uploader
	uploader := manager.NewUploader(client)

	// Create presigner
	presigner := s3.NewPresignClient(client)

	return &Client{
		client:    client,
		uploader:  uploader,
		presigner: presigner,
		config:    cfg,
	}, nil
}

// Upload implements storage.Client
func (c *Client) Upload(ctx context.Context, input *storage.UploadInput) (*storage.UploadOutput, error) {
	if input.Body == nil {
		return nil, fmt.Errorf("body is required")
	}

	// Read body for checksum calculation
	data, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	return c.uploadData(ctx, data, input)
}

// UploadBytes implements storage.Client
func (c *Client) UploadBytes(ctx context.Context, data []byte, input *storage.UploadInput) (*storage.UploadOutput, error) {
	return c.uploadData(ctx, data, input)
}

func (c *Client) uploadData(ctx context.Context, data []byte, input *storage.UploadInput) (*storage.UploadOutput, error) {
	key := input.Key
	if key == "" {
		key = c.generateKey(input)
	}

	// Detect content type
	contentType := input.ContentType
	if contentType == "" {
		contentType = c.detectContentType(input.FileName, data)
	}

	// Compute checksum
	hash := sha256.Sum256(data)
	checksum := hex.EncodeToString(hash[:])

	// Build upload input
	uploadInput := &s3.PutObjectInput{
		Bucket:      aws.String(c.config.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	}

	// Set storage class
	if input.StorageClass != "" {
		uploadInput.StorageClass = types.StorageClass(input.StorageClass)
	}

	// Set metadata
	if len(input.Metadata) > 0 {
		uploadInput.Metadata = input.Metadata
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
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Generate the public URL
	// If PublicURL is configured, use it; otherwise fall back to the S3 location
	url := result.Location
	if c.config.PublicURL != "" {
		url = strings.TrimSuffix(c.config.PublicURL, "/") + "/" + key
	}

	output := &storage.UploadOutput{
		Key:         key,
		ETag:        aws.ToString(result.ETag),
		URL:         url,
		Checksum:    checksum,
		Size:        int64(len(data)),
		ContentType: contentType,
		UploadedAt:  time.Now().UTC(),
	}

	if result.VersionID != nil {
		output.VersionID = *result.VersionID
	}

	return output, nil
}

// Download implements storage.Client
func (c *Client) Download(ctx context.Context, input *storage.DownloadInput) (*storage.DownloadOutput, error) {
	getInput := &s3.GetObjectInput{
		Bucket: aws.String(c.config.Bucket),
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

	result, err := c.client.GetObject(ctx, getInput)
	if err != nil {
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}

	output := &storage.DownloadOutput{
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
	if expiration <= 0 {
		expiration = 1 * time.Hour
	}

	presignResult, err := c.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.config.Bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiration))

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	url := presignResult.URL

	// If PublicURL is configured, replace the internal endpoint with the public one
	// This is needed for Docker/local development where the internal endpoint (e.g., minio:9000)
	// is not accessible from the browser
	if c.config.PublicURL != "" && c.config.Endpoint != "" {
		url = strings.Replace(url, c.config.Endpoint, strings.TrimSuffix(c.config.PublicURL, "/"), 1)
	}

	return url, nil
}

// GetPresignedUploadURL implements storage.Client
func (c *Client) GetPresignedUploadURL(ctx context.Context, key, contentType string, expiration time.Duration) (string, error) {
	if expiration <= 0 {
		expiration = 1 * time.Hour
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(c.config.Bucket),
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

// Delete implements storage.Client
func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.config.Bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return nil
}

// Exists implements storage.Client
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	_, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.config.Bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		var notFound *types.NotFound
		if ok := errors.As(err, &notFound); ok {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// GetObjectInfo implements storage.Client
func (c *Client) GetObjectInfo(ctx context.Context, key string) (*storage.ObjectInfo, error) {
	result, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.config.Bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	info := &storage.ObjectInfo{
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

// List implements storage.Client
func (c *Client) List(ctx context.Context, input *storage.ListInput) (*storage.ListOutput, error) {
	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(c.config.Bucket),
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

	output := &storage.ListOutput{
		Objects:     make([]storage.ObjectInfo, len(result.Contents)),
		IsTruncated: aws.ToBool(result.IsTruncated),
	}

	for i, obj := range result.Contents {
		output.Objects[i] = storage.ObjectInfo{
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

// Copy implements storage.Client
func (c *Client) Copy(ctx context.Context, sourceKey, destKey string) (*storage.CopyOutput, error) {
	result, err := c.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(c.config.Bucket),
		CopySource: aws.String(fmt.Sprintf("%s/%s", c.config.Bucket, sourceKey)),
		Key:        aws.String(destKey),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to copy object: %w", err)
	}

	return &storage.CopyOutput{
		ETag:         aws.ToString(result.CopyObjectResult.ETag),
		LastModified: aws.ToTime(result.CopyObjectResult.LastModified),
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
	// Try to list a single object to verify connectivity
	_, err := c.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(c.config.Bucket),
		MaxKeys: aws.Int32(1),
	})

	if err != nil {
		return fmt.Errorf("S3 health check failed: %w", err)
	}

	logging.Info(ctx, "S3 health check passed", map[string]any{"bucket": c.config.Bucket})
	return nil
}

// Close implements storage.Client
func (c *Client) Close() error {
	// S3 client doesn't require explicit cleanup
	return nil
}

// generateKey generates a unique object key
func (c *Client) generateKey(input *storage.UploadInput) string {
	now := time.Now().UTC()
	ext := filepath.Ext(input.FileName)
	timestamp := now.Format("20060102-150405")
	return fmt.Sprintf("uploads/%d/%02d/%s-%d%s",
		now.Year(),
		now.Month(),
		timestamp,
		now.UnixNano()%1000000,
		ext,
	)
}

// detectContentType detects the content type of a file
func (c *Client) detectContentType(fileName string, data []byte) string {
	// Try to get from extension first
	ext := filepath.Ext(fileName)
	if ext != "" {
		mimeType := mime.TypeByExtension(ext)
		if mimeType != "" {
			return mimeType
		}
	}

	// Try to detect from content
	if len(data) > 0 {
		contentType := http.DetectContentType(data[:min(512, len(data))])
		return contentType
	}

	return "application/octet-stream"
}

// Ensure Client implements storage.Client
var _ storage.Client = (*Client)(nil)
