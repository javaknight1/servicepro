package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

func setupPaymentTermsTestDB(t *testing.T) *gorm.DB {
	t.Skip("Skipping: requires PostgreSQL (SQLite doesn't support uuid and custom types)")
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate the schema
	err = db.AutoMigrate(&models.PaymentTerm{}, &models.Invoice{})
	require.NoError(t, err)

	return db
}

func createPaymentTermForTest(t *testing.T, db *gorm.DB, daysUntilDue int, discountPercent float64, discountDays int) *models.PaymentTerm {
	term := &models.PaymentTerm{
		Name:               "Test Net 30",
		Description:        "Test payment term",
		TermType:           models.PaymentTermNet30,
		DaysUntilDue:       daysUntilDue,
		DiscountPercentage: decimal.NewFromFloat(discountPercent),
		DiscountDays:       discountDays,
		LateFeePercentage:  decimal.NewFromFloat(5.0),
		LateFeeAmount:      decimal.NewFromFloat(25.00),
		GracePeriodDays:    3,
		IsActive:           true,
	}

	err := db.Create(term).Error
	require.NoError(t, err)
	return term
}

func TestPaymentTermsService_CreatePaymentTerm(t *testing.T) {
	db := setupPaymentTermsTestDB(t)
	service := NewPaymentTermsService(db)
	ctx := context.Background()

	t.Run("Create valid payment term", func(t *testing.T) {
		term := &models.PaymentTerm{
			Name:               "Net 30",
			Description:        "Payment due in 30 days",
			TermType:           models.PaymentTermNet30,
			DaysUntilDue:       30,
			DiscountPercentage: decimal.NewFromFloat(2.0),
			DiscountDays:       10,
			LateFeePercentage:  decimal.NewFromFloat(5.0),
			GracePeriodDays:    3,
			IsActive:           true,
		}

		err := service.CreatePaymentTerm(ctx, term)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, term.ID)
	})

	t.Run("Create term with invalid discount percentage", func(t *testing.T) {
		term := &models.PaymentTerm{
			Name:               "Invalid Discount",
			DaysUntilDue:       30,
			DiscountPercentage: decimal.NewFromFloat(150.0), // Invalid: > 100
			DiscountDays:       10,
		}

		err := service.CreatePaymentTerm(ctx, term)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "discount percentage must be between 0 and 100")
	})

	t.Run("Create term with invalid discount days", func(t *testing.T) {
		term := &models.PaymentTerm{
			Name:               "Invalid Discount Days",
			DaysUntilDue:       30,
			DiscountPercentage: decimal.NewFromFloat(2.0),
			DiscountDays:       40, // Invalid: > days_until_due
		}

		err := service.CreatePaymentTerm(ctx, term)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "discount days cannot exceed days until due")
	})

	t.Run("Create default payment term", func(t *testing.T) {
		term1 := &models.PaymentTerm{
			Name:         "Default 1",
			DaysUntilDue: 30,
			IsDefault:    true,
		}

		err := service.CreatePaymentTerm(ctx, term1)
		require.NoError(t, err)

		// Create another default term - should unset the first one
		term2 := &models.PaymentTerm{
			Name:         "Default 2",
			DaysUntilDue: 45,
			IsDefault:    true,
		}

		err = service.CreatePaymentTerm(ctx, term2)
		require.NoError(t, err)

		// Verify only term2 is default
		var updatedTerm1 models.PaymentTerm
		db.First(&updatedTerm1, term1.ID)
		assert.False(t, updatedTerm1.IsDefault)

		var updatedTerm2 models.PaymentTerm
		db.First(&updatedTerm2, term2.ID)
		assert.True(t, updatedTerm2.IsDefault)
	})
}

func TestPaymentTermsService_CalculateDueDate(t *testing.T) {
	db := setupPaymentTermsTestDB(t)
	service := NewPaymentTermsService(db)
	ctx := context.Background()

	term := createPaymentTermForTest(t, db, 30, 2.0, 10)

	tests := []struct {
		name      string
		issueDate time.Time
		timezone  string
		expected  time.Time
	}{
		{
			name:      "UTC timezone",
			issueDate: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			timezone:  "UTC",
			expected:  time.Date(2024, 2, 14, 10, 0, 0, 0, time.UTC),
		},
		{
			name:      "America/New_York timezone",
			issueDate: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			timezone:  "America/New_York",
			expected:  time.Date(2024, 2, 14, 5, 0, 0, 0, getLocation(t, "America/New_York")),
		},
		{
			name:      "Asia/Tokyo timezone",
			issueDate: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			timezone:  "Asia/Tokyo",
			expected:  time.Date(2024, 2, 14, 19, 0, 0, 0, getLocation(t, "Asia/Tokyo")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dueDate, err := service.CalculateDueDate(ctx, tt.issueDate, term.ID, tt.timezone)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, dueDate)
		})
	}
}

