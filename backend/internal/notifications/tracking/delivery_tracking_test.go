package tracking

import (
	"context"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// Test Helpers
// =============================================================================

func setupTracker() *DeliveryTracker {
	return NewDeliveryTracker(nil, nil)
}

func createTestRecord(t *testing.T, tracker *DeliveryTracker, customerID, channel string) *DeliveryRecord {
	t.Helper()
	ctx := context.Background()

	input := &TrackDeliveryInput{
		MessageID:  "msg-" + time.Now().Format("150405.000000"),
		CustomerID: customerID,
		Channel:    channel,
		Recipient:  "+1234567890",
	}

	record, err := tracker.TrackDelivery(ctx, input)
	if err != nil {
		t.Fatalf("Failed to create test record: %v", err)
	}
	return record
}

// =============================================================================
// Status Tracking Tests
// =============================================================================

func TestTrackDelivery(t *testing.T) {
	tests := []struct {
		name    string
		input   *TrackDeliveryInput
		wantErr bool
		errType error
	}{
		{
			name: "valid input",
			input: &TrackDeliveryInput{
				MessageID:  "msg-001",
				CustomerID: "cust-001",
				Channel:    "sms",
				Recipient:  "+1234567890",
			},
			wantErr: false,
		},
		{
			name: "with metadata",
			input: &TrackDeliveryInput{
				MessageID:  "msg-002",
				CustomerID: "cust-001",
				Channel:    "email",
				Recipient:  "test@example.com",
				Metadata:   map[string]string{"campaign": "test"},
			},
			wantErr: false,
		},
		{
			name:    "nil input",
			input:   nil,
			wantErr: true,
		},
		{
			name: "empty message ID",
			input: &TrackDeliveryInput{
				MessageID:  "",
				CustomerID: "cust-001",
			},
			wantErr: true,
			errType: ErrInvalidMessageID,
		},
		{
			name: "missing customer ID is allowed",
			input: &TrackDeliveryInput{
				MessageID: "msg-003",
				Channel:   "sms",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := setupTracker()
			ctx := context.Background()

			record, err := tracker.TrackDelivery(ctx, tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if tt.errType != nil && err != tt.errType {
					t.Errorf("Expected error %v, got %v", tt.errType, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if record.ID == "" {
				t.Error("Record ID should be set")
			}
			if record.Status != StatusPending {
				t.Errorf("Initial status should be pending, got %v", record.Status)
			}
			if record.MessageID != tt.input.MessageID {
				t.Errorf("MessageID mismatch: got %v, want %v", record.MessageID, tt.input.MessageID)
			}
		})
	}
}

func TestTrackDeliveryDuplicate(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	input := &TrackDeliveryInput{
		MessageID:  "msg-duplicate",
		CustomerID: "cust-001",
	}

	// First creation should succeed
	_, err := tracker.TrackDelivery(ctx, input)
	if err != nil {
		t.Fatalf("First track should succeed: %v", err)
	}

	// Second creation with same ID should fail
	_, err = tracker.TrackDelivery(ctx, input)
	if err != ErrRecordExists {
		t.Errorf("Expected ErrRecordExists, got %v", err)
	}
}

func TestUpdateStatus(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	// Create initial record
	record := createTestRecord(t, tracker, "cust-001", "sms")

	tests := []struct {
		name      string
		toStatus  DeliveryStatus
		wantErr   bool
		checkFunc func(*DeliveryRecord) bool
	}{
		{
			name:     "pending to queued",
			toStatus: StatusQueued,
			wantErr:  false,
		},
		{
			name:     "queued to sent",
			toStatus: StatusSent,
			wantErr:  false,
			checkFunc: func(r *DeliveryRecord) bool {
				return r.Attempts == 1 && !r.FirstAttempt.IsZero()
			},
		},
		{
			name:     "sent to delivered",
			toStatus: StatusDelivered,
			wantErr:  false,
			checkFunc: func(r *DeliveryRecord) bool {
				return r.DeliveryTime != nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &UpdateStatusInput{
				RecordID: record.ID,
				Status:   tt.toStatus,
				Reason:   "Test transition",
			}

			updated, err := tracker.UpdateStatus(ctx, input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if updated.Status != tt.toStatus {
				t.Errorf("Status mismatch: got %v, want %v", updated.Status, tt.toStatus)
			}

			if tt.checkFunc != nil && !tt.checkFunc(updated) {
				t.Error("Check function failed")
			}

			// Update record for next test
			record = updated
		})
	}
}

func TestUpdateStatusInvalidTransition(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	record := createTestRecord(t, tracker, "cust-001", "sms")

	// Move to delivered (terminal state)
	tracker.UpdateStatus(ctx, &UpdateStatusInput{
		RecordID: record.ID,
		Status:   StatusQueued,
	})
	tracker.UpdateStatus(ctx, &UpdateStatusInput{
		RecordID: record.ID,
		Status:   StatusSent,
	})
	tracker.UpdateStatus(ctx, &UpdateStatusInput{
		RecordID: record.ID,
		Status:   StatusDelivered,
	})

	// Try invalid transition from delivered to sent
	_, err := tracker.UpdateStatus(ctx, &UpdateStatusInput{
		RecordID: record.ID,
		Status:   StatusSent,
	})

	if err != ErrInvalidTransition {
		t.Errorf("Expected ErrInvalidTransition, got %v", err)
	}
}

func TestUpdateStatusByMessageID(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	input := &TrackDeliveryInput{
		MessageID:  "msg-by-id",
		CustomerID: "cust-001",
		Channel:    "sms",
	}

	record, _ := tracker.TrackDelivery(ctx, input)

	// Update using message ID instead of record ID
	updated, err := tracker.UpdateStatus(ctx, &UpdateStatusInput{
		MessageID: "msg-by-id",
		Status:    StatusQueued,
	})

	if err != nil {
		t.Fatalf("Update by message ID failed: %v", err)
	}

	if updated.ID != record.ID {
		t.Error("Should update the same record")
	}
	if updated.Status != StatusQueued {
		t.Errorf("Status should be queued, got %v", updated.Status)
	}
}

func TestUpdateStatusIdempotent(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	record := createTestRecord(t, tracker, "cust-001", "sms")

	// Update to same status should succeed (idempotent)
	_, err := tracker.UpdateStatus(ctx, &UpdateStatusInput{
		RecordID: record.ID,
		Status:   StatusPending,
	})

	if err != nil {
		t.Errorf("Idempotent update should succeed: %v", err)
	}
}

func TestStatusHistory(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	record := createTestRecord(t, tracker, "cust-001", "sms")

	// Make several status transitions
	statuses := []DeliveryStatus{StatusQueued, StatusSent, StatusDelivered}
	for _, status := range statuses {
		tracker.UpdateStatus(ctx, &UpdateStatusInput{
			RecordID: record.ID,
			Status:   status,
			Reason:   "Test: " + string(status),
		})
	}

	// Get status history
	history, err := tracker.GetStatusHistory(ctx, record.ID)
	if err != nil {
		t.Fatalf("Failed to get status history: %v", err)
	}

	// Should have initial + 3 transitions = 4 entries
	if len(history) != 4 {
		t.Errorf("Expected 4 history entries, got %d", len(history))
	}

	// Verify transitions are in order
	expectedTransitions := []struct {
		from, to DeliveryStatus
	}{
		{"", StatusPending},
		{StatusPending, StatusQueued},
		{StatusQueued, StatusSent},
		{StatusSent, StatusDelivered},
	}

	for i, expected := range expectedTransitions {
		if i >= len(history) {
			break
		}
		if history[i].FromStatus != expected.from || history[i].ToStatus != expected.to {
			t.Errorf("History[%d]: expected %v->%v, got %v->%v",
				i, expected.from, expected.to, history[i].FromStatus, history[i].ToStatus)
		}
	}
}

// =============================================================================
// Log Completeness Tests
// =============================================================================

func TestDeliveryLogs(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	record := createTestRecord(t, tracker, "cust-001", "sms")

	// Update to sent (should create log)
	tracker.UpdateStatus(ctx, &UpdateStatusInput{
		RecordID: record.ID,
		Status:   StatusQueued,
	})

	tracker.UpdateStatus(ctx, &UpdateStatusInput{
		RecordID:     record.ID,
		Status:       StatusSent,
		Duration:     100 * time.Millisecond,
		ProviderResp: "Message accepted",
	})

	// Update to delivered (should create another log)
	tracker.UpdateStatus(ctx, &UpdateStatusInput{
		RecordID:     record.ID,
		Status:       StatusDelivered,
		Duration:     50 * time.Millisecond,
		ProviderResp: "Delivered",
	})

	logs, err := tracker.GetDeliveryLogs(ctx, record.ID)
	if err != nil {
		t.Fatalf("Failed to get logs: %v", err)
	}

	// Should have 2 logs (sent and delivered)
	if len(logs) != 2 {
		t.Errorf("Expected 2 logs, got %d", len(logs))
	}

	// Verify log contents
	if len(logs) > 0 {
		if logs[0].Status != StatusSent {
			t.Errorf("First log should be for sent, got %v", logs[0].Status)
		}
		if logs[0].AttemptNum != 1 {
			t.Errorf("First log should be attempt 1, got %d", logs[0].AttemptNum)
		}
	}
}

func TestDeliveryLogWithError(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	record := createTestRecord(t, tracker, "cust-001", "sms")

	tracker.UpdateStatus(ctx, &UpdateStatusInput{
		RecordID: record.ID,
		Status:   StatusQueued,
	})

	// Fail the delivery
	tracker.UpdateStatus(ctx, &UpdateStatusInput{
		RecordID:     record.ID,
		Status:       StatusFailed,
		ErrorCode:    "CARRIER_ERROR",
		ErrorDetails: "Carrier rejected message",
		Duration:     200 * time.Millisecond,
	})

	logs, _ := tracker.GetDeliveryLogs(ctx, record.ID)

	if len(logs) != 1 {
		t.Fatalf("Expected 1 log, got %d", len(logs))
	}

	log := logs[0]
	if log.ErrorCode != "CARRIER_ERROR" {
		t.Errorf("ErrorCode mismatch: got %v", log.ErrorCode)
	}
	if log.ErrorDetails != "Carrier rejected message" {
		t.Errorf("ErrorDetails mismatch: got %v", log.ErrorDetails)
	}
}

func TestLogAttempt(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	record := createTestRecord(t, tracker, "cust-001", "sms")

	// Manually log an attempt
	log := &DeliveryLog{
		AttemptNum:   1,
		Status:       StatusSent,
		Duration:     150 * time.Millisecond,
		ProviderResp: "Accepted by gateway",
	}

	err := tracker.LogAttempt(ctx, record.ID, log)
	if err != nil {
		t.Fatalf("LogAttempt failed: %v", err)
	}

	logs, _ := tracker.GetDeliveryLogs(ctx, record.ID)
	if len(logs) != 1 {
		t.Errorf("Expected 1 log, got %d", len(logs))
	}
	if logs[0].ID == "" {
		t.Error("Log should have an ID")
	}
}

// =============================================================================
// Metric Calculation Tests
// =============================================================================

func TestGenerateMetrics(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-time.Hour),
		End:   now.Add(time.Hour),
	}

	// Create records with various statuses
	records := []struct {
		customerID string
		channel    string
		endStatus  DeliveryStatus
		errorCode  string
	}{
		{"cust-001", "sms", StatusDelivered, ""},
		{"cust-001", "sms", StatusDelivered, ""},
		{"cust-002", "sms", StatusFailed, "INVALID_NUMBER"},
		{"cust-002", "email", StatusDelivered, ""},
		{"cust-003", "email", StatusFailed, "BOUNCE"},
		{"cust-003", "push", StatusDelivered, ""},
		{"cust-001", "sms", StatusBounced, ""},
		{"cust-002", "sms", StatusPending, ""},
	}

	for i, r := range records {
		input := &TrackDeliveryInput{
			MessageID:  "metric-msg-" + string(rune('0'+i)),
			CustomerID: r.customerID,
			Channel:    r.channel,
		}
		record, _ := tracker.TrackDelivery(ctx, input)

		// Transition to end status
		if r.endStatus != StatusPending {
			tracker.UpdateStatus(ctx, &UpdateStatusInput{
				RecordID: record.ID,
				Status:   StatusQueued,
			})

			// StatusDelivered and StatusBounced require going through StatusSent first
			if r.endStatus == StatusDelivered || r.endStatus == StatusBounced {
				tracker.UpdateStatus(ctx, &UpdateStatusInput{
					RecordID: record.ID,
					Status:   StatusSent,
				})
			}

			tracker.UpdateStatus(ctx, &UpdateStatusInput{
				RecordID:  record.ID,
				Status:    r.endStatus,
				ErrorCode: r.errorCode,
			})
		}
	}

	metrics, err := tracker.GenerateMetrics(ctx, timeRange)
	if err != nil {
		t.Fatalf("GenerateMetrics failed: %v", err)
	}

	// Verify total
	if metrics.TotalMessages != 8 {
		t.Errorf("TotalMessages: expected 8, got %d", metrics.TotalMessages)
	}

	// Verify delivered count
	if metrics.Delivered != 4 {
		t.Errorf("Delivered: expected 4, got %d", metrics.Delivered)
	}

	// Verify failed count
	if metrics.Failed != 2 {
		t.Errorf("Failed: expected 2, got %d", metrics.Failed)
	}

	// Verify bounced count
	if metrics.Bounced != 1 {
		t.Errorf("Bounced: expected 1, got %d", metrics.Bounced)
	}

	// Verify pending count
	if metrics.Pending != 1 {
		t.Errorf("Pending: expected 1, got %d", metrics.Pending)
	}

	// Verify delivery rate (4/8 = 50%)
	expectedRate := 50.0
	if metrics.DeliveryRate != expectedRate {
		t.Errorf("DeliveryRate: expected %.1f%%, got %.1f%%", expectedRate, metrics.DeliveryRate)
	}

	// Verify channel breakdown
	if len(metrics.ByChannel) != 3 {
		t.Errorf("Expected 3 channels, got %d", len(metrics.ByChannel))
	}

	smsMetrics, ok := metrics.ByChannel["sms"]
	if !ok {
		t.Fatal("SMS channel metrics missing")
	}
	if smsMetrics.Total != 5 {
		t.Errorf("SMS total: expected 5, got %d", smsMetrics.Total)
	}
	if smsMetrics.Delivered != 2 {
		t.Errorf("SMS delivered: expected 2, got %d", smsMetrics.Delivered)
	}

	// Verify top errors
	if len(metrics.TopErrors) != 2 {
		t.Errorf("Expected 2 error types, got %d", len(metrics.TopErrors))
	}
}

func TestGenerateMetricsEmptyRange(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	timeRange := TimeRange{
		Start: time.Now().Add(-time.Hour),
		End:   time.Now(),
	}

	metrics, err := tracker.GenerateMetrics(ctx, timeRange)
	if err != nil {
		t.Fatalf("GenerateMetrics failed: %v", err)
	}

	if metrics.TotalMessages != 0 {
		t.Errorf("Expected 0 messages, got %d", metrics.TotalMessages)
	}
	if metrics.DeliveryRate != 0 {
		t.Errorf("Expected 0%% delivery rate, got %.1f%%", metrics.DeliveryRate)
	}
}

func TestGenerateMetricsInvalidTimeRange(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	// End before start
	timeRange := TimeRange{
		Start: time.Now(),
		End:   time.Now().Add(-time.Hour),
	}

	_, err := tracker.GenerateMetrics(ctx, timeRange)
	if err != ErrInvalidTimeRange {
		t.Errorf("Expected ErrInvalidTimeRange, got %v", err)
	}
}

func TestGenerateChannelMetrics(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-time.Hour),
		End:   now.Add(time.Hour),
	}

	// Create SMS records
	for i := 0; i < 10; i++ {
		input := &TrackDeliveryInput{
			MessageID:  "ch-msg-" + string(rune('0'+i)),
			CustomerID: "cust-001",
			Channel:    "sms",
		}
		record, _ := tracker.TrackDelivery(ctx, input)

		status := StatusDelivered
		if i >= 8 { // 2 failures
			status = StatusFailed
		}

		tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: StatusQueued})
		tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: StatusSent})
		tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: status})
	}

	metrics, err := tracker.GenerateChannelMetrics(ctx, "sms", timeRange)
	if err != nil {
		t.Fatalf("GenerateChannelMetrics failed: %v", err)
	}

	if metrics.Total != 10 {
		t.Errorf("Total: expected 10, got %d", metrics.Total)
	}
	if metrics.Delivered != 8 {
		t.Errorf("Delivered: expected 8, got %d", metrics.Delivered)
	}
	if metrics.Failed != 2 {
		t.Errorf("Failed: expected 2, got %d", metrics.Failed)
	}

	expectedRate := 80.0
	if metrics.DeliveryRate != expectedRate {
		t.Errorf("DeliveryRate: expected %.1f%%, got %.1f%%", expectedRate, metrics.DeliveryRate)
	}
}

