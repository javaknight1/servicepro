package recurring

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// Mock repositories
type MockScheduleRepo struct {
	mock.Mock
}

func (m *MockScheduleRepo) GetRecurringByID(id uuid.UUID) (*models.RecurringSchedule, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RecurringSchedule), args.Error(1)
}

func (m *MockScheduleRepo) Create(schedule *models.Schedule) error {
	args := m.Called(schedule)
	return args.Error(0)
}

type MockExceptionRepo struct {
	mock.Mock
}

func (m *MockExceptionRepo) GetExceptionsByPattern(patternID uuid.UUID, startDate, endDate time.Time) ([]ScheduleException, error) {
	args := m.Called(patternID, startDate, endDate)
	return args.Get(0).([]ScheduleException), args.Error(1)
}

type MockHolidayRepo struct {
	mock.Mock
}

func (m *MockHolidayRepo) GetHolidaysByDateRange(startDate, endDate time.Time) ([]Holiday, error) {
	args := m.Called(startDate, endDate)
	return args.Get(0).([]Holiday), args.Error(1)
}

// Implement other required methods
func (m *MockScheduleRepo) GetByID(id uuid.UUID) (*models.Schedule, error) { return nil, nil }
func (m *MockScheduleRepo) Update(schedule *models.Schedule) error         { return nil }
func (m *MockScheduleRepo) Delete(id uuid.UUID) error                      { return nil }
func (m *MockScheduleRepo) List(params *models.ScheduleQueryParams) ([]models.Schedule, int64, error) {
	return nil, 0, nil
}
func (m *MockScheduleRepo) GetByDateRange(startDate, endDate time.Time) ([]models.Schedule, error) {
	return nil, nil
}
func (m *MockScheduleRepo) GetByTechnicianAndDateRange(techID uuid.UUID, startDate, endDate time.Time) ([]models.Schedule, error) {
	return nil, nil
}
func (m *MockScheduleRepo) GetByJobID(jobID uuid.UUID) ([]models.Schedule, error) { return nil, nil }
func (m *MockScheduleRepo) DetectConflicts(schedule *models.Schedule) ([]models.ScheduleConflict, error) {
	return nil, nil
}
func (m *MockScheduleRepo) GetConflicts(scheduleID uuid.UUID) ([]models.ScheduleConflict, error) {
	return nil, nil
}
func (m *MockScheduleRepo) ResolveConflict(conflictID uuid.UUID, resolvedBy uuid.UUID, notes *string) error {
	return nil
}
func (m *MockScheduleRepo) CreateRecurring(recurring *models.RecurringSchedule) error { return nil }
func (m *MockScheduleRepo) UpdateRecurring(recurring *models.RecurringSchedule) error { return nil }
func (m *MockScheduleRepo) DeleteRecurring(id uuid.UUID) error                        { return nil }
func (m *MockScheduleRepo) GetActiveRecurring() ([]models.RecurringSchedule, error)   { return nil, nil }
func (m *MockScheduleRepo) GetScheduleCount() (int64, error)                          { return 0, nil }
func (m *MockScheduleRepo) GetConflictCount() (int64, error)                          { return 0, nil }

func (m *MockExceptionRepo) CreateException(exception *ScheduleException) error { return nil }
func (m *MockExceptionRepo) GetExceptionByID(id uuid.UUID) (*ScheduleException, error) {
	return nil, nil
}
func (m *MockExceptionRepo) UpdateException(exception *ScheduleException) error { return nil }
func (m *MockExceptionRepo) DeleteException(id uuid.UUID) error                 { return nil }
func (m *MockExceptionRepo) ListExceptions(patternID uuid.UUID) ([]ScheduleException, error) {
	return nil, nil
}

func (m *MockHolidayRepo) CreateHoliday(holiday *Holiday) error                   { return nil }
func (m *MockHolidayRepo) GetHolidayByID(id uuid.UUID) (*Holiday, error)          { return nil, nil }
func (m *MockHolidayRepo) GetHolidaysByCountry(country string) ([]Holiday, error) { return nil, nil }
func (m *MockHolidayRepo) UpdateHoliday(holiday *Holiday) error                   { return nil }
func (m *MockHolidayRepo) DeleteHoliday(id uuid.UUID) error                       { return nil }
func (m *MockHolidayRepo) ListHolidays() ([]Holiday, error)                       { return nil, nil }

