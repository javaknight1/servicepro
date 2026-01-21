package tracking

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Errors
// =============================================================================

var (
	ErrRecordNotFound    = errors.New("delivery record not found")
	ErrInvalidMessageID  = errors.New("invalid message ID")
	ErrInvalidStatus     = errors.New("invalid delivery status")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrRecordExists      = errors.New("delivery record already exists")
	ErrInvalidCustomerID = errors.New("invalid customer ID")
	ErrInvalidTimeRange  = errors.New("invalid time range")
)

// =============================================================================
// Delivery Status Constants
// =============================================================================

type DeliveryStatus string

const (
	StatusPending   DeliveryStatus = "pending"
	StatusQueued    DeliveryStatus = "queued"
	StatusSent      DeliveryStatus = "sent"
	StatusDelivered DeliveryStatus = "delivered"
	StatusFailed    DeliveryStatus = "failed"
	StatusBounced   DeliveryStatus = "bounced"
	StatusRejected  DeliveryStatus = "rejected"
	StatusExpired   DeliveryStatus = "expired"
	StatusUnknown   DeliveryStatus = "unknown"
)

// ValidStatuses contains all valid delivery statuses
var ValidStatuses = map[DeliveryStatus]bool{
	StatusPending:   true,
	StatusQueued:    true,
	StatusSent:      true,
	StatusDelivered: true,
	StatusFailed:    true,
	StatusBounced:   true,
	StatusRejected:  true,
	StatusExpired:   true,
	StatusUnknown:   true,
}

// ValidTransitions defines allowed status transitions
var ValidTransitions = map[DeliveryStatus][]DeliveryStatus{
	StatusPending:   {StatusQueued, StatusFailed, StatusRejected, StatusExpired},
	StatusQueued:    {StatusSent, StatusFailed, StatusRejected, StatusExpired},
	StatusSent:      {StatusDelivered, StatusFailed, StatusBounced, StatusExpired, StatusUnknown},
	StatusDelivered: {},                            // Terminal state
	StatusFailed:    {StatusQueued, StatusPending}, // Can retry
	StatusBounced:   {},                            // Terminal state
	StatusRejected:  {},                            // Terminal state
	StatusExpired:   {},                            // Terminal state
	StatusUnknown:   {StatusDelivered, StatusFailed, StatusBounced},
}

// IsTerminalStatus returns true if the status is a terminal state
func IsTerminalStatus(status DeliveryStatus) bool {
	transitions, exists := ValidTransitions[status]
	return exists && len(transitions) == 0
}

// =============================================================================
// Core Types
// =============================================================================

// DeliveryRecord represents a single delivery attempt record
type DeliveryRecord struct {
	ID            string            `json:"id"`
	MessageID     string            `json:"message_id"`
	CustomerID    string            `json:"customer_id"`
	Channel       string            `json:"channel"` // sms, email, push
	Recipient     string            `json:"recipient"`
	Status        DeliveryStatus    `json:"status"`
	Attempts      int               `json:"attempts"`
	MaxAttempts   int               `json:"max_attempts"`
	FirstAttempt  time.Time         `json:"first_attempt"`
	LastAttempt   time.Time         `json:"last_attempt"`
	DeliveryTime  *time.Time        `json:"delivery_time,omitempty"`
	ErrorCode     string            `json:"error_code,omitempty"`
	ErrorDetails  string            `json:"error_details,omitempty"`
	ProviderMsgID string            `json:"provider_msg_id,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// StatusChange represents a status transition event
type StatusChange struct {
	RecordID   string         `json:"record_id"`
	FromStatus DeliveryStatus `json:"from_status"`
	ToStatus   DeliveryStatus `json:"to_status"`
	Reason     string         `json:"reason,omitempty"`
	Timestamp  time.Time      `json:"timestamp"`
}

// DeliveryLog represents a detailed log entry for an attempt
type DeliveryLog struct {
	ID           string            `json:"id"`
	RecordID     string            `json:"record_id"`
	AttemptNum   int               `json:"attempt_num"`
	Status       DeliveryStatus    `json:"status"`
	ErrorCode    string            `json:"error_code,omitempty"`
	ErrorDetails string            `json:"error_details,omitempty"`
	Duration     time.Duration     `json:"duration"`
	ProviderResp string            `json:"provider_response,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
}

