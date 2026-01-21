package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// =============================================================================
// Document Metadata
// =============================================================================

// DocumentMetadata contains metadata for a stored document
type DocumentMetadata struct {
	// Unique identifier
	ID string `json:"id"`

	// Storage key
	Key string `json:"key"`

	// Original filename
	FileName string `json:"file_name"`

	// File size in bytes
	Size int64 `json:"size"`

	// MIME type
	ContentType string `json:"content_type"`

	// Document type
	DocumentType DocumentType `json:"document_type"`

	// Associated entity
	EntityID   string `json:"entity_id,omitempty"`
	EntityType string `json:"entity_type,omitempty"`

	// Timestamps
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	AccessedAt *time.Time `json:"accessed_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	ArchivedAt *time.Time `json:"archived_at,omitempty"`

	// Version info
	Version   int    `json:"version"`
	VersionID string `json:"version_id,omitempty"`
	IsLatest  bool   `json:"is_latest"`

	// Storage info
	StorageClass string `json:"storage_class"`
	Bucket       string `json:"bucket"`
	ETag         string `json:"etag"`

	// Security
	Checksum  string `json:"checksum,omitempty"`
	Encrypted bool   `json:"encrypted"`
	LegalHold bool   `json:"legal_hold"`

	// Access control
	Owner       string   `json:"owner,omitempty"`
	AccessLevel string   `json:"access_level,omitempty"`
	SharedWith  []string `json:"shared_with,omitempty"`

	// Custom metadata
	Tags       map[string]string      `json:"tags,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`

	// Status
	Status string `json:"status"` // "active", "archived", "deleted", "processing"

	// Thumbnails (for images)
	Thumbnails map[string]string `json:"thumbnails,omitempty"`

	// Parent/child relationships
	ParentID    string   `json:"parent_id,omitempty"`
	ChildrenIDs []string `json:"children_ids,omitempty"`

	// Audit
	CreatedBy      string `json:"created_by,omitempty"`
	LastModifiedBy string `json:"last_modified_by,omitempty"`
}

// =============================================================================
// Metadata Store Interface
// =============================================================================

// MetadataStore defines the interface for metadata storage
type MetadataStore interface {
	// Save saves document metadata
	Save(ctx context.Context, metadata *DocumentMetadata) error

	// Get retrieves metadata by ID
	Get(ctx context.Context, id string) (*DocumentMetadata, error)

	// GetByKey retrieves metadata by storage key
	GetByKey(ctx context.Context, key string) (*DocumentMetadata, error)

	// Update updates metadata
	Update(ctx context.Context, metadata *DocumentMetadata) error

	// Delete deletes metadata
	Delete(ctx context.Context, id string) error

	// List lists metadata with optional filtering
	List(ctx context.Context, filter *MetadataFilter) (*MetadataListResult, error)

	// Search searches metadata
	Search(ctx context.Context, query string, filter *MetadataFilter) (*MetadataListResult, error)

	// GetVersions gets all versions of a document
	GetVersions(ctx context.Context, id string) ([]*DocumentMetadata, error)

	// BulkSave saves multiple metadata records
	BulkSave(ctx context.Context, metadata []*DocumentMetadata) error

	// BulkDelete deletes multiple metadata records
	BulkDelete(ctx context.Context, ids []string) error
}

// MetadataFilter contains filter options for listing metadata
type MetadataFilter struct {
	// Filter by document type
	DocumentType DocumentType `json:"document_type,omitempty"`

	// Filter by entity
	EntityID   string `json:"entity_id,omitempty"`
	EntityType string `json:"entity_type,omitempty"`

	// Filter by status
	Status string `json:"status,omitempty"`

	// Filter by owner
	Owner string `json:"owner,omitempty"`

	// Filter by tags
	Tags map[string]string `json:"tags,omitempty"`

	// Filter by date range
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`

	// Filter by size
	MinSize int64 `json:"min_size,omitempty"`
	MaxSize int64 `json:"max_size,omitempty"`

	// Pagination
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Cursor string `json:"cursor,omitempty"`

	// Sorting
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"` // "asc" or "desc"

	// Include deleted
	IncludeDeleted bool `json:"include_deleted,omitempty"`

	// Include archived
	IncludeArchived bool `json:"include_archived,omitempty"`
}

