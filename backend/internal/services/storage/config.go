package storage

import (
	"errors"
	"fmt"
	"time"
)

// =============================================================================
// Errors
// =============================================================================

var (
	ErrInvalidConfig      = errors.New("invalid configuration")
	ErrBucketNotFound     = errors.New("bucket not found")
	ErrObjectNotFound     = errors.New("object not found")
	ErrUploadFailed       = errors.New("upload failed")
	ErrDownloadFailed     = errors.New("download failed")
	ErrDeleteFailed       = errors.New("delete failed")
	ErrMetadataFailed     = errors.New("metadata operation failed")
	ErrBackupFailed       = errors.New("backup operation failed")
	ErrRetentionViolation = errors.New("retention policy violation")
	ErrAccessDenied       = errors.New("access denied")
	ErrFileTooLarge       = errors.New("file exceeds maximum size")
	ErrInvalidFileType    = errors.New("file type not allowed")
	ErrChecksumMismatch   = errors.New("checksum verification failed")
	ErrInvalidContent     = errors.New("invalid content")
)

// =============================================================================
// Configuration Types
// =============================================================================

// Config contains all storage service configuration
type Config struct {
	// AWS Configuration
	AWS AWSConfig `json:"aws"`

	// Storage buckets
	Buckets BucketConfig `json:"buckets"`

	// Upload settings
	Upload UploadConfig `json:"upload"`

	// Security settings
	Security SecurityConfig `json:"security"`

	// Backup settings
	Backup BackupConfig `json:"backup"`

	// Retention settings
	Retention RetentionConfig `json:"retention"`

	// Metadata settings
	Metadata MetadataConfig `json:"metadata"`

	// Performance settings
	Performance PerformanceConfig `json:"performance"`
}