// =============================================================================
// Metrics Types
// =============================================================================

// DeliveryMetrics contains aggregated delivery metrics
type DeliveryMetrics struct {
	TimeRange     TimeRange                  `json:"time_range"`
	TotalMessages int64                      `json:"total_messages"`
	Delivered     int64                      `json:"delivered"`
	Failed        int64                      `json:"failed"`
	Pending       int64                      `json:"pending"`
	Bounced       int64                      `json:"bounced"`
	DeliveryRate  float64                    `json:"delivery_rate"`
	FailureRate   float64                    `json:"failure_rate"`
	BounceRate    float64                    `json:"bounce_rate"`
	AvgAttempts   float64                    `json:"avg_attempts"`
	AvgDeliveryMs float64                    `json:"avg_delivery_time_ms"`
	ByChannel     map[string]*ChannelMetrics `json:"by_channel"`
	ByHour        map[int]*HourlyMetrics     `json:"by_hour"`
	TopErrors     []ErrorMetric              `json:"top_errors"`
	GeneratedAt   time.Time                  `json:"generated_at"`
}

// ChannelMetrics contains metrics for a specific channel
type ChannelMetrics struct {
	Channel       string  `json:"channel"`
	Total         int64   `json:"total"`
	Delivered     int64   `json:"delivered"`
	Failed        int64   `json:"failed"`
	DeliveryRate  float64 `json:"delivery_rate"`
	AvgDeliveryMs float64 `json:"avg_delivery_time_ms"`
}

// HourlyMetrics contains metrics for a specific hour
type HourlyMetrics struct {
	Hour      int   `json:"hour"`
	Total     int64 `json:"total"`
	Delivered int64 `json:"delivered"`
	Failed    int64 `json:"failed"`
}

// ErrorMetric contains error frequency data
type ErrorMetric struct {
	ErrorCode string  `json:"error_code"`
	Count     int64   `json:"count"`
	Percent   float64 `json:"percent"`
}

// TimeRange represents a time range for queries
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// CustomerHistory contains delivery history for a customer
type CustomerHistory struct {
	CustomerID   string            `json:"customer_id"`
	TotalRecords int               `json:"total_records"`
	Records      []*DeliveryRecord `json:"records"`
	Summary      *CustomerSummary  `json:"summary"`
	TimeRange    TimeRange         `json:"time_range"`
}

// CustomerSummary contains summary stats for a customer
type CustomerSummary struct {
	TotalMessages int64            `json:"total_messages"`
	Delivered     int64            `json:"delivered"`
	Failed        int64            `json:"failed"`
	DeliveryRate  float64          `json:"delivery_rate"`
	LastDelivery  *time.Time       `json:"last_delivery,omitempty"`
	ByChannel     map[string]int64 `json:"by_channel"`
}

// =============================================================================
// Query Options
// =============================================================================

// QueryOptions for filtering delivery records
type QueryOptions struct {
	CustomerID string
	Channel    string
	Status     DeliveryStatus
	TimeRange  *TimeRange
	Limit      int
	Offset     int
	SortBy     string
	SortDesc   bool
}

// =============================================================================
// Storage Interface
// =============================================================================

