package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// Test Setup
// =============================================================================

func init() {
	gin.SetMode(gin.TestMode)
}

// =============================================================================
// Mock Implementations
// =============================================================================

// mockStorageService implements StorageService for testing
type mockStorageService struct {
	files         map[string][]byte
	metadata      map[string]*DownloadInfo
	presignedURLs map[string]string
	downloadErr   error
	presignedErr  error
}

func newMockStorageService() *mockStorageService {
	return &mockStorageService{
		files:         make(map[string][]byte),
		metadata:      make(map[string]*DownloadInfo),
		presignedURLs: make(map[string]string),
	}
}

func (m *mockStorageService) Download(ctx context.Context, key string) (io.ReadCloser, *DownloadInfo, error) {
	if m.downloadErr != nil {
		return nil, nil, m.downloadErr
	}
	data, ok := m.files[key]
	if !ok {
		return nil, nil, io.EOF
	}
	info := m.metadata[key]
	if info == nil {
		info = &DownloadInfo{
			Size:        int64(len(data)),
			ContentType: "application/octet-stream",
		}
	}
	return io.NopCloser(bytes.NewReader(data)), info, nil
}

func (m *mockStorageService) GetPresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	if m.presignedErr != nil {
		return "", m.presignedErr
	}
	url, ok := m.presignedURLs[key]
	if !ok {
		url = "https://s3.example.com/presigned/" + key
	}
	return url, nil
}

func (m *mockStorageService) GetObjectInfo(ctx context.Context, key string) (*ObjectInfo, error) {
	data, ok := m.files[key]
	if !ok {
		return nil, io.EOF
	}
	return &ObjectInfo{
		Key:         key,
		Size:        int64(len(data)),
		ContentType: "application/octet-stream",
	}, nil
}

func (m *mockStorageService) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := m.files[key]
	return ok, nil
}

func (m *mockStorageService) addFile(key string, content []byte, contentType string) {
	m.files[key] = content
	m.metadata[key] = &DownloadInfo{
		Size:        int64(len(content)),
		ContentType: contentType,
		ETag:        "test-etag",
	}
}

// mockMetadataService implements MetadataService for testing
type mockMetadataService struct {
	documents map[string]*DocumentMetadata
	getErr    error
}

func newMockMetadataService() *mockMetadataService {
	return &mockMetadataService{
		documents: make(map[string]*DocumentMetadata),
	}
}

func (m *mockMetadataService) Get(ctx context.Context, id string) (*DocumentMetadata, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	doc, ok := m.documents[id]
	if !ok {
		return nil, io.EOF
	}
	return doc, nil
}

func (m *mockMetadataService) GetByKey(ctx context.Context, key string) (*DocumentMetadata, error) {
	for _, doc := range m.documents {
		if doc.Key == key {
			return doc, nil
		}
	}
	return nil, io.EOF
}

func (m *mockMetadataService) UpdateAccess(ctx context.Context, id string) error {
	return nil
}

func (m *mockMetadataService) addDocument(doc *DocumentMetadata) {
	m.documents[doc.ID] = doc
}