func TestPaymentTermsService_CalculateEarlyPaymentDiscount(t *testing.T) {
	db := setupPaymentTermsTestDB(t)
	service := NewPaymentTermsService(db)
	ctx := context.Background()

	term := createPaymentTermForTest(t, db, 30, 2.0, 10) // 2% discount within 10 days

	issueDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	invoiceAmount := decimal.NewFromFloat(1000.00)

	tests := []struct {
		name             string
		paymentDate      time.Time
		expectedDiscount string
		expectedEligible bool
	}{
		{
			name:             "Payment on day 5 - eligible",
			paymentDate:      issueDate.AddDate(0, 0, 5),
			expectedDiscount: "20.00",
			expectedEligible: true,
		},
		{
			name:             "Payment on day 10 - eligible (last day)",
			paymentDate:      issueDate.AddDate(0, 0, 10),
			expectedDiscount: "20.00",
			expectedEligible: true,
		},
		{
			name:             "Payment on day 11 - not eligible",
			paymentDate:      issueDate.AddDate(0, 0, 11),
			expectedDiscount: "0.00",
			expectedEligible: false,
		},
		{
			name:             "Payment on day 30 - not eligible",
			paymentDate:      issueDate.AddDate(0, 0, 30),
			expectedDiscount: "0.00",
			expectedEligible: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			discount, eligible, err := service.CalculateEarlyPaymentDiscount(
				ctx, invoiceAmount, tt.paymentDate, issueDate, term.ID,
			)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedDiscount, discount.String())
			assert.Equal(t, tt.expectedEligible, eligible)
		})
	}
}

func TestPaymentTermsService_CalculateLateFee(t *testing.T) {
	db := setupPaymentTermsTestDB(t)
	service := NewPaymentTermsService(db)
	ctx := context.Background()

	// Create term with 5% late fee + $25 fixed fee, 3-day grace period
	term := createPaymentTermForTest(t, db, 30, 2.0, 10)

	dueDate := time.Date(2024, 2, 14, 0, 0, 0, 0, time.UTC)
	invoiceAmount := decimal.NewFromFloat(1000.00)
	timezone := "UTC"

	tests := []struct {
		name               string
		currentDate        time.Time
		expectedLateFee    string
		expectedApplicable bool
	}{
		{
			name:               "Before due date",
			currentDate:        dueDate.AddDate(0, 0, -1),
			expectedLateFee:    "0.00",
			expectedApplicable: false,
		},
		{
			name:               "On due date",
			currentDate:        dueDate,
			expectedLateFee:    "0.00",
			expectedApplicable: false,
		},
		{
			name:               "Within grace period (day 1)",
			currentDate:        dueDate.AddDate(0, 0, 1),
			expectedLateFee:    "0.00",
			expectedApplicable: false,
		},
		{
			name:               "End of grace period (day 3)",
			currentDate:        dueDate.AddDate(0, 0, 3),
			expectedLateFee:    "0.00",
			expectedApplicable: false,
		},
		{
			name:               "After grace period (day 4) - late fee applies",
			currentDate:        dueDate.AddDate(0, 0, 4),
			expectedLateFee:    "75.00", // 5% of 1000 = 50 + 25 fixed = 75
			expectedApplicable: true,
		},
		{
			name:               "Way overdue (day 30)",
			currentDate:        dueDate.AddDate(0, 0, 30),
			expectedLateFee:    "75.00", // Same late fee applies
			expectedApplicable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lateFee, applicable, err := service.CalculateLateFee(
				ctx, invoiceAmount, tt.currentDate, dueDate, term.ID, timezone,
			)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedLateFee, lateFee.String())
			assert.Equal(t, tt.expectedApplicable, applicable)
		})
	}
}

