package handlers

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// =============================================================================
// Download Handler
// =============================================================================

// DownloadHandler handles document download operations
type DownloadHandler struct {
	storage     StorageService
	metadata    MetadataService
	tracker     *DownloadTracker
	rateLimiter *DownloadRateLimiter
	config      *DownloadConfig
	logger      Logger
}

// DownloadConfig contains download configuration
type DownloadConfig struct {
	// Maximum file size for inline download (larger files redirect to presigned URL)
	MaxInlineSize int64 `json:"max_inline_size"`

	// Presigned URL expiration
	URLExpiration time.Duration `json:"url_expiration"`

	// Maximum files in bulk download
	MaxBulkFiles int `json:"max_bulk_files"`

	// Maximum total size for bulk download
	MaxBulkSize int64 `json:"max_bulk_size"`

	// Enable download tracking
	EnableTracking bool `json:"enable_tracking"`

	// Enable rate limiting
	EnableRateLimiting bool `json:"enable_rate_limiting"`

	// Enable history logging
	EnableHistoryLogging bool `json:"enable_history_logging"`

	// Allowed content types for inline display
	InlineContentTypes []string `json:"inline_content_types"`

	// Force download for these types
	ForceDownloadTypes []string `json:"force_download_types"`
}

// DefaultDownloadConfig returns the default download configuration
func DefaultDownloadConfig() *DownloadConfig {
	return &DownloadConfig{
		MaxInlineSize:        10 * 1024 * 1024, // 10MB
		URLExpiration:        1 * time.Hour,
		MaxBulkFiles:         50,
		MaxBulkSize:          500 * 1024 * 1024, // 500MB
		EnableTracking:       true,
		EnableRateLimiting:   true,
		EnableHistoryLogging: true,
		InlineContentTypes: []string{
			"application/pdf",
			"image/jpeg",
			"image/png",
			"image/gif",
			"image/webp",
			"text/plain",
			"text/html",
		},
		ForceDownloadTypes: []string{
			"application/zip",
			"application/x-rar-compressed",
			"application/x-7z-compressed",
			"application/octet-stream",
		},
	}
}

// StorageService defines the interface for storage operations
type StorageService interface {
	Download(ctx context.Context, key string) (io.ReadCloser, *DownloadInfo, error)
	GetPresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error)
	GetObjectInfo(ctx context.Context, key string) (*ObjectInfo, error)
	Exists(ctx context.Context, key string) (bool, error)
}

// MetadataService defines the interface for metadata operations
type MetadataService interface {
	Get(ctx context.Context, id string) (*DocumentMetadata, error)
	GetByKey(ctx context.Context, key string) (*DocumentMetadata, error)
	UpdateAccess(ctx context.Context, id string) error
}

// DownloadInfo contains information about a downloaded file
type DownloadInfo struct {
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
	Metadata     map[string]string
}

// ObjectInfo contains object metadata
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
}

// DocumentMetadata contains document metadata
type DocumentMetadata struct {
	ID           string
	Key          string
	FileName     string
	Size         int64
	ContentType  string
	DocumentType string
	EntityID     string
	EntityType   string
	CreatedAt    time.Time
	AccessLevel  string
	Owner        string
	SharedWith   []string
}

// Logger interface for logging
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// NewDownloadHandler creates a new download handler
func NewDownloadHandler(
	storage StorageService,
	metadata MetadataService,
	config *DownloadConfig,
	logger Logger,
) *DownloadHandler {
	if config == nil {
		config = DefaultDownloadConfig()
	}

	handler := &DownloadHandler{
		storage:  storage,
		metadata: metadata,
		config:   config,
		logger:   logger,
	}

	if config.EnableTracking {
		handler.tracker = NewDownloadTracker()
	}

	if config.EnableRateLimiting {
		handler.rateLimiter = NewDownloadRateLimiter(DefaultRateLimitConfig())
	}

	return handler
}