// MetadataListResult contains the result of a list operation
type MetadataListResult struct {
	Items      []*DocumentMetadata `json:"items"`
	TotalCount int                 `json:"total_count"`
	Cursor     string              `json:"cursor,omitempty"`
	HasMore    bool                `json:"has_more"`
}

// =============================================================================
// S3 Metadata Store Implementation
// =============================================================================

// S3MetadataStore stores metadata as JSON files in S3
type S3MetadataStore struct {
	client         *S3Client
	metadataBucket string
	prefix         string
	cache          *metadataCache
	mu             sync.RWMutex
}

// metadataCache provides in-memory caching for metadata
type metadataCache struct {
	items   map[string]*cachedMetadata
	maxSize int
	mu      sync.RWMutex
}

type cachedMetadata struct {
	data      *DocumentMetadata
	expiresAt time.Time
}

// NewS3MetadataStore creates a new S3-based metadata store
func NewS3MetadataStore(client *S3Client) *S3MetadataStore {
	return &S3MetadataStore{
		client:         client,
		metadataBucket: client.config.Buckets.Primary,
		prefix:         ".metadata/",
		cache: &metadataCache{
			items:   make(map[string]*cachedMetadata),
			maxSize: 1000,
		},
	}
}

// Save saves document metadata
func (s *S3MetadataStore) Save(ctx context.Context, metadata *DocumentMetadata) error {
	if metadata.ID == "" {
		return fmt.Errorf("%w: metadata ID is required", ErrMetadataFailed)
	}

	metadata.UpdatedAt = time.Now().UTC()
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = metadata.UpdatedAt
	}
	metadata.IsLatest = true
	metadata.Status = "active"

	// Serialize to JSON
	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal metadata: %v", ErrMetadataFailed, err)
	}

	// Generate key
	key := s.metadataKey(metadata.ID)

	// Upload to S3
	_, err = s.client.UploadBytes(ctx, data, &UploadInput{
		Key:          key,
		FileName:     metadata.ID + ".json",
		Size:         int64(len(data)),
		ContentType:  "application/json",
		DocumentType: DocumentTypeAttachment,
	})

	if err != nil {
		return fmt.Errorf("%w: failed to save metadata: %v", ErrMetadataFailed, err)
	}

	// Update cache
	s.cache.set(metadata.ID, metadata)

	return nil
}

// Get retrieves metadata by ID
func (s *S3MetadataStore) Get(ctx context.Context, id string) (*DocumentMetadata, error) {
	// Check cache first
	if cached := s.cache.get(id); cached != nil {
		return cached, nil
	}

	// Download from S3
	key := s.metadataKey(id)
	data, err := s.client.DownloadBytes(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("%w: metadata not found: %v", ErrObjectNotFound, err)
	}

	// Deserialize
	var metadata DocumentMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal metadata: %v", ErrMetadataFailed, err)
	}

	// Update cache
	s.cache.set(id, &metadata)

	return &metadata, nil
}

// GetByKey retrieves metadata by storage key
func (s *S3MetadataStore) GetByKey(ctx context.Context, key string) (*DocumentMetadata, error) {
	// Search for metadata with this key
	result, err := s.List(ctx, &MetadataFilter{Limit: 1})
	if err != nil {
		return nil, err
	}

	for _, item := range result.Items {
		if item.Key == key {
			return item, nil
		}
	}

	return nil, ErrObjectNotFound
}

// Update updates metadata
func (s *S3MetadataStore) Update(ctx context.Context, metadata *DocumentMetadata) error {
	metadata.UpdatedAt = time.Now().UTC()
	metadata.Version++

	return s.Save(ctx, metadata)
}

// Delete deletes metadata (soft delete)
func (s *S3MetadataStore) Delete(ctx context.Context, id string) error {
	// Get current metadata
	metadata, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	// Mark as deleted
	metadata.Status = "deleted"
	now := time.Now().UTC()
	metadata.ArchivedAt = &now

	// Save updated metadata
	if err := s.Save(ctx, metadata); err != nil {
		return err
	}

	// Remove from cache
	s.cache.delete(id)

	return nil
}

