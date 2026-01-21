package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Standard errors
var (
	ErrQueueClosed        = errors.New("queue is closed")
	ErrQueueFull          = errors.New("queue is full")
	ErrJobNotFound        = errors.New("job not found")
	ErrInvalidJob         = errors.New("invalid job")
	ErrDequeueTimeout     = errors.New("dequeue timeout")
	ErrJobExpired         = errors.New("job has expired")
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
)

// JobStatus represents the status of a queued job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusRetrying   JobStatus = "retrying"
	JobStatusDead       JobStatus = "dead"
)

// JobPriority represents job priority levels
type JobPriority int

const (
	PriorityLow      JobPriority = 1
	PriorityNormal   JobPriority = 5
	PriorityHigh     JobPriority = 10
	PriorityCritical JobPriority = 100
)

// EmailJob represents an email job in the queue
type EmailJob struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Priority JobPriority            `json:"priority"`
	Status   JobStatus              `json:"status"`
	Payload  *EmailPayload          `json:"payload"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Timing
	CreatedAt   time.Time  `json:"created_at"`
	ScheduledAt time.Time  `json:"scheduled_at,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`

	// Retry tracking
	Attempts    int        `json:"attempts"`
	MaxRetries  int        `json:"max_retries"`
	LastError   string     `json:"last_error,omitempty"`
	NextRetryAt *time.Time `json:"next_retry_at,omitempty"`

	// Processing
	WorkerID    string   `json:"worker_id,omitempty"`
	ProcessedBy []string `json:"processed_by,omitempty"`
}