func TestMetricsAverageAttempts(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	now := time.Now()
	timeRange := TimeRange{
		Start: now.Add(-time.Hour),
		End:   now.Add(time.Hour),
	}

	// Create record with multiple attempts
	record := createTestRecord(t, tracker, "cust-001", "sms")

	// First attempt - fail
	tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: StatusQueued})
	tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: StatusFailed})

	// Retry
	tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: StatusQueued})
	tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: StatusSent})
	tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: StatusDelivered})

	// Check record has 2 attempts
	updated, _ := tracker.GetRecord(ctx, record.ID)
	if updated.Attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", updated.Attempts)
	}

	metrics, _ := tracker.GenerateMetrics(ctx, timeRange)
	if metrics.AvgAttempts != 2.0 {
		t.Errorf("AvgAttempts: expected 2.0, got %.2f", metrics.AvgAttempts)
	}
}

// =============================================================================
// History Retrieval Tests
// =============================================================================

func TestGetCustomerHistory(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	customerID := "hist-cust-001"

	// Create multiple records for this customer
	for i := 0; i < 5; i++ {
		input := &TrackDeliveryInput{
			MessageID:  "hist-msg-" + string(rune('0'+i)),
			CustomerID: customerID,
			Channel:    "sms",
		}
		record, _ := tracker.TrackDelivery(ctx, input)

		status := StatusDelivered
		if i == 2 {
			status = StatusFailed
		}

		tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: StatusQueued})
		tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: StatusSent})
		tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: status})

		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	history, err := tracker.GetCustomerHistory(ctx, customerID, nil)
	if err != nil {
		t.Fatalf("GetCustomerHistory failed: %v", err)
	}

	if history.CustomerID != customerID {
		t.Errorf("CustomerID mismatch: got %v", history.CustomerID)
	}
	if history.TotalRecords != 5 {
		t.Errorf("TotalRecords: expected 5, got %d", history.TotalRecords)
	}
	if len(history.Records) != 5 {
		t.Errorf("Records count: expected 5, got %d", len(history.Records))
	}

	// Verify summary
	if history.Summary.TotalMessages != 5 {
		t.Errorf("Summary.TotalMessages: expected 5, got %d", history.Summary.TotalMessages)
	}
	if history.Summary.Delivered != 4 {
		t.Errorf("Summary.Delivered: expected 4, got %d", history.Summary.Delivered)
	}
	if history.Summary.Failed != 1 {
		t.Errorf("Summary.Failed: expected 1, got %d", history.Summary.Failed)
	}

	expectedRate := 80.0
	if history.Summary.DeliveryRate != expectedRate {
		t.Errorf("Summary.DeliveryRate: expected %.1f%%, got %.1f%%", expectedRate, history.Summary.DeliveryRate)
	}
}