// RegisterRoutes registers download routes with a Gin router group
func (h *DownloadHandler) RegisterRoutes(r *gin.RouterGroup) {
	documents := r.Group("/documents")
	{
		// Single document download
		documents.GET("/download/:id", h.Download)

		// Get presigned download URL
		documents.GET("/download-url/:id", h.GetDownloadURL)

		// Bulk download
		documents.POST("/bulk-download", h.BulkDownload)

		// Download history
		documents.GET("/history", h.GetHistory)
		documents.GET("/history/:id", h.GetDocumentHistory)

		// Download stats
		documents.GET("/stats", h.GetStats)
	}
}

// =============================================================================
// Download Endpoint
// =============================================================================

// Download handles single document downloads
func (h *DownloadHandler) Download(c *gin.Context) {
	ctx := c.Request.Context()
	documentID := c.Param("id")

	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request", "message": "document ID is required"})
		return
	}

	// Get user info from context
	userInfo := h.getUserInfo(c)

	// Check rate limit
	if h.rateLimiter != nil {
		if !h.rateLimiter.Allow(userInfo.ID, userInfo.IP) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too Many Requests", "message": "rate limit exceeded"})
			return
		}
	}

	// Get document metadata
	meta, err := h.metadata.Get(ctx, documentID)
	if err != nil {
		h.logger.Error("Failed to get document metadata", "id", documentID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found", "message": "document not found"})
		return
	}

	// Check access permissions
	if !h.canAccess(userInfo, meta) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden", "message": "access denied"})
		return
	}

	// Determine download method based on file size
	if meta.Size > h.config.MaxInlineSize {
		// Large file - redirect to presigned URL
		h.redirectToPresignedURL(c, ctx, meta)
		return
	}

	// Small file - stream directly
	h.streamDownload(c, ctx, meta)
}

// streamDownload streams the file directly to the response
func (h *DownloadHandler) streamDownload(c *gin.Context, ctx context.Context, meta *DocumentMetadata) {
	// Download from storage
	reader, info, err := h.storage.Download(ctx, meta.Key)
	if err != nil {
		h.logger.Error("Failed to download file", "key", meta.Key, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error", "message": "download failed"})
		return
	}
	defer reader.Close()

	// Set headers
	h.setDownloadHeaders(c, meta, info, c.Query("inline") == "true")

	// Track download
	h.trackDownload(c, meta, "stream")

	// Update access time
	h.metadata.UpdateAccess(ctx, meta.ID)

	// Stream content
	written, err := io.Copy(c.Writer, reader)
	if err != nil {
		h.logger.Error("Failed to stream file", "key", meta.Key, "error", err)
		return
	}

	h.logger.Info("File downloaded", "id", meta.ID, "size", written)
}

// redirectToPresignedURL redirects to a presigned S3 URL
func (h *DownloadHandler) redirectToPresignedURL(c *gin.Context, ctx context.Context, meta *DocumentMetadata) {
	url, err := h.storage.GetPresignedURL(ctx, meta.Key, h.config.URLExpiration)
	if err != nil {
		h.logger.Error("Failed to generate presigned URL", "key", meta.Key, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error", "message": "failed to generate download URL"})
		return
	}

	// Track download
	h.trackDownload(c, meta, "redirect")

	// Update access time
	h.metadata.UpdateAccess(ctx, meta.ID)

	// Redirect to presigned URL
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// =============================================================================
// Download URL Endpoint
// =============================================================================

// GetDownloadURL returns a presigned download URL
func (h *DownloadHandler) GetDownloadURL(c *gin.Context) {
	ctx := c.Request.Context()
	documentID := c.Param("id")

	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request", "message": "document ID is required"})
		return
	}

	// Get user info
	userInfo := h.getUserInfo(c)

	// Check rate limit
	if h.rateLimiter != nil {
		if !h.rateLimiter.Allow(userInfo.ID, userInfo.IP) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too Many Requests", "message": "rate limit exceeded"})
			return
		}
	}

	// Get document metadata
	meta, err := h.metadata.Get(ctx, documentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found", "message": "document not found"})
		return
	}

	// Check access
	if !h.canAccess(userInfo, meta) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden", "message": "access denied"})
		return
	}

	// Parse custom expiration
	expiration := h.config.URLExpiration
	if expStr := c.Query("expires_in"); expStr != "" {
		if exp, err := time.ParseDuration(expStr); err == nil {
			if exp > 0 && exp <= 24*time.Hour {
				expiration = exp
			}
		}
	}

	// Generate presigned URL
	url, err := h.storage.GetPresignedURL(ctx, meta.Key, expiration)
	if err != nil {
		h.logger.Error("Failed to generate presigned URL", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error", "message": "failed to generate URL"})
		return
	}

	// Track URL generation
	h.trackDownload(c, meta, "url_generated")

	response := DownloadURLResponse{
		URL:       url,
		ExpiresAt: time.Now().Add(expiration),
		FileName:  meta.FileName,
		Size:      meta.Size,
		MimeType:  meta.ContentType,
	}

	c.JSON(http.StatusOK, response)
}

// DownloadURLResponse is the response for download URL requests
type DownloadURLResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
	FileName  string    `json:"file_name"`
	Size      int64     `json:"size"`
	MimeType  string    `json:"mime_type"`
}

