package queue

import (
	"container/heap"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// MockSMSSender implements SMSSender for testing
type MockSMSSender struct {
	mu           sync.Mutex
	sentMessages []*SMSMessage
	sendDelay    time.Duration
	failPhones   map[string]error
	callCount    int32
}

func NewMockSMSSender() *MockSMSSender {
	return &MockSMSSender{
		sentMessages: make([]*SMSMessage, 0),
		failPhones:   make(map[string]error),
	}
}

func (m *MockSMSSender) Send(ctx context.Context, phoneNumber, content string) error {
	atomic.AddInt32(&m.callCount, 1)

	if m.sendDelay > 0 {
		time.Sleep(m.sendDelay)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err, exists := m.failPhones[phoneNumber]; exists {
		return err
	}

	return nil
}

func (m *MockSMSSender) SendBatch(ctx context.Context, messages []*SMSMessage) []SendResult {
	results := make([]SendResult, len(messages))

	for i, msg := range messages {
		err := m.Send(ctx, msg.PhoneNumber, msg.Content)
		results[i] = SendResult{
			MessageID: msg.ID,
			Success:   err == nil,
			Error:     err,
			SentAt:    time.Now(),
		}

		if err == nil {
			m.mu.Lock()
			m.sentMessages = append(m.sentMessages, msg)
			m.mu.Unlock()
		}
	}

	return results
}

func (m *MockSMSSender) SetFailPhone(phone string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failPhones[phone] = err
}

func (m *MockSMSSender) GetSentMessages() []*SMSMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*SMSMessage, len(m.sentMessages))
	copy(result, m.sentMessages)
	return result
}

func (m *MockSMSSender) GetCallCount() int {
	return int(atomic.LoadInt32(&m.callCount))
}

// =============================================================================
// Queue Operation Tests
// =============================================================================

func TestEnqueueMessage(t *testing.T) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)

	tests := []struct {
		name    string
		msg     *SMSMessage
		wantErr bool
	}{
		{
			name: "valid message",
			msg: &SMSMessage{
				PhoneNumber: "+1234567890",
				Content:     "Hello, World!",
				Priority:    PriorityNormal,
			},
			wantErr: false,
		},
		{
			name: "message with all fields",
			msg: &SMSMessage{
				PhoneNumber: "+1234567890",
				Content:     "Test message",
				Priority:    PriorityHigh,
				Metadata: map[string]string{
					"campaign": "test",
				},
				RecipientID: "user-123",
			},
			wantErr: false,
		},
		{
			name:    "nil message",
			msg:     nil,
			wantErr: true,
		},
		{
			name: "empty phone number",
			msg: &SMSMessage{
				PhoneNumber: "",
				Content:     "Hello",
			},
			wantErr: true,
		},
		{
			name: "empty content",
			msg: &SMSMessage{
				PhoneNumber: "+1234567890",
				Content:     "",
			},
			wantErr: true,
		},
		{
			name: "invalid phone number - too short",
			msg: &SMSMessage{
				PhoneNumber: "123",
				Content:     "Hello",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := queue.EnqueueMessage(context.Background(), tt.msg)

			if tt.wantErr && err == nil {
				t.Errorf("EnqueueMessage() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("EnqueueMessage() unexpected error: %v", err)
			}

			if !tt.wantErr && tt.msg != nil {
				// Verify message was enqueued
				if tt.msg.ID == "" {
					t.Errorf("Message ID should be set")
				}
				if tt.msg.Status != StatusPending {
					t.Errorf("Message status should be pending, got %s", tt.msg.Status)
				}
			}
		})
	}
}

func TestEnqueueBatch(t *testing.T) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)
	ctx := context.Background()

	messages := []*SMSMessage{
		{PhoneNumber: "+1234567890", Content: "Message 1"},
		{PhoneNumber: "", Content: "Invalid"},
		{PhoneNumber: "+1234567891", Content: "Message 2"},
		{PhoneNumber: "+1234567892", Content: "Message 3"},
	}

	ids, errs := queue.EnqueueBatch(ctx, messages)

	// Check results
	if ids[0] == "" {
		t.Errorf("First message should be enqueued")
	}
	if errs[1] == nil {
		t.Errorf("Second message should fail (empty phone)")
	}
	if ids[2] == "" {
		t.Errorf("Third message should be enqueued")
	}
	if ids[3] == "" {
		t.Errorf("Fourth message should be enqueued")
	}

	// Check queue size
	if queue.GetQueueSize() != 3 {
		t.Errorf("Queue size should be 3, got %d", queue.GetQueueSize())
	}
}