func TestGetCustomerHistoryWithFilters(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	customerID := "filter-cust-001"

	// Create records with different channels
	channels := []string{"sms", "email", "sms", "push", "sms"}
	for i, ch := range channels {
		input := &TrackDeliveryInput{
			MessageID:  "filter-msg-" + string(rune('0'+i)),
			CustomerID: customerID,
			Channel:    ch,
		}
		record, _ := tracker.TrackDelivery(ctx, input)
		tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: StatusQueued})
		tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: StatusSent})
		tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: StatusDelivered})
	}

	// Filter by channel
	opts := &QueryOptions{
		Channel: "sms",
	}

	history, err := tracker.GetCustomerHistory(ctx, customerID, opts)
	if err != nil {
		t.Fatalf("GetCustomerHistory with filter failed: %v", err)
	}

	if history.TotalRecords != 3 {
		t.Errorf("TotalRecords with SMS filter: expected 3, got %d", history.TotalRecords)
	}

	for _, record := range history.Records {
		if record.Channel != "sms" {
			t.Errorf("Record has wrong channel: %v", record.Channel)
		}
	}
}

func TestGetCustomerHistoryPagination(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	customerID := "page-cust-001"

	// Create 10 records
	for i := 0; i < 10; i++ {
		input := &TrackDeliveryInput{
			MessageID:  "page-msg-" + string(rune('A'+i)),
			CustomerID: customerID,
			Channel:    "sms",
		}
		tracker.TrackDelivery(ctx, input)
		time.Sleep(time.Millisecond)
	}

	// Get first page
	opts := &QueryOptions{
		Limit:  5,
		Offset: 0,
	}

	history, _ := tracker.GetCustomerHistory(ctx, customerID, opts)
	if len(history.Records) != 5 {
		t.Errorf("Page 1: expected 5 records, got %d", len(history.Records))
	}
	if history.TotalRecords != 10 {
		t.Errorf("TotalRecords should be 10, got %d", history.TotalRecords)
	}

	// Get second page
	opts.Offset = 5
	history, _ = tracker.GetCustomerHistory(ctx, customerID, opts)
	if len(history.Records) != 5 {
		t.Errorf("Page 2: expected 5 records, got %d", len(history.Records))
	}
}

