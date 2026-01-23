package storage

import (
	"io"
	"time"
)

// Provider represents a storage service provider
type Provider string

const (
	ProviderS3   Provider = "s3"
	ProviderR2   Provider = "r2"
	ProviderMock Provider = "mock"
)

// StorageClass represents storage tier/class
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

// UploadInput contains parameters for uploading a file
type UploadInput struct {
	// File content - required for Upload, ignored for UploadBytes
	Body io.Reader

	// Key is the object key (path). If empty, one will be generated.
	Key string

	// FileName is the original filename (used for key generation and metadata)
	FileName string

	// Size is the file size in bytes (for validation)
	Size int64

	// ContentType is the MIME type. Auto-detected if empty.
	ContentType string

	// Metadata is custom metadata to store with the object
	Metadata map[string]string

	// Tags are object tags
	Tags map[string]string

	// StorageClass specifies the storage class/tier
	StorageClass StorageClass

	// CacheControl sets the Cache-Control header
	CacheControl string

	// ContentDisposition sets the Content-Disposition header
	ContentDisposition string

	// Encrypt enables server-side encryption (overrides config)
	Encrypt *bool
}

// UploadOutput contains the result of an upload operation
type UploadOutput struct {
	// Key is the object key
	Key string

	// VersionID is the object version (if versioning enabled)
	VersionID string

	// ETag is the entity tag
	ETag string

	// URL is the object URL
	URL string

	// Checksum is the SHA256 checksum (if computed)
	Checksum string

	// Size is the uploaded size in bytes
	Size int64

	// ContentType is the content type
	ContentType string

	// UploadedAt is when the upload completed
	UploadedAt time.Time
}

// DownloadInput contains parameters for downloading a file
type DownloadInput struct {
	// Key is the object key - required
	Key string

	// VersionID specifies a specific version to download
	VersionID string

	// Range specifies a byte range (e.g., "bytes=0-1023")
	Range string

	// IfModifiedSince returns the object only if modified since this time
	IfModifiedSince *time.Time

	// IfNoneMatch returns the object only if ETag doesn't match
	IfNoneMatch string
}

// DownloadOutput contains the result of a download operation
type DownloadOutput struct {
	// Body is the object content - caller must close
	Body io.ReadCloser

	// Size is the content size in bytes
	Size int64

	// ContentType is the MIME type
	ContentType string

	// ETag is the entity tag
	ETag string

	// LastModified is when the object was last modified
	LastModified time.Time

	// VersionID is the object version
	VersionID string

	// Metadata is custom metadata stored with the object
	Metadata map[string]string

	// CacheControl is the Cache-Control header
	CacheControl string

	// ContentDisposition is the Content-Disposition header
	ContentDisposition string
}

// ObjectInfo contains metadata about an object
type ObjectInfo struct {
	// Key is the object key
	Key string

	// Size is the object size in bytes
	Size int64

	// ContentType is the MIME type
	ContentType string

	// ETag is the entity tag
	ETag string

	// LastModified is when the object was last modified
	LastModified time.Time

	// VersionID is the object version
	VersionID string

	// StorageClass is the storage class
	StorageClass string

	// Metadata is custom metadata
	Metadata map[string]string
}

// ListInput contains parameters for listing objects
type ListInput struct {
	// Prefix filters objects to those starting with this prefix
	Prefix string

	// Delimiter groups results by this delimiter (usually "/")
	Delimiter string

	// MaxKeys limits the number of results
	MaxKeys int32

	// ContinuationToken continues from a previous list
	ContinuationToken string

	// StartAfter starts listing after this key
	StartAfter string
}

// ListOutput contains the result of a list operation
type ListOutput struct {
	// Objects is the list of objects
	Objects []ObjectInfo

	// CommonPrefixes lists "directory" prefixes when delimiter is used
	CommonPrefixes []string

	// ContinuationToken is used to continue listing
	ContinuationToken string

	// IsTruncated indicates if there are more results
	IsTruncated bool
}

// CopyOutput contains the result of a copy operation
type CopyOutput struct {
	// ETag is the entity tag of the new object
	ETag string

	// LastModified is when the copy completed
	LastModified time.Time

	// VersionID is the new object's version
	VersionID string
}

// NewUploadInput creates a new UploadInput with the given key
func NewUploadInput(key string) *UploadInput {
	return &UploadInput{
		Key:      key,
		Metadata: make(map[string]string),
		Tags:     make(map[string]string),
	}
}

// WithBody sets the body
func (u *UploadInput) WithBody(body io.Reader) *UploadInput {
	u.Body = body
	return u
}

// WithFileName sets the filename
func (u *UploadInput) WithFileName(fileName string) *UploadInput {
	u.FileName = fileName
	return u
}

// WithContentType sets the content type
func (u *UploadInput) WithContentType(contentType string) *UploadInput {
	u.ContentType = contentType
	return u
}

// WithSize sets the size
func (u *UploadInput) WithSize(size int64) *UploadInput {
	u.Size = size
	return u
}

// WithMetadata adds metadata
func (u *UploadInput) WithMetadata(key, value string) *UploadInput {
	if u.Metadata == nil {
		u.Metadata = make(map[string]string)
	}
	u.Metadata[key] = value
	return u
}

// WithTag adds a tag
func (u *UploadInput) WithTag(key, value string) *UploadInput {
	if u.Tags == nil {
		u.Tags = make(map[string]string)
	}
	u.Tags[key] = value
	return u
}

// WithStorageClass sets the storage class
func (u *UploadInput) WithStorageClass(class StorageClass) *UploadInput {
	u.StorageClass = class
	return u
}

// WithCacheControl sets the cache control header
func (u *UploadInput) WithCacheControl(cacheControl string) *UploadInput {
	u.CacheControl = cacheControl
	return u
}