func TestScheduleMessage(t *testing.T) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)
	ctx := context.Background()

	// Schedule for future
	futureTime := time.Now().Add(time.Hour)
	msg := &SMSMessage{
		PhoneNumber: "+1234567890",
		Content:     "Scheduled message",
	}

	err := queue.ScheduleMessage(ctx, msg, futureTime)
	if err != nil {
		t.Errorf("ScheduleMessage() failed: %v", err)
	}

	if msg.Status != StatusScheduled {
		t.Errorf("Status should be scheduled, got %s", msg.Status)
	}

	if !msg.ScheduledTime.Equal(futureTime) {
		t.Errorf("ScheduledTime should be %v, got %v", futureTime, msg.ScheduledTime)
	}

	// Schedule for past should fail
	pastTime := time.Now().Add(-time.Hour)
	msg2 := &SMSMessage{
		PhoneNumber: "+1234567890",
		Content:     "Past message",
	}

	err = queue.ScheduleMessage(ctx, msg2, pastTime)
	if err != ErrScheduleInPast {
		t.Errorf("ScheduleMessage() should fail for past time, got %v", err)
	}
}

func TestCancelMessage(t *testing.T) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)
	ctx := context.Background()

	// Enqueue a message
	msg := &SMSMessage{
		PhoneNumber: "+1234567890",
		Content:     "Cancel me",
	}

	err := queue.EnqueueMessage(ctx, msg)
	if err != nil {
		t.Fatalf("EnqueueMessage() failed: %v", err)
	}

	// Cancel it
	err = queue.CancelMessage(ctx, msg.ID)
	if err != nil {
		t.Errorf("CancelMessage() failed: %v", err)
	}

	// Verify status
	retrieved, err := queue.GetMessage(ctx, msg.ID)
	if err != nil {
		t.Errorf("GetMessage() failed: %v", err)
	}
	if retrieved.Status != StatusCancelled {
		t.Errorf("Status should be cancelled, got %s", retrieved.Status)
	}

	// Cancel non-existent message
	err = queue.CancelMessage(ctx, "non-existent")
	if err != ErrMessageNotFound {
		t.Errorf("CancelMessage() should return ErrMessageNotFound")
	}
}

func TestGetMessage(t *testing.T) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)
	ctx := context.Background()

	msg := &SMSMessage{
		PhoneNumber: "+1234567890",
		Content:     "Test message",
		Priority:    PriorityHigh,
	}

	queue.EnqueueMessage(ctx, msg)

	// Get existing message
	retrieved, err := queue.GetMessage(ctx, msg.ID)
	if err != nil {
		t.Errorf("GetMessage() failed: %v", err)
	}

	if retrieved.Content != msg.Content {
		t.Errorf("Content mismatch")
	}

	// Get non-existent message
	_, err = queue.GetMessage(ctx, "non-existent")
	if err != ErrMessageNotFound {
		t.Errorf("GetMessage() should return ErrMessageNotFound")
	}
}

func TestQueueFull(t *testing.T) {
	config := &QueueConfig{
		MaxQueueSize:    3,
		BatchSize:       10,
		ProcessInterval: time.Second,
		MessageTTL:      time.Hour,
		RetryPolicy:     DefaultRetryPolicy(),
		WorkerCount:     1,
	}

	sender := NewMockSMSSender()
	queue := NewSMSQueue(config, sender)
	ctx := context.Background()

	// Fill the queue
	for i := 0; i < 3; i++ {
		err := queue.EnqueueMessage(ctx, &SMSMessage{
			PhoneNumber: "+1234567890",
			Content:     "Message",
		})
		if err != nil {
			t.Fatalf("EnqueueMessage() failed: %v", err)
		}
	}

	// Next should fail
	err := queue.EnqueueMessage(ctx, &SMSMessage{
		PhoneNumber: "+1234567890",
		Content:     "Overflow",
	})

	if err != ErrQueueFull {
		t.Errorf("Expected ErrQueueFull, got %v", err)
	}
}

// =============================================================================
// Priority Handling Tests
// =============================================================================