// DeliveryStore interface for pluggable storage backends
type DeliveryStore interface {
	SaveRecord(ctx context.Context, record *DeliveryRecord) error
	GetRecord(ctx context.Context, id string) (*DeliveryRecord, error)
	GetRecordByMessageID(ctx context.Context, messageID string) (*DeliveryRecord, error)
	UpdateRecord(ctx context.Context, record *DeliveryRecord) error
	QueryRecords(ctx context.Context, opts QueryOptions) ([]*DeliveryRecord, error)
	CountRecords(ctx context.Context, opts QueryOptions) (int64, error)
	SaveLog(ctx context.Context, log *DeliveryLog) error
	GetLogs(ctx context.Context, recordID string) ([]*DeliveryLog, error)
	SaveStatusChange(ctx context.Context, change *StatusChange) error
	GetStatusHistory(ctx context.Context, recordID string) ([]*StatusChange, error)
}

// =============================================================================
// In-Memory Store Implementation
// =============================================================================

// MemoryStore provides an in-memory implementation of DeliveryStore
type MemoryStore struct {
	records       map[string]*DeliveryRecord
	recordsByMsg  map[string]string // messageID -> recordID
	logs          map[string][]*DeliveryLog
	statusHistory map[string][]*StatusChange
	mu            sync.RWMutex
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		records:       make(map[string]*DeliveryRecord),
		recordsByMsg:  make(map[string]string),
		logs:          make(map[string][]*DeliveryLog),
		statusHistory: make(map[string][]*StatusChange),
	}
}

func (s *MemoryStore) SaveRecord(ctx context.Context, record *DeliveryRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.records[record.ID]; exists {
		return ErrRecordExists
	}

	// Check for duplicate MessageID
	if record.MessageID != "" {
		if _, exists := s.recordsByMsg[record.MessageID]; exists {
			return ErrRecordExists
		}
	}

	s.records[record.ID] = record
	if record.MessageID != "" {
		s.recordsByMsg[record.MessageID] = record.ID
	}
	return nil
}

func (s *MemoryStore) GetRecord(ctx context.Context, id string) (*DeliveryRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, exists := s.records[id]
	if !exists {
		return nil, ErrRecordNotFound
	}
	return record, nil
}

func (s *MemoryStore) GetRecordByMessageID(ctx context.Context, messageID string) (*DeliveryRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	recordID, exists := s.recordsByMsg[messageID]
	if !exists {
		return nil, ErrRecordNotFound
	}
	return s.records[recordID], nil
}

func (s *MemoryStore) UpdateRecord(ctx context.Context, record *DeliveryRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.records[record.ID]; !exists {
		return ErrRecordNotFound
	}
	s.records[record.ID] = record
	return nil
}

func (s *MemoryStore) QueryRecords(ctx context.Context, opts QueryOptions) ([]*DeliveryRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*DeliveryRecord

	for _, record := range s.records {
		if s.matchesQuery(record, opts) {
			results = append(results, record)
		}
	}

	// Sort results
	sort.Slice(results, func(i, j int) bool {
		if opts.SortBy == "created_at" {
			if opts.SortDesc {
				return results[i].CreatedAt.After(results[j].CreatedAt)
			}
			return results[i].CreatedAt.Before(results[j].CreatedAt)
		}
		// Default sort by updated_at desc
		if opts.SortDesc {
			return results[i].UpdatedAt.After(results[j].UpdatedAt)
		}
		return results[i].UpdatedAt.Before(results[j].UpdatedAt)
	})

	// Apply pagination
	if opts.Offset > 0 && opts.Offset < len(results) {
		results = results[opts.Offset:]
	} else if opts.Offset >= len(results) {
		return []*DeliveryRecord{}, nil
	}

	if opts.Limit > 0 && opts.Limit < len(results) {
		results = results[:opts.Limit]
	}

	return results, nil
}

func (s *MemoryStore) CountRecords(ctx context.Context, opts QueryOptions) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int64
	for _, record := range s.records {
		if s.matchesQuery(record, opts) {
			count++
		}
	}
	return count, nil
}

func (s *MemoryStore) matchesQuery(record *DeliveryRecord, opts QueryOptions) bool {
	if opts.CustomerID != "" && record.CustomerID != opts.CustomerID {
		return false
	}
	if opts.Channel != "" && record.Channel != opts.Channel {
		return false
	}
	if opts.Status != "" && record.Status != opts.Status {
		return false
	}
	if opts.TimeRange != nil {
		if record.CreatedAt.Before(opts.TimeRange.Start) || record.CreatedAt.After(opts.TimeRange.End) {
			return false
		}
	}
	return true
}