// Test daily recurrence generation
func TestGenerateSchedules_Daily(t *testing.T) {
	scheduleRepo := new(MockScheduleRepo)
	exceptionRepo := new(MockExceptionRepo)
	holidayRepo := new(MockHolidayRepo)

	generator := NewScheduleGenerator(scheduleRepo, exceptionRepo, holidayRepo)

	jobID := uuid.New()
	patternID := uuid.New()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	recurring := &models.RecurringSchedule{
		ID:              patternID,
		Title:           "Daily Maintenance",
		RecurrenceType:  models.RecurrenceDaily,
		StartDate:       startDate,
		Interval:        1,
		TimeStart:       "09:00",
		TimeEnd:         "10:00",
		JobTemplateID:   &jobID,
		IsActive:        true,
		CreatedBy:       uuid.New(),
		AssignedTechIDs: []uuid.UUID{uuid.New()},
	}

	scheduleRepo.On("GetRecurringByID", patternID).Return(recurring, nil)
	exceptionRepo.On("GetExceptionsByPattern", patternID, mock.Anything, mock.Anything).Return([]ScheduleException{}, nil)
	scheduleRepo.On("Create", mock.Anything).Return(nil)

	req := &GenerationRequest{
		RecurringScheduleID: patternID,
		StartDate:           startDate,
		EndDate:             startDate.AddDate(0, 0, 7), // 7 days
		SkipHolidays:        false,
		SkipWeekends:        false,
		MaxOccurrences:      10,
	}

	ctx := context.Background()
	result, err := generator.GenerateSchedules(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 8, result.GeneratedCount) // 7 days + start day
	assert.Equal(t, 0, result.SkippedCount)

	scheduleRepo.AssertExpectations(t)
}

// Test weekly recurrence generation
func TestGenerateSchedules_Weekly(t *testing.T) {
	scheduleRepo := new(MockScheduleRepo)
	exceptionRepo := new(MockExceptionRepo)
	holidayRepo := new(MockHolidayRepo)

	generator := NewScheduleGenerator(scheduleRepo, exceptionRepo, holidayRepo)

	jobID := uuid.New()
	patternID := uuid.New()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // Monday

	recurring := &models.RecurringSchedule{
		ID:              patternID,
		Title:           "Weekly Team Meeting",
		RecurrenceType:  models.RecurrenceWeekly,
		StartDate:       startDate,
		Interval:        1,
		DaysOfWeek:      []int{1, 3, 5}, // Mon, Wed, Fri
		TimeStart:       "14:00",
		TimeEnd:         "15:00",
		JobTemplateID:   &jobID,
		IsActive:        true,
		CreatedBy:       uuid.New(),
		AssignedTechIDs: []uuid.UUID{uuid.New()},
	}

	scheduleRepo.On("GetRecurringByID", patternID).Return(recurring, nil)
	exceptionRepo.On("GetExceptionsByPattern", patternID, mock.Anything, mock.Anything).Return([]ScheduleException{}, nil)
	scheduleRepo.On("Create", mock.Anything).Return(nil)

	req := &GenerationRequest{
		RecurringScheduleID: patternID,
		StartDate:           startDate,
		EndDate:             startDate.AddDate(0, 0, 14), // 2 weeks
		SkipHolidays:        false,
		SkipWeekends:        false,
		MaxOccurrences:      10,
	}

	ctx := context.Background()
	result, err := generator.GenerateSchedules(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 7, result.GeneratedCount) // 3 days/week * 2 weeks + Mon Jan 15 (end date is inclusive)
	assert.Equal(t, 0, result.SkippedCount)

	scheduleRepo.AssertExpectations(t)
}

// Test monthly recurrence generation
func TestGenerateSchedules_Monthly(t *testing.T) {
	scheduleRepo := new(MockScheduleRepo)
	exceptionRepo := new(MockExceptionRepo)
	holidayRepo := new(MockHolidayRepo)

	generator := NewScheduleGenerator(scheduleRepo, exceptionRepo, holidayRepo)

	jobID := uuid.New()
	patternID := uuid.New()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	dayOfMonth := 15

	recurring := &models.RecurringSchedule{
		ID:              patternID,
		Title:           "Monthly Review",
		RecurrenceType:  models.RecurrenceMonthly,
		StartDate:       startDate,
		Interval:        1,
		DayOfMonth:      &dayOfMonth,
		TimeStart:       "10:00",
		TimeEnd:         "11:00",
		JobTemplateID:   &jobID,
		IsActive:        true,
		CreatedBy:       uuid.New(),
		AssignedTechIDs: []uuid.UUID{uuid.New()},
	}

	scheduleRepo.On("GetRecurringByID", patternID).Return(recurring, nil)
	exceptionRepo.On("GetExceptionsByPattern", patternID, mock.Anything, mock.Anything).Return([]ScheduleException{}, nil)
	scheduleRepo.On("Create", mock.Anything).Return(nil)

	req := &GenerationRequest{
		RecurringScheduleID: patternID,
		StartDate:           startDate,
		EndDate:             startDate.AddDate(0, 3, 0), // 3 months
		SkipHolidays:        false,
		SkipWeekends:        false,
		MaxOccurrences:      10,
	}

	ctx := context.Background()
	result, err := generator.GenerateSchedules(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, result.GeneratedCount) // 3 months
	assert.Equal(t, 0, result.SkippedCount)

	scheduleRepo.AssertExpectations(t)
}