// AWSConfig contains AWS-specific settings
type AWSConfig struct {
	// Region for S3 operations
	Region string `json:"region"`

	// Custom endpoint (for S3-compatible services like MinIO)
	Endpoint string `json:"endpoint"`

	// Use path-style addressing (required for some S3-compatible services)
	UsePathStyle bool `json:"use_path_style"`

	// Access credentials (optional - uses default credential chain if empty)
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`

	// Session token (for temporary credentials)
	SessionToken string `json:"session_token"`

	// Profile name (for credential file)
	Profile string `json:"profile"`

	// Disable SSL (for local testing only)
	DisableSSL bool `json:"disable_ssl"`
}

// BucketConfig defines the storage buckets
type BucketConfig struct {
	// Primary bucket for documents
	Primary string `json:"primary"`

	// Backup bucket for redundancy
	Backup string `json:"backup"`

	// Archive bucket for long-term storage
	Archive string `json:"archive"`

	// Temporary bucket for uploads in progress
	Temporary string `json:"temporary"`

	// Auto-create buckets if they don't exist
	AutoCreate bool `json:"auto_create"`
}

// UploadConfig contains upload-related settings
type UploadConfig struct {
	// Maximum file size in bytes (default: 100MB)
	MaxFileSize int64 `json:"max_file_size"`

	// Allowed MIME types (empty = all types allowed)
	AllowedMimeTypes []string `json:"allowed_mime_types"`

	// Blocked MIME types
	BlockedMimeTypes []string `json:"blocked_mime_types"`

	// Allowed file extensions (empty = all extensions allowed)
	AllowedExtensions []string `json:"allowed_extensions"`

	// Multipart upload threshold (files larger than this use multipart)
	MultipartThreshold int64 `json:"multipart_threshold"`

	// Multipart chunk size
	MultipartChunkSize int64 `json:"multipart_chunk_size"`

	// Upload timeout
	Timeout time.Duration `json:"timeout"`

	// Enable server-side encryption
	EnableEncryption bool `json:"enable_encryption"`

	// KMS key ID for encryption (if using KMS)
	KMSKeyID string `json:"kms_key_id"`

	// Verify checksum after upload
	VerifyChecksum bool `json:"verify_checksum"`

	// Generate thumbnails for images
	GenerateThumbnails bool `json:"generate_thumbnails"`

	// Thumbnail sizes
	ThumbnailSizes []ThumbnailSize `json:"thumbnail_sizes"`
}

// ThumbnailSize defines a thumbnail dimension
type ThumbnailSize struct {
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// SecurityConfig contains security-related settings
type SecurityConfig struct {
	// Enable server-side encryption
	ServerSideEncryption string `json:"server_side_encryption"` // "AES256", "aws:kms", ""

	// KMS key ID for SSE-KMS
	KMSKeyID string `json:"kms_key_id"`

	// Enable bucket versioning
	EnableVersioning bool `json:"enable_versioning"`

	// Presigned URL expiration
	PresignedURLExpiration time.Duration `json:"presigned_url_expiration"`

	// Enable access logging
	EnableAccessLogging bool `json:"enable_access_logging"`

	// Access log bucket
	AccessLogBucket string `json:"access_log_bucket"`

	// Block public access
	BlockPublicAccess bool `json:"block_public_access"`

	// Enable CORS
	EnableCORS bool `json:"enable_cors"`

	// CORS allowed origins
	CORSAllowedOrigins []string `json:"cors_allowed_origins"`

	// Require secure transport (HTTPS only)
	RequireSecureTransport bool `json:"require_secure_transport"`

	// Enable object lock
	EnableObjectLock bool `json:"enable_object_lock"`
}

// BackupConfig contains backup-related settings
type BackupConfig struct {
	// Enable automatic backups
	Enabled bool `json:"enabled"`

	// Backup schedule (cron expression)
	Schedule string `json:"schedule"`

	// Backup to different region
	CrossRegion bool `json:"cross_region"`

	// Cross-region backup destination
	CrossRegionDestination string `json:"cross_region_destination"`

	// Number of backup copies to keep
	MaxCopies int `json:"max_copies"`

	// Backup compression
	EnableCompression bool `json:"enable_compression"`

	// Compression level (1-9)
	CompressionLevel int `json:"compression_level"`

	// Incremental backups
	Incremental bool `json:"incremental"`

	// Full backup interval (for incremental mode)
	FullBackupInterval time.Duration `json:"full_backup_interval"`

	// Verify backup integrity
	VerifyIntegrity bool `json:"verify_integrity"`

	// Notification on backup completion
	NotifyOnComplete bool `json:"notify_on_complete"`

	// Notification on backup failure
	NotifyOnFailure bool `json:"notify_on_failure"`

	// Notification endpoints
	NotificationEndpoints []string `json:"notification_endpoints"`
}

// RetentionConfig contains retention policy settings
type RetentionConfig struct {
	// Enable retention policies
	Enabled bool `json:"enabled"`

	// Default retention period
	DefaultRetention time.Duration `json:"default_retention"`

	// Document type specific retention
	DocumentRetention map[string]time.Duration `json:"document_retention"`

	// Archive after period
	ArchiveAfter time.Duration `json:"archive_after"`

	// Delete after archive period
	DeleteAfterArchive time.Duration `json:"delete_after_archive"`

	// Enable lifecycle rules
	EnableLifecycle bool `json:"enable_lifecycle"`

	// Transition to Glacier after
	GlacierTransition time.Duration `json:"glacier_transition"`

	// Transition to Deep Archive after
	DeepArchiveTransition time.Duration `json:"deep_archive_transition"`

	// Minimum retention period (cannot delete before this)
	MinRetention time.Duration `json:"min_retention"`

	// Legal hold support
	EnableLegalHold bool `json:"enable_legal_hold"`

	// Exempt document types from deletion
	ExemptTypes []string `json:"exempt_types"`
}

// MetadataConfig contains metadata settings
type MetadataConfig struct {
	// Enable metadata storage
	Enabled bool `json:"enabled"`

	// Metadata storage backend ("dynamodb", "rds", "s3")
	Backend string `json:"backend"`

	// DynamoDB table name (if using DynamoDB)
	DynamoDBTable string `json:"dynamodb_table"`

	// Index common fields
	IndexFields []string `json:"index_fields"`

	// Enable full-text search
	EnableSearch bool `json:"enable_search"`

	// Search engine ("elasticsearch", "opensearch", "none")
	SearchEngine string `json:"search_engine"`

	// Search endpoint
	SearchEndpoint string `json:"search_endpoint"`

	// Enable metadata versioning
	EnableVersioning bool `json:"enable_versioning"`

	// Max versions to keep
	MaxVersions int `json:"max_versions"`

	// Custom metadata fields
	CustomFields []CustomMetadataField `json:"custom_fields"`
}

// CustomMetadataField defines a custom metadata field
type CustomMetadataField struct {
	Name         string `json:"name"`
	Type         string `json:"type"` // "string", "number", "date", "boolean", "array"
	Required     bool   `json:"required"`
	Indexed      bool   `json:"indexed"`
	Searchable   bool   `json:"searchable"`
	DefaultValue string `json:"default_value"`
}

// PerformanceConfig contains performance settings
type PerformanceConfig struct {
	// Max concurrent uploads
	MaxConcurrentUploads int `json:"max_concurrent_uploads"`

	// Max concurrent downloads
	MaxConcurrentDownloads int `json:"max_concurrent_downloads"`

	// Connection pool size
	ConnectionPoolSize int `json:"connection_pool_size"`

	// Request timeout
	RequestTimeout time.Duration `json:"request_timeout"`

	// Retry configuration
	MaxRetries    int           `json:"max_retries"`
	RetryDelay    time.Duration `json:"retry_delay"`
	MaxRetryDelay time.Duration `json:"max_retry_delay"`

	// Enable request caching
	EnableCache bool `json:"enable_cache"`

	// Cache TTL
	CacheTTL time.Duration `json:"cache_ttl"`

	// Cache size limit
	CacheMaxSize int64 `json:"cache_max_size"`

	// Use transfer acceleration
	EnableAcceleration bool `json:"enable_acceleration"`

	// Buffer pool size
	BufferPoolSize int `json:"buffer_pool_size"`
}

// =============================================================================
// Default Configuration
// =============================================================================

// DefaultConfig returns the default storage configuration
func DefaultConfig() *Config {
	return &Config{
		AWS: AWSConfig{
			Region:       "us-east-1",
			UsePathStyle: false,
		},
		Buckets: BucketConfig{
			Primary:    "servicepro-documents",
			Backup:     "servicepro-documents-backup",
			Archive:    "servicepro-documents-archive",
			Temporary:  "servicepro-documents-temp",
			AutoCreate: false,
		},
		Upload: UploadConfig{
			MaxFileSize:        100 * 1024 * 1024, // 100MB
			MultipartThreshold: 5 * 1024 * 1024,   // 5MB
			MultipartChunkSize: 5 * 1024 * 1024,   // 5MB
			Timeout:            30 * time.Minute,
			EnableEncryption:   true,
			VerifyChecksum:     true,
			AllowedMimeTypes: []string{
				"application/pdf",
				"image/jpeg",
				"image/png",
				"image/gif",
				"image/webp",
				"application/msword",
				"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
				"application/vnd.ms-excel",
				"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				"text/plain",
				"text/csv",
			},
			ThumbnailSizes: []ThumbnailSize{
				{Name: "small", Width: 150, Height: 150},
				{Name: "medium", Width: 300, Height: 300},
				{Name: "large", Width: 600, Height: 600},
			},
		},
		Security: SecurityConfig{
			ServerSideEncryption:   "AES256",
			EnableVersioning:       true,
			PresignedURLExpiration: 1 * time.Hour,
			EnableAccessLogging:    true,
			BlockPublicAccess:      true,
			EnableCORS:             true,
			RequireSecureTransport: true,
			CORSAllowedOrigins:     []string{"*"},
		},
		Backup: BackupConfig{
			Enabled:            true,
			Schedule:           "0 2 * * *", // Daily at 2 AM
			MaxCopies:          7,
			EnableCompression:  true,
			CompressionLevel:   6,
			Incremental:        true,
			FullBackupInterval: 7 * 24 * time.Hour, // Weekly
			VerifyIntegrity:    true,
			NotifyOnFailure:    true,
		},
		Retention: RetentionConfig{
			Enabled:          true,
			DefaultRetention: 7 * 365 * 24 * time.Hour, // 7 years
			DocumentRetention: map[string]time.Duration{
				"invoice":        7 * 365 * 24 * time.Hour,  // 7 years
				"quote":          1 * 365 * 24 * time.Hour,  // 1 year
				"service_report": 5 * 365 * 24 * time.Hour,  // 5 years
				"contract":       10 * 365 * 24 * time.Hour, // 10 years
				"temporary":      30 * 24 * time.Hour,       // 30 days
			},
			ArchiveAfter:          2 * 365 * 24 * time.Hour, // 2 years
			DeleteAfterArchive:    5 * 365 * 24 * time.Hour, // 5 years after archive
			EnableLifecycle:       true,
			GlacierTransition:     1 * 365 * 24 * time.Hour, // 1 year
			DeepArchiveTransition: 3 * 365 * 24 * time.Hour, // 3 years
			MinRetention:          30 * 24 * time.Hour,      // 30 days
			EnableLegalHold:       true,
		},
		Metadata: MetadataConfig{
			Enabled:          true,
			Backend:          "s3",
			EnableVersioning: true,
			MaxVersions:      10,
			IndexFields:      []string{"document_type", "customer_id", "created_at", "status"},
		},
		Performance: PerformanceConfig{
			MaxConcurrentUploads:   10,
			MaxConcurrentDownloads: 20,
			ConnectionPoolSize:     50,
			RequestTimeout:         30 * time.Second,
			MaxRetries:             3,
			RetryDelay:             1 * time.Second,
			MaxRetryDelay:          30 * time.Second,
			EnableCache:            true,
			CacheTTL:               5 * time.Minute,
			CacheMaxSize:           100 * 1024 * 1024, // 100MB
			BufferPoolSize:         32,
		},
	}
}

// =============================================================================
// Configuration Validation
// =============================================================================

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.AWS.Region == "" {
		return fmt.Errorf("%w: AWS region is required", ErrInvalidConfig)
	}

	if c.Buckets.Primary == "" {
		return fmt.Errorf("%w: primary bucket is required", ErrInvalidConfig)
	}

	if c.Upload.MaxFileSize <= 0 {
		c.Upload.MaxFileSize = 100 * 1024 * 1024 // Default 100MB
	}

	if c.Upload.MultipartThreshold <= 0 {
		c.Upload.MultipartThreshold = 5 * 1024 * 1024 // Default 5MB
	}

	if c.Upload.MultipartChunkSize <= 0 {
		c.Upload.MultipartChunkSize = 5 * 1024 * 1024 // Default 5MB
	}

	if c.Security.PresignedURLExpiration <= 0 {
		c.Security.PresignedURLExpiration = 1 * time.Hour
	}

	if c.Performance.MaxConcurrentUploads <= 0 {
		c.Performance.MaxConcurrentUploads = 10
	}

	if c.Performance.MaxConcurrentDownloads <= 0 {
		c.Performance.MaxConcurrentDownloads = 20
	}

	if c.Performance.MaxRetries < 0 {
		c.Performance.MaxRetries = 3
	}

	if c.Backup.Enabled {
		if c.Backup.Schedule == "" {
			c.Backup.Schedule = "0 2 * * *"
		}
		if c.Backup.MaxCopies <= 0 {
			c.Backup.MaxCopies = 7
		}
	}

	if c.Retention.Enabled && c.Retention.DefaultRetention <= 0 {
		c.Retention.DefaultRetention = 7 * 365 * 24 * time.Hour
	}

	return nil
}

// =============================================================================
// Configuration Builder
// =============================================================================

// ConfigBuilder provides a fluent interface for building configuration
type ConfigBuilder struct {
	config *Config
}

// NewConfigBuilder creates a new configuration builder
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: DefaultConfig(),
	}
}

// WithRegion sets the AWS region
func (b *ConfigBuilder) WithRegion(region string) *ConfigBuilder {
	b.config.AWS.Region = region
	return b
}

// WithEndpoint sets a custom endpoint (for S3-compatible services)
func (b *ConfigBuilder) WithEndpoint(endpoint string) *ConfigBuilder {
	b.config.AWS.Endpoint = endpoint
	b.config.AWS.UsePathStyle = true
	return b
}

// WithCredentials sets AWS credentials
func (b *ConfigBuilder) WithCredentials(accessKey, secretKey string) *ConfigBuilder {
	b.config.AWS.AccessKeyID = accessKey
	b.config.AWS.SecretAccessKey = secretKey
	return b
}

// WithPrimaryBucket sets the primary bucket
func (b *ConfigBuilder) WithPrimaryBucket(bucket string) *ConfigBuilder {
	b.config.Buckets.Primary = bucket
	return b
}

// WithBackupBucket sets the backup bucket
func (b *ConfigBuilder) WithBackupBucket(bucket string) *ConfigBuilder {
	b.config.Buckets.Backup = bucket
	return b
}

// WithMaxFileSize sets the maximum file size
func (b *ConfigBuilder) WithMaxFileSize(size int64) *ConfigBuilder {
	b.config.Upload.MaxFileSize = size
	return b
}

// WithEncryption enables server-side encryption
func (b *ConfigBuilder) WithEncryption(encType string, kmsKeyID string) *ConfigBuilder {
	b.config.Security.ServerSideEncryption = encType
	b.config.Security.KMSKeyID = kmsKeyID
	return b
}

// WithVersioning enables bucket versioning
func (b *ConfigBuilder) WithVersioning(enabled bool) *ConfigBuilder {
	b.config.Security.EnableVersioning = enabled
	return b
}

// WithBackup configures backup settings
func (b *ConfigBuilder) WithBackup(enabled bool, schedule string) *ConfigBuilder {
	b.config.Backup.Enabled = enabled
	b.config.Backup.Schedule = schedule
	return b
}

// WithRetention sets the default retention period
func (b *ConfigBuilder) WithRetention(duration time.Duration) *ConfigBuilder {
	b.config.Retention.Enabled = true
	b.config.Retention.DefaultRetention = duration
	return b
}

// Build validates and returns the configuration
func (b *ConfigBuilder) Build() (*Config, error) {
	if err := b.config.Validate(); err != nil {
		return nil, err
	}
	return b.config, nil
}

// =============================================================================
// Document Types
// =============================================================================

// DocumentType represents the type of document
type DocumentType string

const (
	DocumentTypeInvoice       DocumentType = "invoice"
	DocumentTypeQuote         DocumentType = "quote"
	DocumentTypeServiceReport DocumentType = "service_report"
	DocumentTypeContract      DocumentType = "contract"
	DocumentTypeReceipt       DocumentType = "receipt"
	DocumentTypeWorkOrder     DocumentType = "work_order"
	DocumentTypePhoto         DocumentType = "photo"
	DocumentTypeAttachment    DocumentType = "attachment"
	DocumentTypeTemporary     DocumentType = "temporary"
)

// String returns the string representation
func (d DocumentType) String() string {
	return string(d)
}

// IsValid checks if the document type is valid
func (d DocumentType) IsValid() bool {
	switch d {
	case DocumentTypeInvoice, DocumentTypeQuote, DocumentTypeServiceReport,
		DocumentTypeContract, DocumentTypeReceipt, DocumentTypeWorkOrder,
		DocumentTypePhoto, DocumentTypeAttachment, DocumentTypeTemporary:
		return true
	default:
		return false
	}
}

// =============================================================================
// Storage Classes
// =============================================================================

// StorageClass represents S3 storage classes
type StorageClass string

const (
	StorageClassStandard           StorageClass = "STANDARD"
	StorageClassIntelligentTiering StorageClass = "INTELLIGENT_TIERING"
	StorageClassStandardIA         StorageClass = "STANDARD_IA"
	StorageClassOneZoneIA          StorageClass = "ONEZONE_IA"
	StorageClassGlacier            StorageClass = "GLACIER"
	StorageClassGlacierIR          StorageClass = "GLACIER_IR"
	StorageClassDeepArchive        StorageClass = "DEEP_ARCHIVE"
)