func (s *MemoryStore) SaveLog(ctx context.Context, log *DeliveryLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logs[log.RecordID] = append(s.logs[log.RecordID], log)
	return nil
}

func (s *MemoryStore) GetLogs(ctx context.Context, recordID string) ([]*DeliveryLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logs, exists := s.logs[recordID]
	if !exists {
		return []*DeliveryLog{}, nil
	}
	return logs, nil
}

func (s *MemoryStore) SaveStatusChange(ctx context.Context, change *StatusChange) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.statusHistory[change.RecordID] = append(s.statusHistory[change.RecordID], change)
	return nil
}

func (s *MemoryStore) GetStatusHistory(ctx context.Context, recordID string) ([]*StatusChange, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, exists := s.statusHistory[recordID]
	if !exists {
		return []*StatusChange{}, nil
	}
	return history, nil
}

// =============================================================================
// Delivery Tracker
// =============================================================================

// TrackerConfig contains configuration for the delivery tracker
type TrackerConfig struct {
	MaxAttempts   int
	EnableLogging bool
	EnableHistory bool
	RetentionDays int
}

// DefaultTrackerConfig returns default configuration
func DefaultTrackerConfig() *TrackerConfig {
	return &TrackerConfig{
		MaxAttempts:   3,
		EnableLogging: true,
		EnableHistory: true,
		RetentionDays: 30,
	}
}

// DeliveryTracker manages delivery tracking
type DeliveryTracker struct {
	store  DeliveryStore
	config *TrackerConfig
	mu     sync.RWMutex
}

// NewDeliveryTracker creates a new delivery tracker
func NewDeliveryTracker(store DeliveryStore, config *TrackerConfig) *DeliveryTracker {
	if store == nil {
		store = NewMemoryStore()
	}
	if config == nil {
		config = DefaultTrackerConfig()
	}
	return &DeliveryTracker{
		store:  store,
		config: config,
	}
}

// =============================================================================
// Core Tracking Methods
// =============================================================================

// TrackDeliveryInput contains input for tracking a new delivery
type TrackDeliveryInput struct {
	MessageID  string
	CustomerID string
	Channel    string
	Recipient  string
	Metadata   map[string]string
}

// TrackDelivery creates a new delivery record
func (t *DeliveryTracker) TrackDelivery(ctx context.Context, input *TrackDeliveryInput) (*DeliveryRecord, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}
	if input.MessageID == "" {
		return nil, ErrInvalidMessageID
	}

	now := time.Now()
	record := &DeliveryRecord{
		ID:          uuid.New().String(),
		MessageID:   input.MessageID,
		CustomerID:  input.CustomerID,
		Channel:     input.Channel,
		Recipient:   input.Recipient,
		Status:      StatusPending,
		Attempts:    0,
		MaxAttempts: t.config.MaxAttempts,
		Metadata:    input.Metadata,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := t.store.SaveRecord(ctx, record); err != nil {
		return nil, err
	}

	// Log initial status
	if t.config.EnableHistory {
		change := &StatusChange{
			RecordID:   record.ID,
			FromStatus: "",
			ToStatus:   StatusPending,
			Reason:     "Initial tracking",
			Timestamp:  now,
		}
		t.store.SaveStatusChange(ctx, change)
	}

	return record, nil
}

// UpdateStatusInput contains input for updating delivery status
type UpdateStatusInput struct {
	RecordID      string
	MessageID     string // Alternative to RecordID
	Status        DeliveryStatus
	ErrorCode     string
	ErrorDetails  string
	ProviderMsgID string
	ProviderResp  string
	Duration      time.Duration
	Reason        string
}

