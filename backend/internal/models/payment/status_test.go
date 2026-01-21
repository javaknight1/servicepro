package payment

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPaymentStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status PaymentStatus
		want   bool
	}{
		{"Valid status - pending", StatusPending, true},
		{"Valid status - succeeded", StatusSucceeded, true},
		{"Valid status - failed", StatusFailed, true},
		{"Valid status - refunded", StatusRefunded, true},
		{"Invalid status", PaymentStatus("invalid"), false},
		{"Empty status", PaymentStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.IsValid())
		})
	}
}

func TestPaymentStatus_GetCategory(t *testing.T) {
	tests := []struct {
		name   string
		status PaymentStatus
		want   StatusCategory
	}{
		{"Pending -> Initial", StatusPending, CategoryInitial},
		{"Awaiting Payment -> Initial", StatusAwaitingPayment, CategoryInitial},
		{"Processing -> In Progress", StatusProcessing, CategoryInProgress},
		{"Authorized -> In Progress", StatusAuthorized, CategoryInProgress},
		{"Succeeded -> Success", StatusSucceeded, CategorySuccess},
		{"Paid -> Success", StatusPaid, CategorySuccess},
		{"Failed -> Failure", StatusFailed, CategoryFailure},
		{"Declined -> Failure", StatusDeclined, CategoryFailure},
		{"Refunded -> Refund", StatusRefunded, CategoryRefund},
		{"Partial Refund -> Refund", StatusPartiallyRefunded, CategoryRefund},
		{"Disputed -> Dispute", StatusDisputed, CategoryDispute},
		{"Charged Back -> Dispute", StatusChargedBack, CategoryDispute},
		{"On Hold -> OnHold", StatusOnHold, CategoryOnHold},
		{"Under Review -> OnHold", StatusUnderReview, CategoryOnHold},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.GetCategory())
		})
	}
}

func TestPaymentStatus_IsFinal(t *testing.T) {
	tests := []struct {
		name   string
		status PaymentStatus
		want   bool
	}{
		{"Succeeded is final", StatusSucceeded, true},
		{"Paid is final", StatusPaid, true},
		{"Failed is final", StatusFailed, true},
		{"Declined is final", StatusDeclined, true},
		{"Expired is final", StatusExpired, true},
		{"Canceled is final", StatusCanceled, true},
		{"Refunded is final", StatusRefunded, true},
		{"Charged Back is final", StatusChargedBack, true},
		{"Pending is not final", StatusPending, false},
		{"Processing is not final", StatusProcessing, false},
		{"Authorized is not final", StatusAuthorized, false},
		{"On Hold is not final", StatusOnHold, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.IsFinal())
		})
	}
}

func TestPaymentStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name       string
		fromStatus PaymentStatus
		toStatus   PaymentStatus
		want       bool
	}{
		// Valid transitions from Pending
		{"Pending to Awaiting Payment", StatusPending, StatusAwaitingPayment, true},
		{"Pending to Processing", StatusPending, StatusProcessing, true},
		{"Pending to Canceled", StatusPending, StatusCanceled, true},
		{"Pending to Failed", StatusPending, StatusFailed, true},
		{"Pending to Expired", StatusPending, StatusExpired, true},

		// Invalid transitions from Pending
		{"Pending to Succeeded (invalid)", StatusPending, StatusSucceeded, false},
		{"Pending to Refunded (invalid)", StatusPending, StatusRefunded, false},

		// Valid transitions from Processing
		{"Processing to Authorized", StatusProcessing, StatusAuthorized, true},
		{"Processing to Succeeded", StatusProcessing, StatusSucceeded, true},
		{"Processing to Failed", StatusProcessing, StatusFailed, true},
		{"Processing to Requires Action", StatusProcessing, StatusRequiresAction, true},

		// Invalid transitions from Processing
		{"Processing to Pending (invalid)", StatusProcessing, StatusPending, false},
		{"Processing to Refunded (invalid)", StatusProcessing, StatusRefunded, false},

		// Valid transitions from Authorized
		{"Authorized to Captured", StatusAuthorized, StatusCaptured, true},
		{"Authorized to Succeeded", StatusAuthorized, StatusSucceeded, true},
		{"Authorized to Canceled", StatusAuthorized, StatusCanceled, true},
		{"Authorized to Expired", StatusAuthorized, StatusExpired, true},

		// Valid transitions from Succeeded
		{"Succeeded to Refund Requested", StatusSucceeded, StatusRefundRequested, true},
		{"Succeeded to Disputed", StatusSucceeded, StatusDisputed, true},
		{"Succeeded to Partially Refunded", StatusSucceeded, StatusPartiallyRefunded, true},
		{"Succeeded to Refunded", StatusSucceeded, StatusRefunded, true},

		// Final state transitions (mostly invalid)
		{"Failed to Processing (invalid)", StatusFailed, StatusProcessing, false},
		{"Canceled to Succeeded (invalid)", StatusCanceled, StatusSucceeded, false},
		{"Refunded to Processing (invalid)", StatusRefunded, StatusProcessing, false},

		// Refund flow
		{"Refund Requested to Refund Processing", StatusRefundRequested, StatusRefundProcessing, true},
		{"Refund Processing to Refunded", StatusRefundProcessing, StatusRefunded, true},
		{"Refund Processing to Partially Refunded", StatusRefundProcessing, StatusPartiallyRefunded, true},
		{"Partially Refunded to Refunded", StatusPartiallyRefunded, StatusRefunded, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.fromStatus.CanTransitionTo(tt.toStatus))
		})
	}
}