func TestPriorityOrdering(t *testing.T) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)
	ctx := context.Background()

	// Enqueue messages with different priorities in random order
	priorities := []int{PriorityLow, PriorityUrgent, PriorityNormal, PriorityHigh, PriorityBulk}
	var messages []*SMSMessage
	for _, priority := range priorities {
		msg := &SMSMessage{
			PhoneNumber: "+1234567890",
			Content:     "Message",
			Priority:    priority,
		}
		queue.EnqueueMessage(ctx, msg)
		messages = append(messages, msg)
	}

	// Set all messages to have the same ScheduledTime to test pure priority ordering
	// The heap compares ScheduledTime first, so we need them equal for priority to matter
	sameTime := time.Now().Add(-time.Millisecond)
	queue.pqMu.Lock()
	for _, msg := range messages {
		msg.ScheduledTime = sameTime
	}
	// Rebuild the heap with updated scheduled times
	heap.Init(&queue.pq)
	queue.pqMu.Unlock()

	// Process and check order - should be sorted by priority
	batch := queue.getNextBatch(5)

	// Verify messages are sorted by priority (ascending - lower number = higher priority)
	for i := 0; i < len(batch)-1; i++ {
		if batch[i].Priority > batch[i+1].Priority {
			t.Errorf("Messages not sorted by priority: msg[%d].Priority=%d > msg[%d].Priority=%d",
				i, batch[i].Priority, i+1, batch[i+1].Priority)
		}
	}
}

func TestPriorityFIFOWithinSamePriority(t *testing.T) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)
	ctx := context.Background()

	// Enqueue messages with same priority
	for i := 0; i < 5; i++ {
		msg := &SMSMessage{
			PhoneNumber: "+1234567890",
			Content:     "Message " + string(rune('A'+i)),
			Priority:    PriorityNormal,
		}
		queue.EnqueueMessage(ctx, msg)
		time.Sleep(time.Millisecond) // Ensure different creation times
	}

	// Process and check FIFO order
	batch := queue.getNextBatch(5)

	for i := 0; i < len(batch)-1; i++ {
		if batch[i].CreatedAt.After(batch[i+1].CreatedAt) {
			t.Errorf("Messages should be in FIFO order within same priority")
		}
	}
}

func TestUpdatePriority(t *testing.T) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)
	ctx := context.Background()

	msg := &SMSMessage{
		PhoneNumber: "+1234567890",
		Content:     "Test",
		Priority:    PriorityLow,
	}

	queue.EnqueueMessage(ctx, msg)

	// Update priority
	err := queue.UpdatePriority(msg.ID, PriorityUrgent)
	if err != nil {
		t.Errorf("UpdatePriority() failed: %v", err)
	}

	// Verify
	retrieved, _ := queue.GetMessage(ctx, msg.ID)
	if retrieved.Priority != PriorityUrgent {
		t.Errorf("Priority should be urgent, got %d", retrieved.Priority)
	}

	// Update non-existent
	err = queue.UpdatePriority("non-existent", PriorityHigh)
	if err != ErrMessageNotFound {
		t.Errorf("UpdatePriority() should return ErrMessageNotFound")
	}
}

func TestScheduledMessagePriority(t *testing.T) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)
	ctx := context.Background()

	// Enqueue messages with different scheduled times
	futureTime := time.Now().Add(time.Hour)
	soonTime := time.Now().Add(time.Minute)

	messages := []*SMSMessage{
		{PhoneNumber: "+1234567890", Content: "Later", Priority: PriorityUrgent},
		{PhoneNumber: "+1234567891", Content: "Now", Priority: PriorityLow},
		{PhoneNumber: "+1234567892", Content: "Soon", Priority: PriorityHigh},
	}

	// Enqueue first - these will get default ScheduledTime = now
	queue.EnqueueMessage(ctx, messages[1]) // Now - will be ready immediately

	// Set future times manually for the others after enqueueing
	messages[0].ScheduledTime = futureTime
	messages[2].ScheduledTime = soonTime

	// Force message status to scheduled and push to queue
	queue.messagesMu.Lock()
	queue.messages[messages[0].ID] = messages[0]
	queue.messagesMu.Unlock()

	// Wait a tiny bit to ensure the "Now" message is ready
	time.Sleep(10 * time.Millisecond)

	// Only the "Now" message should be ready
	batch := queue.getNextBatch(3)

	if len(batch) != 1 {
		t.Errorf("Expected 1 message ready, got %d", len(batch))
		return
	}

	if batch[0].Content != "Now" {
		t.Errorf("Expected 'Now' message, got '%s'", batch[0].Content)
	}
}