// UpdateStatus updates the status of a delivery record
func (t *DeliveryTracker) UpdateStatus(ctx context.Context, input *UpdateStatusInput) (*DeliveryRecord, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	// Validate status
	if !ValidStatuses[input.Status] {
		return nil, ErrInvalidStatus
	}

	// Get the record
	var record *DeliveryRecord
	var err error

	if input.RecordID != "" {
		record, err = t.store.GetRecord(ctx, input.RecordID)
	} else if input.MessageID != "" {
		record, err = t.store.GetRecordByMessageID(ctx, input.MessageID)
	} else {
		return nil, ErrInvalidMessageID
	}

	if err != nil {
		return nil, err
	}

	// Validate transition
	if !t.isValidTransition(record.Status, input.Status) {
		return nil, ErrInvalidTransition
	}

	oldStatus := record.Status
	now := time.Now()

	// Update record
	record.Status = input.Status
	record.UpdatedAt = now

	if input.ErrorCode != "" {
		record.ErrorCode = input.ErrorCode
	}
	if input.ErrorDetails != "" {
		record.ErrorDetails = input.ErrorDetails
	}
	if input.ProviderMsgID != "" {
		record.ProviderMsgID = input.ProviderMsgID
	}

	// Track attempts for send operations - only on actual transition (not idempotent updates)
	if (input.Status == StatusSent || input.Status == StatusFailed) && oldStatus != input.Status {
		record.Attempts++
		record.LastAttempt = now
		if record.FirstAttempt.IsZero() {
			record.FirstAttempt = now
		}
	}

	// Set delivery time for successful delivery
	if input.Status == StatusDelivered {
		record.DeliveryTime = &now
	}

	if err := t.store.UpdateRecord(ctx, record); err != nil {
		return nil, err
	}

	// Log the status change
	if t.config.EnableHistory {
		change := &StatusChange{
			RecordID:   record.ID,
			FromStatus: oldStatus,
			ToStatus:   input.Status,
			Reason:     input.Reason,
			Timestamp:  now,
		}
		t.store.SaveStatusChange(ctx, change)
	}

	// Create delivery log
	if t.config.EnableLogging && (input.Status == StatusSent || input.Status == StatusFailed || input.Status == StatusDelivered) {
		log := &DeliveryLog{
			ID:           uuid.New().String(),
			RecordID:     record.ID,
			AttemptNum:   record.Attempts,
			Status:       input.Status,
			ErrorCode:    input.ErrorCode,
			ErrorDetails: input.ErrorDetails,
			Duration:     input.Duration,
			ProviderResp: input.ProviderResp,
			Timestamp:    now,
		}
		t.store.SaveLog(ctx, log)
	}

	return record, nil
}

func (t *DeliveryTracker) isValidTransition(from, to DeliveryStatus) bool {
	// Allow same status (idempotent updates)
	if from == to {
		return true
	}

	validTargets, exists := ValidTransitions[from]
	if !exists {
		return false
	}

	for _, valid := range validTargets {
		if valid == to {
			return true
		}
	}
	return false
}

// LogAttempt logs a delivery attempt
func (t *DeliveryTracker) LogAttempt(ctx context.Context, recordID string, log *DeliveryLog) error {
	if recordID == "" {
		return ErrInvalidMessageID
	}
	if log == nil {
		return errors.New("log cannot be nil")
	}

	log.ID = uuid.New().String()
	log.RecordID = recordID
	log.Timestamp = time.Now()

	return t.store.SaveLog(ctx, log)
}

// GetRecord retrieves a delivery record by ID
func (t *DeliveryTracker) GetRecord(ctx context.Context, id string) (*DeliveryRecord, error) {
	if id == "" {
		return nil, ErrInvalidMessageID
	}
	return t.store.GetRecord(ctx, id)
}

// GetRecordByMessageID retrieves a delivery record by message ID
func (t *DeliveryTracker) GetRecordByMessageID(ctx context.Context, messageID string) (*DeliveryRecord, error) {
	if messageID == "" {
		return nil, ErrInvalidMessageID
	}
	return t.store.GetRecordByMessageID(ctx, messageID)
}