func TestStatusTransition_Validate(t *testing.T) {
	tests := []struct {
		name       string
		transition *StatusTransition
		wantErr    bool
		errMsg     string
	}{
		{
			name: "Valid transition",
			transition: &StatusTransition{
				PaymentID:  newTestUUID(),
				FromStatus: StatusPending,
				ToStatus:   StatusProcessing,
			},
			wantErr: false,
		},
		{
			name: "Missing payment ID",
			transition: &StatusTransition{
				ToStatus: StatusProcessing,
			},
			wantErr: true,
			errMsg:  "payment_id is required",
		},
		{
			name: "Invalid to status",
			transition: &StatusTransition{
				PaymentID: newTestUUID(),
				ToStatus:  PaymentStatus("invalid"),
			},
			wantErr: true,
			errMsg:  "invalid target status",
		},
		{
			name: "Invalid from status",
			transition: &StatusTransition{
				PaymentID:  newTestUUID(),
				FromStatus: PaymentStatus("invalid"),
				ToStatus:   StatusProcessing,
			},
			wantErr: true,
			errMsg:  "invalid from status",
		},
		{
			name: "Invalid transition without force",
			transition: &StatusTransition{
				PaymentID:  newTestUUID(),
				FromStatus: StatusSucceeded,
				ToStatus:   StatusPending,
				Force:      false,
			},
			wantErr: true,
			errMsg:  "invalid status transition",
		},
		{
			name: "Invalid transition with force",
			transition: &StatusTransition{
				PaymentID:  newTestUUID(),
				FromStatus: StatusSucceeded,
				ToStatus:   StatusPending,
				Force:      true,
			},
			wantErr: false, // Force bypasses validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.transition.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func newTestUUID() uuid.UUID {
	return uuid.New()
}

func TestStatusMetadata(t *testing.T) {
	metadata := StatusMetadata{
		Reason:          "Payment declined by bank",
		ErrorCode:       "card_declined",
		ErrorMessage:    "Your card was declined",
		ProviderStatus:  "declined",
		ProviderMessage: "Insufficient funds",
		CustomFields: map[string]string{
			"decline_code": "insufficient_funds",
			"bank_code":    "51",
		},
	}

	assert.Equal(t, "Payment declined by bank", metadata.Reason)
	assert.Equal(t, "card_declined", metadata.ErrorCode)
	assert.Equal(t, 2, len(metadata.CustomFields))
	assert.Equal(t, "insufficient_funds", metadata.CustomFields["decline_code"])
}

func TestPaymentStatusRecord(t *testing.T) {
	record := &PaymentStatusRecord{
		PaymentID: newTestUUID(),
		Status:    StatusProcessing,
		Version:   1,
	}

	// Test category is set correctly
	assert.Equal(t, CategoryInProgress, record.Status.GetCategory())

	// Test status string representation
	assert.Equal(t, "processing", record.Status.String())
}

func TestStatusQuery(t *testing.T) {
	paymentID1 := newTestUUID()
	paymentID2 := newTestUUID()

	query := &StatusQuery{
		PaymentIDs:   []uuid.UUID{paymentID1, paymentID2},
		Statuses:     []PaymentStatus{StatusPending, StatusProcessing},
		Categories:   []StatusCategory{CategoryInitial, CategoryInProgress},
		FinalOnly:    false,
		NonFinalOnly: true,
		Page:         1,
		PageSize:     20,
	}

	assert.Len(t, query.PaymentIDs, 2)
	assert.Len(t, query.Statuses, 2)
	assert.Len(t, query.Categories, 2)
	assert.True(t, query.NonFinalOnly)
	assert.False(t, query.FinalOnly)
}

func TestStatusStatistics(t *testing.T) {
	stats := &StatusStatistics{
		TotalPayments: 100,
		StatusCounts: map[PaymentStatus]int64{
			StatusSucceeded: 80,
			StatusFailed:    15,
			StatusPending:   5,
		},
		CategoryCounts: map[StatusCategory]int64{
			CategorySuccess: 80,
			CategoryFailure: 15,
			CategoryInitial: 5,
		},
		SuccessRate: 80.0,
		FailureRate: 15.0,
		RefundRate:  5.0,
		DisputeRate: 2.0,
	}

	assert.Equal(t, int64(100), stats.TotalPayments)
	assert.Equal(t, int64(80), stats.StatusCounts[StatusSucceeded])
	assert.Equal(t, 80.0, stats.SuccessRate)
	assert.Equal(t, int64(80), stats.CategoryCounts[CategorySuccess])
}

func TestStatusSummary(t *testing.T) {
	summary := &StatusSummary{
		PaymentID:       newTestUUID(),
		CurrentStatus:   StatusSucceeded,
		StatusCategory:  CategorySuccess,
		IsFinal:         true,
		TransitionCount: 5,
	}

	assert.True(t, summary.IsFinal)
	assert.Equal(t, StatusSucceeded, summary.CurrentStatus)
	assert.Equal(t, CategorySuccess, summary.StatusCategory)
	assert.Equal(t, 5, summary.TransitionCount)
}

func TestStatusHealth(t *testing.T) {
	health := &StatusHealth{
		IsHealthy:           false,
		StuckPaymentsCount:  10,
		OldPendingCount:     5,
		LongProcessingCount: 3,
		RecommendedActions: []string{
			"Review stuck payments",
			"Check payment processor status",
			"Review old pending payments",
		},
	}

	assert.False(t, health.IsHealthy)
	assert.Equal(t, int64(10), health.StuckPaymentsCount)
	assert.Len(t, health.RecommendedActions, 3)
}

// Benchmark tests
func BenchmarkPaymentStatus_CanTransitionTo(b *testing.B) {
	from := StatusPending
	to := StatusProcessing

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = from.CanTransitionTo(to)
	}
}

func BenchmarkPaymentStatus_GetCategory(b *testing.B) {
	status := StatusProcessing

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = status.GetCategory()
	}
}

func BenchmarkPaymentStatus_IsValid(b *testing.B) {
	status := StatusProcessing

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = status.IsValid()
	}
}