// =============================================================================
// Bulk Download Endpoint
// =============================================================================

// BulkDownloadRequest is the request for bulk downloads
type BulkDownloadRequest struct {
	DocumentIDs []string `json:"document_ids"`
	FileName    string   `json:"file_name,omitempty"`
}

// BulkDownload handles bulk document downloads as a zip file
func (h *DownloadHandler) BulkDownload(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse request
	var req BulkDownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request", "message": "invalid request body"})
		return
	}

	if len(req.DocumentIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request", "message": "document_ids is required"})
		return
	}

	if len(req.DocumentIDs) > h.config.MaxBulkFiles {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": fmt.Sprintf("maximum %d files allowed per bulk download", h.config.MaxBulkFiles),
		})
		return
	}

	// Get user info
	userInfo := h.getUserInfo(c)

	// Check rate limit (bulk downloads count as multiple requests)
	if h.rateLimiter != nil {
		for range req.DocumentIDs {
			if !h.rateLimiter.Allow(userInfo.ID, userInfo.IP) {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too Many Requests", "message": "rate limit exceeded"})
				return
			}
		}
	}

	// Validate documents and calculate total size
	var documents []*DocumentMetadata
	var totalSize int64

	for _, id := range req.DocumentIDs {
		meta, err := h.metadata.Get(ctx, id)
		if err != nil {
			h.logger.Warn("Document not found for bulk download", "id", id)
			continue
		}

		if !h.canAccess(userInfo, meta) {
			h.logger.Warn("Access denied for bulk download", "id", id, "user", userInfo.ID)
			continue
		}

		totalSize += meta.Size
		if totalSize > h.config.MaxBulkSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Bad Request",
				"message": fmt.Sprintf("total size exceeds maximum of %d bytes", h.config.MaxBulkSize),
			})
			return
		}

		documents = append(documents, meta)
	}

	if len(documents) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found", "message": "no accessible documents found"})
		return
	}

	// Generate zip filename
	zipFileName := req.FileName
	if zipFileName == "" {
		zipFileName = fmt.Sprintf("documents_%s.zip", time.Now().Format("20060102_150405"))
	}
	if !strings.HasSuffix(zipFileName, ".zip") {
		zipFileName += ".zip"
	}

	// Set response headers for zip download
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, zipFileName))
	c.Header("Cache-Control", "no-store")

	// Create zip writer
	zipWriter := zip.NewWriter(c.Writer)
	defer zipWriter.Close()

	// Track unique filenames to avoid duplicates
	usedNames := make(map[string]int)

	// Add each document to the zip
	for _, meta := range documents {
		// Download file
		reader, _, err := h.storage.Download(ctx, meta.Key)
		if err != nil {
			h.logger.Error("Failed to download file for bulk", "key", meta.Key, "error", err)
			continue
		}

		// Generate unique filename
		fileName := h.getUniqueFileName(meta.FileName, usedNames)

		// Create zip entry
		zipEntry, err := zipWriter.Create(fileName)
		if err != nil {
			reader.Close()
			h.logger.Error("Failed to create zip entry", "file", fileName, "error", err)
			continue
		}

		// Copy content
		_, err = io.Copy(zipEntry, reader)
		reader.Close()

		if err != nil {
			h.logger.Error("Failed to write to zip", "file", fileName, "error", err)
			continue
		}

		// Track download
		h.trackDownload(c, meta, "bulk")

		// Update access time
		h.metadata.UpdateAccess(ctx, meta.ID)
	}

	h.logger.Info("Bulk download completed", "files", len(documents), "user", userInfo.ID)
}

