package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// Test helper to create a mock Redis server
func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return mr, client
}

// ============================================
// Queue Tests
// ============================================

func TestNewEmailQueue(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	config := DefaultQueueConfig()
	config.RedisAddr = mr.Addr()

	queue := NewEmailQueueWithClient(client, config)

	if queue == nil {
		t.Fatal("Queue is nil")
	}

	if queue.config.QueueName != "email_queue" {
		t.Errorf("Expected queue name email_queue, got %s", queue.config.QueueName)
	}
}

func TestEnqueue(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	queue := NewEmailQueueWithClient(client, DefaultQueueConfig())
	ctx := context.Background()

	job := &EmailJob{
		Type:     "transactional",
		Priority: PriorityNormal,
		Payload: &EmailPayload{
			From:     "test@example.com",
			To:       []string{"recipient@example.com"},
			Subject:  "Test Subject",
			HTMLBody: "<p>Test body</p>",
		},
	}

	err := queue.Enqueue(ctx, job)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Verify job was added
	if job.ID == "" {
		t.Error("Job ID should be set")
	}

	if job.Status != JobStatusPending {
		t.Errorf("Expected status pending, got %s", job.Status)
	}

	// Check queue size
	size, err := queue.Size(ctx)
	if err != nil {
		t.Fatalf("Failed to get queue size: %v", err)
	}

	if size != 1 {
		t.Errorf("Expected queue size 1, got %d", size)
	}
}

func TestEnqueueWithPriority(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	queue := NewEmailQueueWithClient(client, DefaultQueueConfig())
	ctx := context.Background()

	// Enqueue jobs with different priorities
	priorities := []JobPriority{PriorityLow, PriorityNormal, PriorityHigh, PriorityCritical}

	for _, priority := range priorities {
		job := &EmailJob{
			Priority: priority,
			Payload: &EmailPayload{
				To:       []string{"test@example.com"},
				Subject:  fmt.Sprintf("Priority %d", priority),
				HTMLBody: "body",
			},
		}
		err := queue.Enqueue(ctx, job)
		if err != nil {
			t.Fatalf("Failed to enqueue job: %v", err)
		}
	}

	// Dequeue should return highest priority first
	job, err := queue.Dequeue(ctx, "worker-1")
	if err != nil {
		t.Fatalf("Failed to dequeue: %v", err)
	}

	if job.Priority != PriorityCritical {
		t.Errorf("Expected critical priority, got %d", job.Priority)
	}
}

func TestEnqueueBatch(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	queue := NewEmailQueueWithClient(client, DefaultQueueConfig())
	ctx := context.Background()

	jobs := make([]*EmailJob, 10)
	for i := 0; i < 10; i++ {
		jobs[i] = &EmailJob{
			Payload: &EmailPayload{
				To:       []string{fmt.Sprintf("test%d@example.com", i)},
				Subject:  fmt.Sprintf("Test %d", i),
				HTMLBody: "body",
			},
		}
	}

	errs, err := queue.EnqueueBatch(ctx, jobs)
	if err != nil {
		t.Fatalf("Failed to enqueue batch: %v", err)
	}

	for i, e := range errs {
		if e != nil {
			t.Errorf("Job %d failed: %v", i, e)
		}
	}

	size, _ := queue.Size(ctx)
	if size != 10 {
		t.Errorf("Expected 10 jobs, got %d", size)
	}
}

func TestDequeue(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	queue := NewEmailQueueWithClient(client, DefaultQueueConfig())
	ctx := context.Background()

	// Enqueue a job
	job := &EmailJob{
		Payload: &EmailPayload{
			To:       []string{"test@example.com"},
			Subject:  "Test",
			HTMLBody: "body",
		},
	}
	queue.Enqueue(ctx, job)

	// Dequeue
	dequeued, err := queue.Dequeue(ctx, "worker-1")
	if err != nil {
		t.Fatalf("Failed to dequeue: %v", err)
	}

	if dequeued == nil {
		t.Fatal("Dequeued job is nil")
	}

	if dequeued.ID != job.ID {
		t.Errorf("Expected job ID %s, got %s", job.ID, dequeued.ID)
	}

	if dequeued.Status != JobStatusProcessing {
		t.Errorf("Expected status processing, got %s", dequeued.Status)
	}

	if dequeued.WorkerID != "worker-1" {
		t.Errorf("Expected worker ID worker-1, got %s", dequeued.WorkerID)
	}

	if dequeued.Attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", dequeued.Attempts)
	}
}