// HardDelete permanently deletes metadata
func (s *S3MetadataStore) HardDelete(ctx context.Context, id string) error {
	key := s.metadataKey(id)

	if err := s.client.Delete(ctx, key); err != nil {
		return fmt.Errorf("%w: failed to delete metadata: %v", ErrMetadataFailed, err)
	}

	s.cache.delete(id)

	return nil
}

// List lists metadata with optional filtering
func (s *S3MetadataStore) List(ctx context.Context, filter *MetadataFilter) (*MetadataListResult, error) {
	if filter == nil {
		filter = &MetadataFilter{Limit: 100}
	}

	if filter.Limit <= 0 {
		filter.Limit = 100
	}

	// List all metadata files
	objects, err := s.client.ListAll(ctx, s.prefix)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list metadata: %v", ErrMetadataFailed, err)
	}

	// Load and filter metadata
	var items []*DocumentMetadata
	for _, obj := range objects {
		if !strings.HasSuffix(obj.Key, ".json") {
			continue
		}

		// Download metadata
		data, err := s.client.DownloadBytes(ctx, obj.Key)
		if err != nil {
			continue
		}

		var metadata DocumentMetadata
		if err := json.Unmarshal(data, &metadata); err != nil {
			continue
		}

		// Apply filters
		if !s.matchesFilter(&metadata, filter) {
			continue
		}

		items = append(items, &metadata)
	}

	// Sort
	s.sortMetadata(items, filter.SortBy, filter.SortOrder)

	// Apply pagination
	totalCount := len(items)
	if filter.Offset > 0 && filter.Offset < len(items) {
		items = items[filter.Offset:]
	}

	hasMore := len(items) > filter.Limit
	if len(items) > filter.Limit {
		items = items[:filter.Limit]
	}

	return &MetadataListResult{
		Items:      items,
		TotalCount: totalCount,
		HasMore:    hasMore,
	}, nil
}

// Search searches metadata (basic implementation)
func (s *S3MetadataStore) Search(ctx context.Context, query string, filter *MetadataFilter) (*MetadataListResult, error) {
	// Get all items first
	result, err := s.List(ctx, &MetadataFilter{Limit: 10000})
	if err != nil {
		return nil, err
	}

	// Filter by search query
	query = strings.ToLower(query)
	var matched []*DocumentMetadata

	for _, item := range result.Items {
		if s.matchesSearch(item, query) {
			matched = append(matched, item)
		}
	}

	// Apply additional filters
	if filter != nil {
		var filtered []*DocumentMetadata
		for _, item := range matched {
			if s.matchesFilter(item, filter) {
				filtered = append(filtered, item)
			}
		}
		matched = filtered
	}

	// Apply pagination
	limit := 100
	if filter != nil && filter.Limit > 0 {
		limit = filter.Limit
	}

	hasMore := len(matched) > limit
	if len(matched) > limit {
		matched = matched[:limit]
	}

	return &MetadataListResult{
		Items:      matched,
		TotalCount: len(matched),
		HasMore:    hasMore,
	}, nil
}

// GetVersions gets all versions of a document
func (s *S3MetadataStore) GetVersions(ctx context.Context, id string) ([]*DocumentMetadata, error) {
	// List version metadata files
	prefix := s.prefix + "versions/" + id + "/"
	objects, err := s.client.ListAll(ctx, prefix)
	if err != nil {
		return nil, err
	}

	var versions []*DocumentMetadata
	for _, obj := range objects {
		data, err := s.client.DownloadBytes(ctx, obj.Key)
		if err != nil {
			continue
		}

		var metadata DocumentMetadata
		if err := json.Unmarshal(data, &metadata); err != nil {
			continue
		}

		versions = append(versions, &metadata)
	}

	// Sort by version number
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})

	return versions, nil
}

// BulkSave saves multiple metadata records
func (s *S3MetadataStore) BulkSave(ctx context.Context, metadata []*DocumentMetadata) error {
	for _, m := range metadata {
		if err := s.Save(ctx, m); err != nil {
			return err
		}
	}
	return nil
}