// GetDeliveryLogs retrieves all logs for a delivery record
func (t *DeliveryTracker) GetDeliveryLogs(ctx context.Context, recordID string) ([]*DeliveryLog, error) {
	if recordID == "" {
		return nil, ErrInvalidMessageID
	}
	return t.store.GetLogs(ctx, recordID)
}

// GetStatusHistory retrieves status change history for a record
func (t *DeliveryTracker) GetStatusHistory(ctx context.Context, recordID string) ([]*StatusChange, error) {
	if recordID == "" {
		return nil, ErrInvalidMessageID
	}
	return t.store.GetStatusHistory(ctx, recordID)
}

// =============================================================================
// Metrics Generation
// =============================================================================

// GenerateMetrics generates delivery metrics for a time range
func (t *DeliveryTracker) GenerateMetrics(ctx context.Context, timeRange TimeRange) (*DeliveryMetrics, error) {
	if timeRange.End.Before(timeRange.Start) {
		return nil, ErrInvalidTimeRange
	}

	opts := QueryOptions{
		TimeRange: &timeRange,
	}

	records, err := t.store.QueryRecords(ctx, opts)
	if err != nil {
		return nil, err
	}

	metrics := &DeliveryMetrics{
		TimeRange:   timeRange,
		ByChannel:   make(map[string]*ChannelMetrics),
		ByHour:      make(map[int]*HourlyMetrics),
		GeneratedAt: time.Now(),
	}

	// Error tracking
	errorCounts := make(map[string]int64)
	var totalDeliveryTimeMs int64
	var deliveredWithTime int64

	for _, record := range records {
		metrics.TotalMessages++

		// Count by status
		switch record.Status {
		case StatusDelivered:
			metrics.Delivered++
			if record.DeliveryTime != nil && !record.FirstAttempt.IsZero() {
				deliveryMs := record.DeliveryTime.Sub(record.FirstAttempt).Milliseconds()
				totalDeliveryTimeMs += deliveryMs
				deliveredWithTime++
			}
		case StatusFailed:
			metrics.Failed++
			if record.ErrorCode != "" {
				errorCounts[record.ErrorCode]++
			}
		case StatusPending, StatusQueued, StatusSent:
			metrics.Pending++
		case StatusBounced:
			metrics.Bounced++
		}

		// Track by channel
		if record.Channel != "" {
			cm, exists := metrics.ByChannel[record.Channel]
			if !exists {
				cm = &ChannelMetrics{Channel: record.Channel}
				metrics.ByChannel[record.Channel] = cm
			}
			cm.Total++
			if record.Status == StatusDelivered {
				cm.Delivered++
			} else if record.Status == StatusFailed {
				cm.Failed++
			}
		}

		// Track by hour
		hour := record.CreatedAt.Hour()
		hm, exists := metrics.ByHour[hour]
		if !exists {
			hm = &HourlyMetrics{Hour: hour}
			metrics.ByHour[hour] = hm
		}
		hm.Total++
		if record.Status == StatusDelivered {
			hm.Delivered++
		} else if record.Status == StatusFailed {
			hm.Failed++
		}

		// Track total attempts for average
		metrics.AvgAttempts += float64(record.Attempts)
	}

	// Calculate rates
	if metrics.TotalMessages > 0 {
		metrics.DeliveryRate = float64(metrics.Delivered) / float64(metrics.TotalMessages) * 100
		metrics.FailureRate = float64(metrics.Failed) / float64(metrics.TotalMessages) * 100
		metrics.BounceRate = float64(metrics.Bounced) / float64(metrics.TotalMessages) * 100
		metrics.AvgAttempts = metrics.AvgAttempts / float64(metrics.TotalMessages)
	}

	// Calculate average delivery time
	if deliveredWithTime > 0 {
		metrics.AvgDeliveryMs = float64(totalDeliveryTimeMs) / float64(deliveredWithTime)
	}

	// Calculate channel rates
	for _, cm := range metrics.ByChannel {
		if cm.Total > 0 {
			cm.DeliveryRate = float64(cm.Delivered) / float64(cm.Total) * 100
		}
	}

	// Build top errors
	metrics.TopErrors = t.buildTopErrors(errorCounts, metrics.Failed)

	return metrics, nil
}