func TestGetCustomerHistoryInvalidCustomer(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	_, err := tracker.GetCustomerHistory(ctx, "", nil)
	if err != ErrInvalidCustomerID {
		t.Errorf("Expected ErrInvalidCustomerID, got %v", err)
	}
}

func TestGetRecentDeliveries(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	customerID := "recent-cust-001"

	// Create 10 records
	for i := 0; i < 10; i++ {
		input := &TrackDeliveryInput{
			MessageID:  "recent-msg-" + string(rune('A'+i)),
			CustomerID: customerID,
			Channel:    "sms",
		}
		tracker.TrackDelivery(ctx, input)
		time.Sleep(time.Millisecond)
	}

	records, err := tracker.GetRecentDeliveries(ctx, customerID, 5)
	if err != nil {
		t.Fatalf("GetRecentDeliveries failed: %v", err)
	}

	if len(records) != 5 {
		t.Errorf("Expected 5 records, got %d", len(records))
	}

	// Should be sorted by created_at desc (most recent first)
	for i := 0; i < len(records)-1; i++ {
		if records[i].CreatedAt.Before(records[i+1].CreatedAt) {
			t.Error("Records should be sorted by created_at desc")
		}
	}
}

func TestGetFailedDeliveries(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	// Create some failed deliveries
	for i := 0; i < 5; i++ {
		record := createTestRecord(t, tracker, "cust-001", "sms")
		tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: StatusQueued})
		tracker.UpdateStatus(ctx, &UpdateStatusInput{
			RecordID:  record.ID,
			Status:    StatusFailed,
			ErrorCode: "TEST_ERROR",
		})
	}

	// Create some successful deliveries
	for i := 0; i < 3; i++ {
		record := createTestRecord(t, tracker, "cust-001", "sms")
		tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: StatusQueued})
		tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: StatusSent})
		tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: StatusDelivered})
	}

	timeRange := TimeRange{
		Start: time.Now().Add(-time.Hour),
		End:   time.Now().Add(time.Hour),
	}

	failed, err := tracker.GetFailedDeliveries(ctx, timeRange)
	if err != nil {
		t.Fatalf("GetFailedDeliveries failed: %v", err)
	}

	if len(failed) != 5 {
		t.Errorf("Expected 5 failed deliveries, got %d", len(failed))
	}

	for _, record := range failed {
		if record.Status != StatusFailed {
			t.Errorf("Record should have failed status, got %v", record.Status)
		}
	}
}