// =============================================================================
// Retry Scenario Tests
// =============================================================================

func TestRetryPolicy(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:   3,
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     time.Second,
		Multiplier:   2.0,
		JitterFactor: 0,
	}

	tests := []struct {
		retryCount    int
		expectedDelay time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 0}, // Max retries reached
	}

	for _, tt := range tests {
		delay := policy.CalculateNextRetry(tt.retryCount)
		if delay != tt.expectedDelay {
			t.Errorf("Retry %d: delay = %v, want %v", tt.retryCount, delay, tt.expectedDelay)
		}
	}
}

func TestRetryWithMaxDelayCap(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:   10,
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     500 * time.Millisecond,
		Multiplier:   2.0,
		JitterFactor: 0,
	}

	// After several retries, delay should cap at MaxDelay
	delay := policy.CalculateNextRetry(5) // Would be 3.2s without cap
	if delay > policy.MaxDelay {
		t.Errorf("Delay should be capped at %v, got %v", policy.MaxDelay, delay)
	}
}

func TestMessageRetry(t *testing.T) {
	config := &QueueConfig{
		MaxQueueSize:    1000,
		BatchSize:       10,
		ProcessInterval: 10 * time.Millisecond,
		MessageTTL:      time.Hour,
		RetryPolicy: &RetryPolicy{
			MaxRetries:   2,
			BaseDelay:    10 * time.Millisecond,
			MaxDelay:     100 * time.Millisecond,
			Multiplier:   2.0,
			JitterFactor: 0,
		},
		WorkerCount: 1,
	}

	sender := NewMockSMSSender()
	sender.SetFailPhone("+1234567890", errors.New("temporary failure"))

	queue := NewSMSQueue(config, sender)
	ctx := context.Background()

	msg := &SMSMessage{
		PhoneNumber: "+1234567890",
		Content:     "Retry test",
		Priority:    PriorityNormal,
	}

	err := queue.EnqueueMessage(ctx, msg)
	if err != nil {
		t.Fatalf("EnqueueMessage() failed: %v", err)
	}

	// Process the message (will fail)
	queue.ProcessQueue(ctx)

	// Check retry was scheduled
	retrieved, _ := queue.GetMessage(ctx, msg.ID)
	if retrieved.RetryCount != 1 {
		t.Errorf("RetryCount should be 1, got %d", retrieved.RetryCount)
	}
	if retrieved.Status != StatusRetrying {
		t.Errorf("Status should be retrying, got %s", retrieved.Status)
	}
}

func TestMaxRetriesExceeded(t *testing.T) {
	config := &QueueConfig{
		MaxQueueSize:    1000,
		BatchSize:       10,
		ProcessInterval: 10 * time.Millisecond,
		MessageTTL:      time.Hour,
		RetryPolicy: &RetryPolicy{
			MaxRetries:   1,
			BaseDelay:    1 * time.Millisecond,
			MaxDelay:     10 * time.Millisecond,
			Multiplier:   2.0,
			JitterFactor: 0,
		},
		WorkerCount: 1,
	}

	sender := NewMockSMSSender()
	sender.SetFailPhone("+1234567890", errors.New("permanent failure"))

	queue := NewSMSQueue(config, sender)
	ctx := context.Background()

	failedCalled := false
	queue.SetCallbacks(nil, func(msg *SMSMessage, err error) {
		failedCalled = true
	}, nil)

	msg := &SMSMessage{
		PhoneNumber: "+1234567890",
		Content:     "Will fail",
		Priority:    PriorityNormal,
	}

	queue.EnqueueMessage(ctx, msg)

	// First attempt
	queue.ProcessQueue(ctx)

	// Wait and process retry
	time.Sleep(10 * time.Millisecond)
	queue.HandleRetries(ctx)
	queue.ProcessQueue(ctx)

	// Check final state
	retrieved, _ := queue.GetMessage(ctx, msg.ID)
	if retrieved.Status != StatusFailed {
		t.Errorf("Status should be failed, got %s", retrieved.Status)
	}

	if !failedCalled {
		t.Errorf("OnFailed callback should have been called")
	}
}