func (t *DeliveryTracker) buildTopErrors(errorCounts map[string]int64, totalFailed int64) []ErrorMetric {
	var errors []ErrorMetric
	for code, count := range errorCounts {
		percent := 0.0
		if totalFailed > 0 {
			percent = float64(count) / float64(totalFailed) * 100
		}
		errors = append(errors, ErrorMetric{
			ErrorCode: code,
			Count:     count,
			Percent:   percent,
		})
	}

	// Sort by count descending
	sort.Slice(errors, func(i, j int) bool {
		return errors[i].Count > errors[j].Count
	})

	// Return top 10
	if len(errors) > 10 {
		errors = errors[:10]
	}

	return errors
}

// GenerateChannelMetrics generates metrics for a specific channel
func (t *DeliveryTracker) GenerateChannelMetrics(ctx context.Context, channel string, timeRange TimeRange) (*ChannelMetrics, error) {
	if channel == "" {
		return nil, errors.New("channel cannot be empty")
	}
	if timeRange.End.Before(timeRange.Start) {
		return nil, ErrInvalidTimeRange
	}

	opts := QueryOptions{
		Channel:   channel,
		TimeRange: &timeRange,
	}

	records, err := t.store.QueryRecords(ctx, opts)
	if err != nil {
		return nil, err
	}

	cm := &ChannelMetrics{Channel: channel}
	var totalDeliveryTimeMs int64
	var deliveredWithTime int64

	for _, record := range records {
		cm.Total++
		if record.Status == StatusDelivered {
			cm.Delivered++
			if record.DeliveryTime != nil && !record.FirstAttempt.IsZero() {
				deliveryMs := record.DeliveryTime.Sub(record.FirstAttempt).Milliseconds()
				totalDeliveryTimeMs += deliveryMs
				deliveredWithTime++
			}
		} else if record.Status == StatusFailed {
			cm.Failed++
		}
	}

	if cm.Total > 0 {
		cm.DeliveryRate = float64(cm.Delivered) / float64(cm.Total) * 100
	}
	if deliveredWithTime > 0 {
		cm.AvgDeliveryMs = float64(totalDeliveryTimeMs) / float64(deliveredWithTime)
	}

	return cm, nil
}

// =============================================================================
// Customer History Management
// =============================================================================

// GetCustomerHistory retrieves delivery history for a customer
func (t *DeliveryTracker) GetCustomerHistory(ctx context.Context, customerID string, opts *QueryOptions) (*CustomerHistory, error) {
	if customerID == "" {
		return nil, ErrInvalidCustomerID
	}

	queryOpts := QueryOptions{
		CustomerID: customerID,
		SortBy:     "created_at",
		SortDesc:   true,
	}

	if opts != nil {
		if opts.Channel != "" {
			queryOpts.Channel = opts.Channel
		}
		if opts.Status != "" {
			queryOpts.Status = opts.Status
		}
		if opts.TimeRange != nil {
			queryOpts.TimeRange = opts.TimeRange
		}
		if opts.Limit > 0 {
			queryOpts.Limit = opts.Limit
		}
		if opts.Offset > 0 {
			queryOpts.Offset = opts.Offset
		}
	}

	records, err := t.store.QueryRecords(ctx, queryOpts)
	if err != nil {
		return nil, err
	}

	// Get total count without pagination
	countOpts := QueryOptions{
		CustomerID: customerID,
		Channel:    queryOpts.Channel,
		Status:     queryOpts.Status,
		TimeRange:  queryOpts.TimeRange,
	}
	totalCount, err := t.store.CountRecords(ctx, countOpts)
	if err != nil {
		return nil, err
	}

	history := &CustomerHistory{
		CustomerID:   customerID,
		TotalRecords: int(totalCount),
		Records:      records,
		Summary:      t.buildCustomerSummary(records),
	}

	if queryOpts.TimeRange != nil {
		history.TimeRange = *queryOpts.TimeRange
	}

	return history, nil
}