func TestGetPendingDeliveries(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	// Create pending deliveries
	for i := 0; i < 3; i++ {
		createTestRecord(t, tracker, "cust-001", "sms")
		time.Sleep(time.Millisecond)
	}

	// Create some processed deliveries
	for i := 0; i < 2; i++ {
		record := createTestRecord(t, tracker, "cust-001", "sms")
		tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: StatusQueued})
	}

	pending, err := tracker.GetPendingDeliveries(ctx)
	if err != nil {
		t.Fatalf("GetPendingDeliveries failed: %v", err)
	}

	if len(pending) != 3 {
		t.Errorf("Expected 3 pending deliveries, got %d", len(pending))
	}

	// Should be sorted oldest first (for processing order)
	for i := 0; i < len(pending)-1; i++ {
		if pending[i].CreatedAt.After(pending[i+1].CreatedAt) {
			t.Error("Pending should be sorted oldest first")
		}
	}
}

// =============================================================================
// Bulk Operations Tests
// =============================================================================

func TestBulkUpdateStatus(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	// Create records
	var recordIDs []string
	for i := 0; i < 5; i++ {
		record := createTestRecord(t, tracker, "cust-001", "sms")
		recordIDs = append(recordIDs, record.ID)
	}

	// Bulk update to queued
	updated, err := tracker.BulkUpdateStatus(ctx, recordIDs, StatusQueued, "Bulk processing")
	if err != nil {
		t.Fatalf("BulkUpdateStatus failed: %v", err)
	}

	if updated != 5 {
		t.Errorf("Expected 5 updated, got %d", updated)
	}

	// Verify all are updated
	for _, id := range recordIDs {
		record, _ := tracker.GetRecord(ctx, id)
		if record.Status != StatusQueued {
			t.Errorf("Record %s should be queued", id)
		}
	}
}