func TestHandleRetries(t *testing.T) {
	config := DefaultQueueConfig()
	config.RetryPolicy.BaseDelay = time.Millisecond
	config.RetryPolicy.MaxDelay = 10 * time.Millisecond

	sender := NewMockSMSSender()
	queue := NewSMSQueue(config, sender)
	ctx := context.Background()

	// Manually add a message to retry queue
	msg := &SMSMessage{
		ID:          "test-retry",
		PhoneNumber: "+1234567890",
		Content:     "Retry me",
		Status:      StatusRetrying,
		RetryCount:  1,
	}

	nextRetry := time.Now().Add(-time.Second) // Already past
	msg.NextRetryAt = &nextRetry

	queue.messagesMu.Lock()
	queue.messages[msg.ID] = msg
	queue.messagesMu.Unlock()

	queue.retryQueueMu.Lock()
	queue.retryQueue = append(queue.retryQueue, msg)
	queue.retryQueueMu.Unlock()

	// Handle retries
	queue.HandleRetries(ctx)

	// Check message was re-queued
	if msg.Status != StatusPending {
		t.Errorf("Status should be pending, got %s", msg.Status)
	}

	if queue.GetPendingCount() != 1 {
		t.Errorf("Pending count should be 1, got %d", queue.GetPendingCount())
	}
}

// =============================================================================
// Batch Processing Tests
// =============================================================================

func TestBatchProcess(t *testing.T) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)
	ctx := context.Background()

	// Enqueue 10 messages
	for i := 0; i < 10; i++ {
		queue.EnqueueMessage(ctx, &SMSMessage{
			PhoneNumber: "+1234567890",
			Content:     "Batch message",
			Priority:    PriorityNormal,
		})
	}

	// Process batch of 5
	processed, err := queue.BatchProcess(ctx, 5)
	if err != nil {
		t.Errorf("BatchProcess() failed: %v", err)
	}

	if processed != 5 {
		t.Errorf("Should process 5 messages, got %d", processed)
	}

	// Check sent messages
	sent := sender.GetSentMessages()
	if len(sent) != 5 {
		t.Errorf("Should send 5 messages, got %d", len(sent))
	}

	// Process remaining
	processed, _ = queue.BatchProcess(ctx, 10)
	if processed != 5 {
		t.Errorf("Should process remaining 5 messages, got %d", processed)
	}
}

func TestBatchProcessEmpty(t *testing.T) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)
	ctx := context.Background()

	processed, err := queue.BatchProcess(ctx, 10)
	if err != nil {
		t.Errorf("BatchProcess() should not error on empty queue")
	}

	if processed != 0 {
		t.Errorf("Should process 0 messages on empty queue")
	}
}

func TestBatchProcessMixedResults(t *testing.T) {
	sender := NewMockSMSSender()
	sender.SetFailPhone("+1234567899", errors.New("failed"))

	// Use config with 0 retries to ensure immediate failure
	config := DefaultQueueConfig()
	config.RetryPolicy.MaxRetries = 0

	queue := NewSMSQueue(config, sender)
	ctx := context.Background()

	// Enqueue mix of success and fail
	queue.EnqueueMessage(ctx, &SMSMessage{
		PhoneNumber: "+1234567890",
		Content:     "Success 1",
	})
	queue.EnqueueMessage(ctx, &SMSMessage{
		PhoneNumber: "+1234567899", // This one will fail
		Content:     "Will fail",
	})
	queue.EnqueueMessage(ctx, &SMSMessage{
		PhoneNumber: "+1234567891",
		Content:     "Success 2",
	})

	batchSent := 0
	batchFailed := 0

	queue.SetCallbacks(nil, nil, func(sent, failed int) {
		batchSent = sent
		batchFailed = failed
	})

	queue.BatchProcess(ctx, 10)

	if batchSent != 2 {
		t.Errorf("Should have 2 successful, got %d", batchSent)
	}

	if batchFailed != 1 {
		t.Errorf("Should have 1 failed, got %d", batchFailed)
	}
}