// Test skip weekends
func TestGenerateSchedules_SkipWeekends(t *testing.T) {
	scheduleRepo := new(MockScheduleRepo)
	exceptionRepo := new(MockExceptionRepo)
	holidayRepo := new(MockHolidayRepo)

	generator := NewScheduleGenerator(scheduleRepo, exceptionRepo, holidayRepo)

	jobID := uuid.New()
	patternID := uuid.New()
	startDate := time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC) // Saturday

	recurring := &models.RecurringSchedule{
		ID:              patternID,
		Title:           "Daily Task",
		RecurrenceType:  models.RecurrenceDaily,
		StartDate:       startDate,
		Interval:        1,
		TimeStart:       "09:00",
		TimeEnd:         "10:00",
		JobTemplateID:   &jobID,
		IsActive:        true,
		CreatedBy:       uuid.New(),
		AssignedTechIDs: []uuid.UUID{uuid.New()},
	}

	scheduleRepo.On("GetRecurringByID", patternID).Return(recurring, nil)
	exceptionRepo.On("GetExceptionsByPattern", patternID, mock.Anything, mock.Anything).Return([]ScheduleException{}, nil)
	scheduleRepo.On("Create", mock.Anything).Return(nil)

	req := &GenerationRequest{
		RecurringScheduleID: patternID,
		StartDate:           startDate,
		EndDate:             startDate.AddDate(0, 0, 7), // 1 week
		SkipHolidays:        false,
		SkipWeekends:        true,
		MaxOccurrences:      10,
	}

	ctx := context.Background()
	result, err := generator.GenerateSchedules(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 5, result.GeneratedCount) // 5 weekdays (Mon-Fri)
	assert.Equal(t, 3, result.SkippedCount)   // Jan 6 (Sat), Jan 7 (Sun), Jan 13 (Sat) - end date is inclusive
	assert.Len(t, result.SkippedDates, 3)

	scheduleRepo.AssertExpectations(t)
}

// Test skip holidays
func TestGenerateSchedules_SkipHolidays(t *testing.T) {
	scheduleRepo := new(MockScheduleRepo)
	exceptionRepo := new(MockExceptionRepo)
	holidayRepo := new(MockHolidayRepo)

	generator := NewScheduleGenerator(scheduleRepo, exceptionRepo, holidayRepo)

	jobID := uuid.New()
	patternID := uuid.New()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	recurring := &models.RecurringSchedule{
		ID:              patternID,
		Title:           "Daily Task",
		RecurrenceType:  models.RecurrenceDaily,
		StartDate:       startDate,
		Interval:        1,
		TimeStart:       "09:00",
		TimeEnd:         "10:00",
		JobTemplateID:   &jobID,
		IsActive:        true,
		CreatedBy:       uuid.New(),
		AssignedTechIDs: []uuid.UUID{uuid.New()},
	}

	// New Year's Day is a holiday
	holidays := []Holiday{
		{
			ID:          uuid.New(),
			Name:        "New Year's Day",
			Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Country:     "US",
			IsRecurring: true,
		},
	}

	scheduleRepo.On("GetRecurringByID", patternID).Return(recurring, nil)
	exceptionRepo.On("GetExceptionsByPattern", patternID, mock.Anything, mock.Anything).Return([]ScheduleException{}, nil)
	holidayRepo.On("GetHolidaysByDateRange", mock.Anything, mock.Anything).Return(holidays, nil)
	scheduleRepo.On("Create", mock.Anything).Return(nil)

	req := &GenerationRequest{
		RecurringScheduleID: patternID,
		StartDate:           startDate,
		EndDate:             startDate.AddDate(0, 0, 3), // 3 days
		SkipHolidays:        true,
		SkipWeekends:        false,
		MaxOccurrences:      10,
	}

	ctx := context.Background()
	result, err := generator.GenerateSchedules(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, result.GeneratedCount) // Jan 2, 3, 4
	assert.Equal(t, 1, result.SkippedCount)   // Jan 1 (New Year's)
	assert.Len(t, result.SkippedDates, 1)
	assert.Contains(t, result.SkippedDates[0].Reason, "New Year's Day")

	scheduleRepo.AssertExpectations(t)
}