// mockLogger implements Logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, keysAndValues ...interface{}) {}
func (m *mockLogger) Info(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Warn(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Error(msg string, keysAndValues ...interface{}) {}

// =============================================================================
// Test Helpers
// =============================================================================

func setupTestHandler() (*DownloadHandler, *mockStorageService, *mockMetadataService) {
	storage := newMockStorageService()
	metadata := newMockMetadataService()
	logger := &mockLogger{}

	config := DefaultDownloadConfig()
	config.MaxInlineSize = 1024 * 1024 // 1MB for tests

	handler := NewDownloadHandler(storage, metadata, config, logger)

	return handler, storage, metadata
}

func setupDownloadTestRouter(handler *DownloadHandler) *gin.Engine {
	r := gin.New()
	api := r.Group("/api")
	handler.RegisterRoutes(api)
	return r
}

func createTestRequest(method, path string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("X-User-ID", "user-123")
	req.Header.Set("X-User-Email", "test@example.com")
	req.Header.Set("X-User-Role", "user")
	return req
}

func createAdminRequest(method, path string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("X-User-ID", "admin-1")
	req.Header.Set("X-User-Email", "admin@example.com")
	req.Header.Set("X-User-Role", "admin")
	return req
}

// =============================================================================
// Download Endpoint Tests
// =============================================================================

func TestDownload_Success(t *testing.T) {
	handler, storage, metadata := setupTestHandler()
	router := setupDownloadTestRouter(handler)

	// Setup test data
	content := []byte("test file content")
	storage.addFile("documents/test.pdf", content, "application/pdf")
	metadata.addDocument(&DocumentMetadata{
		ID:          "doc-1",
		Key:         "documents/test.pdf",
		FileName:    "test.pdf",
		Size:        int64(len(content)),
		ContentType: "application/pdf",
		Owner:       "user-123",
		AccessLevel: "private",
	})

	req := createTestRequest("GET", "/api/documents/download/doc-1", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	if rr.Header().Get("Content-Type") != "application/pdf" {
		t.Errorf("Expected Content-Type application/pdf, got %s", rr.Header().Get("Content-Type"))
	}

	if !strings.Contains(rr.Header().Get("Content-Disposition"), "test.pdf") {
		t.Errorf("Expected Content-Disposition to contain filename, got %s", rr.Header().Get("Content-Disposition"))
	}

	if rr.Body.String() != string(content) {
		t.Errorf("Expected body %s, got %s", string(content), rr.Body.String())
	}
}

func TestDownload_NotFound(t *testing.T) {
	handler, _, _ := setupTestHandler()
	router := setupDownloadTestRouter(handler)

	req := createTestRequest("GET", "/api/documents/download/nonexistent", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}
}

func TestDownload_AccessDenied(t *testing.T) {
	handler, storage, metadata := setupTestHandler()
	router := setupDownloadTestRouter(handler)

	// Setup test data with different owner
	content := []byte("test file content")
	storage.addFile("documents/test.pdf", content, "application/pdf")
	metadata.addDocument(&DocumentMetadata{
		ID:          "doc-1",
		Key:         "documents/test.pdf",
		FileName:    "test.pdf",
		Size:        int64(len(content)),
		ContentType: "application/pdf",
		Owner:       "other-user",
		AccessLevel: "private",
	})

	req := createTestRequest("GET", "/api/documents/download/doc-1", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rr.Code)
	}
}

func TestDownload_AdminAccess(t *testing.T) {
	handler, storage, metadata := setupTestHandler()
	router := setupDownloadTestRouter(handler)

	// Setup test data with different owner
	content := []byte("test file content")
	storage.addFile("documents/test.pdf", content, "application/pdf")
	metadata.addDocument(&DocumentMetadata{
		ID:          "doc-1",
		Key:         "documents/test.pdf",
		FileName:    "test.pdf",
		Size:        int64(len(content)),
		ContentType: "application/pdf",
		Owner:       "other-user",
		AccessLevel: "private",
	})

	// Admin should be able to access any document
	req := createAdminRequest("GET", "/api/documents/download/doc-1", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestDownload_PublicAccess(t *testing.T) {
	handler, storage, metadata := setupTestHandler()
	router := setupDownloadTestRouter(handler)

	content := []byte("public content")
	storage.addFile("documents/public.pdf", content, "application/pdf")
	metadata.addDocument(&DocumentMetadata{
		ID:          "doc-1",
		Key:         "documents/public.pdf",
		FileName:    "public.pdf",
		Size:        int64(len(content)),
		ContentType: "application/pdf",
		Owner:       "other-user",
		AccessLevel: "public",
	})

	// Anonymous user can access public document
	req := httptest.NewRequest("GET", "/api/documents/download/doc-1", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

// =============================================================================
// Download URL Endpoint Tests
// =============================================================================

func TestGetDownloadURL_Success(t *testing.T) {
	handler, storage, metadata := setupTestHandler()
	router := setupDownloadTestRouter(handler)

	content := []byte("test content")
	storage.addFile("documents/test.pdf", content, "application/pdf")
	metadata.addDocument(&DocumentMetadata{
		ID:          "doc-1",
		Key:         "documents/test.pdf",
		FileName:    "test.pdf",
		Size:        int64(len(content)),
		ContentType: "application/pdf",
		Owner:       "user-123",
		AccessLevel: "private",
	})

	req := createTestRequest("GET", "/api/documents/download-url/doc-1", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var resp DownloadURLResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.URL == "" {
		t.Error("Expected non-empty URL")
	}
	if resp.FileName != "test.pdf" {
		t.Errorf("Expected filename test.pdf, got %s", resp.FileName)
	}
}

func TestGetDownloadURL_CustomExpiration(t *testing.T) {
	handler, storage, metadata := setupTestHandler()
	router := setupDownloadTestRouter(handler)

	content := []byte("test content")
	storage.addFile("documents/test.pdf", content, "application/pdf")
	metadata.addDocument(&DocumentMetadata{
		ID:          "doc-1",
		Key:         "documents/test.pdf",
		FileName:    "test.pdf",
		Size:        int64(len(content)),
		ContentType: "application/pdf",
		Owner:       "user-123",
		AccessLevel: "private",
	})

	req := createTestRequest("GET", "/api/documents/download-url/doc-1?expires_in=2h", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var resp DownloadURLResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check expiration is approximately 2 hours from now
	expectedExp := time.Now().Add(2 * time.Hour)
	if resp.ExpiresAt.Before(expectedExp.Add(-1*time.Minute)) || resp.ExpiresAt.After(expectedExp.Add(1*time.Minute)) {
		t.Errorf("Expected expiration around 2 hours from now, got %v", resp.ExpiresAt)
	}
}

// =============================================================================
// Bulk Download Tests
// =============================================================================

func TestBulkDownload_Success(t *testing.T) {
	handler, storage, metadata := setupTestHandler()
	router := setupDownloadTestRouter(handler)

	// Add multiple documents
	for i := 1; i <= 3; i++ {
		id := string(rune('0' + i))
		content := []byte("content for file " + id)
		key := "documents/file" + id + ".pdf"
		storage.addFile(key, content, "application/pdf")
		metadata.addDocument(&DocumentMetadata{
			ID:          "doc-" + id,
			Key:         key,
			FileName:    "file" + id + ".pdf",
			Size:        int64(len(content)),
			ContentType: "application/pdf",
			Owner:       "user-123",
			AccessLevel: "private",
		})
	}

	body := `{"document_ids": ["doc-1", "doc-2", "doc-3"]}`
	req := createTestRequest("POST", "/api/documents/bulk-download", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	if rr.Header().Get("Content-Type") != "application/zip" {
		t.Errorf("Expected Content-Type application/zip, got %s", rr.Header().Get("Content-Type"))
	}

	// Verify zip contains files
	zipReader, err := zip.NewReader(bytes.NewReader(rr.Body.Bytes()), int64(rr.Body.Len()))
	if err != nil {
		t.Fatalf("Failed to read zip: %v", err)
	}

	if len(zipReader.File) != 3 {
		t.Errorf("Expected 3 files in zip, got %d", len(zipReader.File))
	}
}

func TestBulkDownload_TooManyFiles(t *testing.T) {
	handler, _, _ := setupTestHandler()
	handler.config.MaxBulkFiles = 2
	router := setupDownloadTestRouter(handler)

	body := `{"document_ids": ["doc-1", "doc-2", "doc-3"]}`
	req := createTestRequest("POST", "/api/documents/bulk-download", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestBulkDownload_EmptyDocumentIDs(t *testing.T) {
	handler, _, _ := setupTestHandler()
	router := setupDownloadTestRouter(handler)

	body := `{"document_ids": []}`
	req := createTestRequest("POST", "/api/documents/bulk-download", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestBulkDownload_CustomFilename(t *testing.T) {
	handler, storage, metadata := setupTestHandler()
	router := setupDownloadTestRouter(handler)

	content := []byte("test content")
	storage.addFile("documents/test.pdf", content, "application/pdf")
	metadata.addDocument(&DocumentMetadata{
		ID:          "doc-1",
		Key:         "documents/test.pdf",
		FileName:    "test.pdf",
		Size:        int64(len(content)),
		ContentType: "application/pdf",
		Owner:       "user-123",
		AccessLevel: "private",
	})

	body := `{"document_ids": ["doc-1"], "file_name": "my_archive"}`
	req := createTestRequest("POST", "/api/documents/bulk-download", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	disposition := rr.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, "my_archive.zip") {
		t.Errorf("Expected Content-Disposition to contain my_archive.zip, got %s", disposition)
	}
}

// =============================================================================
// History Endpoint Tests
// =============================================================================

func TestGetHistory_Success(t *testing.T) {
	handler, _, _ := setupTestHandler()
	router := setupDownloadTestRouter(handler)

	// Record some downloads
	for i := 0; i < 5; i++ {
		handler.tracker.Record(&DownloadRecord{
			ID:         "record-" + string(rune('0'+i)),
			DocumentID: "doc-" + string(rune('0'+i)),
			FileName:   "file.pdf",
			UserID:     "user-123",
			Method:     "stream",
			Timestamp:  time.Now().Add(-time.Duration(i) * time.Hour),
		})
	}

	req := createTestRequest("GET", "/api/documents/history?limit=3", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var resp HistoryResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(resp.Items))
	}

	if resp.TotalCount != 5 {
		t.Errorf("Expected total count 5, got %d", resp.TotalCount)
	}
}

func TestGetDocumentHistory_Success(t *testing.T) {
	handler, _, _ := setupTestHandler()
	router := setupDownloadTestRouter(handler)

	// Record downloads for a specific document
	for i := 0; i < 3; i++ {
		handler.tracker.Record(&DownloadRecord{
			ID:         "record-" + string(rune('0'+i)),
			DocumentID: "doc-1",
			FileName:   "file.pdf",
			UserID:     "user-" + string(rune('0'+i)),
			Method:     "stream",
			Timestamp:  time.Now().Add(-time.Duration(i) * time.Hour),
		})
	}

	req := createTestRequest("GET", "/api/documents/history/doc-1", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var resp HistoryResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(resp.Items))
	}
}

// =============================================================================
// Stats Endpoint Tests
// =============================================================================

func TestGetStats_Success(t *testing.T) {
	handler, _, _ := setupTestHandler()
	router := setupDownloadTestRouter(handler)

	// Record some downloads
	for i := 0; i < 10; i++ {
		handler.tracker.Record(&DownloadRecord{
			ID:          "record-" + string(rune('0'+i)),
			DocumentID:  "doc-" + string(rune('0'+i%3)),
			FileName:    "file.pdf",
			FileSize:    1024,
			UserID:      "user-123",
			Method:      "stream",
			ContentType: "application/pdf",
			Timestamp:   time.Now().Add(-time.Duration(i) * time.Hour),
		})
	}

	req := createTestRequest("GET", "/api/documents/stats?period=7d", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var stats DownloadStats
	if err := json.NewDecoder(rr.Body).Decode(&stats); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if stats.TotalDownloads != 10 {
		t.Errorf("Expected 10 total downloads, got %d", stats.TotalDownloads)
	}

	if stats.TotalBytes != 10240 {
		t.Errorf("Expected 10240 total bytes, got %d", stats.TotalBytes)
	}
}

// =============================================================================
// Rate Limiting Tests
// =============================================================================

func TestRateLimiting(t *testing.T) {
	handler, storage, metadata := setupTestHandler()

	// Configure strict rate limit
	handler.rateLimiter = NewDownloadRateLimiter(&RateLimitConfig{
		UserRequestsPerMinute: 2,
		UserBurstSize:         2,
		IPRequestsPerMinute:   2,
		IPBurstSize:           2,
		CleanupInterval:       0, // Disable cleanup
	})

	router := setupDownloadTestRouter(handler)

	content := []byte("test content")
	storage.addFile("documents/test.pdf", content, "application/pdf")
	metadata.addDocument(&DocumentMetadata{
		ID:          "doc-1",
		Key:         "documents/test.pdf",
		FileName:    "test.pdf",
		Size:        int64(len(content)),
		ContentType: "application/pdf",
		Owner:       "user-123",
		AccessLevel: "private",
	})

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		req := createTestRequest("GET", "/api/documents/download/doc-1", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i, rr.Code)
		}
	}

	// Third request should be rate limited
	req := createTestRequest("GET", "/api/documents/download/doc-1", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", rr.Code)
	}
}

func TestRateLimiting_Whitelist(t *testing.T) {
	handler, storage, metadata := setupTestHandler()

	handler.rateLimiter = NewDownloadRateLimiter(&RateLimitConfig{
		UserRequestsPerMinute: 1,
		UserBurstSize:         1,
		WhitelistedUsers:      []string{"user-123"},
		CleanupInterval:       0,
	})

	router := setupDownloadTestRouter(handler)

	content := []byte("test content")
	storage.addFile("documents/test.pdf", content, "application/pdf")
	metadata.addDocument(&DocumentMetadata{
		ID:          "doc-1",
		Key:         "documents/test.pdf",
		FileName:    "test.pdf",
		Size:        int64(len(content)),
		ContentType: "application/pdf",
		Owner:       "user-123",
		AccessLevel: "private",
	})

	// Whitelisted user can make multiple requests
	for i := 0; i < 5; i++ {
		req := createTestRequest("GET", "/api/documents/download/doc-1", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i, rr.Code)
		}
	}
}

// =============================================================================
// Tracking Tests
// =============================================================================

func TestDownloadTracking(t *testing.T) {
	handler, storage, metadata := setupTestHandler()
	router := setupDownloadTestRouter(handler)

	content := []byte("test content")
	storage.addFile("documents/test.pdf", content, "application/pdf")
	metadata.addDocument(&DocumentMetadata{
		ID:          "doc-1",
		Key:         "documents/test.pdf",
		FileName:    "test.pdf",
		Size:        int64(len(content)),
		ContentType: "application/pdf",
		Owner:       "user-123",
		AccessLevel: "private",
	})

	req := createTestRequest("GET", "/api/documents/download/doc-1", nil)
	req.Header.Set("User-Agent", "Test-Agent/1.0")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rr.Code)
	}

	// Check that download was tracked
	history := handler.tracker.GetUserHistory("user-123", nil)
	if len(history) != 1 {
		t.Fatalf("Expected 1 history record, got %d", len(history))
	}

	record := history[0]
	if record.DocumentID != "doc-1" {
		t.Errorf("Expected document ID doc-1, got %s", record.DocumentID)
	}
	if record.UserID != "user-123" {
		t.Errorf("Expected user ID user-123, got %s", record.UserID)
	}
	if record.Method != "stream" {
		t.Errorf("Expected method stream, got %s", record.Method)
	}
	if record.UserAgent != "Test-Agent/1.0" {
		t.Errorf("Expected user agent Test-Agent/1.0, got %s", record.UserAgent)
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestCanAccess(t *testing.T) {
	handler, _, _ := setupTestHandler()

	tests := []struct {
		name     string
		user     *UserInfo
		meta     *DocumentMetadata
		expected bool
	}{
		{
			name:     "Admin access",
			user:     &UserInfo{ID: "admin-1", Role: "admin"},
			meta:     &DocumentMetadata{Owner: "other", AccessLevel: "private"},
			expected: true,
		},
		{
			name:     "Owner access",
			user:     &UserInfo{ID: "user-1", Role: "user"},
			meta:     &DocumentMetadata{Owner: "user-1", AccessLevel: "private"},
			expected: true,
		},
		{
			name:     "Shared with user",
			user:     &UserInfo{ID: "user-1", Role: "user"},
			meta:     &DocumentMetadata{Owner: "other", AccessLevel: "private", SharedWith: []string{"user-1"}},
			expected: true,
		},
		{
			name:     "Public access",
			user:     &UserInfo{ID: "anonymous"},
			meta:     &DocumentMetadata{Owner: "other", AccessLevel: "public"},
			expected: true,
		},
		{
			name:     "Authenticated access",
			user:     &UserInfo{ID: "user-1", Role: "user"},
			meta:     &DocumentMetadata{Owner: "other", AccessLevel: "authenticated"},
			expected: true,
		},
		{
			name:     "Anonymous denied authenticated",
			user:     &UserInfo{ID: "anonymous"},
			meta:     &DocumentMetadata{Owner: "other", AccessLevel: "authenticated"},
			expected: false,
		},
		{
			name:     "Private access denied",
			user:     &UserInfo{ID: "user-1", Role: "user"},
			meta:     &DocumentMetadata{Owner: "other", AccessLevel: "private"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.canAccess(tt.user, tt.meta)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsInlineAllowed(t *testing.T) {
	handler, _, _ := setupTestHandler()

	tests := []struct {
		contentType string
		expected    bool
	}{
		{"application/pdf", true},
		{"image/jpeg", true},
		{"image/png", true},
		{"text/plain", true},
		{"application/zip", false},
		{"application/octet-stream", false},
		{"application/x-msdownload", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := handler.isInlineAllowed(tt.contentType)
			if result != tt.expected {
				t.Errorf("Expected %v for %s, got %v", tt.expected, tt.contentType, result)
			}
		})
	}
}

func TestGetUniqueFileName(t *testing.T) {
	handler, _, _ := setupTestHandler()

	used := make(map[string]int)

	// First use should return original name
	name1 := handler.getUniqueFileName("test.pdf", used)
	if name1 != "test.pdf" {
		t.Errorf("Expected test.pdf, got %s", name1)
	}

	// Second use should return numbered name
	name2 := handler.getUniqueFileName("test.pdf", used)
	if name2 != "test_1.pdf" {
		t.Errorf("Expected test_1.pdf, got %s", name2)
	}

	// Third use should return next numbered name
	name3 := handler.getUniqueFileName("test.pdf", used)
	if name3 != "test_2.pdf" {
		t.Errorf("Expected test_2.pdf, got %s", name3)
	}
}

// =============================================================================
// Tracker Unit Tests
// =============================================================================

func TestDownloadTracker_Record(t *testing.T) {
	tracker := NewDownloadTracker()

	record := &DownloadRecord{
		ID:         "rec-1",
		DocumentID: "doc-1",
		UserID:     "user-1",
		Method:     "stream",
	}

	tracker.Record(record)

	if tracker.RecordCount() != 1 {
		t.Errorf("Expected 1 record, got %d", tracker.RecordCount())
	}

	if tracker.UserCount() != 1 {
		t.Errorf("Expected 1 user, got %d", tracker.UserCount())
	}

	if tracker.DocumentCount() != 1 {
		t.Errorf("Expected 1 document, got %d", tracker.DocumentCount())
	}
}

func TestDownloadTracker_Clear(t *testing.T) {
	tracker := NewDownloadTracker()

	for i := 0; i < 5; i++ {
		tracker.Record(&DownloadRecord{
			ID:         "rec-" + string(rune('0'+i)),
			DocumentID: "doc-" + string(rune('0'+i)),
			UserID:     "user-" + string(rune('0'+i)),
		})
	}

	tracker.Clear()

	if tracker.RecordCount() != 0 {
		t.Errorf("Expected 0 records after clear, got %d", tracker.RecordCount())
	}
}

// =============================================================================
// Rate Limiter Unit Tests
// =============================================================================

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewDownloadRateLimiter(&RateLimitConfig{
		UserRequestsPerMinute: 2,
		UserBurstSize:         2,
		IPRequestsPerMinute:   2,
		IPBurstSize:           2,
		CleanupInterval:       0,
	})

	// Should allow first 2 requests
	if !rl.Allow("user-1", "1.2.3.4") {
		t.Error("Expected first request to be allowed")
	}
	if !rl.Allow("user-1", "1.2.3.4") {
		t.Error("Expected second request to be allowed")
	}

	// Third should be denied
	if rl.Allow("user-1", "1.2.3.4") {
		t.Error("Expected third request to be denied")
	}
}

func TestRateLimiter_Whitelist(t *testing.T) {
	rl := NewDownloadRateLimiter(&RateLimitConfig{
		UserRequestsPerMinute: 1,
		UserBurstSize:         1,
		WhitelistedUsers:      []string{"admin-1"},
		CleanupInterval:       0,
	})

	// Whitelisted user should always be allowed
	for i := 0; i < 10; i++ {
		if !rl.Allow("admin-1", "1.2.3.4") {
			t.Errorf("Request %d: Expected whitelisted user to be allowed", i)
		}
	}
}

func TestRateLimiter_Stats(t *testing.T) {
	rl := NewDownloadRateLimiter(&RateLimitConfig{
		UserRequestsPerMinute: 60,
		UserBurstSize:         10,
		IPRequestsPerMinute:   30,
		IPBurstSize:           5,
		CleanupInterval:       0,
	})

	// Make some requests
	rl.Allow("user-1", "1.2.3.4")
	rl.Allow("user-2", "5.6.7.8")

	stats := rl.Stats()

	if stats.ActiveUserBuckets != 2 {
		t.Errorf("Expected 2 active user buckets, got %d", stats.ActiveUserBuckets)
	}
}