func TestBatchProcessEfficiency(t *testing.T) {
	sender := NewMockSMSSender()
	sender.sendDelay = time.Millisecond

	config := DefaultQueueConfig()
	config.BatchSize = 100

	queue := NewSMSQueue(config, sender)
	ctx := context.Background()

	// Enqueue many messages
	for i := 0; i < 100; i++ {
		queue.EnqueueMessage(ctx, &SMSMessage{
			PhoneNumber: "+1234567890",
			Content:     "Efficiency test",
		})
	}

	start := time.Now()
	queue.BatchProcess(ctx, 100)
	elapsed := time.Since(start)

	// With batch processing, 100 messages should take roughly 100ms (1ms each)
	// Not 100 * 1ms * overhead
	if elapsed > 500*time.Millisecond {
		t.Errorf("Batch processing too slow: %v", elapsed)
	}

	metrics := queue.GetMetrics()
	if metrics.BatchesProcessed != 1 {
		t.Errorf("Should record 1 batch processed, got %d", metrics.BatchesProcessed)
	}
}

// =============================================================================
// Queue Lifecycle Tests
// =============================================================================

func TestQueueStartStop(t *testing.T) {
	sender := NewMockSMSSender()
	config := DefaultQueueConfig()
	config.ProcessInterval = 10 * time.Millisecond

	queue := NewSMSQueue(config, sender)

	// Start
	err := queue.Start()
	if err != nil {
		t.Errorf("Start() failed: %v", err)
	}

	if !queue.IsRunning() {
		t.Errorf("Queue should be running")
	}

	// Double start should be idempotent
	err = queue.Start()
	if err != nil {
		t.Errorf("Double Start() failed: %v", err)
	}

	// Enqueue and let it process
	queue.EnqueueMessage(context.Background(), &SMSMessage{
		PhoneNumber: "+1234567890",
		Content:     "Auto processed",
	})

	time.Sleep(50 * time.Millisecond)

	// Stop
	err = queue.Stop()
	if err != nil {
		t.Errorf("Stop() failed: %v", err)
	}

	if queue.IsRunning() {
		t.Errorf("Queue should not be running")
	}

	// Double stop should be idempotent
	err = queue.Stop()
	if err != nil {
		t.Errorf("Double Stop() failed: %v", err)
	}
}

func TestDrain(t *testing.T) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)
	ctx := context.Background()

	// Enqueue messages
	for i := 0; i < 5; i++ {
		queue.EnqueueMessage(ctx, &SMSMessage{
			PhoneNumber: "+1234567890",
			Content:     "Drain test",
		})
	}

	// Drain
	drained := queue.Drain()

	if len(drained) != 5 {
		t.Errorf("Should drain 5 messages, got %d", len(drained))
	}

	if queue.GetPendingCount() != 0 {
		t.Errorf("Queue should be empty after drain")
	}
}

func TestClear(t *testing.T) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)
	ctx := context.Background()

	// Enqueue messages
	for i := 0; i < 5; i++ {
		queue.EnqueueMessage(ctx, &SMSMessage{
			PhoneNumber: "+1234567890",
			Content:     "Clear test",
		})
	}

	// Clear
	queue.Clear()

	if queue.GetQueueSize() != 0 {
		t.Errorf("Queue should be empty after clear")
	}

	if queue.GetPendingCount() != 0 {
		t.Errorf("Pending count should be 0 after clear")
	}
}

// =============================================================================
// Metrics Tests
// =============================================================================

func TestMetrics(t *testing.T) {
	sender := NewMockSMSSender()
	sender.SetFailPhone("+1234567899", errors.New("failed"))

	// Config with 0 retries for immediate failure
	config := DefaultQueueConfig()
	config.RetryPolicy.MaxRetries = 0

	queue := NewSMSQueue(config, sender)
	ctx := context.Background()

	// Enqueue messages (both with valid phone numbers)
	err1 := queue.EnqueueMessage(ctx, &SMSMessage{
		PhoneNumber: "+1234567890",
		Content:     "Success",
	})
	err2 := queue.EnqueueMessage(ctx, &SMSMessage{
		PhoneNumber: "+1234567899", // Will fail at send time
		Content:     "Fail",
	})

	if err1 != nil || err2 != nil {
		t.Fatalf("Enqueue failed: %v, %v", err1, err2)
	}

	metrics := queue.GetMetrics()
	if metrics.MessagesEnqueued != 2 {
		t.Errorf("MessagesEnqueued should be 2, got %d", metrics.MessagesEnqueued)
	}

	// Process
	queue.BatchProcess(ctx, 10)

	metrics = queue.GetMetrics()
	if metrics.MessagesProcessed != 2 {
		t.Errorf("MessagesProcessed should be 2, got %d", metrics.MessagesProcessed)
	}

	if metrics.MessagesSent != 1 {
		t.Errorf("MessagesSent should be 1, got %d", metrics.MessagesSent)
	}

	if metrics.MessagesFailed != 1 {
		t.Errorf("MessagesFailed should be 1, got %d", metrics.MessagesFailed)
	}
}