// Test with exceptions
func TestGenerateSchedules_WithExceptions(t *testing.T) {
	scheduleRepo := new(MockScheduleRepo)
	exceptionRepo := new(MockExceptionRepo)
	holidayRepo := new(MockHolidayRepo)

	generator := NewScheduleGenerator(scheduleRepo, exceptionRepo, holidayRepo)

	jobID := uuid.New()
	patternID := uuid.New()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	recurring := &models.RecurringSchedule{
		ID:              patternID,
		Title:           "Daily Task",
		RecurrenceType:  models.RecurrenceDaily,
		StartDate:       startDate,
		Interval:        1,
		TimeStart:       "09:00",
		TimeEnd:         "10:00",
		JobTemplateID:   &jobID,
		IsActive:        true,
		CreatedBy:       uuid.New(),
		AssignedTechIDs: []uuid.UUID{uuid.New()},
	}

	// Exception on Jan 3
	exceptions := []ScheduleException{
		{
			ID:                 uuid.New(),
			RecurringPatternID: patternID,
			ExceptionDate:      time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
			ExceptionType:      "skip",
			Reason:             "Technician unavailable",
		},
	}

	scheduleRepo.On("GetRecurringByID", patternID).Return(recurring, nil)
	exceptionRepo.On("GetExceptionsByPattern", patternID, mock.Anything, mock.Anything).Return(exceptions, nil)
	scheduleRepo.On("Create", mock.Anything).Return(nil)

	req := &GenerationRequest{
		RecurringScheduleID: patternID,
		StartDate:           startDate,
		EndDate:             startDate.AddDate(0, 0, 4), // 4 days
		SkipHolidays:        false,
		SkipWeekends:        false,
		MaxOccurrences:      10,
	}

	ctx := context.Background()
	result, err := generator.GenerateSchedules(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 4, result.GeneratedCount) // Jan 1, 2, 4, 5
	assert.Equal(t, 1, result.SkippedCount)   // Jan 3
	assert.Len(t, result.SkippedDates, 1)
	assert.Contains(t, result.SkippedDates[0].Reason, "Technician unavailable")

	scheduleRepo.AssertExpectations(t)
}

// Test max occurrences
func TestGenerateSchedules_MaxOccurrences(t *testing.T) {
	scheduleRepo := new(MockScheduleRepo)
	exceptionRepo := new(MockExceptionRepo)
	holidayRepo := new(MockHolidayRepo)

	generator := NewScheduleGenerator(scheduleRepo, exceptionRepo, holidayRepo)

	jobID := uuid.New()
	patternID := uuid.New()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	recurring := &models.RecurringSchedule{
		ID:              patternID,
		Title:           "Daily Task",
		RecurrenceType:  models.RecurrenceDaily,
		StartDate:       startDate,
		Interval:        1,
		TimeStart:       "09:00",
		TimeEnd:         "10:00",
		JobTemplateID:   &jobID,
		IsActive:        true,
		CreatedBy:       uuid.New(),
		AssignedTechIDs: []uuid.UUID{uuid.New()},
	}

	scheduleRepo.On("GetRecurringByID", patternID).Return(recurring, nil)
	exceptionRepo.On("GetExceptionsByPattern", patternID, mock.Anything, mock.Anything).Return([]ScheduleException{}, nil)
	scheduleRepo.On("Create", mock.Anything).Return(nil)

	req := &GenerationRequest{
		RecurringScheduleID: patternID,
		StartDate:           startDate,
		EndDate:             startDate.AddDate(0, 1, 0), // 1 month
		SkipHolidays:        false,
		SkipWeekends:        false,
		MaxOccurrences:      5, // Limit to 5
	}

	ctx := context.Background()
	result, err := generator.GenerateSchedules(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 5, result.GeneratedCount) // Limited to 5
	assert.Equal(t, 0, result.SkippedCount)

	scheduleRepo.AssertExpectations(t)
}