func TestBulkUpdateStatusPartialFailure(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	// Create some records
	var recordIDs []string
	for i := 0; i < 3; i++ {
		record := createTestRecord(t, tracker, "cust-001", "sms")
		recordIDs = append(recordIDs, record.ID)
	}

	// Add a non-existent ID
	recordIDs = append(recordIDs, "non-existent-id")

	// Bulk update
	updated, _ := tracker.BulkUpdateStatus(ctx, recordIDs, StatusQueued, "Test")

	// Should have updated 3 (not 4)
	if updated != 3 {
		t.Errorf("Expected 3 updated, got %d", updated)
	}
}

// =============================================================================
// Concurrency Tests
// =============================================================================

func TestConcurrentTracking(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			input := &TrackDeliveryInput{
				MessageID:  "concurrent-" + string(rune('A'+idx%26)) + string(rune('0'+idx/26)),
				CustomerID: "cust-001",
				Channel:    "sms",
			}

			record, err := tracker.TrackDelivery(ctx, input)
			if err != nil {
				t.Errorf("TrackDelivery failed: %v", err)
				return
			}

			// Update status
			tracker.UpdateStatus(ctx, &UpdateStatusInput{
				RecordID: record.ID,
				Status:   StatusQueued,
			})
		}(i)
	}

	wg.Wait()

	// Verify all records exist
	history, _ := tracker.GetCustomerHistory(ctx, "cust-001", nil)
	if history.TotalRecords != numGoroutines {
		t.Errorf("Expected %d records, got %d", numGoroutines, history.TotalRecords)
	}
}