// BulkDelete deletes multiple metadata records
func (s *S3MetadataStore) BulkDelete(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := s.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// =============================================================================
// Helper Methods
// =============================================================================

func (s *S3MetadataStore) metadataKey(id string) string {
	return s.prefix + id + ".json"
}

func (s *S3MetadataStore) matchesFilter(m *DocumentMetadata, f *MetadataFilter) bool {
	// Document type filter
	if f.DocumentType != "" && m.DocumentType != f.DocumentType {
		return false
	}

	// Entity filters
	if f.EntityID != "" && m.EntityID != f.EntityID {
		return false
	}
	if f.EntityType != "" && m.EntityType != f.EntityType {
		return false
	}

	// Status filter
	if f.Status != "" && m.Status != f.Status {
		return false
	}

	// Owner filter
	if f.Owner != "" && m.Owner != f.Owner {
		return false
	}

	// Exclude deleted unless specified
	if !f.IncludeDeleted && m.Status == "deleted" {
		return false
	}

	// Exclude archived unless specified
	if !f.IncludeArchived && m.Status == "archived" {
		return false
	}

	// Date range filters
	if f.CreatedAfter != nil && m.CreatedAt.Before(*f.CreatedAfter) {
		return false
	}
	if f.CreatedBefore != nil && m.CreatedAt.After(*f.CreatedBefore) {
		return false
	}

	// Size filters
	if f.MinSize > 0 && m.Size < f.MinSize {
		return false
	}
	if f.MaxSize > 0 && m.Size > f.MaxSize {
		return false
	}

	// Tag filters
	if len(f.Tags) > 0 {
		for k, v := range f.Tags {
			if m.Tags[k] != v {
				return false
			}
		}
	}

	return true
}

func (s *S3MetadataStore) matchesSearch(m *DocumentMetadata, query string) bool {
	// Search in filename
	if strings.Contains(strings.ToLower(m.FileName), query) {
		return true
	}

	// Search in entity ID
	if strings.Contains(strings.ToLower(m.EntityID), query) {
		return true
	}

	// Search in tags
	for k, v := range m.Tags {
		if strings.Contains(strings.ToLower(k), query) ||
			strings.Contains(strings.ToLower(v), query) {
			return true
		}
	}

	return false
}

func (s *S3MetadataStore) sortMetadata(items []*DocumentMetadata, sortBy, order string) {
	if sortBy == "" {
		sortBy = "created_at"
	}

	desc := strings.ToLower(order) == "desc"

	sort.Slice(items, func(i, j int) bool {
		var less bool

		switch sortBy {
		case "file_name":
			less = items[i].FileName < items[j].FileName
		case "size":
			less = items[i].Size < items[j].Size
		case "updated_at":
			less = items[i].UpdatedAt.Before(items[j].UpdatedAt)
		default: // created_at
			less = items[i].CreatedAt.Before(items[j].CreatedAt)
		}

		if desc {
			return !less
		}
		return less
	})
}

// Cache methods
func (c *metadataCache) get(id string) *DocumentMetadata {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if cached, ok := c.items[id]; ok {
		if time.Now().Before(cached.expiresAt) {
			return cached.data
		}
		// Expired, will be cleaned up later
	}

	return nil
}

func (c *metadataCache) set(id string, data *DocumentMetadata) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict if at capacity
	if len(c.items) >= c.maxSize {
		// Simple eviction: remove first expired item or random item
		for k, v := range c.items {
			if time.Now().After(v.expiresAt) {
				delete(c.items, k)
				break
			}
		}
	}

	c.items[id] = &cachedMetadata{
		data:      data,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
}

func (c *metadataCache) delete(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, id)
}

// =============================================================================
// Metadata Manager
// =============================================================================

// MetadataManager provides high-level metadata operations
type MetadataManager struct {
	store    MetadataStore
	s3Client *S3Client
	config   *Config
}

// NewMetadataManager creates a new metadata manager
func NewMetadataManager(s3Client *S3Client) *MetadataManager {
	return &MetadataManager{
		store:    NewS3MetadataStore(s3Client),
		s3Client: s3Client,
		config:   s3Client.config,
	}
}