// getUniqueFileName generates a unique filename for zip entries
func (h *DownloadHandler) getUniqueFileName(name string, used map[string]int) string {
	if _, exists := used[name]; !exists {
		used[name] = 1
		return name
	}

	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	for {
		count := used[name]
		used[name] = count + 1
		newName := fmt.Sprintf("%s_%d%s", base, count, ext)
		if _, exists := used[newName]; !exists {
			used[newName] = 1
			return newName
		}
	}
}

// =============================================================================
// History Endpoints
// =============================================================================

// GetHistory returns download history for the current user
func (h *DownloadHandler) GetHistory(c *gin.Context) {
	if h.tracker == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Not Implemented", "message": "download tracking is disabled"})
		return
	}

	userInfo := h.getUserInfo(c)

	// Parse pagination
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}

	// Parse date filters
	var startDate, endDate *time.Time
	if start := c.Query("start_date"); start != "" {
		if t, err := time.Parse("2006-01-02", start); err == nil {
			startDate = &t
		}
	}
	if end := c.Query("end_date"); end != "" {
		if t, err := time.Parse("2006-01-02", end); err == nil {
			endOfDay := t.Add(24*time.Hour - time.Second)
			endDate = &endOfDay
		}
	}

	// Get history
	history := h.tracker.GetUserHistory(userInfo.ID, &HistoryFilter{
		Limit:     limit,
		Offset:    offset,
		StartDate: startDate,
		EndDate:   endDate,
	})

	c.JSON(http.StatusOK, HistoryResponse{
		Items:      history,
		TotalCount: h.tracker.GetUserHistoryCount(userInfo.ID),
		Limit:      limit,
		Offset:     offset,
	})
}

// HistoryResponse is the response for history requests
type HistoryResponse struct {
	Items      []*DownloadRecord `json:"items"`
	TotalCount int               `json:"total_count"`
	Limit      int               `json:"limit"`
	Offset     int               `json:"offset"`
}

// GetDocumentHistory returns download history for a specific document
func (h *DownloadHandler) GetDocumentHistory(c *gin.Context) {
	if h.tracker == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Not Implemented", "message": "download tracking is disabled"})
		return
	}

	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request", "message": "document ID is required"})
		return
	}

	// Parse pagination
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}

	// Get document history
	history := h.tracker.GetDocumentHistory(documentID, &HistoryFilter{
		Limit:  limit,
		Offset: offset,
	})

	c.JSON(http.StatusOK, HistoryResponse{
		Items:      history,
		TotalCount: h.tracker.GetDocumentHistoryCount(documentID),
		Limit:      limit,
		Offset:     offset,
	})
}

// =============================================================================
// Stats Endpoint
// =============================================================================

// GetStats returns download statistics
func (h *DownloadHandler) GetStats(c *gin.Context) {
	if h.tracker == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Not Implemented", "message": "download tracking is disabled"})
		return
	}

	userInfo := h.getUserInfo(c)

	// Parse time range
	period := c.DefaultQuery("period", "30d")

	var since time.Time
	switch period {
	case "24h":
		since = time.Now().Add(-24 * time.Hour)
	case "7d":
		since = time.Now().Add(-7 * 24 * time.Hour)
	case "30d":
		since = time.Now().Add(-30 * 24 * time.Hour)
	case "90d":
		since = time.Now().Add(-90 * 24 * time.Hour)
	default:
		since = time.Now().Add(-30 * 24 * time.Hour)
	}

	stats := h.tracker.GetStats(userInfo.ID, since)

	c.JSON(http.StatusOK, stats)
}

// =============================================================================
// Helper Methods
// =============================================================================

// UserInfo contains information about the requesting user
type UserInfo struct {
	ID      string
	Email   string
	Role    string
	IP      string
	Headers map[string]string
}

