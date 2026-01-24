package queue

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// Standard errors
var (
	ErrQueueFull         = errors.New("queue is full")
	ErrQueueStopped      = errors.New("queue has been stopped")
	ErrMessageNotFound   = errors.New("message not found")
	ErrInvalidMessage    = errors.New("invalid message")
	ErrInvalidPhone      = errors.New("invalid phone number")
	ErrMaxRetriesReached = errors.New("maximum retries reached")
	ErrScheduleInPast    = errors.New("scheduled time is in the past")
	ErrBatchEmpty        = errors.New("batch is empty")
)

// Priority levels
const (
	PriorityUrgent = 0 // Highest priority (OTP, alerts)
	PriorityHigh   = 1
	PriorityNormal = 2
	PriorityLow    = 3
	PriorityBulk   = 4 // Lowest priority (marketing)
)

// Message status
type MessageStatus string

const (
	StatusPending    MessageStatus = "pending"
	StatusScheduled  MessageStatus = "scheduled"
	StatusProcessing MessageStatus = "processing"
	StatusSent       MessageStatus = "sent"
	StatusFailed     MessageStatus = "failed"
	StatusRetrying   MessageStatus = "retrying"
	StatusCancelled  MessageStatus = "cancelled"
	StatusExpired    MessageStatus = "expired"
)

// Default configuration values
const (
	DefaultMaxQueueSize    = 100000
	DefaultBatchSize       = 100
	DefaultMaxRetries      = 3
	DefaultBaseRetryDelay  = 1 * time.Second
	DefaultMaxRetryDelay   = 5 * time.Minute
	DefaultMessageTTL      = 24 * time.Hour
	DefaultProcessInterval = 100 * time.Millisecond
)

// SMSMessage represents a message in the queue
type SMSMessage struct {
	ID            string            `json:"id"`
	PhoneNumber   string            `json:"phone_number"`
	Content       string            `json:"content"`
	Priority      int               `json:"priority"`
	Status        MessageStatus     `json:"status"`
	ScheduledTime time.Time         `json:"scheduled_time"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	SentAt        *time.Time        `json:"sent_at,omitempty"`
	RetryCount    int               `json:"retry_count"`
	NextRetryAt   *time.Time        `json:"next_retry_at,omitempty"`
	ExpiresAt     time.Time         `json:"expires_at"`
	LastError     string            `json:"last_error,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	TemplateID    string            `json:"template_id,omitempty"`
	RecipientID   string            `json:"recipient_id,omitempty"`

	// Internal tracking
	index int // Index in the priority queue heap
}