// CreateFromUpload creates metadata from an upload result
func (m *MetadataManager) CreateFromUpload(ctx context.Context, upload *UploadOutput, input *UploadInput) (*DocumentMetadata, error) {
	metadata := &DocumentMetadata{
		ID:           generateID(),
		Key:          upload.Key,
		FileName:     input.FileName,
		Size:         upload.Size,
		ContentType:  upload.ContentType,
		DocumentType: input.DocumentType,
		EntityID:     input.EntityID,
		EntityType:   input.EntityType,
		CreatedAt:    upload.UploadedAt,
		UpdatedAt:    upload.UploadedAt,
		Version:      1,
		VersionID:    upload.VersionID,
		IsLatest:     true,
		StorageClass: string(input.StorageClass),
		Bucket:       m.config.Buckets.Primary,
		ETag:         upload.ETag,
		Checksum:     upload.Checksum,
		Encrypted:    m.config.Upload.EnableEncryption,
		Status:       "active",
		Tags:         input.Tags,
	}

	// Calculate expiration based on retention policy
	if m.config.Retention.Enabled {
		retention := m.config.Retention.DefaultRetention
		if r, ok := m.config.Retention.DocumentRetention[string(input.DocumentType)]; ok {
			retention = r
		}
		expiresAt := time.Now().Add(retention)
		metadata.ExpiresAt = &expiresAt
	}

	if err := m.store.Save(ctx, metadata); err != nil {
		return nil, err
	}

	return metadata, nil
}

// UpdateAccess updates the last accessed time
func (m *MetadataManager) UpdateAccess(ctx context.Context, id string) error {
	metadata, err := m.store.Get(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	metadata.AccessedAt = &now

	return m.store.Update(ctx, metadata)
}

// SetLegalHold sets or removes legal hold on a document
func (m *MetadataManager) SetLegalHold(ctx context.Context, id string, hold bool) error {
	metadata, err := m.store.Get(ctx, id)
	if err != nil {
		return err
	}

	metadata.LegalHold = hold

	// Update S3 object lock if enabled
	if m.config.Security.EnableObjectLock {
		status := "OFF"
		if hold {
			status = "ON"
		}

		_, err = m.s3Client.client.PutObjectLegalHold(ctx, &s3.PutObjectLegalHoldInput{
			Bucket: aws.String(m.config.Buckets.Primary),
			Key:    aws.String(metadata.Key),
			LegalHold: &types.ObjectLockLegalHold{
				Status: types.ObjectLockLegalHoldStatus(status),
			},
		})

		if err != nil {
			return fmt.Errorf("failed to set legal hold: %w", err)
		}
	}

	return m.store.Update(ctx, metadata)
}

// Share shares a document with users
func (m *MetadataManager) Share(ctx context.Context, id string, userIDs []string) error {
	metadata, err := m.store.Get(ctx, id)
	if err != nil {
		return err
	}

	// Add new users to shared list
	shared := make(map[string]bool)
	for _, u := range metadata.SharedWith {
		shared[u] = true
	}
	for _, u := range userIDs {
		shared[u] = true
	}

	metadata.SharedWith = make([]string, 0, len(shared))
	for u := range shared {
		metadata.SharedWith = append(metadata.SharedWith, u)
	}

	return m.store.Update(ctx, metadata)
}

// Unshare removes sharing for a document
func (m *MetadataManager) Unshare(ctx context.Context, id string, userIDs []string) error {
	metadata, err := m.store.Get(ctx, id)
	if err != nil {
		return err
	}

	// Remove specified users
	toRemove := make(map[string]bool)
	for _, u := range userIDs {
		toRemove[u] = true
	}

	var remaining []string
	for _, u := range metadata.SharedWith {
		if !toRemove[u] {
			remaining = append(remaining, u)
		}
	}

	metadata.SharedWith = remaining

	return m.store.Update(ctx, metadata)
}

// GetStore returns the underlying metadata store
func (m *MetadataManager) GetStore() MetadataStore {
	return m.store
}

// =============================================================================
// ID Generation
// =============================================================================

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
