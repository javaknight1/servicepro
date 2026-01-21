package handlers

import (
	"sync"
	"time"
)

// =============================================================================
// Download Tracking System
// =============================================================================

// DownloadTracker tracks download events and history
type DownloadTracker struct {
	mu sync.RWMutex

	// Records by user ID
	userRecords map[string][]*DownloadRecord

	// Records by document ID
	documentRecords map[string][]*DownloadRecord

	// All records (for stats)
	allRecords []*DownloadRecord

	// Configuration
	maxRecordsPerUser     int
	maxRecordsPerDocument int
	maxTotalRecords       int

	// Optional persistence callback
	persistFunc func(*DownloadRecord) error
}

// DownloadRecord represents a single download event
type DownloadRecord struct {
	ID          string    `json:"id"`
	DocumentID  string    `json:"document_id"`
	DocumentKey string    `json:"document_key"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	ContentType string    `json:"content_type"`
	UserID      string    `json:"user_id"`
	UserEmail   string    `json:"user_email,omitempty"`
	UserIP      string    `json:"user_ip"`
	UserAgent   string    `json:"user_agent,omitempty"`
	Referer     string    `json:"referer,omitempty"`
	Method      string    `json:"method"` // stream, redirect, url_generated, bulk
	Timestamp   time.Time `json:"timestamp"`
	Duration    int64     `json:"duration_ms,omitempty"` // Download duration in milliseconds
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
}

// HistoryFilter contains filters for history queries
type HistoryFilter struct {
	Limit        int
	Offset       int
	StartDate    *time.Time
	EndDate      *time.Time
	Method       string
	DocumentType string
}

// DownloadStats contains download statistics
type DownloadStats struct {
	TotalDownloads    int64            `json:"total_downloads"`
	TotalBytes        int64            `json:"total_bytes"`
	UniqueDocuments   int              `json:"unique_documents"`
	UniqueUsers       int              `json:"unique_users"`
	DownloadsByDay    map[string]int64 `json:"downloads_by_day"`
	DownloadsByType   map[string]int64 `json:"downloads_by_type"`
	DownloadsByMethod map[string]int64 `json:"downloads_by_method"`
	TopDocuments      []DocumentStat   `json:"top_documents"`
	AverageFileSize   int64            `json:"average_file_size"`
	Period            string           `json:"period"`
	StartDate         time.Time        `json:"start_date"`
	EndDate           time.Time        `json:"end_date"`
}

// DocumentStat contains stats for a single document
type DocumentStat struct {
	DocumentID    string `json:"document_id"`
	FileName      string `json:"file_name"`
	DownloadCount int64  `json:"download_count"`
	TotalBytes    int64  `json:"total_bytes"`
}

// TrackerConfig contains tracker configuration
type TrackerConfig struct {
	MaxRecordsPerUser     int
	MaxRecordsPerDocument int
	MaxTotalRecords       int
	PersistFunc           func(*DownloadRecord) error
}

// DefaultTrackerConfig returns default tracker configuration
func DefaultTrackerConfig() *TrackerConfig {
	return &TrackerConfig{
		MaxRecordsPerUser:     1000,
		MaxRecordsPerDocument: 1000,
		MaxTotalRecords:       100000,
	}
}

// NewDownloadTracker creates a new download tracker
func NewDownloadTracker() *DownloadTracker {
	return NewDownloadTrackerWithConfig(DefaultTrackerConfig())
}

// NewDownloadTrackerWithConfig creates a new download tracker with custom config
func NewDownloadTrackerWithConfig(config *TrackerConfig) *DownloadTracker {
	if config == nil {
		config = DefaultTrackerConfig()
	}

	return &DownloadTracker{
		userRecords:           make(map[string][]*DownloadRecord),
		documentRecords:       make(map[string][]*DownloadRecord),
		allRecords:            make([]*DownloadRecord, 0),
		maxRecordsPerUser:     config.MaxRecordsPerUser,
		maxRecordsPerDocument: config.MaxRecordsPerDocument,
		maxTotalRecords:       config.MaxTotalRecords,
		persistFunc:           config.PersistFunc,
	}
}

// Record records a download event
func (t *DownloadTracker) Record(record *DownloadRecord) {
	if record == nil {
		return
	}

	// Set defaults
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now().UTC()
	}
	if !record.Success && record.Error == "" {
		record.Success = true
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Add to user records
	t.userRecords[record.UserID] = t.appendWithLimit(
		t.userRecords[record.UserID],
		record,
		t.maxRecordsPerUser,
	)

	// Add to document records
	t.documentRecords[record.DocumentID] = t.appendWithLimit(
		t.documentRecords[record.DocumentID],
		record,
		t.maxRecordsPerDocument,
	)

	// Add to all records
	t.allRecords = t.appendWithLimit(t.allRecords, record, t.maxTotalRecords)

	// Persist if callback is set
	if t.persistFunc != nil {
		go t.persistFunc(record)
	}
}

// appendWithLimit appends a record while respecting max limit
func (t *DownloadTracker) appendWithLimit(records []*DownloadRecord, record *DownloadRecord, maxSize int) []*DownloadRecord {
	records = append(records, record)
	if len(records) > maxSize {
		// Remove oldest records (first 10% when limit exceeded)
		removeCount := maxSize / 10
		if removeCount < 1 {
			removeCount = 1
		}
		records = records[removeCount:]
	}
	return records
}

// GetUserHistory returns download history for a user
func (t *DownloadTracker) GetUserHistory(userID string, filter *HistoryFilter) []*DownloadRecord {
	t.mu.RLock()
	records := t.userRecords[userID]
	t.mu.RUnlock()

	return t.filterRecords(records, filter)
}

// GetUserHistoryCount returns the total count of user history records
func (t *DownloadTracker) GetUserHistoryCount(userID string) int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.userRecords[userID])
}

// GetDocumentHistory returns download history for a document
func (t *DownloadTracker) GetDocumentHistory(documentID string, filter *HistoryFilter) []*DownloadRecord {
	t.mu.RLock()
	records := t.documentRecords[documentID]
	t.mu.RUnlock()

	return t.filterRecords(records, filter)
}

// GetDocumentHistoryCount returns the total count of document history records
func (t *DownloadTracker) GetDocumentHistoryCount(documentID string) int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.documentRecords[documentID])
}

// filterRecords applies filters and pagination to records
func (t *DownloadTracker) filterRecords(records []*DownloadRecord, filter *HistoryFilter) []*DownloadRecord {
	if filter == nil {
		filter = &HistoryFilter{Limit: 50}
	}

	// Apply date filters
	var filtered []*DownloadRecord
	for i := len(records) - 1; i >= 0; i-- { // Reverse order (newest first)
		r := records[i]

		// Check start date
		if filter.StartDate != nil && r.Timestamp.Before(*filter.StartDate) {
			continue
		}

		// Check end date
		if filter.EndDate != nil && r.Timestamp.After(*filter.EndDate) {
			continue
		}

		// Check method
		if filter.Method != "" && r.Method != filter.Method {
			continue
		}

		filtered = append(filtered, r)
	}

	// Apply offset
	if filter.Offset > 0 {
		if filter.Offset >= len(filtered) {
			return []*DownloadRecord{}
		}
		filtered = filtered[filter.Offset:]
	}

	// Apply limit
	if filter.Limit > 0 && len(filtered) > filter.Limit {
		filtered = filtered[:filter.Limit]
	}

	return filtered
}

// GetStats returns download statistics for a user since a given time
func (t *DownloadTracker) GetStats(userID string, since time.Time) *DownloadStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := &DownloadStats{
		DownloadsByDay:    make(map[string]int64),
		DownloadsByType:   make(map[string]int64),
		DownloadsByMethod: make(map[string]int64),
		StartDate:         since,
		EndDate:           time.Now().UTC(),
	}

	// Determine which records to analyze
	var records []*DownloadRecord
	if userID != "" && userID != "admin" {
		records = t.userRecords[userID]
	} else {
		records = t.allRecords
	}

	// Track unique documents and users
	uniqueDocs := make(map[string]bool)
	uniqueUsers := make(map[string]bool)
	docStats := make(map[string]*DocumentStat)

	for _, r := range records {
		if r.Timestamp.Before(since) {
			continue
		}

		stats.TotalDownloads++
		stats.TotalBytes += r.FileSize

		// Track by day
		dayKey := r.Timestamp.Format("2006-01-02")
		stats.DownloadsByDay[dayKey]++

		// Track by content type
		if r.ContentType != "" {
			stats.DownloadsByType[r.ContentType]++
		}

		// Track by method
		if r.Method != "" {
			stats.DownloadsByMethod[r.Method]++
		}

		// Track unique documents
		uniqueDocs[r.DocumentID] = true

		// Track unique users
		uniqueUsers[r.UserID] = true

		// Track document stats
		if ds, ok := docStats[r.DocumentID]; ok {
			ds.DownloadCount++
			ds.TotalBytes += r.FileSize
		} else {
			docStats[r.DocumentID] = &DocumentStat{
				DocumentID:    r.DocumentID,
				FileName:      r.FileName,
				DownloadCount: 1,
				TotalBytes:    r.FileSize,
			}
		}
	}

	stats.UniqueDocuments = len(uniqueDocs)
	stats.UniqueUsers = len(uniqueUsers)

	if stats.TotalDownloads > 0 {
		stats.AverageFileSize = stats.TotalBytes / stats.TotalDownloads
	}

	// Get top documents (top 10)
	stats.TopDocuments = t.getTopDocuments(docStats, 10)

	return stats
}

// getTopDocuments returns the top N documents by download count
func (t *DownloadTracker) getTopDocuments(docStats map[string]*DocumentStat, limit int) []DocumentStat {
	// Convert to slice
	docs := make([]DocumentStat, 0, len(docStats))
	for _, ds := range docStats {
		docs = append(docs, *ds)
	}

	// Sort by download count (simple bubble sort for small lists)
	for i := 0; i < len(docs); i++ {
		for j := i + 1; j < len(docs); j++ {
			if docs[j].DownloadCount > docs[i].DownloadCount {
				docs[i], docs[j] = docs[j], docs[i]
			}
		}
	}

	// Return top N
	if len(docs) > limit {
		docs = docs[:limit]
	}

	return docs
}

// Clear clears all tracking data
func (t *DownloadTracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.userRecords = make(map[string][]*DownloadRecord)
	t.documentRecords = make(map[string][]*DownloadRecord)
	t.allRecords = make([]*DownloadRecord, 0)
}

// ClearUser clears tracking data for a specific user
func (t *DownloadTracker) ClearUser(userID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.userRecords, userID)

	// Also remove from allRecords
	filtered := make([]*DownloadRecord, 0)
	for _, r := range t.allRecords {
		if r.UserID != userID {
			filtered = append(filtered, r)
		}
	}
	t.allRecords = filtered
}

// ClearDocument clears tracking data for a specific document
func (t *DownloadTracker) ClearDocument(documentID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.documentRecords, documentID)

	// Also remove from allRecords
	filtered := make([]*DownloadRecord, 0)
	for _, r := range t.allRecords {
		if r.DocumentID != documentID {
			filtered = append(filtered, r)
		}
	}
	t.allRecords = filtered
}

// GetRecentDownloads returns the most recent N downloads
func (t *DownloadTracker) GetRecentDownloads(limit int) []*DownloadRecord {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if limit <= 0 || limit > len(t.allRecords) {
		limit = len(t.allRecords)
	}

	// Return newest records (at the end of slice)
	start := len(t.allRecords) - limit
	if start < 0 {
		start = 0
	}

	// Create copy in reverse order (newest first)
	result := make([]*DownloadRecord, 0, limit)
	for i := len(t.allRecords) - 1; i >= start; i-- {
		result = append(result, t.allRecords[i])
	}

	return result
}

// SetPersistFunc sets the persistence callback function
func (t *DownloadTracker) SetPersistFunc(fn func(*DownloadRecord) error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.persistFunc = fn
}

// RecordCount returns the total number of records
func (t *DownloadTracker) RecordCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.allRecords)
}

// UserCount returns the number of unique users with records
func (t *DownloadTracker) UserCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.userRecords)
}

// DocumentCount returns the number of unique documents with records
func (t *DownloadTracker) DocumentCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.documentRecords)
}
