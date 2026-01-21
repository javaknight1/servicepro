package storage

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// Config Tests
// =============================================================================

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Check AWS defaults
	if cfg.AWS.Region != "us-east-1" {
		t.Errorf("expected region us-east-1, got %s", cfg.AWS.Region)
	}

	// Check bucket defaults
	if cfg.Buckets.Primary == "" {
		t.Error("primary bucket should have default value")
	}

	// Check upload defaults
	if cfg.Upload.MaxFileSize != 100*1024*1024 {
		t.Errorf("expected max file size 100MB, got %d", cfg.Upload.MaxFileSize)
	}

	// Check security defaults
	if cfg.Security.ServerSideEncryption != "AES256" {
		t.Errorf("expected encryption AES256, got %s", cfg.Security.ServerSideEncryption)
	}

	// Check backup defaults
	if !cfg.Backup.Enabled {
		t.Error("backup should be enabled by default")
	}

	// Check retention defaults
	if !cfg.Retention.Enabled {
		t.Error("retention should be enabled by default")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		modify    func(*Config)
		wantError bool
	}{
		{
			name:      "valid default config",
			modify:    func(c *Config) {},
			wantError: false,
		},
		{
			name: "missing region",
			modify: func(c *Config) {
				c.AWS.Region = ""
			},
			wantError: true,
		},
		{
			name: "missing primary bucket",
			modify: func(c *Config) {
				c.Buckets.Primary = ""
			},
			wantError: true,
		},
		{
			name: "zero max file size gets corrected",
			modify: func(c *Config) {
				c.Upload.MaxFileSize = 0
			},
			wantError: false,
		},
		{
			name: "negative max concurrent gets corrected",
			modify: func(c *Config) {
				c.Performance.MaxConcurrentUploads = -1
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)

			err := cfg.Validate()

			if tt.wantError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestConfigBuilder(t *testing.T) {
	cfg, err := NewConfigBuilder().
		WithRegion("us-west-2").
		WithPrimaryBucket("my-bucket").
		WithMaxFileSize(50*1024*1024).
		WithEncryption("aws:kms", "key-id").
		WithVersioning(true).
		WithBackup(true, "0 0 * * *").
		WithRetention(365 * 24 * time.Hour).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if cfg.AWS.Region != "us-west-2" {
		t.Errorf("expected region us-west-2, got %s", cfg.AWS.Region)
	}

	if cfg.Buckets.Primary != "my-bucket" {
		t.Errorf("expected bucket my-bucket, got %s", cfg.Buckets.Primary)
	}

	if cfg.Upload.MaxFileSize != 50*1024*1024 {
		t.Errorf("expected max file size 50MB, got %d", cfg.Upload.MaxFileSize)
	}

	if cfg.Security.ServerSideEncryption != "aws:kms" {
		t.Error("expected aws:kms encryption")
	}

	if !cfg.Security.EnableVersioning {
		t.Error("expected versioning to be enabled")
	}
}

// =============================================================================
// Document Type Tests
// =============================================================================

func TestDocumentType(t *testing.T) {
	validTypes := []DocumentType{
		DocumentTypeInvoice,
		DocumentTypeQuote,
		DocumentTypeServiceReport,
		DocumentTypeContract,
		DocumentTypeReceipt,
		DocumentTypeWorkOrder,
		DocumentTypePhoto,
		DocumentTypeAttachment,
		DocumentTypeTemporary,
	}

	for _, docType := range validTypes {
		if !docType.IsValid() {
			t.Errorf("expected %s to be valid", docType)
		}
	}

	invalidType := DocumentType("invalid")
	if invalidType.IsValid() {
		t.Error("expected invalid type to be invalid")
	}
}

func TestDocumentTypeString(t *testing.T) {
	tests := []struct {
		docType DocumentType
		want    string
	}{
		{DocumentTypeInvoice, "invoice"},
		{DocumentTypeQuote, "quote"},
		{DocumentTypeServiceReport, "service_report"},
	}

	for _, tt := range tests {
		if tt.docType.String() != tt.want {
			t.Errorf("expected %s, got %s", tt.want, tt.docType.String())
		}
	}
}

// =============================================================================
// Upload Input Validation Tests
// =============================================================================

func TestUploadInputValidation(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name      string
		input     *UploadInput
		wantError bool
		errorType error
	}{
		{
			name: "valid input",
			input: &UploadInput{
				Body:        bytes.NewReader([]byte("test")),
				FileName:    "test.pdf",
				Size:        4,
				ContentType: "application/pdf",
			},
			wantError: false,
		},
		{
			name: "nil body",
			input: &UploadInput{
				FileName: "test.pdf",
				Size:     4,
			},
			wantError: true,
			errorType: ErrInvalidContent,
		},
		{
			name: "file too large",
			input: &UploadInput{
				Body:     bytes.NewReader([]byte("test")),
				FileName: "test.pdf",
				Size:     200 * 1024 * 1024, // 200MB
			},
			wantError: true,
			errorType: ErrFileTooLarge,
		},
		{
			name: "blocked mime type",
			input: &UploadInput{
				Body:        bytes.NewReader([]byte("test")),
				FileName:    "test.exe",
				Size:        4,
				ContentType: "application/x-msdownload",
			},
			wantError: false, // Not in blocked list by default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock client for validation
			client := &S3Client{config: cfg}
			err := client.validateUpload(tt.input)

			if tt.wantError {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// =============================================================================
// Key Generation Tests
// =============================================================================

func TestGenerateKey(t *testing.T) {
	cfg := DefaultConfig()
	client := &S3Client{config: cfg}

	input := &UploadInput{
		FileName:     "test-document.pdf",
		DocumentType: DocumentTypeInvoice,
		EntityID:     "customer-123",
	}

	key := client.generateKey(input)

	// Should contain document type
	if !strings.HasPrefix(key, "invoice/") {
		t.Errorf("key should start with document type, got %s", key)
	}

	// Should contain year and month
	now := time.Now()
	if !strings.Contains(key, now.Format("2006")) {
		t.Error("key should contain year")
	}

	// Should end with extension
	if !strings.HasSuffix(key, ".pdf") {
		t.Error("key should end with file extension")
	}
}

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple.pdf", "simple"},
		{"file with spaces.pdf", "file_with_spaces"},
		{"file/with/slashes.pdf", "file_with_slashes"},
		{"file:with:colons.pdf", "file_with_colons"},
		{"file<with>special*chars?.pdf", "file_with_special_chars_"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// Remove extension for test
			name := strings.TrimSuffix(tt.input, ".pdf")
			got := sanitizeFileName(name)
			if got != tt.want {
				t.Errorf("sanitizeFileName(%q) = %q, want %q", name, got, tt.want)
			}
		})
	}
}

// =============================================================================
// Metadata Tests
// =============================================================================

func TestDocumentMetadata(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(365 * 24 * time.Hour)

	meta := &DocumentMetadata{
		ID:           "test-123",
		Key:          "invoice/2024/01/test.pdf",
		FileName:     "test.pdf",
		Size:         1024,
		ContentType:  "application/pdf",
		DocumentType: DocumentTypeInvoice,
		EntityID:     "customer-456",
		EntityType:   "customer",
		CreatedAt:    now,
		UpdatedAt:    now,
		ExpiresAt:    &expiresAt,
		Version:      1,
		IsLatest:     true,
		Status:       "active",
		Encrypted:    true,
		Tags: map[string]string{
			"category": "billing",
		},
	}

	if meta.ID == "" {
		t.Error("ID should not be empty")
	}

	if meta.DocumentType != DocumentTypeInvoice {
		t.Error("document type should be invoice")
	}

	if meta.ExpiresAt == nil {
		t.Error("expires at should not be nil")
	}

	if len(meta.Tags) != 1 {
		t.Error("should have 1 tag")
	}
}

func TestMetadataFilter(t *testing.T) {
	now := time.Now()
	before := now.Add(-24 * time.Hour)

	filter := &MetadataFilter{
		DocumentType:  DocumentTypeInvoice,
		EntityID:      "customer-123",
		Status:        "active",
		CreatedAfter:  &before,
		CreatedBefore: &now,
		MinSize:       100,
		MaxSize:       10000,
		Limit:         50,
		Offset:        10,
		SortBy:        "created_at",
		SortOrder:     "desc",
	}

	if filter.DocumentType != DocumentTypeInvoice {
		t.Error("document type should be invoice")
	}

	if filter.Limit != 50 {
		t.Errorf("expected limit 50, got %d", filter.Limit)
	}

	if filter.SortOrder != "desc" {
		t.Error("sort order should be desc")
	}
}

func TestMetadataMatchesFilter(t *testing.T) {
	store := &S3MetadataStore{}

	now := time.Now()
	meta := &DocumentMetadata{
		ID:           "test-1",
		DocumentType: DocumentTypeInvoice,
		EntityID:     "customer-123",
		Status:       "active",
		CreatedAt:    now,
		Size:         500,
	}

	tests := []struct {
		name   string
		filter *MetadataFilter
		want   bool
	}{
		{
			name:   "empty filter matches all",
			filter: &MetadataFilter{},
			want:   true,
		},
		{
			name: "matching document type",
			filter: &MetadataFilter{
				DocumentType: DocumentTypeInvoice,
			},
			want: true,
		},
		{
			name: "non-matching document type",
			filter: &MetadataFilter{
				DocumentType: DocumentTypeQuote,
			},
			want: false,
		},
		{
			name: "matching entity ID",
			filter: &MetadataFilter{
				EntityID: "customer-123",
			},
			want: true,
		},
		{
			name: "non-matching entity ID",
			filter: &MetadataFilter{
				EntityID: "customer-456",
			},
			want: false,
		},
		{
			name: "deleted status excluded by default",
			filter: &MetadataFilter{
				Status: "active",
			},
			want: true,
		},
		{
			name: "size in range",
			filter: &MetadataFilter{
				MinSize: 100,
				MaxSize: 1000,
			},
			want: true,
		},
		{
			name: "size out of range",
			filter: &MetadataFilter{
				MinSize: 1000,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := store.matchesFilter(meta, tt.filter)
			if got != tt.want {
				t.Errorf("matchesFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// Backup Tests
// =============================================================================

func TestBackupInfo(t *testing.T) {
	now := time.Now()
	completed := now.Add(5 * time.Minute)

	info := &BackupInfo{
		ID:             "20240115-020000",
		Type:           BackupTypeFull,
		Status:         BackupStatusCompleted,
		StartedAt:      now,
		CompletedAt:    &completed,
		ObjectCount:    100,
		TotalSize:      1024 * 1024 * 100, // 100MB
		CompressedSize: 1024 * 1024 * 50,  // 50MB
		SourceBucket:   "primary",
		DestBucket:     "backup",
	}

	if info.Type != BackupTypeFull {
		t.Error("backup type should be full")
	}

	if info.Status != BackupStatusCompleted {
		t.Error("backup status should be completed")
	}

	// Check compression ratio
	ratio := float64(info.CompressedSize) / float64(info.TotalSize)
	if ratio > 1.0 {
		t.Error("compressed size should be smaller than original")
	}
}

func TestBackupManifest(t *testing.T) {
	manifest := &BackupManifest{
		Version:   "1.0",
		CreatedAt: time.Now(),
		Objects: []BackupObjectInfo{
			{
				Key:          "invoice/2024/01/test1.pdf",
				Size:         1024,
				ETag:         "abc123",
				LastModified: time.Now(),
				StorageClass: string(StorageClassStandard),
				BackupKey:    "backups/20240115-020000/invoice/2024/01/test1.pdf",
			},
			{
				Key:          "invoice/2024/01/test2.pdf",
				Size:         2048,
				ETag:         "def456",
				LastModified: time.Now(),
				StorageClass: string(StorageClassStandard),
				BackupKey:    "backups/20240115-020000/invoice/2024/01/test2.pdf",
			},
		},
	}

	if manifest.Version != "1.0" {
		t.Error("manifest version should be 1.0")
	}

	if len(manifest.Objects) != 2 {
		t.Errorf("expected 2 objects, got %d", len(manifest.Objects))
	}

	totalSize := int64(0)
	for _, obj := range manifest.Objects {
		totalSize += obj.Size
	}

	if totalSize != 3072 {
		t.Errorf("expected total size 3072, got %d", totalSize)
	}
}

func TestBackupTypes(t *testing.T) {
	types := []BackupType{
		BackupTypeFull,
		BackupTypeIncremental,
		BackupTypeDifferential,
	}

	for _, bt := range types {
		if string(bt) == "" {
			t.Error("backup type should not be empty")
		}
	}
}

func TestBackupStatuses(t *testing.T) {
	statuses := []BackupStatus{
		BackupStatusPending,
		BackupStatusRunning,
		BackupStatusCompleted,
		BackupStatusFailed,
		BackupStatusCancelled,
	}

	for _, status := range statuses {
		if string(status) == "" {
			t.Error("backup status should not be empty")
		}
	}
}

// =============================================================================
// Retention Tests
// =============================================================================

func TestRetentionConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Check default retention
	if cfg.Retention.DefaultRetention != 7*365*24*time.Hour {
		t.Error("default retention should be 7 years")
	}

	// Check document-specific retention
	invoiceRetention := cfg.Retention.DocumentRetention["invoice"]
	if invoiceRetention != 7*365*24*time.Hour {
		t.Error("invoice retention should be 7 years")
	}

	quoteRetention := cfg.Retention.DocumentRetention["quote"]
	if quoteRetention != 1*365*24*time.Hour {
		t.Error("quote retention should be 1 year")
	}

	tempRetention := cfg.Retention.DocumentRetention["temporary"]
	if tempRetention != 30*24*time.Hour {
		t.Error("temporary retention should be 30 days")
	}
}

func TestRetentionPolicyLookup(t *testing.T) {
	cfg := DefaultConfig()
	service := &RetentionService{config: cfg}

	tests := []struct {
		docType  DocumentType
		expected time.Duration
	}{
		{DocumentTypeInvoice, 7 * 365 * 24 * time.Hour},
		{DocumentTypeQuote, 1 * 365 * 24 * time.Hour},
		{DocumentTypeServiceReport, 5 * 365 * 24 * time.Hour},
		{DocumentTypeContract, 10 * 365 * 24 * time.Hour},
		{DocumentTypeTemporary, 30 * 24 * time.Hour},
		{DocumentTypePhoto, 7 * 365 * 24 * time.Hour}, // Uses default
	}

	for _, tt := range tests {
		t.Run(string(tt.docType), func(t *testing.T) {
			got := service.GetRetentionPolicy(tt.docType)
			if got != tt.expected {
				t.Errorf("GetRetentionPolicy(%s) = %v, want %v",
					tt.docType, got, tt.expected)
			}
		})
	}
}

func TestExemptTypes(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Retention.ExemptTypes = []string{"contract", "legal"}

	service := &RetentionService{config: cfg}

	if !service.isExemptType(DocumentTypeContract) {
		t.Error("contract should be exempt")
	}

	if !service.isExemptType(DocumentType("legal")) {
		t.Error("legal should be exempt")
	}

	if service.isExemptType(DocumentTypeInvoice) {
		t.Error("invoice should not be exempt")
	}
}

// =============================================================================
// Storage Class Tests
// =============================================================================

func TestStorageClasses(t *testing.T) {
	classes := []StorageClass{
		StorageClassStandard,
		StorageClassIntelligentTiering,
		StorageClassStandardIA,
		StorageClassOneZoneIA,
		StorageClassGlacier,
		StorageClassGlacierIR,
		StorageClassDeepArchive,
	}

	for _, class := range classes {
		if string(class) == "" {
			t.Error("storage class should not be empty")
		}
	}
}

// =============================================================================
// Error Tests
// =============================================================================

func TestErrors(t *testing.T) {
	errors := []error{
		ErrInvalidConfig,
		ErrBucketNotFound,
		ErrObjectNotFound,
		ErrUploadFailed,
		ErrDownloadFailed,
		ErrDeleteFailed,
		ErrMetadataFailed,
		ErrBackupFailed,
		ErrRetentionViolation,
		ErrAccessDenied,
		ErrFileTooLarge,
		ErrInvalidFileType,
		ErrChecksumMismatch,
	}

	for _, err := range errors {
		if err == nil {
			t.Error("error should not be nil")
		}
		if err.Error() == "" {
			t.Error("error message should not be empty")
		}
	}
}

// =============================================================================
// Cache Tests
// =============================================================================

func TestMetadataCache(t *testing.T) {
	cache := &metadataCache{
		items:   make(map[string]*cachedMetadata),
		maxSize: 3,
	}

	// Test set and get
	meta1 := &DocumentMetadata{ID: "1", FileName: "test1.pdf"}
	cache.set("1", meta1)

	got := cache.get("1")
	if got == nil {
		t.Error("should get cached item")
	}
	if got.FileName != "test1.pdf" {
		t.Error("cached item should match")
	}

	// Test miss
	got = cache.get("nonexistent")
	if got != nil {
		t.Error("should return nil for missing item")
	}

	// Test delete
	cache.delete("1")
	got = cache.get("1")
	if got != nil {
		t.Error("should return nil after delete")
	}

	// Test capacity
	cache.set("a", &DocumentMetadata{ID: "a"})
	cache.set("b", &DocumentMetadata{ID: "b"})
	cache.set("c", &DocumentMetadata{ID: "c"})
	cache.set("d", &DocumentMetadata{ID: "d"}) // Should trigger eviction

	if len(cache.items) > 3 {
		t.Error("cache should not exceed max size")
	}
}

// =============================================================================
// Integration Test Helpers
// =============================================================================

// MockS3Client provides a mock S3 client for testing
type MockS3Client struct {
	objects map[string][]byte
	mu      sync.RWMutex
}

func NewMockS3Client() *MockS3Client {
	return &MockS3Client{
		objects: make(map[string][]byte),
	}
}

func (m *MockS3Client) Put(key string, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[key] = data
}

func (m *MockS3Client) Get(key string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.objects[key]
	return data, ok
}

func (m *MockS3Client) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.objects, key)
}

func (m *MockS3Client) List(prefix string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var keys []string
	for key := range m.objects {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	return keys
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkGenerateKey(b *testing.B) {
	cfg := DefaultConfig()
	client := &S3Client{config: cfg}

	input := &UploadInput{
		FileName:     "test-document.pdf",
		DocumentType: DocumentTypeInvoice,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.generateKey(input)
	}
}

func BenchmarkSanitizeFileName(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sanitizeFileName("test file with spaces and special <chars>.pdf")
	}
}

func BenchmarkMetadataCache(b *testing.B) {
	cache := &metadataCache{
		items:   make(map[string]*cachedMetadata),
		maxSize: 1000,
	}

	meta := &DocumentMetadata{ID: "test", FileName: "test.pdf"}

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cache.set("test", meta)
		}
	})

	b.Run("Get", func(b *testing.B) {
		cache.set("test", meta)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cache.get("test")
		}
	})
}