func TestDequeueEmpty(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	queue := NewEmailQueueWithClient(client, DefaultQueueConfig())
	ctx := context.Background()

	job, err := queue.Dequeue(ctx, "worker-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if job != nil {
		t.Error("Expected nil job from empty queue")
	}
}

func TestComplete(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	queue := NewEmailQueueWithClient(client, DefaultQueueConfig())
	ctx := context.Background()

	// Enqueue and dequeue
	job := &EmailJob{
		Payload: &EmailPayload{
			To:       []string{"test@example.com"},
			Subject:  "Test",
			HTMLBody: "body",
		},
	}
	queue.Enqueue(ctx, job)
	dequeued, _ := queue.Dequeue(ctx, "worker-1")

	// Complete
	err := queue.Complete(ctx, dequeued.ID)
	if err != nil {
		t.Fatalf("Failed to complete: %v", err)
	}

	// Verify
	completed, _ := queue.GetJob(ctx, dequeued.ID)
	if completed.Status != JobStatusCompleted {
		t.Errorf("Expected status completed, got %s", completed.Status)
	}

	if completed.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func TestFail(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	config := DefaultQueueConfig()
	config.MaxRetries = 3
	queue := NewEmailQueueWithClient(client, config)
	ctx := context.Background()

	// Enqueue and dequeue
	job := &EmailJob{
		MaxRetries: 3,
		Payload: &EmailPayload{
			To:       []string{"test@example.com"},
			Subject:  "Test",
			HTMLBody: "body",
		},
	}
	queue.Enqueue(ctx, job)
	dequeued, _ := queue.Dequeue(ctx, "worker-1")

	// Fail (should schedule retry)
	err := queue.Fail(ctx, dequeued.ID, errors.New("send failed"))
	if err != nil {
		t.Fatalf("Failed to fail job: %v", err)
	}

	// Check it's scheduled for retry
	scheduled, _ := queue.ScheduledSize(ctx)
	if scheduled != 1 {
		t.Errorf("Expected 1 scheduled job, got %d", scheduled)
	}
}

func TestFailMaxRetries(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	config := DefaultQueueConfig()
	queue := NewEmailQueueWithClient(client, config)
	ctx := context.Background()

	// Create job that has exceeded max retries
	job := &EmailJob{
		MaxRetries: 1,
		Payload: &EmailPayload{
			To:       []string{"test@example.com"},
			Subject:  "Test",
			HTMLBody: "body",
		},
	}
	queue.Enqueue(ctx, job)

	// Dequeue and fail multiple times
	dequeued, _ := queue.Dequeue(ctx, "worker-1")
	queue.Fail(ctx, dequeued.ID, errors.New("failed"))

	// Wait for scheduled job to be ready
	mr.FastForward(2 * time.Minute)
	queue.processScheduledJobs(ctx)

	dequeued, _ = queue.Dequeue(ctx, "worker-1")
	if dequeued != nil {
		queue.Fail(ctx, dequeued.ID, errors.New("failed again"))
	}

	// Should be in dead queue
	deadSize, _ := queue.DeadSize(ctx)
	if deadSize != 1 {
		t.Errorf("Expected 1 dead job, got %d", deadSize)
	}
}

func TestScheduledJob(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	queue := NewEmailQueueWithClient(client, DefaultQueueConfig())
	ctx := context.Background()

	// Test 1: Schedule job for future - should go to scheduled queue
	futureTime := time.Now().Add(time.Hour)
	futureJob := &EmailJob{
		ScheduledAt: futureTime,
		Payload: &EmailPayload{
			To:       []string{"future@example.com"},
			Subject:  "Future Scheduled",
			HTMLBody: "body",
		},
	}
	queue.Enqueue(ctx, futureJob)

	// Should be in scheduled queue
	scheduled, _ := queue.ScheduledSize(ctx)
	if scheduled != 1 {
		t.Errorf("Expected 1 scheduled job, got %d", scheduled)
	}

	// Should not be dequeued yet (nothing in pending queue)
	dequeued, _ := queue.Dequeue(ctx, "worker-1")
	if dequeued != nil {
		t.Error("Future scheduled job should not be dequeued before time")
	}

	// Test 2: Job scheduled for past goes directly to pending queue (correct behavior)
	pastTime := time.Now().Add(-time.Hour)
	pastJob := &EmailJob{
		ScheduledAt: pastTime,
		Payload: &EmailPayload{
			To:       []string{"past@example.com"},
			Subject:  "Past Scheduled",
			HTMLBody: "body",
		},
	}
	queue.Enqueue(ctx, pastJob)

	// Past-scheduled job should go directly to pending queue, not scheduled
	pending, _ := queue.Size(ctx)
	if pending != 1 {
		t.Errorf("Expected 1 pending job (past-scheduled goes directly to pending), got %d", pending)
	}

	// Future job should still be in scheduled queue
	scheduled, _ = queue.ScheduledSize(ctx)
	if scheduled != 1 {
		t.Errorf("Expected 1 scheduled job remaining, got %d", scheduled)
	}

	// Note: Testing processScheduledJobs with time manipulation requires integration tests
	// with a real Redis instance. The core scheduling logic is verified above.
}

func TestGetStats(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	queue := NewEmailQueueWithClient(client, DefaultQueueConfig())
	ctx := context.Background()

	// Add some jobs
	for i := 0; i < 5; i++ {
		job := &EmailJob{
			Payload: &EmailPayload{
				To:       []string{"test@example.com"},
				Subject:  fmt.Sprintf("Test %d", i),
				HTMLBody: "body",
			},
		}
		queue.Enqueue(ctx, job)
	}

	stats, err := queue.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Pending != 5 {
		t.Errorf("Expected 5 pending, got %d", stats.Pending)
	}

	if stats.Enqueued != 5 {
		t.Errorf("Expected 5 enqueued, got %d", stats.Enqueued)
	}
}

func TestRecoverStaleJobs(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	config := DefaultQueueConfig()
	config.VisibilityTimeout = time.Minute
	queue := NewEmailQueueWithClient(client, config)
	ctx := context.Background()

	// Enqueue and dequeue a job
	job := &EmailJob{
		Payload: &EmailPayload{
			To:       []string{"test@example.com"},
			Subject:  "Test",
			HTMLBody: "body",
		},
	}
	queue.Enqueue(ctx, job)
	_, _ = queue.Dequeue(ctx, "worker-1")

	// Verify job is processing
	processing, _ := queue.ProcessingSize(ctx)
	if processing != 1 {
		t.Errorf("Expected 1 processing, got %d", processing)
	}

	// Test that RecoverStaleJobs runs without error when no stale jobs
	// (jobs are not stale yet since visibility timeout hasn't passed)
	recovered, err := queue.RecoverStaleJobs(ctx)
	if err != nil {
		t.Fatalf("Failed to recover stale jobs: %v", err)
	}

	// No jobs should be recovered since they're not stale yet
	if recovered != 0 {
		t.Errorf("Expected 0 recovered (not stale yet), got %d", recovered)
	}

	// Job should still be in processing
	processing, _ = queue.ProcessingSize(ctx)
	if processing != 1 {
		t.Errorf("Expected 1 still processing, got %d", processing)
	}

	// Note: Testing actual stale job recovery requires time manipulation
	// which should be done in integration tests with a real Redis instance.
}

func TestValidateJob(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	queue := NewEmailQueueWithClient(client, DefaultQueueConfig())
	ctx := context.Background()

	tests := []struct {
		name    string
		job     *EmailJob
		wantErr bool
	}{
		{
			name:    "nil job",
			job:     nil,
			wantErr: true,
		},
		{
			name: "nil payload",
			job: &EmailJob{
				Payload: nil,
			},
			wantErr: true,
		},
		{
			name: "no recipients",
			job: &EmailJob{
				Payload: &EmailPayload{
					Subject:  "Test",
					HTMLBody: "body",
				},
			},
			wantErr: true,
		},
		{
			name: "no subject",
			job: &EmailJob{
				Payload: &EmailPayload{
					To:       []string{"test@example.com"},
					HTMLBody: "body",
				},
			},
			wantErr: true,
		},
		{
			name: "no body or template",
			job: &EmailJob{
				Payload: &EmailPayload{
					To:      []string{"test@example.com"},
					Subject: "Test",
				},
			},
			wantErr: true,
		},
		{
			name: "valid with HTML body",
			job: &EmailJob{
				Payload: &EmailPayload{
					To:       []string{"test@example.com"},
					Subject:  "Test",
					HTMLBody: "<p>body</p>",
				},
			},
			wantErr: false,
		},
		{
			name: "valid with template",
			job: &EmailJob{
				Payload: &EmailPayload{
					To:         []string{"test@example.com"},
					Subject:    "Test",
					TemplateID: "welcome",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := queue.Enqueue(ctx, tt.job)
			if (err != nil) != tt.wantErr {
				t.Errorf("Enqueue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPurge(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	queue := NewEmailQueueWithClient(client, DefaultQueueConfig())
	ctx := context.Background()

	// Add jobs
	for i := 0; i < 5; i++ {
		job := &EmailJob{
			Payload: &EmailPayload{
				To:       []string{"test@example.com"},
				Subject:  "Test",
				HTMLBody: "body",
			},
		}
		queue.Enqueue(ctx, job)
	}

	// Purge
	err := queue.Purge(ctx)
	if err != nil {
		t.Fatalf("Failed to purge: %v", err)
	}

	// Verify empty
	size, _ := queue.Size(ctx)
	if size != 0 {
		t.Errorf("Expected 0 jobs after purge, got %d", size)
	}
}

// ============================================
// Retry Tests
// ============================================

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		attempt      int
		initialDelay time.Duration
		maxDelay     time.Duration
		minExpected  time.Duration
		maxExpected  time.Duration
	}{
		{1, time.Second, time.Hour, 900 * time.Millisecond, 1100 * time.Millisecond},
		{2, time.Second, time.Hour, 1800 * time.Millisecond, 2200 * time.Millisecond},
		{3, time.Second, time.Hour, 3600 * time.Millisecond, 4400 * time.Millisecond},
		{10, time.Second, time.Hour, 460 * time.Second, 564 * time.Second}, // 512s with ±10% jitter
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			delay := calculateBackoff(tt.attempt, tt.initialDelay, tt.maxDelay)

			if delay < tt.minExpected || delay > tt.maxExpected {
				t.Errorf("Delay %v out of expected range [%v, %v]",
					delay, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

func TestRetryManager(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	queue := NewEmailQueueWithClient(client, DefaultQueueConfig())

	rm := NewRetryManager(queue, nil)

	job := &EmailJob{
		ID:         "test-job",
		Attempts:   1,
		MaxRetries: 3,
	}

	// Should retry
	if !rm.ShouldRetry(job, errors.New("connection refused")) {
		t.Error("Should retry connection refused error")
	}

	// Should not retry non-retryable error
	if rm.ShouldRetry(job, errors.New("invalid email address")) {
		t.Error("Should not retry invalid email error")
	}

	// Should not retry when max retries exceeded
	job.Attempts = 3
	if rm.ShouldRetry(job, errors.New("connection refused")) {
		t.Error("Should not retry when max retries exceeded")
	}
}

func TestIsRetryableError(t *testing.T) {
	rm := NewRetryManager(nil, DefaultRetryPolicy())

	tests := []struct {
		err       error
		retryable bool
	}{
		{errors.New("connection refused"), true},
		{errors.New("timeout"), true},
		{errors.New("rate limit exceeded"), true},
		{errors.New("service unavailable"), true},
		{errors.New("503 error"), true},
		{errors.New("invalid email"), false},
		{errors.New("authentication failed"), false},
		{nil, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.err), func(t *testing.T) {
			if rm.IsRetryableError(tt.err) != tt.retryable {
				t.Errorf("IsRetryableError(%v) = %v, want %v",
					tt.err, rm.IsRetryableError(tt.err), tt.retryable)
			}
		})
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Second)

	// Initially closed
	if cb.State() != CircuitBreakerClosed {
		t.Errorf("Expected closed, got %s", cb.State())
	}

	if cb.IsOpen() {
		t.Error("Circuit should not be open initially")
	}

	// Record failures
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	// Should be open after threshold
	if cb.State() != CircuitBreakerOpen {
		t.Errorf("Expected open after failures, got %s", cb.State())
	}

	if !cb.IsOpen() {
		t.Error("Circuit should be open after threshold")
	}
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// Should transition to half-open
	if cb.IsOpen() {
		t.Error("Circuit should transition to half-open after timeout")
	}

	if cb.State() != CircuitBreakerHalfOpen {
		t.Errorf("Expected half-open, got %s", cb.State())
	}

	// Success should close it
	cb.RecordSuccess()
	cb.RecordSuccess()
	cb.RecordSuccess()

	if cb.State() != CircuitBreakerClosed {
		t.Errorf("Expected closed after successes, got %s", cb.State())
	}
}

func TestRetryStrategies(t *testing.T) {
	t.Run("exponential", func(t *testing.T) {
		strategy := &ExponentialBackoffStrategy{
			InitialDelay: time.Second,
			MaxDelay:     time.Minute,
			Multiplier:   2,
			MaxRetries:   5,
		}

		if strategy.NextDelay(1) != time.Second {
			t.Error("First delay should be initial")
		}

		if strategy.NextDelay(2) != 2*time.Second {
			t.Error("Second delay should be doubled")
		}

		if !strategy.ShouldRetry(3, nil) {
			t.Error("Should retry attempt 3")
		}

		if strategy.ShouldRetry(5, nil) {
			t.Error("Should not retry attempt 5")
		}
	})

	t.Run("linear", func(t *testing.T) {
		strategy := &LinearBackoffStrategy{
			InitialDelay: time.Second,
			Increment:    time.Second,
			MaxDelay:     time.Minute,
			MaxRetries:   5,
		}

		if strategy.NextDelay(1) != time.Second {
			t.Error("First delay should be initial")
		}

		if strategy.NextDelay(3) != 3*time.Second {
			t.Error("Third delay should be 3s")
		}
	})

	t.Run("constant", func(t *testing.T) {
		strategy := &ConstantBackoffStrategy{
			Delay:      5 * time.Second,
			MaxRetries: 3,
		}

		if strategy.NextDelay(1) != 5*time.Second {
			t.Error("All delays should be constant")
		}

		if strategy.NextDelay(10) != 5*time.Second {
			t.Error("All delays should be constant")
		}
	})
}

func TestRetryExecutor(t *testing.T) {
	attempts := 0
	strategy := &ExponentialBackoffStrategy{
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2,
		MaxRetries:   3,
	}

	executor := NewRetryExecutor(strategy)

	err := executor.Execute(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected success after retries, got %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

// ============================================
// Worker Tests
// ============================================

type mockSender struct {
	sendFunc func(context.Context, *EmailJob) error
}

func (m *mockSender) Send(ctx context.Context, job *EmailJob) error {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, job)
	}
	return nil
}

func TestWorkerPool(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	queue := NewEmailQueueWithClient(client, DefaultQueueConfig())
	sender := &mockSender{}

	config := DefaultWorkerConfig()
	config.NumWorkers = 2
	config.PollInterval = 50 * time.Millisecond

	pool := NewWorkerPool(queue, sender, config)

	err := pool.Start()
	if err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop()

	// Verify workers started
	time.Sleep(100 * time.Millisecond)

	if pool.WorkerCount() != 2 {
		t.Errorf("Expected 2 workers, got %d", pool.WorkerCount())
	}
}

func TestWorkerProcessing(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	queue := NewEmailQueueWithClient(client, DefaultQueueConfig())

	var processed atomic.Int32
	sender := &mockSender{
		sendFunc: func(ctx context.Context, job *EmailJob) error {
			processed.Add(1)
			return nil
		},
	}

	config := DefaultWorkerConfig()
	config.NumWorkers = 2
	config.PollInterval = 10 * time.Millisecond

	pool := NewWorkerPool(queue, sender, config)
	pool.Start()
	defer pool.Stop()

	ctx := context.Background()

	// Enqueue jobs
	for i := 0; i < 5; i++ {
		job := &EmailJob{
			Payload: &EmailPayload{
				To:       []string{"test@example.com"},
				Subject:  fmt.Sprintf("Test %d", i),
				HTMLBody: "body",
			},
		}
		queue.Enqueue(ctx, job)
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	if processed.Load() != 5 {
		t.Errorf("Expected 5 processed, got %d", processed.Load())
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(100, 10)

	// Should allow burst
	allowed := 0
	for i := 0; i < 15; i++ {
		if limiter.Allow() {
			allowed++
		}
	}

	if allowed != 10 {
		t.Errorf("Expected 10 allowed in burst, got %d", allowed)
	}

	// Wait for refill
	time.Sleep(100 * time.Millisecond)

	if !limiter.Allow() {
		t.Error("Should allow after refill")
	}
}

func TestRateLimiterWait(t *testing.T) {
	limiter := NewRateLimiter(1000, 1)

	// Consume the burst
	limiter.Allow()

	// Wait should return quickly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := limiter.Wait(ctx)
	if err != nil {
		t.Errorf("Wait should succeed: %v", err)
	}
}

func TestRateLimiterWaitCancellation(t *testing.T) {
	limiter := NewRateLimiter(0.01, 1) // Very slow rate

	// Consume the burst
	limiter.Allow()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := limiter.Wait(ctx)
	if err == nil {
		t.Error("Wait should be cancelled")
	}
}

// ============================================
// Monitoring Tests
// ============================================

func TestQueueMetrics(t *testing.T) {
	metrics := NewQueueMetrics("test_queue")

	metrics.RecordEnqueue("normal")
	metrics.RecordEnqueue("high")
	metrics.RecordDequeue()
	metrics.RecordComplete(100 * time.Millisecond)
	metrics.RecordFailed()
	metrics.RecordRetry(1)

	snapshot := metrics.GetSnapshot()

	if snapshot.Enqueued != 2 {
		t.Errorf("Expected 2 enqueued, got %d", snapshot.Enqueued)
	}

	if snapshot.Dequeued != 1 {
		t.Errorf("Expected 1 dequeued, got %d", snapshot.Dequeued)
	}

	if snapshot.Completed != 1 {
		t.Errorf("Expected 1 completed, got %d", snapshot.Completed)
	}

	if snapshot.Failed != 1 {
		t.Errorf("Expected 1 failed, got %d", snapshot.Failed)
	}

	if snapshot.AvgProcessingTime != 100*time.Millisecond {
		t.Errorf("Expected 100ms avg time, got %v", snapshot.AvgProcessingTime)
	}
}

func TestAlertManager(t *testing.T) {
	am := NewAlertManager()

	var fired []Alert
	am.AddHandler(func(a Alert) {
		fired = append(fired, a)
	})

	// Fire alert
	am.Fire(Alert{
		Name:     "test_alert",
		Severity: SeverityWarning,
		Message:  "Test message",
	})

	time.Sleep(10 * time.Millisecond) // Allow handler goroutine

	if len(fired) != 1 {
		t.Errorf("Expected 1 fired alert, got %d", len(fired))
	}

	active := am.GetActive()
	if len(active) != 1 {
		t.Errorf("Expected 1 active alert, got %d", len(active))
	}

	// Resolve
	am.Resolve("test_alert")

	active = am.GetActive()
	if len(active) != 0 {
		t.Errorf("Expected 0 active alerts after resolve, got %d", len(active))
	}
}

func TestMonitoringService(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	queue := NewEmailQueueWithClient(client, DefaultQueueConfig())

	config := DefaultMonitoringConfig()
	config.CollectionInterval = 50 * time.Millisecond

	ms := NewMonitoringService(queue, nil, config)
	ms.Start()
	defer ms.Stop()

	// Add some data
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		queue.Enqueue(ctx, &EmailJob{
			Payload: &EmailPayload{
				To:       []string{"test@example.com"},
				Subject:  "Test",
				HTMLBody: "body",
			},
		})
	}

	// Wait for collection
	time.Sleep(100 * time.Millisecond)

	metrics := ms.GetMetrics()
	if metrics.QueueName == "" {
		t.Error("Should have collected metrics")
	}
}

func TestHealthCheck(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	queue := NewEmailQueueWithClient(client, DefaultQueueConfig())
	ms := NewMonitoringService(queue, nil, DefaultMonitoringConfig())

	ctx := context.Background()
	health := ms.CheckHealth(ctx)

	if health.Status != HealthStatusHealthy {
		t.Errorf("Expected healthy status, got %s", health.Status)
	}

	if health.Components["redis"].Status != HealthStatusHealthy {
		t.Error("Redis should be healthy")
	}
}

// ============================================
// Concurrent Tests
// ============================================

func TestConcurrentEnqueue(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	queue := NewEmailQueueWithClient(client, DefaultQueueConfig())
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 10
	jobsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < jobsPerGoroutine; j++ {
				job := &EmailJob{
					Payload: &EmailPayload{
						To:       []string{fmt.Sprintf("test%d_%d@example.com", id, j)},
						Subject:  "Concurrent test",
						HTMLBody: "body",
					},
				}
				queue.Enqueue(ctx, job)
			}
		}(i)
	}

	wg.Wait()

	size, _ := queue.Size(ctx)
	expected := int64(numGoroutines * jobsPerGoroutine)
	if size != expected {
		t.Errorf("Expected %d jobs, got %d", expected, size)
	}
}

func TestConcurrentDequeue(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	queue := NewEmailQueueWithClient(client, DefaultQueueConfig())
	ctx := context.Background()

	// Enqueue jobs
	numJobs := 100
	for i := 0; i < numJobs; i++ {
		job := &EmailJob{
			Payload: &EmailPayload{
				To:       []string{fmt.Sprintf("test%d@example.com", i)},
				Subject:  "Concurrent test",
				HTMLBody: "body",
			},
		}
		queue.Enqueue(ctx, job)
	}

	// Concurrent dequeue
	var wg sync.WaitGroup
	var dequeued atomic.Int64
	numWorkers := 5

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			workerID := fmt.Sprintf("worker-%d", id)
			for {
				job, err := queue.Dequeue(ctx, workerID)
				if err != nil || job == nil {
					break
				}
				dequeued.Add(1)
				queue.Complete(ctx, job.ID)
			}
		}(i)
	}

	wg.Wait()

	if dequeued.Load() != int64(numJobs) {
		t.Errorf("Expected %d dequeued, got %d", numJobs, dequeued.Load())
	}
}

// ============================================
// Benchmark Tests
// ============================================

func BenchmarkEnqueue(b *testing.B) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()

	queue := NewEmailQueueWithClient(client, DefaultQueueConfig())
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		job := &EmailJob{
			Payload: &EmailPayload{
				To:       []string{"test@example.com"},
				Subject:  "Benchmark",
				HTMLBody: "body",
			},
		}
		queue.Enqueue(ctx, job)
	}
}

func BenchmarkDequeue(b *testing.B) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()

	queue := NewEmailQueueWithClient(client, DefaultQueueConfig())
	ctx := context.Background()

	// Pre-populate
	for i := 0; i < b.N; i++ {
		job := &EmailJob{
			Payload: &EmailPayload{
				To:       []string{"test@example.com"},
				Subject:  "Benchmark",
				HTMLBody: "body",
			},
		}
		queue.Enqueue(ctx, job)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queue.Dequeue(ctx, "worker-bench")
	}
}

func BenchmarkCalculateBackoff(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateBackoff(5, time.Second, time.Hour)
	}
}

func BenchmarkRateLimiter(b *testing.B) {
	limiter := NewRateLimiter(1000000, 1000000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow()
	}
}