func (t *DeliveryTracker) buildCustomerSummary(records []*DeliveryRecord) *CustomerSummary {
	summary := &CustomerSummary{
		ByChannel: make(map[string]int64),
	}

	var lastDelivery *time.Time

	for _, record := range records {
		summary.TotalMessages++

		if record.Status == StatusDelivered {
			summary.Delivered++
			if record.DeliveryTime != nil {
				if lastDelivery == nil || record.DeliveryTime.After(*lastDelivery) {
					lastDelivery = record.DeliveryTime
				}
			}
		} else if record.Status == StatusFailed || record.Status == StatusBounced || record.Status == StatusRejected {
			summary.Failed++
		}

		if record.Channel != "" {
			summary.ByChannel[record.Channel]++
		}
	}

	if summary.TotalMessages > 0 {
		summary.DeliveryRate = float64(summary.Delivered) / float64(summary.TotalMessages) * 100
	}
	summary.LastDelivery = lastDelivery

	return summary
}

// GetRecentDeliveries retrieves recent deliveries for a customer
func (t *DeliveryTracker) GetRecentDeliveries(ctx context.Context, customerID string, limit int) ([]*DeliveryRecord, error) {
	if customerID == "" {
		return nil, ErrInvalidCustomerID
	}
	if limit <= 0 {
		limit = 10
	}

	opts := QueryOptions{
		CustomerID: customerID,
		Limit:      limit,
		SortBy:     "created_at",
		SortDesc:   true,
	}

	return t.store.QueryRecords(ctx, opts)
}

// GetFailedDeliveries retrieves failed deliveries for retry analysis
func (t *DeliveryTracker) GetFailedDeliveries(ctx context.Context, timeRange TimeRange) ([]*DeliveryRecord, error) {
	if timeRange.End.Before(timeRange.Start) {
		return nil, ErrInvalidTimeRange
	}

	opts := QueryOptions{
		Status:    StatusFailed,
		TimeRange: &timeRange,
		SortDesc:  true,
	}

	return t.store.QueryRecords(ctx, opts)
}

// GetPendingDeliveries retrieves pending deliveries
func (t *DeliveryTracker) GetPendingDeliveries(ctx context.Context) ([]*DeliveryRecord, error) {
	opts := QueryOptions{
		Status:   StatusPending,
		SortDesc: false, // Oldest first
	}

	return t.store.QueryRecords(ctx, opts)
}

// =============================================================================
// Bulk Operations
// =============================================================================

// BulkUpdateStatus updates status for multiple records
func (t *DeliveryTracker) BulkUpdateStatus(ctx context.Context, recordIDs []string, status DeliveryStatus, reason string) (int, error) {
	if !ValidStatuses[status] {
		return 0, ErrInvalidStatus
	}

	updated := 0
	for _, id := range recordIDs {
		input := &UpdateStatusInput{
			RecordID: id,
			Status:   status,
			Reason:   reason,
		}
		_, err := t.UpdateStatus(ctx, input)
		if err == nil {
			updated++
		}
	}

	return updated, nil
}

// CleanupOldRecords removes records older than retention period
func (t *DeliveryTracker) CleanupOldRecords(ctx context.Context) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -t.config.RetentionDays)

	opts := QueryOptions{
		TimeRange: &TimeRange{
			Start: time.Time{}, // Beginning of time
			End:   cutoff,
		},
	}

	// This is a simplified implementation - a real one would delete records
	count, err := t.store.CountRecords(ctx, opts)
	if err != nil {
		return 0, err
	}

	return count, nil
}