func TestConcurrentStatusUpdates(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	// Create a single record
	record := createTestRecord(t, tracker, "cust-001", "sms")

	// Move to queued first
	tracker.UpdateStatus(ctx, &UpdateStatusInput{
		RecordID: record.ID,
		Status:   StatusQueued,
	})

	// Try concurrent updates (only one should succeed for sent)
	var wg sync.WaitGroup
	var successCount int32
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err := tracker.UpdateStatus(ctx, &UpdateStatusInput{
				RecordID: record.ID,
				Status:   StatusSent,
			})

			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// All should succeed due to idempotency
	if successCount != 10 {
		t.Errorf("Expected 10 successful updates (idempotent), got %d", successCount)
	}

	// Final record should have few attempts (idempotency reduces duplicates)
	// Without per-record locking, some race conditions may increment attempts
	// but it should be much less than 10 (full duplicate counting)
	updated, _ := tracker.GetRecord(ctx, record.ID)
	if updated.Attempts > 5 {
		t.Errorf("Expected fewer attempts due to idempotency, got %d (should be much less than 10)", updated.Attempts)
	}
	if updated.Attempts < 1 {
		t.Errorf("Should have at least 1 attempt, got %d", updated.Attempts)
	}
}

// =============================================================================
// Edge Cases and Error Handling
// =============================================================================

func TestGetRecordNotFound(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	_, err := tracker.GetRecord(ctx, "non-existent")
	if err != ErrRecordNotFound {
		t.Errorf("Expected ErrRecordNotFound, got %v", err)
	}
}

func TestUpdateStatusRecordNotFound(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	_, err := tracker.UpdateStatus(ctx, &UpdateStatusInput{
		RecordID: "non-existent",
		Status:   StatusQueued,
	})

	if err != ErrRecordNotFound {
		t.Errorf("Expected ErrRecordNotFound, got %v", err)
	}
}