func TestQueueStats(t *testing.T) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)
	ctx := context.Background()

	// Enqueue messages with different priorities
	queue.EnqueueMessage(ctx, &SMSMessage{
		PhoneNumber: "+1234567890",
		Content:     "Urgent",
		Priority:    PriorityUrgent,
	})
	queue.EnqueueMessage(ctx, &SMSMessage{
		PhoneNumber: "+1234567891",
		Content:     "Normal 1",
		Priority:    PriorityNormal,
	})
	queue.EnqueueMessage(ctx, &SMSMessage{
		PhoneNumber: "+1234567892",
		Content:     "Normal 2",
		Priority:    PriorityNormal,
	})

	stats := queue.GetQueueStats()

	if stats.TotalMessages != 3 {
		t.Errorf("TotalMessages should be 3, got %d", stats.TotalMessages)
	}

	if stats.ByStatus[StatusPending] != 3 {
		t.Errorf("ByStatus[pending] should be 3, got %d", stats.ByStatus[StatusPending])
	}

	if stats.ByPriority[PriorityUrgent] != 1 {
		t.Errorf("ByPriority[urgent] should be 1, got %d", stats.ByPriority[PriorityUrgent])
	}

	if stats.ByPriority[PriorityNormal] != 2 {
		t.Errorf("ByPriority[normal] should be 2, got %d", stats.ByPriority[PriorityNormal])
	}
}

// =============================================================================
// Message Expiration Tests
// =============================================================================

func TestMessageExpiration(t *testing.T) {
	config := DefaultQueueConfig()
	config.MessageTTL = 10 * time.Millisecond

	sender := NewMockSMSSender()
	queue := NewSMSQueue(config, sender)
	ctx := context.Background()

	// Enqueue message - it will get short TTL
	msg := &SMSMessage{
		PhoneNumber: "+1234567890",
		Content:     "Will expire",
	}
	queue.EnqueueMessage(ctx, msg)

	// Wait for expiration (TTL is 10ms)
	time.Sleep(20 * time.Millisecond)

	// Manually set ExpiresAt to past to simulate expiration
	queue.messagesMu.Lock()
	if m, exists := queue.messages[msg.ID]; exists {
		m.ExpiresAt = time.Now().Add(-time.Millisecond)
	}
	queue.messagesMu.Unlock()

	// Try to process - message should be expired
	batch := queue.getNextBatch(10)

	// Message should be expired and not in batch
	if len(batch) != 0 {
		t.Errorf("Expired message should not be in batch, got %d messages", len(batch))
	}

	metrics := queue.GetMetrics()
	if metrics.MessagesExpired != 1 {
		t.Errorf("MessagesExpired should be 1, got %d", metrics.MessagesExpired)
	}
}

// =============================================================================
// Query Tests
// =============================================================================

func TestGetMessagesByStatus(t *testing.T) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)
	ctx := context.Background()

	// Enqueue messages
	queue.EnqueueMessage(ctx, &SMSMessage{
		PhoneNumber: "+1234567890",
		Content:     "Message 1",
	})
	queue.EnqueueMessage(ctx, &SMSMessage{
		PhoneNumber: "+1234567891",
		Content:     "Message 2",
	})

	// Cancel one
	msgs := queue.GetMessagesByStatus(StatusPending)
	queue.CancelMessage(ctx, msgs[0].ID)

	// Check by status
	pending := queue.GetMessagesByStatus(StatusPending)
	if len(pending) != 1 {
		t.Errorf("Should have 1 pending, got %d", len(pending))
	}

	cancelled := queue.GetMessagesByStatus(StatusCancelled)
	if len(cancelled) != 1 {
		t.Errorf("Should have 1 cancelled, got %d", len(cancelled))
	}
}