func TestPaymentTermsService_GetPaymentStatus(t *testing.T) {
	db := setupPaymentTermsTestDB(t)
	service := NewPaymentTermsService(db)
	ctx := context.Background()

	term := createPaymentTermForTest(t, db, 30, 2.0, 10)

	issueDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	dueDate := time.Date(2024, 2, 14, 0, 0, 0, 0, time.UTC)
	invoiceAmount := decimal.NewFromFloat(1000.00)
	timezone := "UTC"

	tests := []struct {
		name                     string
		currentDate              time.Time
		expectedStatus           string
		expectedDiscountEligible bool
		expectedInGracePeriod    bool
		expectedLateFeeApplied   bool
		expectedNotification     bool
		expectedNotificationType string
	}{
		{
			name:                     "Early payment eligible (within discount period)",
			currentDate:              issueDate.AddDate(0, 0, 5),
			expectedStatus:           "early",
			expectedDiscountEligible: true,
			expectedInGracePeriod:    false,
			expectedLateFeeApplied:   false,
			expectedNotification:     true,
			expectedNotificationType: "early_payment_discount_available",
		},
		{
			name:                     "On time payment (after discount, before due)",
			currentDate:              issueDate.AddDate(0, 0, 20),
			expectedStatus:           "on_time",
			expectedDiscountEligible: false,
			expectedInGracePeriod:    false,
			expectedLateFeeApplied:   false,
			expectedNotification:     false,
		},
		{
			name:                     "Due soon (7 days before due)",
			currentDate:              dueDate.AddDate(0, 0, -7),
			expectedStatus:           "on_time",
			expectedDiscountEligible: false,
			expectedInGracePeriod:    false,
			expectedLateFeeApplied:   false,
			expectedNotification:     true,
			expectedNotificationType: "payment_due_soon",
		},
		{
			name:                     "In grace period",
			currentDate:              dueDate.AddDate(0, 0, 2),
			expectedStatus:           "grace_period",
			expectedDiscountEligible: false,
			expectedInGracePeriod:    true,
			expectedLateFeeApplied:   false,
			expectedNotification:     true,
			expectedNotificationType: "payment_in_grace_period",
		},
		{
			name:                     "Overdue (after grace period)",
			currentDate:              dueDate.AddDate(0, 0, 5),
			expectedStatus:           "overdue",
			expectedDiscountEligible: false,
			expectedInGracePeriod:    false,
			expectedLateFeeApplied:   true,
			expectedNotification:     true,
			expectedNotificationType: "payment_overdue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := service.GetPaymentStatus(
				ctx, invoiceAmount, issueDate, dueDate, term.ID, tt.currentDate, timezone,
			)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, status.Status)
			assert.Equal(t, tt.expectedDiscountEligible, status.DiscountEligible)
			assert.Equal(t, tt.expectedInGracePeriod, status.InGracePeriod)
			assert.Equal(t, tt.expectedLateFeeApplied, status.LateFeeApplied)
			assert.Equal(t, tt.expectedNotification, status.NotificationRequired)
			if tt.expectedNotification {
				assert.Equal(t, tt.expectedNotificationType, status.NotificationType)
			}
		})
	}
}

func TestPaymentTermsService_CalculatePaymentDetails(t *testing.T) {
	db := setupPaymentTermsTestDB(t)
	service := NewPaymentTermsService(db)
	ctx := context.Background()

	term := createPaymentTermForTest(t, db, 30, 2.0, 10)

	issueDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	invoiceAmount := decimal.NewFromFloat(1000.00)
	timezone := "UTC"

	t.Run("Calculate details without payment date", func(t *testing.T) {
		result, err := service.CalculatePaymentDetails(ctx, invoiceAmount, issueDate, term.ID, nil, timezone)
		require.NoError(t, err)

		expectedDueDate := time.Date(2024, 2, 14, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expectedDueDate, result.DueDate)
		assert.NotNil(t, result.DiscountAvailableUntil)
		assert.Equal(t, "20.00", result.DiscountAmount.String()) // Potential discount
		assert.Equal(t, "0.00", result.LateFeeAmount.String())
		assert.Equal(t, "1000.00", result.TotalDue.String())
	})

	t.Run("Calculate details with early payment", func(t *testing.T) {
		earlyPaymentDate := issueDate.AddDate(0, 0, 5)
		result, err := service.CalculatePaymentDetails(ctx, invoiceAmount, issueDate, term.ID, &earlyPaymentDate, timezone)
		require.NoError(t, err)

		assert.Equal(t, "20.00", result.DiscountAmount.String())
		assert.True(t, result.IsEarlyPaymentEligible)
		assert.Equal(t, "0.00", result.LateFeeAmount.String())
		assert.Equal(t, "980.00", result.TotalDue.String()) // 1000 - 20 discount
		assert.False(t, result.IsOverdue)
	})

	t.Run("Calculate details with late payment", func(t *testing.T) {
		dueDate := time.Date(2024, 2, 14, 0, 0, 0, 0, time.UTC)
		latePaymentDate := dueDate.AddDate(0, 0, 5) // 5 days late (past grace period)

		result, err := service.CalculatePaymentDetails(ctx, invoiceAmount, issueDate, term.ID, &latePaymentDate, timezone)
		require.NoError(t, err)

		assert.Equal(t, "0.00", result.DiscountAmount.String())
		assert.False(t, result.IsEarlyPaymentEligible)
		assert.Equal(t, "75.00", result.LateFeeAmount.String()) // 5% + $25
		assert.Equal(t, "1075.00", result.TotalDue.String())    // 1000 + 75 late fee
		assert.True(t, result.IsOverdue)
		assert.Equal(t, 5, result.DaysOverdue)
	})
}