// EmailPayload contains the email content and recipients
type EmailPayload struct {
	From        string            `json:"from"`
	To          []string          `json:"to"`
	CC          []string          `json:"cc,omitempty"`
	BCC         []string          `json:"bcc,omitempty"`
	ReplyTo     string            `json:"reply_to,omitempty"`
	Subject     string            `json:"subject"`
	HTMLBody    string            `json:"html_body"`
	TextBody    string            `json:"text_body,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Attachments []Attachment      `json:"attachments,omitempty"`

	// Template info
	TemplateID   string                 `json:"template_id,omitempty"`
	TemplateData map[string]interface{} `json:"template_data,omitempty"`

	// Tracking
	TrackOpens  bool     `json:"track_opens"`
	TrackClicks bool     `json:"track_clicks"`
	Tags        []string `json:"tags,omitempty"`
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Content     []byte `json:"content"`
	ContentID   string `json:"content_id,omitempty"` // For inline attachments
}

// QueueConfig holds queue configuration
type QueueConfig struct {
	// Redis connection
	RedisAddr     string `json:"redis_addr"`
	RedisPassword string `json:"redis_password"`
	RedisDB       int    `json:"redis_db"`

	// Queue settings
	QueueName string `json:"queue_name"`
	MaxSize   int64  `json:"max_size"`

	// Processing
	VisibilityTimeout time.Duration `json:"visibility_timeout"`
	ProcessingTimeout time.Duration `json:"processing_timeout"`

	// Retry settings
	MaxRetries    int           `json:"max_retries"`
	RetryDelay    time.Duration `json:"retry_delay"`
	MaxRetryDelay time.Duration `json:"max_retry_delay"`

	// Rate limiting
	RateLimit  float64 `json:"rate_limit"` // Jobs per second
	BurstLimit int     `json:"burst_limit"`

	// Job TTL
	JobTTL       time.Duration `json:"job_ttl"`
	CompletedTTL time.Duration `json:"completed_ttl"`
	FailedTTL    time.Duration `json:"failed_ttl"`
}

// DefaultQueueConfig returns default configuration
func DefaultQueueConfig() *QueueConfig {
	return &QueueConfig{
		RedisAddr:         "localhost:6379",
		RedisPassword:     "",
		RedisDB:           0,
		QueueName:         "email_queue",
		MaxSize:           100000,
		VisibilityTimeout: 5 * time.Minute,
		ProcessingTimeout: 30 * time.Second,
		MaxRetries:        3,
		RetryDelay:        time.Minute,
		MaxRetryDelay:     time.Hour,
		RateLimit:         100, // 100 emails per second
		BurstLimit:        200,
		JobTTL:            24 * time.Hour,
		CompletedTTL:      time.Hour,
		FailedTTL:         7 * 24 * time.Hour,
	}
}

// Redis key prefixes
const (
	keyPrefix        = "eq:"
	keyQueue         = keyPrefix + "queue:"
	keyPriorityQueue = keyPrefix + "pqueue:"
	keyProcessing    = keyPrefix + "processing:"
	keyScheduled     = keyPrefix + "scheduled:"
	keyDead          = keyPrefix + "dead:"
	keyJob           = keyPrefix + "job:"
	keyStats         = keyPrefix + "stats:"
	keyRateLimit     = keyPrefix + "ratelimit:"
)

// EmailQueue manages the email job queue
type EmailQueue struct {
	client  *redis.Client
	config  *QueueConfig
	metrics *QueueMetrics

	// State
	closed bool
	mu     sync.RWMutex

	// Callbacks
	onEnqueue  func(*EmailJob)
	onDequeue  func(*EmailJob)
	onComplete func(*EmailJob)
	onFail     func(*EmailJob, error)
}

// NewEmailQueue creates a new email queue
func NewEmailQueue(config *QueueConfig) (*EmailQueue, error) {
	if config == nil {
		config = DefaultQueueConfig()
	}

	client := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	q := &EmailQueue{
		client:  client,
		config:  config,
		metrics: NewQueueMetrics(config.QueueName),
	}

	return q, nil
}

// NewEmailQueueWithClient creates a queue with an existing Redis client
func NewEmailQueueWithClient(client *redis.Client, config *QueueConfig) *EmailQueue {
	if config == nil {
		config = DefaultQueueConfig()
	}

	return &EmailQueue{
		client:  client,
		config:  config,
		metrics: NewQueueMetrics(config.QueueName),
	}
}

// SetCallbacks sets queue event callbacks
func (q *EmailQueue) SetCallbacks(onEnqueue, onDequeue, onComplete func(*EmailJob), onFail func(*EmailJob, error)) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.onEnqueue = onEnqueue
	q.onDequeue = onDequeue
	q.onComplete = onComplete
	q.onFail = onFail
}

// Enqueue adds a job to the queue
func (q *EmailQueue) Enqueue(ctx context.Context, job *EmailJob) error {
	q.mu.RLock()
	if q.closed {
		q.mu.RUnlock()
		return ErrQueueClosed
	}
	q.mu.RUnlock()

	if err := q.validateJob(job); err != nil {
		return err
	}

	// Set defaults
	if job.ID == "" {
		job.ID = generateJobID()
	}
	if job.Status == "" {
		job.Status = JobStatusPending
	}
	if job.Priority == 0 {
		job.Priority = PriorityNormal
	}
	if job.MaxRetries == 0 {
		job.MaxRetries = q.config.MaxRetries
	}
	job.CreatedAt = time.Now()

	// Set expiration
	if job.ExpiresAt == nil && q.config.JobTTL > 0 {
		expiry := time.Now().Add(q.config.JobTTL)
		job.ExpiresAt = &expiry
	}

	// Check queue size
	size, err := q.Size(ctx)
	if err != nil {
		return fmt.Errorf("failed to check queue size: %w", err)
	}
	if size >= q.config.MaxSize {
		q.metrics.RecordEnqueueRejected("queue_full")
		return ErrQueueFull
	}

	// Serialize job
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Use pipeline for atomic operation
	pipe := q.client.Pipeline()

	// Store job data
	jobKey := keyJob + job.ID
	pipe.Set(ctx, jobKey, data, q.config.JobTTL)

	// Add to appropriate queue
	if !job.ScheduledAt.IsZero() && job.ScheduledAt.After(time.Now()) {
		// Scheduled job - add to sorted set with score as timestamp
		pipe.ZAdd(ctx, keyScheduled+q.config.QueueName, redis.Z{
			Score:  float64(job.ScheduledAt.Unix()),
			Member: job.ID,
		})
	} else {
		// Immediate job - add to priority queue
		pipe.ZAdd(ctx, keyPriorityQueue+q.config.QueueName, redis.Z{
			Score:  float64(job.Priority),
			Member: job.ID,
		})
	}

	// Update stats
	pipe.Incr(ctx, keyStats+q.config.QueueName+":enqueued")
	pipe.Incr(ctx, keyStats+q.config.QueueName+":total")

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	q.metrics.RecordEnqueue(fmt.Sprintf("%d", job.Priority))

	if q.onEnqueue != nil {
		q.onEnqueue(job)
	}

	return nil
}

// EnqueueBatch adds multiple jobs to the queue
func (q *EmailQueue) EnqueueBatch(ctx context.Context, jobs []*EmailJob) ([]error, error) {
	q.mu.RLock()
	if q.closed {
		q.mu.RUnlock()
		return nil, ErrQueueClosed
	}
	q.mu.RUnlock()

	errs := make([]error, len(jobs))
	pipe := q.client.Pipeline()

	for i, job := range jobs {
		if err := q.validateJob(job); err != nil {
			errs[i] = err
			continue
		}

		// Set defaults
		if job.ID == "" {
			job.ID = generateJobID()
		}
		if job.Status == "" {
			job.Status = JobStatusPending
		}
		if job.Priority == 0 {
			job.Priority = PriorityNormal
		}
		if job.MaxRetries == 0 {
			job.MaxRetries = q.config.MaxRetries
		}
		job.CreatedAt = time.Now()

		if job.ExpiresAt == nil && q.config.JobTTL > 0 {
			expiry := time.Now().Add(q.config.JobTTL)
			job.ExpiresAt = &expiry
		}

		data, err := json.Marshal(job)
		if err != nil {
			errs[i] = fmt.Errorf("failed to marshal job: %w", err)
			continue
		}

		jobKey := keyJob + job.ID
		pipe.Set(ctx, jobKey, data, q.config.JobTTL)

		if !job.ScheduledAt.IsZero() && job.ScheduledAt.After(time.Now()) {
			pipe.ZAdd(ctx, keyScheduled+q.config.QueueName, redis.Z{
				Score:  float64(job.ScheduledAt.Unix()),
				Member: job.ID,
			})
		} else {
			pipe.ZAdd(ctx, keyPriorityQueue+q.config.QueueName, redis.Z{
				Score:  float64(job.Priority),
				Member: job.ID,
			})
		}
	}

	pipe.IncrBy(ctx, keyStats+q.config.QueueName+":enqueued", int64(len(jobs)))
	pipe.IncrBy(ctx, keyStats+q.config.QueueName+":total", int64(len(jobs)))

	_, err := pipe.Exec(ctx)
	if err != nil {
		return errs, fmt.Errorf("failed to execute batch enqueue: %w", err)
	}

	return errs, nil
}

// Dequeue retrieves and locks a job for processing
func (q *EmailQueue) Dequeue(ctx context.Context, workerID string) (*EmailJob, error) {
	q.mu.RLock()
	if q.closed {
		q.mu.RUnlock()
		return nil, ErrQueueClosed
	}
	q.mu.RUnlock()

	// First, move any ready scheduled jobs to the main queue
	if err := q.processScheduledJobs(ctx); err != nil {
		// Log but don't fail
	}

	// Get highest priority job (ZPOPMAX)
	results, err := q.client.ZPopMax(ctx, keyPriorityQueue+q.config.QueueName, 1).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // No jobs available
		}
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	jobID := results[0].Member.(string)

	// Get job data
	job, err := q.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	// Check if expired
	if job.ExpiresAt != nil && time.Now().After(*job.ExpiresAt) {
		q.moveToDeadQueue(ctx, job, "expired")
		q.metrics.RecordJobExpired()
		return nil, ErrJobExpired
	}

	// Update job status
	now := time.Now()
	job.Status = JobStatusProcessing
	job.StartedAt = &now
	job.WorkerID = workerID
	job.Attempts++
	job.ProcessedBy = append(job.ProcessedBy, workerID)

	// Save updated job and add to processing set
	data, _ := json.Marshal(job)
	pipe := q.client.Pipeline()
	pipe.Set(ctx, keyJob+job.ID, data, q.config.JobTTL)
	pipe.ZAdd(ctx, keyProcessing+q.config.QueueName, redis.Z{
		Score:  float64(time.Now().Add(q.config.VisibilityTimeout).Unix()),
		Member: job.ID,
	})
	pipe.Incr(ctx, keyStats+q.config.QueueName+":dequeued")

	_, err = pipe.Exec(ctx)
	if err != nil {
		// Put job back in queue
		q.client.ZAdd(ctx, keyPriorityQueue+q.config.QueueName, redis.Z{
			Score:  float64(job.Priority),
			Member: job.ID,
		})
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}

	q.metrics.RecordDequeue()

	if q.onDequeue != nil {
		q.onDequeue(job)
	}

	return job, nil
}

// Complete marks a job as completed
func (q *EmailQueue) Complete(ctx context.Context, jobID string) error {
	job, err := q.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	now := time.Now()
	job.Status = JobStatusCompleted
	job.CompletedAt = &now
	job.WorkerID = ""

	data, _ := json.Marshal(job)
	pipe := q.client.Pipeline()

	// Update job with shorter TTL
	pipe.Set(ctx, keyJob+job.ID, data, q.config.CompletedTTL)

	// Remove from processing queue
	pipe.ZRem(ctx, keyProcessing+q.config.QueueName, job.ID)

	// Update stats
	pipe.Incr(ctx, keyStats+q.config.QueueName+":completed")

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	q.metrics.RecordComplete(time.Since(*job.StartedAt))

	if q.onComplete != nil {
		q.onComplete(job)
	}

	return nil
}

// Fail marks a job as failed
func (q *EmailQueue) Fail(ctx context.Context, jobID string, jobErr error) error {
	job, err := q.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	job.LastError = jobErr.Error()
	job.WorkerID = ""

	// Remove from processing queue
	q.client.ZRem(ctx, keyProcessing+q.config.QueueName, job.ID)

	// Check if should retry
	if job.Attempts < job.MaxRetries {
		return q.scheduleRetry(ctx, job)
	}

	// Move to dead queue
	job.Status = JobStatusDead
	return q.moveToDeadQueue(ctx, job, "max_retries_exceeded")
}

// GetJob retrieves a job by ID
func (q *EmailQueue) GetJob(ctx context.Context, jobID string) (*EmailJob, error) {
	data, err := q.client.Get(ctx, keyJob+jobID).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	var job EmailJob
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// Size returns the number of pending jobs
func (q *EmailQueue) Size(ctx context.Context) (int64, error) {
	return q.client.ZCard(ctx, keyPriorityQueue+q.config.QueueName).Result()
}

// ScheduledSize returns the number of scheduled jobs
func (q *EmailQueue) ScheduledSize(ctx context.Context) (int64, error) {
	return q.client.ZCard(ctx, keyScheduled+q.config.QueueName).Result()
}

// ProcessingSize returns the number of jobs being processed
func (q *EmailQueue) ProcessingSize(ctx context.Context) (int64, error) {
	return q.client.ZCard(ctx, keyProcessing+q.config.QueueName).Result()
}

// DeadSize returns the number of dead jobs
func (q *EmailQueue) DeadSize(ctx context.Context) (int64, error) {
	return q.client.ZCard(ctx, keyDead+q.config.QueueName).Result()
}

// Purge removes all jobs from the queue
func (q *EmailQueue) Purge(ctx context.Context) error {
	pipe := q.client.Pipeline()
	pipe.Del(ctx, keyPriorityQueue+q.config.QueueName)
	pipe.Del(ctx, keyScheduled+q.config.QueueName)
	pipe.Del(ctx, keyProcessing+q.config.QueueName)
	pipe.Del(ctx, keyDead+q.config.QueueName)

	_, err := pipe.Exec(ctx)
	return err
}

// Close closes the queue
func (q *EmailQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil
	}

	q.closed = true
	return q.client.Close()
}

// Metrics returns the queue metrics
func (q *EmailQueue) Metrics() *QueueMetrics {
	return q.metrics
}

// validateJob validates a job before enqueueing
func (q *EmailQueue) validateJob(job *EmailJob) error {
	if job == nil {
		return ErrInvalidJob
	}

	if job.Payload == nil {
		return fmt.Errorf("%w: payload is required", ErrInvalidJob)
	}

	if len(job.Payload.To) == 0 {
		return fmt.Errorf("%w: at least one recipient is required", ErrInvalidJob)
	}

	if job.Payload.Subject == "" {
		return fmt.Errorf("%w: subject is required", ErrInvalidJob)
	}

	if job.Payload.HTMLBody == "" && job.Payload.TextBody == "" && job.Payload.TemplateID == "" {
		return fmt.Errorf("%w: body or template is required", ErrInvalidJob)
	}

	return nil
}

// processScheduledJobs moves ready scheduled jobs to the main queue
func (q *EmailQueue) processScheduledJobs(ctx context.Context) error {
	now := float64(time.Now().Unix())

	// Get all scheduled jobs that are ready
	jobIDs, err := q.client.ZRangeByScore(ctx, keyScheduled+q.config.QueueName, &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%f", now),
	}).Result()
	if err != nil {
		return err
	}

	if len(jobIDs) == 0 {
		return nil
	}

	pipe := q.client.Pipeline()

	for _, jobID := range jobIDs {
		// Get job to determine priority
		job, err := q.GetJob(ctx, jobID)
		if err != nil {
			continue
		}

		// Move to priority queue
		pipe.ZRem(ctx, keyScheduled+q.config.QueueName, jobID)
		pipe.ZAdd(ctx, keyPriorityQueue+q.config.QueueName, redis.Z{
			Score:  float64(job.Priority),
			Member: jobID,
		})
	}

	_, err = pipe.Exec(ctx)
	return err
}

// moveToDeadQueue moves a job to the dead letter queue
func (q *EmailQueue) moveToDeadQueue(ctx context.Context, job *EmailJob, reason string) error {
	job.Status = JobStatusDead
	if job.Metadata == nil {
		job.Metadata = make(map[string]interface{})
	}
	job.Metadata["dead_reason"] = reason
	job.Metadata["dead_at"] = time.Now()

	data, _ := json.Marshal(job)
	pipe := q.client.Pipeline()

	pipe.Set(ctx, keyJob+job.ID, data, q.config.FailedTTL)
	pipe.ZAdd(ctx, keyDead+q.config.QueueName, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: job.ID,
	})
	pipe.Incr(ctx, keyStats+q.config.QueueName+":dead")

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to move job to dead queue: %w", err)
	}

	q.metrics.RecordDead(reason)

	if q.onFail != nil {
		q.onFail(job, ErrMaxRetriesExceeded)
	}

	return nil
}

// scheduleRetry schedules a job for retry
func (q *EmailQueue) scheduleRetry(ctx context.Context, job *EmailJob) error {
	job.Status = JobStatusRetrying

	// Calculate next retry time with exponential backoff
	delay := calculateBackoff(job.Attempts, q.config.RetryDelay, q.config.MaxRetryDelay)
	nextRetry := time.Now().Add(delay)
	job.NextRetryAt = &nextRetry

	data, _ := json.Marshal(job)
	pipe := q.client.Pipeline()

	pipe.Set(ctx, keyJob+job.ID, data, q.config.JobTTL)
	pipe.ZAdd(ctx, keyScheduled+q.config.QueueName, redis.Z{
		Score:  float64(nextRetry.Unix()),
		Member: job.ID,
	})
	pipe.Incr(ctx, keyStats+q.config.QueueName+":retries")

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to schedule retry: %w", err)
	}

	q.metrics.RecordRetry(job.Attempts)

	return nil
}

// RecoverStaleJobs moves stale processing jobs back to the queue
func (q *EmailQueue) RecoverStaleJobs(ctx context.Context) (int, error) {
	now := float64(time.Now().Unix())

	// Get all jobs that have exceeded visibility timeout
	jobIDs, err := q.client.ZRangeByScore(ctx, keyProcessing+q.config.QueueName, &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%f", now),
	}).Result()
	if err != nil {
		return 0, err
	}

	if len(jobIDs) == 0 {
		return 0, nil
	}

	recovered := 0
	pipe := q.client.Pipeline()

	for _, jobID := range jobIDs {
		job, err := q.GetJob(ctx, jobID)
		if err != nil {
			continue
		}

		// Reset job status
		job.Status = JobStatusPending
		job.WorkerID = ""
		job.LastError = "recovered from stale processing"

		data, _ := json.Marshal(job)
		pipe.Set(ctx, keyJob+job.ID, data, q.config.JobTTL)
		pipe.ZRem(ctx, keyProcessing+q.config.QueueName, jobID)
		pipe.ZAdd(ctx, keyPriorityQueue+q.config.QueueName, redis.Z{
			Score:  float64(job.Priority),
			Member: jobID,
		})

		recovered++
	}

	if recovered > 0 {
		pipe.IncrBy(ctx, keyStats+q.config.QueueName+":recovered", int64(recovered))
		_, err = pipe.Exec(ctx)
		if err != nil {
			return 0, err
		}

		q.metrics.RecordRecovered(recovered)
	}

	return recovered, nil
}

// GetStats returns queue statistics
func (q *EmailQueue) GetStats(ctx context.Context) (*QueueStats, error) {
	pipe := q.client.Pipeline()

	pending := pipe.ZCard(ctx, keyPriorityQueue+q.config.QueueName)
	scheduled := pipe.ZCard(ctx, keyScheduled+q.config.QueueName)
	processing := pipe.ZCard(ctx, keyProcessing+q.config.QueueName)
	dead := pipe.ZCard(ctx, keyDead+q.config.QueueName)

	enqueued := pipe.Get(ctx, keyStats+q.config.QueueName+":enqueued")
	dequeued := pipe.Get(ctx, keyStats+q.config.QueueName+":dequeued")
	completed := pipe.Get(ctx, keyStats+q.config.QueueName+":completed")
	retries := pipe.Get(ctx, keyStats+q.config.QueueName+":retries")

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	stats := &QueueStats{
		Pending:    pending.Val(),
		Scheduled:  scheduled.Val(),
		Processing: processing.Val(),
		Dead:       dead.Val(),
	}

	if v, err := enqueued.Int64(); err == nil {
		stats.Enqueued = v
	}
	if v, err := dequeued.Int64(); err == nil {
		stats.Dequeued = v
	}
	if v, err := completed.Int64(); err == nil {
		stats.Completed = v
	}
	if v, err := retries.Int64(); err == nil {
		stats.Retries = v
	}

	return stats, nil
}

// QueueStats holds queue statistics
type QueueStats struct {
	Pending    int64 `json:"pending"`
	Scheduled  int64 `json:"scheduled"`
	Processing int64 `json:"processing"`
	Dead       int64 `json:"dead"`
	Enqueued   int64 `json:"enqueued"`
	Dequeued   int64 `json:"dequeued"`
	Completed  int64 `json:"completed"`
	Retries    int64 `json:"retries"`
}

// ListDeadJobs returns dead jobs with pagination
func (q *EmailQueue) ListDeadJobs(ctx context.Context, offset, limit int64) ([]*EmailJob, error) {
	jobIDs, err := q.client.ZRevRange(ctx, keyDead+q.config.QueueName, offset, offset+limit-1).Result()
	if err != nil {
		return nil, err
	}

	jobs := make([]*EmailJob, 0, len(jobIDs))
	for _, id := range jobIDs {
		job, err := q.GetJob(ctx, id)
		if err == nil {
			jobs = append(jobs, job)
		}
	}

	return jobs, nil
}

// RetryDeadJob moves a dead job back to the queue
func (q *EmailQueue) RetryDeadJob(ctx context.Context, jobID string) error {
	job, err := q.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	if job.Status != JobStatusDead {
		return fmt.Errorf("job is not in dead queue")
	}

	// Reset job
	job.Status = JobStatusPending
	job.Attempts = 0
	job.LastError = ""
	job.NextRetryAt = nil
	job.WorkerID = ""

	data, _ := json.Marshal(job)
	pipe := q.client.Pipeline()

	pipe.Set(ctx, keyJob+job.ID, data, q.config.JobTTL)
	pipe.ZRem(ctx, keyDead+q.config.QueueName, jobID)
	pipe.ZAdd(ctx, keyPriorityQueue+q.config.QueueName, redis.Z{
		Score:  float64(job.Priority),
		Member: jobID,
	})

	_, err = pipe.Exec(ctx)
	return err
}

// DeleteDeadJob permanently removes a dead job
func (q *EmailQueue) DeleteDeadJob(ctx context.Context, jobID string) error {
	pipe := q.client.Pipeline()
	pipe.Del(ctx, keyJob+jobID)
	pipe.ZRem(ctx, keyDead+q.config.QueueName, jobID)
	_, err := pipe.Exec(ctx)
	return err
}