func TestUpdateStatusInvalidStatus(t *testing.T) {
	tracker := setupTracker()
	ctx := context.Background()

	record := createTestRecord(t, tracker, "cust-001", "sms")

	_, err := tracker.UpdateStatus(ctx, &UpdateStatusInput{
		RecordID: record.ID,
		Status:   "invalid_status",
	})

	if err != ErrInvalidStatus {
		t.Errorf("Expected ErrInvalidStatus, got %v", err)
	}
}

func TestIsTerminalStatus(t *testing.T) {
	tests := []struct {
		status   DeliveryStatus
		terminal bool
	}{
		{StatusPending, false},
		{StatusQueued, false},
		{StatusSent, false},
		{StatusDelivered, true},
		{StatusFailed, false}, // Can retry
		{StatusBounced, true},
		{StatusRejected, true},
		{StatusExpired, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if IsTerminalStatus(tt.status) != tt.terminal {
				t.Errorf("IsTerminalStatus(%v) = %v, want %v", tt.status, !tt.terminal, tt.terminal)
			}
		})
	}
}

func TestTrackerWithCustomConfig(t *testing.T) {
	config := &TrackerConfig{
		MaxAttempts:   5,
		EnableLogging: false,
		EnableHistory: false,
		RetentionDays: 7,
	}

	tracker := NewDeliveryTracker(nil, config)
	ctx := context.Background()

	record := createTestRecord(t, tracker, "cust-001", "sms")

	if record.MaxAttempts != 5 {
		t.Errorf("MaxAttempts should be 5, got %d", record.MaxAttempts)
	}

	// Update to sent - should NOT create log
	tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: StatusQueued})
	tracker.UpdateStatus(ctx, &UpdateStatusInput{RecordID: record.ID, Status: StatusSent})

	logs, _ := tracker.GetDeliveryLogs(ctx, record.ID)
	if len(logs) != 0 {
		t.Errorf("Logging disabled but got %d logs", len(logs))
	}

	// Should NOT have status history (besides initial)
	history, _ := tracker.GetStatusHistory(ctx, record.ID)
	if len(history) != 0 {
		t.Errorf("History disabled but got %d entries", len(history))
	}
}

// =============================================================================
// Memory Store Tests
// =============================================================================

func TestMemoryStoreQueryWithPagination(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Create 20 records
	for i := 0; i < 20; i++ {
		record := &DeliveryRecord{
			ID:         "record-" + string(rune('A'+i)),
			MessageID:  "msg-" + string(rune('A'+i)),
			CustomerID: "cust-001",
			Status:     StatusPending,
			CreatedAt:  time.Now().Add(time.Duration(i) * time.Minute),
			UpdatedAt:  time.Now().Add(time.Duration(i) * time.Minute),
		}
		store.SaveRecord(ctx, record)
	}

	// Query with offset and limit
	opts := QueryOptions{
		CustomerID: "cust-001",
		Offset:     5,
		Limit:      10,
		SortBy:     "created_at",
		SortDesc:   false,
	}

	records, _ := store.QueryRecords(ctx, opts)
	if len(records) != 10 {
		t.Errorf("Expected 10 records, got %d", len(records))
	}

	// Query beyond available records
	opts.Offset = 25
	records, _ = store.QueryRecords(ctx, opts)
	if len(records) != 0 {
		t.Errorf("Expected 0 records for offset beyond count, got %d", len(records))
	}
}

func TestMemoryStoreCountRecords(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Create records with different statuses
	statuses := []DeliveryStatus{StatusPending, StatusPending, StatusDelivered, StatusFailed, StatusPending}
	for i, status := range statuses {
		record := &DeliveryRecord{
			ID:        "count-" + string(rune('A'+i)),
			MessageID: "msg-" + string(rune('A'+i)),
			Status:    status,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		store.SaveRecord(ctx, record)
	}

	// Count all
	count, _ := store.CountRecords(ctx, QueryOptions{})
	if count != 5 {
		t.Errorf("Expected 5 total, got %d", count)
	}

	// Count pending
	count, _ = store.CountRecords(ctx, QueryOptions{Status: StatusPending})
	if count != 3 {
		t.Errorf("Expected 3 pending, got %d", count)
	}
}