func TestPaymentTermsService_TimeZoneHandling(t *testing.T) {
	db := setupPaymentTermsTestDB(t)
	service := NewPaymentTermsService(db)
	ctx := context.Background()

	term := createPaymentTermForTest(t, db, 30, 2.0, 10)

	// Issue date at midnight UTC
	issueDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	timezones := []string{
		"UTC",
		"America/New_York",
		"America/Los_Angeles",
		"Europe/London",
		"Asia/Tokyo",
		"Australia/Sydney",
	}

	for _, tz := range timezones {
		t.Run("Timezone: "+tz, func(t *testing.T) {
			dueDate, err := service.CalculateDueDate(ctx, issueDate, term.ID, tz)
			require.NoError(t, err)

			// Verify the due date is 30 days later
			duration := dueDate.Sub(issueDate)
			expectedDuration := time.Hour * 24 * 30
			assert.Equal(t, expectedDuration, duration)

			// Verify timezone is correct
			loc, err := time.LoadLocation(tz)
			require.NoError(t, err)
			assert.Equal(t, loc.String(), dueDate.Location().String())
		})
	}
}

func TestPaymentTermsService_UpdatePaymentTerm(t *testing.T) {
	db := setupPaymentTermsTestDB(t)
	service := NewPaymentTermsService(db)
	ctx := context.Background()

	term := createPaymentTermForTest(t, db, 30, 2.0, 10)

	t.Run("Update payment term", func(t *testing.T) {
		updates := &models.PaymentTerm{
			Name:               "Updated Net 45",
			DaysUntilDue:       45,
			DiscountPercentage: decimal.NewFromFloat(3.0),
			DiscountDays:       15,
		}

		err := service.UpdatePaymentTerm(ctx, term.ID, updates)
		require.NoError(t, err)

		// Verify updates
		updated, err := service.GetPaymentTerm(ctx, term.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Net 45", updated.Name)
		assert.Equal(t, 45, updated.DaysUntilDue)
		assert.Equal(t, "3.00", updated.DiscountPercentage.String())
		assert.Equal(t, 15, updated.DiscountDays)
	})
}

func TestPaymentTermsService_DeletePaymentTerm(t *testing.T) {
	db := setupPaymentTermsTestDB(t)
	service := NewPaymentTermsService(db)
	ctx := context.Background()

	t.Run("Delete unused payment term", func(t *testing.T) {
		term := createPaymentTermForTest(t, db, 30, 2.0, 10)

		err := service.DeletePaymentTerm(ctx, term.ID)
		assert.NoError(t, err)

		// Verify deletion
		_, err = service.GetPaymentTerm(ctx, term.ID)
		assert.Error(t, err)
	})

	t.Run("Cannot delete payment term in use", func(t *testing.T) {
		term := createPaymentTermForTest(t, db, 30, 2.0, 10)

		// Create an invoice using this payment term
		invoice := &models.Invoice{
			InvoiceNumber: "INV-001",
			CustomerID:    uuid.New(),
			PaymentTermID: &term.ID,
			IssueDate:     time.Now(),
			DueDate:       time.Now().AddDate(0, 0, 30),
			Status:        models.InvoiceStatusDraft,
			TotalAmount:   decimal.NewFromFloat(1000),
		}
		db.Create(invoice)

		err := service.DeletePaymentTerm(ctx, term.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete payment term")
	})
}

func TestPaymentTermsService_GetDefaultPaymentTerm(t *testing.T) {
	db := setupPaymentTermsTestDB(t)
	service := NewPaymentTermsService(db)
	ctx := context.Background()

	t.Run("Get default payment term", func(t *testing.T) {
		// Create non-default term
		term1 := createPaymentTermForTest(t, db, 15, 0, 0)
		term1.IsDefault = false
		db.Save(term1)

		// Create default term
		term2 := createPaymentTermForTest(t, db, 30, 2.0, 10)
		term2.IsDefault = true
		db.Save(term2)

		defaultTerm, err := service.GetDefaultPaymentTerm(ctx)
		require.NoError(t, err)
		assert.Equal(t, term2.ID, defaultTerm.ID)
	})

	t.Run("No default payment term found", func(t *testing.T) {
		db2 := setupPaymentTermsTestDB(t)
		service2 := NewPaymentTermsService(db2)

		_, err := service2.GetDefaultPaymentTerm(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no default payment term found")
	})
}

// Helper function to get time.Location
func getLocation(t *testing.T, tz string) *time.Location {
	loc, err := time.LoadLocation(tz)
	require.NoError(t, err)
	return loc
}