// getUserInfo extracts user information from the gin context
func (h *DownloadHandler) getUserInfo(c *gin.Context) *UserInfo {
	// In a real implementation, this would extract from JWT or session
	info := &UserInfo{
		ID:      c.GetHeader("X-User-ID"),
		Email:   c.GetHeader("X-User-Email"),
		Role:    c.GetHeader("X-User-Role"),
		IP:      c.ClientIP(),
		Headers: make(map[string]string),
	}

	if info.ID == "" {
		info.ID = "anonymous"
	}

	// Capture relevant headers for tracking
	headers := []string{"User-Agent", "Referer", "Accept-Language"}
	for _, h := range headers {
		if v := c.GetHeader(h); v != "" {
			info.Headers[h] = v
		}
	}

	return info
}

// canAccess checks if a user can access a document
func (h *DownloadHandler) canAccess(user *UserInfo, meta *DocumentMetadata) bool {
	// Admin can access everything
	if user.Role == "admin" {
		return true
	}

	// Owner can access
	if meta.Owner == user.ID {
		return true
	}

	// Check if shared with user
	for _, sharedWith := range meta.SharedWith {
		if sharedWith == user.ID {
			return true
		}
	}

	// Check access level
	switch meta.AccessLevel {
	case "public":
		return true
	case "authenticated":
		return user.ID != "anonymous"
	case "private":
		return meta.Owner == user.ID
	default:
		return false
	}
}

// setDownloadHeaders sets appropriate headers for file download
func (h *DownloadHandler) setDownloadHeaders(c *gin.Context, meta *DocumentMetadata, info *DownloadInfo, inline bool) {
	// Content-Type
	contentType := meta.ContentType
	if contentType == "" && info != nil {
		contentType = info.ContentType
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)

	// Content-Length
	if info != nil && info.Size > 0 {
		c.Header("Content-Length", strconv.FormatInt(info.Size, 10))
	} else if meta.Size > 0 {
		c.Header("Content-Length", strconv.FormatInt(meta.Size, 10))
	}

	// Content-Disposition
	disposition := "attachment"
	if inline && h.isInlineAllowed(contentType) {
		disposition = "inline"
	} else if h.isForceDownload(contentType) {
		disposition = "attachment"
	}
	c.Header("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, meta.FileName))

	// ETag
	if info != nil && info.ETag != "" {
		c.Header("ETag", info.ETag)
	}

	// Last-Modified
	if info != nil && !info.LastModified.IsZero() {
		c.Header("Last-Modified", info.LastModified.UTC().Format(http.TimeFormat))
	}

	// Cache control
	c.Header("Cache-Control", "private, max-age=3600")

	// Security headers
	c.Header("X-Content-Type-Options", "nosniff")
}

// isInlineAllowed checks if content type can be displayed inline
func (h *DownloadHandler) isInlineAllowed(contentType string) bool {
	for _, allowed := range h.config.InlineContentTypes {
		if strings.EqualFold(contentType, allowed) {
			return true
		}
	}
	return false
}

// isForceDownload checks if content type should force download
func (h *DownloadHandler) isForceDownload(contentType string) bool {
	for _, forceType := range h.config.ForceDownloadTypes {
		if strings.EqualFold(contentType, forceType) {
			return true
		}
	}
	return false
}

// trackDownload records a download event
func (h *DownloadHandler) trackDownload(c *gin.Context, meta *DocumentMetadata, method string) {
	if h.tracker == nil {
		return
	}

	userInfo := h.getUserInfo(c)

	record := &DownloadRecord{
		ID:          uuid.New().String(),
		DocumentID:  meta.ID,
		DocumentKey: meta.Key,
		FileName:    meta.FileName,
		FileSize:    meta.Size,
		ContentType: meta.ContentType,
		UserID:      userInfo.ID,
		UserEmail:   userInfo.Email,
		UserIP:      userInfo.IP,
		UserAgent:   userInfo.Headers["User-Agent"],
		Referer:     userInfo.Headers["Referer"],
		Method:      method,
		Timestamp:   time.Now().UTC(),
	}

	h.tracker.Record(record)
}