func TestGetMessagesByPriority(t *testing.T) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)
	ctx := context.Background()

	// Enqueue messages with different priorities
	queue.EnqueueMessage(ctx, &SMSMessage{
		PhoneNumber: "+1234567890",
		Content:     "High 1",
		Priority:    PriorityHigh,
	})
	queue.EnqueueMessage(ctx, &SMSMessage{
		PhoneNumber: "+1234567891",
		Content:     "Normal",
		Priority:    PriorityNormal,
	})
	queue.EnqueueMessage(ctx, &SMSMessage{
		PhoneNumber: "+1234567892",
		Content:     "High 2",
		Priority:    PriorityHigh,
	})

	high := queue.GetMessagesByPriority(PriorityHigh)
	if len(high) != 2 {
		t.Errorf("Should have 2 high priority, got %d", len(high))
	}

	normal := queue.GetMessagesByPriority(PriorityNormal)
	if len(normal) != 1 {
		t.Errorf("Should have 1 normal priority, got %d", len(normal))
	}
}

// =============================================================================
// Concurrency Tests
// =============================================================================

func TestConcurrentEnqueue(t *testing.T) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 10
	messagesPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				queue.EnqueueMessage(ctx, &SMSMessage{
					PhoneNumber: "+1234567890",
					Content:     "Concurrent message",
					Priority:    PriorityNormal,
				})
			}
		}(i)
	}

	wg.Wait()

	expected := numGoroutines * messagesPerGoroutine
	if queue.GetQueueSize() != expected {
		t.Errorf("Queue size should be %d, got %d", expected, queue.GetQueueSize())
	}
}

func TestConcurrentProcessing(t *testing.T) {
	sender := NewMockSMSSender()
	config := DefaultQueueConfig()
	config.WorkerCount = 4
	config.ProcessInterval = time.Millisecond

	queue := NewSMSQueue(config, sender)
	ctx := context.Background()

	// Enqueue many messages
	for i := 0; i < 100; i++ {
		queue.EnqueueMessage(ctx, &SMSMessage{
			PhoneNumber: "+1234567890",
			Content:     "Process me",
		})
	}

	// Start queue
	queue.Start()

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Stop
	queue.Stop()

	// Check all processed
	sent := sender.GetSentMessages()
	if len(sent) != 100 {
		t.Errorf("Should send all 100 messages, got %d", len(sent))
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkEnqueueMessage(b *testing.B) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queue.EnqueueMessage(ctx, &SMSMessage{
			PhoneNumber: "+1234567890",
			Content:     "Benchmark message",
			Priority:    PriorityNormal,
		})
	}
}

func BenchmarkGetNextBatch(b *testing.B) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)
	ctx := context.Background()

	// Pre-populate queue
	for i := 0; i < 10000; i++ {
		queue.EnqueueMessage(ctx, &SMSMessage{
			PhoneNumber: "+1234567890",
			Content:     "Benchmark message",
			Priority:    i % 5, // Varied priorities
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queue.getNextBatch(100)
		// Re-add messages for next iteration
		for j := 0; j < 100; j++ {
			queue.EnqueueMessage(ctx, &SMSMessage{
				PhoneNumber: "+1234567890",
				Content:     "Benchmark message",
				Priority:    j % 5,
			})
		}
	}
}

func BenchmarkBatchProcess(b *testing.B) {
	sender := NewMockSMSSender()
	config := DefaultQueueConfig()
	queue := NewSMSQueue(config, sender)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Re-populate queue for each iteration
		for j := 0; j < 100; j++ {
			queue.EnqueueMessage(ctx, &SMSMessage{
				PhoneNumber: "+1234567890",
				Content:     "Benchmark message",
			})
		}
		b.StartTimer()

		queue.BatchProcess(ctx, 100)
	}
}

func BenchmarkConcurrentEnqueue(b *testing.B) {
	sender := NewMockSMSSender()
	queue := NewSMSQueue(nil, sender)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			queue.EnqueueMessage(ctx, &SMSMessage{
				PhoneNumber: "+1234567890",
				Content:     "Benchmark message",
				Priority:    PriorityNormal,
			})
		}
	})
}

func BenchmarkRetryPolicyCalculation(b *testing.B) {
	policy := DefaultRetryPolicy()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		policy.CalculateNextRetry(i % 3)
	}
}