// RetryPolicy defines the retry behavior
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	BaseDelay       time.Duration `json:"base_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	Multiplier      float64       `json:"multiplier"`
	JitterFactor    float64       `json:"jitter_factor"`
	RetryableErrors []string      `json:"retryable_errors,omitempty"`
}

// DefaultRetryPolicy returns the default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:   DefaultMaxRetries,
		BaseDelay:    DefaultBaseRetryDelay,
		MaxDelay:     DefaultMaxRetryDelay,
		Multiplier:   2.0,
		JitterFactor: 0.2,
	}
}

// CalculateNextRetry calculates the next retry time
func (p *RetryPolicy) CalculateNextRetry(retryCount int) time.Duration {
	if retryCount >= p.MaxRetries {
		return 0
	}

	delay := float64(p.BaseDelay) * math.Pow(p.Multiplier, float64(retryCount))
	if delay > float64(p.MaxDelay) {
		delay = float64(p.MaxDelay)
	}

	// Add jitter
	jitter := delay * p.JitterFactor * (rand.Float64()*2 - 1)
	delay += jitter

	return time.Duration(delay)
}

// QueueConfig holds the queue configuration
type QueueConfig struct {
	MaxQueueSize    int           `json:"max_queue_size"`
	BatchSize       int           `json:"batch_size"`
	ProcessInterval time.Duration `json:"process_interval"`
	MessageTTL      time.Duration `json:"message_ttl"`
	RetryPolicy     *RetryPolicy  `json:"retry_policy"`
	WorkerCount     int           `json:"worker_count"`
	EnableMetrics   bool          `json:"enable_metrics"`
}

// DefaultQueueConfig returns the default configuration
func DefaultQueueConfig() *QueueConfig {
	return &QueueConfig{
		MaxQueueSize:    DefaultMaxQueueSize,
		BatchSize:       DefaultBatchSize,
		ProcessInterval: DefaultProcessInterval,
		MessageTTL:      DefaultMessageTTL,
		RetryPolicy:     DefaultRetryPolicy(),
		WorkerCount:     4,
		EnableMetrics:   true,
	}
}

// SMSSender is the interface for sending SMS messages
type SMSSender interface {
	Send(ctx context.Context, phoneNumber, content string) error
	SendBatch(ctx context.Context, messages []*SMSMessage) []SendResult
}

// SendResult holds the result of sending a message
type SendResult struct {
	MessageID string
	Success   bool
	Error     error
	SentAt    time.Time
}

// MessagePriorityQueue implements heap.Interface for priority queue
type MessagePriorityQueue []*SMSMessage

func (pq MessagePriorityQueue) Len() int { return len(pq) }

func (pq MessagePriorityQueue) Less(i, j int) bool {
	// First compare by scheduled time if both are scheduled
	if !pq[i].ScheduledTime.IsZero() && !pq[j].ScheduledTime.IsZero() {
		if !pq[i].ScheduledTime.Equal(pq[j].ScheduledTime) {
			return pq[i].ScheduledTime.Before(pq[j].ScheduledTime)
		}
	}
	// Then by priority (lower number = higher priority)
	if pq[i].Priority != pq[j].Priority {
		return pq[i].Priority < pq[j].Priority
	}
	// Finally by creation time (FIFO within same priority)
	return pq[i].CreatedAt.Before(pq[j].CreatedAt)
}

func (pq MessagePriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *MessagePriorityQueue) Push(x interface{}) {
	n := len(*pq)
	msg := x.(*SMSMessage)
	msg.index = n
	*pq = append(*pq, msg)
}

func (pq *MessagePriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	msg := old[n-1]
	old[n-1] = nil // avoid memory leak
	msg.index = -1
	*pq = old[0 : n-1]
	return msg
}

// SMSQueue manages the SMS message queue
type SMSQueue struct {
	config *QueueConfig
	sender SMSSender

	// Priority queue
	pq   MessagePriorityQueue
	pqMu sync.RWMutex

	// Message lookup
	messages   map[string]*SMSMessage
	messagesMu sync.RWMutex

	// Retry queue
	retryQueue   []*SMSMessage
	retryQueueMu sync.Mutex

	// State
	running int32
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	// Metrics
	metrics *QueueMetrics

	// Callbacks
	onMessageSent   func(*SMSMessage)
	onMessageFailed func(*SMSMessage, error)
	onBatchComplete func(sent, failed int)
}

// QueueMetrics tracks queue performance
type QueueMetrics struct {
	mu sync.RWMutex

	MessagesEnqueued    int64
	MessagesProcessed   int64
	MessagesSent        int64
	MessagesFailed      int64
	MessagesRetried     int64
	MessagesExpired     int64
	MessagesCancelled   int64
	BatchesProcessed    int64
	TotalProcessingTime time.Duration
	CurrentQueueSize    int64
	PeakQueueSize       int64
}

// NewSMSQueue creates a new SMS queue
func NewSMSQueue(config *QueueConfig, sender SMSSender) *SMSQueue {
	if config == nil {
		config = DefaultQueueConfig()
	}
	if config.RetryPolicy == nil {
		config.RetryPolicy = DefaultRetryPolicy()
	}

	ctx, cancel := context.WithCancel(context.Background())

	q := &SMSQueue{
		config:     config,
		sender:     sender,
		pq:         make(MessagePriorityQueue, 0),
		messages:   make(map[string]*SMSMessage),
		retryQueue: make([]*SMSMessage, 0),
		ctx:        ctx,
		cancel:     cancel,
		metrics:    &QueueMetrics{},
	}

	heap.Init(&q.pq)

	return q
}

// Start begins processing the queue
func (q *SMSQueue) Start() error {
	if !atomic.CompareAndSwapInt32(&q.running, 0, 1) {
		return nil // Already running
	}

	// Start worker goroutines
	for i := 0; i < q.config.WorkerCount; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}

	// Start retry processor
	q.wg.Add(1)
	go q.retryProcessor()

	// Start cleanup routine
	q.wg.Add(1)
	go q.cleanupRoutine()

	return nil
}

// Stop gracefully stops the queue
func (q *SMSQueue) Stop() error {
	if !atomic.CompareAndSwapInt32(&q.running, 1, 0) {
		return nil // Not running
	}

	q.cancel()
	q.wg.Wait()

	return nil
}

// IsRunning returns whether the queue is running
func (q *SMSQueue) IsRunning() bool {
	return atomic.LoadInt32(&q.running) == 1
}

// EnqueueMessage adds a message to the queue
func (q *SMSQueue) EnqueueMessage(ctx context.Context, msg *SMSMessage) error {
	if err := q.validateMessage(msg); err != nil {
		return err
	}

	q.messagesMu.Lock()
	currentSize := len(q.messages)
	q.messagesMu.Unlock()

	if currentSize >= q.config.MaxQueueSize {
		return ErrQueueFull
	}

	// Set defaults
	now := time.Now()
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = now
	}
	msg.UpdatedAt = now
	msg.Status = StatusPending

	if msg.ScheduledTime.IsZero() {
		msg.ScheduledTime = now
	} else if msg.ScheduledTime.Before(now) {
		return ErrScheduleInPast
	} else {
		msg.Status = StatusScheduled
	}

	if msg.ExpiresAt.IsZero() {
		msg.ExpiresAt = now.Add(q.config.MessageTTL)
	}

	// Store message and get current queue size
	q.messagesMu.Lock()
	q.messages[msg.ID] = msg
	queueSize := int64(len(q.messages))
	q.messagesMu.Unlock()

	// Add to priority queue
	q.pqMu.Lock()
	heap.Push(&q.pq, msg)
	q.pqMu.Unlock()

	// Update metrics
	q.metrics.mu.Lock()
	q.metrics.MessagesEnqueued++
	q.metrics.CurrentQueueSize = queueSize
	if q.metrics.CurrentQueueSize > q.metrics.PeakQueueSize {
		q.metrics.PeakQueueSize = q.metrics.CurrentQueueSize
	}
	q.metrics.mu.Unlock()

	return nil
}

// EnqueueBatch adds multiple messages to the queue
func (q *SMSQueue) EnqueueBatch(ctx context.Context, messages []*SMSMessage) ([]string, []error) {
	ids := make([]string, len(messages))
	errs := make([]error, len(messages))

	for i, msg := range messages {
		err := q.EnqueueMessage(ctx, msg)
		if err != nil {
			errs[i] = err
		} else {
			ids[i] = msg.ID
		}
	}

	return ids, errs
}

// ScheduleMessage schedules a message for future delivery
func (q *SMSQueue) ScheduleMessage(ctx context.Context, msg *SMSMessage, scheduledTime time.Time) error {
	if scheduledTime.Before(time.Now()) {
		return ErrScheduleInPast
	}

	msg.ScheduledTime = scheduledTime
	msg.Status = StatusScheduled

	return q.EnqueueMessage(ctx, msg)
}

// CancelMessage cancels a pending message
func (q *SMSQueue) CancelMessage(ctx context.Context, messageID string) error {
	q.messagesMu.Lock()
	msg, exists := q.messages[messageID]
	if !exists {
		q.messagesMu.Unlock()
		return ErrMessageNotFound
	}

	if msg.Status == StatusSent || msg.Status == StatusProcessing {
		q.messagesMu.Unlock()
		return fmt.Errorf("cannot cancel message in status: %s", msg.Status)
	}

	msg.Status = StatusCancelled
	msg.UpdatedAt = time.Now()
	q.messagesMu.Unlock()

	q.metrics.mu.Lock()
	q.metrics.MessagesCancelled++
	q.metrics.mu.Unlock()

	return nil
}

// GetMessage retrieves a message by ID
func (q *SMSQueue) GetMessage(ctx context.Context, messageID string) (*SMSMessage, error) {
	q.messagesMu.RLock()
	defer q.messagesMu.RUnlock()

	msg, exists := q.messages[messageID]
	if !exists {
		return nil, ErrMessageNotFound
	}

	// Return a copy
	copy := *msg
	return &copy, nil
}

// GetQueueSize returns the current queue size
func (q *SMSQueue) GetQueueSize() int {
	q.messagesMu.RLock()
	defer q.messagesMu.RUnlock()
	return len(q.messages)
}

// GetPendingCount returns the count of pending messages
func (q *SMSQueue) GetPendingCount() int {
	q.pqMu.RLock()
	defer q.pqMu.RUnlock()
	return q.pq.Len()
}

// ProcessQueue processes messages in the queue
func (q *SMSQueue) ProcessQueue(ctx context.Context) error {
	batch := q.getNextBatch(q.config.BatchSize)
	if len(batch) == 0 {
		return nil
	}

	return q.processBatch(ctx, batch)
}

// getNextBatch retrieves the next batch of messages ready for processing
func (q *SMSQueue) getNextBatch(maxSize int) []*SMSMessage {
	q.pqMu.Lock()
	defer q.pqMu.Unlock()

	now := time.Now()
	batch := make([]*SMSMessage, 0, maxSize)

	for q.pq.Len() > 0 && len(batch) < maxSize {
		// Peek at the next message
		msg := q.pq[0]

		// Skip cancelled or expired messages
		if msg.Status == StatusCancelled || msg.Status == StatusExpired {
			heap.Pop(&q.pq)
			continue
		}

		// Check if scheduled time has arrived
		if msg.ScheduledTime.After(now) {
			break // No more messages ready
		}

		// Check expiration
		if msg.ExpiresAt.Before(now) {
			msg.Status = StatusExpired
			heap.Pop(&q.pq)
			q.metrics.mu.Lock()
			q.metrics.MessagesExpired++
			q.metrics.mu.Unlock()
			continue
		}

		// Pop and add to batch
		heap.Pop(&q.pq)
		msg.Status = StatusProcessing
		msg.UpdatedAt = now
		batch = append(batch, msg)
	}

	return batch
}

// processBatch processes a batch of messages
func (q *SMSQueue) processBatch(ctx context.Context, batch []*SMSMessage) error {
	if len(batch) == 0 {
		return ErrBatchEmpty
	}

	start := time.Now()

	// Use batch send if available, otherwise send individually
	var results []SendResult
	if q.sender != nil {
		results = q.sender.SendBatch(ctx, batch)
	} else {
		results = make([]SendResult, len(batch))
		for i, msg := range batch {
			results[i] = SendResult{
				MessageID: msg.ID,
				Success:   true,
				SentAt:    time.Now(),
			}
		}
	}

	// Process results
	sentCount := 0
	failedCount := 0

	for i, result := range results {
		msg := batch[i]
		q.handleSendResult(msg, result)

		if result.Success {
			sentCount++
		} else {
			failedCount++
		}
	}

	// Update metrics
	elapsed := time.Since(start)
	q.metrics.mu.Lock()
	q.metrics.BatchesProcessed++
	q.metrics.MessagesProcessed += int64(len(batch))
	q.metrics.TotalProcessingTime += elapsed
	q.metrics.mu.Unlock()

	// Callback
	if q.onBatchComplete != nil {
		q.onBatchComplete(sentCount, failedCount)
	}

	return nil
}

// handleSendResult processes the result of sending a message
func (q *SMSQueue) handleSendResult(msg *SMSMessage, result SendResult) {
	q.messagesMu.Lock()
	defer q.messagesMu.Unlock()

	now := time.Now()

	if result.Success {
		msg.Status = StatusSent
		msg.SentAt = &result.SentAt
		msg.UpdatedAt = now

		q.metrics.mu.Lock()
		q.metrics.MessagesSent++
		q.metrics.CurrentQueueSize--
		q.metrics.mu.Unlock()

		if q.onMessageSent != nil {
			q.onMessageSent(msg)
		}
	} else {
		msg.LastError = result.Error.Error()
		msg.UpdatedAt = now

		// Check if should retry
		if q.shouldRetry(msg, result.Error) {
			q.scheduleRetry(msg)
		} else {
			msg.Status = StatusFailed

			q.metrics.mu.Lock()
			q.metrics.MessagesFailed++
			q.metrics.CurrentQueueSize--
			q.metrics.mu.Unlock()

			if q.onMessageFailed != nil {
				q.onMessageFailed(msg, result.Error)
			}
		}
	}
}

// shouldRetry determines if a message should be retried
func (q *SMSQueue) shouldRetry(msg *SMSMessage, err error) bool {
	if msg.RetryCount >= q.config.RetryPolicy.MaxRetries {
		return false
	}

	// Check if error is retryable
	if len(q.config.RetryPolicy.RetryableErrors) > 0 {
		errStr := err.Error()
		for _, retryable := range q.config.RetryPolicy.RetryableErrors {
			if errStr == retryable {
				return true
			}
		}
		return false
	}

	// Default: retry all errors
	return true
}

// scheduleRetry schedules a message for retry
func (q *SMSQueue) scheduleRetry(msg *SMSMessage) {
	msg.RetryCount++
	msg.Status = StatusRetrying

	delay := q.config.RetryPolicy.CalculateNextRetry(msg.RetryCount - 1)
	nextRetry := time.Now().Add(delay)
	msg.NextRetryAt = &nextRetry

	q.retryQueueMu.Lock()
	q.retryQueue = append(q.retryQueue, msg)
	q.retryQueueMu.Unlock()

	q.metrics.mu.Lock()
	q.metrics.MessagesRetried++
	q.metrics.mu.Unlock()
}

// HandleRetries processes messages ready for retry
func (q *SMSQueue) HandleRetries(ctx context.Context) error {
	q.retryQueueMu.Lock()

	now := time.Now()
	readyForRetry := make([]*SMSMessage, 0)
	stillWaiting := make([]*SMSMessage, 0)

	for _, msg := range q.retryQueue {
		if msg.NextRetryAt != nil && msg.NextRetryAt.Before(now) {
			readyForRetry = append(readyForRetry, msg)
		} else {
			stillWaiting = append(stillWaiting, msg)
		}
	}

	q.retryQueue = stillWaiting
	q.retryQueueMu.Unlock()

	// Re-queue messages for retry
	for _, msg := range readyForRetry {
		msg.Status = StatusPending
		msg.ScheduledTime = now

		q.pqMu.Lock()
		heap.Push(&q.pq, msg)
		q.pqMu.Unlock()
	}

	return nil
}

// BatchProcess processes messages in batches with the given batch size
func (q *SMSQueue) BatchProcess(ctx context.Context, batchSize int) (int, error) {
	if batchSize <= 0 {
		batchSize = q.config.BatchSize
	}

	batch := q.getNextBatch(batchSize)
	if len(batch) == 0 {
		return 0, nil
	}

	err := q.processBatch(ctx, batch)
	return len(batch), err
}

// worker is the main processing loop for a worker goroutine
func (q *SMSQueue) worker(id int) {
	defer q.wg.Done()

	ticker := time.NewTicker(q.config.ProcessInterval)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
			// Get and process a single message or small batch
			batch := q.getNextBatch(q.config.BatchSize / q.config.WorkerCount)
			if len(batch) > 0 {
				q.processBatch(q.ctx, batch)
			}
		}
	}
}

// retryProcessor handles retry queue processing
func (q *SMSQueue) retryProcessor() {
	defer q.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
			q.HandleRetries(q.ctx)
		}
	}
}

// cleanupRoutine cleans up expired and processed messages
func (q *SMSQueue) cleanupRoutine() {
	defer q.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
			q.cleanup()
		}
	}
}

// cleanup removes old processed messages
func (q *SMSQueue) cleanup() {
	q.messagesMu.Lock()
	defer q.messagesMu.Unlock()

	cutoff := time.Now().Add(-time.Hour)

	for id, msg := range q.messages {
		// Remove sent/failed/cancelled messages older than cutoff
		if (msg.Status == StatusSent || msg.Status == StatusFailed ||
			msg.Status == StatusCancelled || msg.Status == StatusExpired) &&
			msg.UpdatedAt.Before(cutoff) {
			delete(q.messages, id)
		}
	}

	// Update current size metric
	q.metrics.mu.Lock()
	q.metrics.CurrentQueueSize = int64(len(q.messages))
	q.metrics.mu.Unlock()
}

// GetMetrics returns the current queue metrics
func (q *SMSQueue) GetMetrics() *QueueMetrics {
	q.metrics.mu.RLock()
	defer q.metrics.mu.RUnlock()

	return &QueueMetrics{
		MessagesEnqueued:    q.metrics.MessagesEnqueued,
		MessagesProcessed:   q.metrics.MessagesProcessed,
		MessagesSent:        q.metrics.MessagesSent,
		MessagesFailed:      q.metrics.MessagesFailed,
		MessagesRetried:     q.metrics.MessagesRetried,
		MessagesExpired:     q.metrics.MessagesExpired,
		MessagesCancelled:   q.metrics.MessagesCancelled,
		BatchesProcessed:    q.metrics.BatchesProcessed,
		TotalProcessingTime: q.metrics.TotalProcessingTime,
		CurrentQueueSize:    q.metrics.CurrentQueueSize,
		PeakQueueSize:       q.metrics.PeakQueueSize,
	}
}

// GetQueueStats returns detailed queue statistics
func (q *SMSQueue) GetQueueStats() *QueueStats {
	q.messagesMu.RLock()
	defer q.messagesMu.RUnlock()

	stats := &QueueStats{
		TotalMessages:  len(q.messages),
		ByStatus:       make(map[MessageStatus]int),
		ByPriority:     make(map[int]int),
		OldestPending:  nil,
		RetryQueueSize: 0,
	}

	var oldestPending *time.Time

	for _, msg := range q.messages {
		stats.ByStatus[msg.Status]++
		stats.ByPriority[msg.Priority]++

		if msg.Status == StatusPending || msg.Status == StatusScheduled {
			if oldestPending == nil || msg.CreatedAt.Before(*oldestPending) {
				t := msg.CreatedAt
				oldestPending = &t
			}
		}
	}

	stats.OldestPending = oldestPending

	q.retryQueueMu.Lock()
	stats.RetryQueueSize = len(q.retryQueue)
	q.retryQueueMu.Unlock()

	return stats
}

// QueueStats holds queue statistics
type QueueStats struct {
	TotalMessages  int                   `json:"total_messages"`
	ByStatus       map[MessageStatus]int `json:"by_status"`
	ByPriority     map[int]int           `json:"by_priority"`
	OldestPending  *time.Time            `json:"oldest_pending,omitempty"`
	RetryQueueSize int                   `json:"retry_queue_size"`
}

// SetCallbacks sets the callback functions
func (q *SMSQueue) SetCallbacks(
	onSent func(*SMSMessage),
	onFailed func(*SMSMessage, error),
	onBatch func(sent, failed int),
) {
	q.onMessageSent = onSent
	q.onMessageFailed = onFailed
	q.onBatchComplete = onBatch
}

// validateMessage validates a message before enqueuing
func (q *SMSQueue) validateMessage(msg *SMSMessage) error {
	if msg == nil {
		return ErrInvalidMessage
	}

	if msg.PhoneNumber == "" {
		return ErrInvalidPhone
	}

	if msg.Content == "" {
		return fmt.Errorf("%w: content cannot be empty", ErrInvalidMessage)
	}

	// Basic phone number validation (E.164 format)
	if len(msg.PhoneNumber) < 10 || len(msg.PhoneNumber) > 15 {
		return ErrInvalidPhone
	}

	return nil
}

// Drain removes and returns all pending messages (for graceful shutdown)
func (q *SMSQueue) Drain() []*SMSMessage {
	q.pqMu.Lock()
	defer q.pqMu.Unlock()

	messages := make([]*SMSMessage, 0, q.pq.Len())
	for q.pq.Len() > 0 {
		msg := heap.Pop(&q.pq).(*SMSMessage)
		messages = append(messages, msg)
	}

	return messages
}

// Clear removes all messages from the queue
func (q *SMSQueue) Clear() {
	q.pqMu.Lock()
	q.pq = make(MessagePriorityQueue, 0)
	heap.Init(&q.pq)
	q.pqMu.Unlock()

	q.messagesMu.Lock()
	q.messages = make(map[string]*SMSMessage)
	q.messagesMu.Unlock()

	q.retryQueueMu.Lock()
	q.retryQueue = make([]*SMSMessage, 0)
	q.retryQueueMu.Unlock()

	q.metrics.mu.Lock()
	q.metrics.CurrentQueueSize = 0
	q.metrics.mu.Unlock()
}

// UpdatePriority updates the priority of a pending message
func (q *SMSQueue) UpdatePriority(messageID string, newPriority int) error {
	q.messagesMu.Lock()
	msg, exists := q.messages[messageID]
	if !exists {
		q.messagesMu.Unlock()
		return ErrMessageNotFound
	}

	if msg.Status != StatusPending && msg.Status != StatusScheduled {
		q.messagesMu.Unlock()
		return fmt.Errorf("cannot update priority for message in status: %s", msg.Status)
	}

	msg.Priority = newPriority
	msg.UpdatedAt = time.Now()
	q.messagesMu.Unlock()

	// Reheapify the priority queue
	q.pqMu.Lock()
	if msg.index >= 0 && msg.index < q.pq.Len() {
		heap.Fix(&q.pq, msg.index)
	}
	q.pqMu.Unlock()

	return nil
}

// GetMessagesByStatus returns all messages with the given status
func (q *SMSQueue) GetMessagesByStatus(status MessageStatus) []*SMSMessage {
	q.messagesMu.RLock()
	defer q.messagesMu.RUnlock()

	var result []*SMSMessage
	for _, msg := range q.messages {
		if msg.Status == status {
			copy := *msg
			result = append(result, &copy)
		}
	}

	return result
}

// GetMessagesByPriority returns all messages with the given priority
func (q *SMSQueue) GetMessagesByPriority(priority int) []*SMSMessage {
	q.messagesMu.RLock()
	defer q.messagesMu.RUnlock()

	var result []*SMSMessage
	for _, msg := range q.messages {
		if msg.Priority == priority {
			copy := *msg
			result = append(result, &copy)
		}
	}

	return result
}
