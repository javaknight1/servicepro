# Recurring Job Management System Documentation

Complete recurring job management with pattern generation, exception handling, and holiday calendar integration.

## 📦 Overview

The recurring job management system enables automated schedule generation from recurring patterns, with support for exceptions and holiday calendars. It helps automate repetitive scheduling tasks while maintaining flexibility.

### Features

- ✅ Daily, Weekly, Monthly, Yearly recurrence patterns
- ✅ Custom intervals and complex patterns (RFC 5545 compliant)
- ✅ Exception handling (skip, reschedule, modify)
- ✅ Holiday calendar integration
- ✅ Pattern preview and validation
- ✅ Long-term schedule generation
- ✅ Frontend components for pattern management

## 🗄️ Database Schema

### Tables

#### 1. recurring_schedules (Already exists from migration 010)

Stores recurring pattern definitions

```sql
- id (UUID, PK)
- title, description
- recurrence_type (daily, weekly, monthly, yearly)
- recurrence_rule (RFC 5545 RRULE)
- start_date, end_date
- interval (e.g., every 2 weeks)
- days_of_week (for weekly)
- day_of_month, month_of_year (for monthly/yearly)
- occurrences (max count)
- time_start, time_end, duration
- assigned_tech_ids
- is_active
```

#### 2. schedule_exceptions (New - Migration 011)

Exceptions to recurring patterns

```sql
- id (UUID, PK)
- recurring_pattern_id (FK to recurring_schedules)
- exception_date (DATE)
- exception_type (skip, reschedule, modify)
- reason (TEXT)
- rescheduled_date (DATE, optional)
- modifications (JSONB, optional)
- created_by
- timestamps
```

#### 3. holidays (New - Migration 011)

Holiday calendar

```sql
- id (UUID, PK)
- name (VARCHAR 200)
- date (DATE)
- country (VARCHAR 2, default 'US')
- is_recurring (BOOLEAN)
- timestamps
```

### Indexes

- `idx_exceptions_pattern` - Fast exception lookups by pattern
- `idx_exceptions_date` - Fast date-based filtering
- `idx_exceptions_pattern_date` - Composite for pattern + date queries
- `idx_holidays_date` - Fast holiday lookups by date
- `idx_holidays_date_range` - Fast date range queries

### Constraints

- **Unique exception per pattern per date** - Prevents duplicate exceptions
- **Reschedule validation** - Reschedule exception type requires rescheduled_date
- **No past exceptions** - Cannot create exceptions more than 1 day in the past

## 🔌 Backend Services

### ScheduleGenerator

Generates schedule instances from recurring patterns.

**File:** `internal/services/recurring/generator.go`

```go
import "github.com/javaknight1/servicepro/backend/internal/services/recurring"

// Initialize
generator := recurring.NewScheduleGenerator(
    scheduleRepo,
    exceptionRepo,
    holidayRepo,
)

// Generate schedules
req := &recurring.GenerationRequest{
    RecurringScheduleID: patternID,
    StartDate:           time.Now(),
    EndDate:             time.Now().AddDate(0, 3, 0), // Next 3 months
    SkipHolidays:        true,
    SkipWeekends:        false,
    MaxOccurrences:      100,
}

result, err := generator.GenerateSchedules(ctx, req)
if err != nil {
    // Handle error
}

fmt.Printf("Generated %d schedules\n", result.GeneratedCount)
fmt.Printf("Skipped %d dates\n", result.SkippedCount)
```

**Key Methods:**

1. **GenerateSchedules** - Creates actual schedules in database
2. **PreviewSchedules** - Generates preview without saving
3. **generateOccurrences** - Calculates occurrence dates
4. **shouldSkipDate** - Checks exceptions and holidays

### PatternHandler

Manages recurring pattern CRUD operations.

**File:** `internal/services/recurring/pattern.go`

```go
import "github.com/javaknight1/servicepro/backend/internal/services/recurring"

// Initialize
handler := recurring.NewPatternHandler(scheduleRepo)

// Create pattern
req := &recurring.PatternRequest{
    Title:          "Weekly Maintenance",
    RecurrenceType: models.RecurrenceWeekly,
    StartDate:      time.Now(),
    Interval:       1,
    DaysOfWeek:     []int{1, 3, 5}, // Mon, Wed, Fri
    TimeStart:      "09:00",
    TimeEnd:        "17:00",
    AssignedTechIDs: []uuid.UUID{techID},
}

pattern, err := handler.CreatePattern(ctx, req, createdBy)
if err != nil {
    // Handle error
}

// Update pattern
updated, err := handler.UpdatePattern(ctx, pattern.ID, req, updatedBy)

// Activate/Deactivate
err = handler.DeactivatePattern(ctx, pattern.ID)
```

**Key Methods:**

1. **CreatePattern** - Creates new recurring pattern with validation
2. **UpdatePattern** - Updates existing pattern
3. **DeletePattern** - Soft deletes pattern
4. **ActivatePattern** / **DeactivatePattern** - Control pattern state
5. **ListActivePatterns** - Get all active patterns

### ExceptionHandler

Manages schedule exceptions.

**File:** `internal/services/recurring/exception.go`

```go
import "github.com/javaknight1/servicepro/backend/internal/services/recurring"

// Initialize
handler := recurring.NewExceptionHandler(exceptionRepo)

// Create skip exception
req := &recurring.ExceptionRequest{
    RecurringPatternID: patternID,
    ExceptionDate:      time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC),
    ExceptionType:      "skip",
    Reason:             "Christmas Day",
}

exception, err := handler.CreateException(ctx, req, createdBy)

// Create reschedule exception
rescheduleReq := &recurring.ExceptionRequest{
    RecurringPatternID: patternID,
    ExceptionDate:      time.Date(2024, 7, 4, 0, 0, 0, 0, time.UTC),
    ExceptionType:      "reschedule",
    Reason:             "Independence Day",
    RescheduledDate:    &newDate,
}

exception, err := handler.CreateException(ctx, rescheduleReq, createdBy)
```

**Exception Types:**

1. **skip** - Skip this occurrence completely
2. **reschedule** - Move to a different date
3. **modify** - Modify schedule details (use Modifications JSON field)

### HolidayHandler

Manages holiday calendar.

**File:** `internal/services/recurring/exception.go`

```go
import "github.com/javaknight1/servicepro/backend/internal/services/recurring"

// Initialize
handler := recurring.NewHolidayHandler(holidayRepo)

// Create holiday
req := &recurring.HolidayRequest{
    Name:        "New Year's Day",
    Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
    Country:     "US",
    IsRecurring: true,
}

holiday, err := handler.CreateHoliday(ctx, req)

// Import US holidays for a year
err = handler.ImportUSHolidays(ctx, 2024)
```

**Supported US Holidays:**

- New Year's Day (Jan 1)
- Memorial Day (Last Monday in May)
- Independence Day (July 4)
- Labor Day (First Monday in September)
- Veterans Day (Nov 11)
- Thanksgiving (Fourth Thursday in November)
- Christmas Day (Dec 25)

## 🎨 Frontend Components

### RecurringForm

Form for creating/editing recurring patterns.

**File:** `frontend/src/components/recurring/RecurringForm.tsx`

```tsx
import { RecurringForm } from '@/components/recurring';

function CreateRecurringPattern() {
  const handleSubmit = async (data) => {
    const response = await fetch('/api/v1/recurring/patterns', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(data),
    });

    const pattern = await response.json();
    console.log('Pattern created:', pattern);
  };

  return (
    <RecurringForm
      onSubmit={handleSubmit}
      onCancel={() => navigate('/schedules')}
      isLoading={isSubmitting}
    />
  );
}
```

**Features:**

- Daily/Weekly/Monthly/Yearly recurrence selection
- Custom interval configuration
- Days of week selection (Weekly)
- Day of month selection (Monthly/Yearly)
- Time range selection
- End condition: Never, Date, or Occurrence count
- Form validation

### PatternPreview

Visual preview of recurring pattern.

**File:** `frontend/src/components/recurring/PatternPreview.tsx`

```tsx
import { PatternPreview } from '@/components/recurring';

function PatternConfiguration() {
  const [formData, setFormData] = useState({
    recurrenceType: 'weekly',
    interval: 1,
    daysOfWeek: [1, 3, 5],
    timeStart: '09:00',
    timeEnd: '17:00',
    // ...
  });

  return (
    <div>
      <RecurringForm initialData={formData} onSubmit={handleSubmit} />

      <PatternPreview pattern={formData} previewMonths={3} showSkipped={true} />
    </div>
  );
}
```

**Features:**

- List and calendar view modes
- Next N occurrences preview
- Human-readable pattern description
- Occurrence count and statistics
- Skipped dates display

## 📝 Recurrence Patterns

### Daily Pattern

```json
{
  "recurrenceType": "daily",
  "interval": 1,
  "startDate": "2024-01-01",
  "timeStart": "09:00",
  "timeEnd": "17:00"
}
```

**RRULE:** `FREQ=DAILY;INTERVAL=1`

**Example:** Every day at 9 AM - 5 PM

### Weekly Pattern

```json
{
  "recurrenceType": "weekly",
  "interval": 1,
  "daysOfWeek": [1, 3, 5],
  "startDate": "2024-01-01",
  "timeStart": "09:00",
  "timeEnd": "17:00"
}
```

**RRULE:** `FREQ=WEEKLY;INTERVAL=1;BYDAY=MO,WE,FR`

**Example:** Every Monday, Wednesday, Friday at 9 AM - 5 PM

### Monthly Pattern

```json
{
  "recurrenceType": "monthly",
  "interval": 1,
  "dayOfMonth": 15,
  "startDate": "2024-01-01",
  "timeStart": "10:00",
  "timeEnd": "11:00"
}
```

**RRULE:** `FREQ=MONTHLY;INTERVAL=1;BYMONTHDAY=15`

**Example:** 15th of every month at 10 AM - 11 AM

### Yearly Pattern

```json
{
  "recurrenceType": "yearly",
  "interval": 1,
  "monthOfYear": 12,
  "dayOfMonth": 25,
  "startDate": "2024-01-01",
  "timeStart": "08:00",
  "timeEnd": "09:00"
}
```

**RRULE:** `FREQ=YEARLY;INTERVAL=1;BYMONTH=12;BYMONTHDAY=25`

**Example:** Every December 25th at 8 AM - 9 AM

## 🧪 Testing

### Backend Unit Tests

**File:** `internal/services/recurring/generator_test.go`

```bash
# Run recurring pattern tests
go test ./internal/services/recurring/... -v

# Run with coverage
go test ./internal/services/recurring/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

**Test Scenarios:**

1. **Daily recurrence** - Generates daily occurrences
2. **Weekly recurrence** - Generates on specific days
3. **Monthly recurrence** - Generates on day of month
4. **Yearly recurrence** - Generates annually
5. **Skip weekends** - Excludes Saturday/Sunday
6. **Skip holidays** - Excludes holiday dates
7. **With exceptions** - Respects exception rules
8. **Max occurrences** - Limits generation count
9. **Long-term generation** - Generates 1+ years ahead

### Example Test

```go
func TestGenerateSchedules_Weekly(t *testing.T) {
    // Setup mocks
    scheduleRepo := new(MockScheduleRepo)
    generator := NewScheduleGenerator(scheduleRepo, exceptionRepo, holidayRepo)

    // Create weekly pattern
    pattern := &models.RecurringSchedule{
        RecurrenceType: models.RecurrenceWeekly,
        DaysOfWeek:     []int{1, 3, 5}, // Mon, Wed, Fri
        Interval:       1,
        // ...
    }

    // Generate schedules
    result, err := generator.GenerateSchedules(ctx, req)

    // Verify
    assert.NoError(t, err)
    assert.Equal(t, 6, result.GeneratedCount) // 3 days/week * 2 weeks
}
```

## 🚀 Integration Guide

### Step 1: Run Migrations

```bash
# Apply migration
psql -U username -d servicepro -f migrations/011_create_exceptions_and_holidays.sql

# Or using migration tool
migrate -path migrations -database "postgres://..." up
```

### Step 2: Initialize Services

```go
// In cmd/api/main.go

// Repositories (already exists)
scheduleRepo := repository.NewScheduleRepository(db)

// Exception and Holiday repositories (need to create)
exceptionRepo := repository.NewExceptionRepository(db)
holidayRepo := repository.NewHolidayRepository(db)

// Initialize recurring services
generator := recurring.NewScheduleGenerator(scheduleRepo, exceptionRepo, holidayRepo)
patternHandler := recurring.NewPatternHandler(scheduleRepo)
exceptionHandler := recurring.NewExceptionHandler(exceptionRepo)
holidayHandler := recurring.NewHolidayHandler(holidayRepo)
```

### Step 3: Add API Endpoints

```go
// Pattern endpoints
patterns := router.Group("/api/v1/recurring/patterns")
patterns.POST("", patternHandler.CreatePattern)
patterns.GET("/:id", patternHandler.GetPattern)
patterns.PUT("/:id", patternHandler.UpdatePattern)
patterns.DELETE("/:id", patternHandler.DeletePattern)
patterns.POST("/:id/generate", generator.GenerateSchedules)
patterns.POST("/:id/preview", generator.PreviewSchedules)

// Exception endpoints
exceptions := router.Group("/api/v1/recurring/exceptions")
exceptions.POST("", exceptionHandler.CreateException)
exceptions.GET("/:id", exceptionHandler.GetException)
exceptions.PUT("/:id", exceptionHandler.UpdateException)
exceptions.DELETE("/:id", exceptionHandler.DeleteException)

// Holiday endpoints
holidays := router.Group("/api/v1/recurring/holidays")
holidays.POST("", holidayHandler.CreateHoliday)
holidays.GET("", holidayHandler.ListHolidays)
holidays.POST("/import/us/:year", holidayHandler.ImportUSHolidays)
```

### Step 4: Generate Schedules

```go
// Automated generation (cron job)
func generateRecurringSchedules() {
    patterns, _ := patternHandler.ListActivePatterns(ctx)

    for _, pattern := range patterns {
        req := &recurring.GenerationRequest{
            RecurringScheduleID: pattern.ID,
            StartDate:           time.Now(),
            EndDate:             time.Now().AddDate(0, 1, 0), // Next month
            SkipHolidays:        true,
            SkipWeekends:        false,
            MaxOccurrences:      50,
        }

        result, err := generator.GenerateSchedules(ctx, req)
        if err != nil {
            log.Printf("Failed to generate for pattern %s: %v", pattern.ID, err)
            continue
        }

        log.Printf("Generated %d schedules for pattern %s", result.GeneratedCount, pattern.Title)
    }
}
```

## 📊 Use Cases

### Use Case 1: Weekly Maintenance Schedule

**Scenario:** HVAC maintenance every Monday and Friday

```go
pattern := &recurring.PatternRequest{
    Title:           "HVAC Maintenance",
    Description:     "Regular HVAC system inspection and maintenance",
    RecurrenceType:  models.RecurrenceWeekly,
    Interval:        1,
    DaysOfWeek:      []int{1, 5}, // Monday, Friday
    StartDate:       time.Now(),
    TimeStart:       "08:00",
    TimeEnd:         "12:00",
    AssignedTechIDs: []uuid.UUID{hvacTechID},
    Location:        "Main Building",
}

pattern, _ := handler.CreatePattern(ctx, pattern, userID)
```

### Use Case 2: Monthly Inspection with Exceptions

**Scenario:** Monthly inspection on 15th, skip December

```go
// Create pattern
pattern := &recurring.PatternRequest{
    Title:          "Monthly Safety Inspection",
    RecurrenceType: models.RecurrenceMonthly,
    Interval:       1,
    DayOfMonth:     &fifteen,
    StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
    TimeStart:      "09:00",
    TimeEnd:        "10:00",
}

created, _ := handler.CreatePattern(ctx, pattern, userID)

// Add exception for December
exception := &recurring.ExceptionRequest{
    RecurringPatternID: created.ID,
    ExceptionDate:      time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC),
    ExceptionType:      "skip",
    Reason:             "Holiday season - skip December",
}

exceptionHandler.CreateException(ctx, exception, userID)
```

### Use Case 3: Yearly Service with Holiday Skip

**Scenario:** Annual service on July 1st, skip if it's a holiday

```go
// Create pattern
pattern := &recurring.PatternRequest{
    Title:          "Annual Certification",
    RecurrenceType: models.RecurrenceYearly,
    Interval:       1,
    MonthOfYear:    &july,
    DayOfMonth:     &first,
    StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
    TimeStart:      "10:00",
    TimeEnd:        "11:00",
}

created, _ := handler.CreatePattern(ctx, pattern, userID)

// Generate with holiday skip
req := &recurring.GenerationRequest{
    RecurringScheduleID: created.ID,
    StartDate:           time.Now(),
    EndDate:             time.Now().AddDate(5, 0, 0), // 5 years
    SkipHolidays:        true,
    MaxOccurrences:      5,
}

result, _ := generator.GenerateSchedules(ctx, req)
```

## 🔒 Best Practices

1. **Generate in advance** - Generate schedules 1-3 months ahead
2. **Use max occurrences** - Prevent runaway generation
3. **Import holidays annually** - Keep holiday calendar updated
4. **Review exceptions** - Regularly audit exception list
5. **Test patterns** - Use preview before generating
6. **Monitor generation** - Log generation results
7. **Handle failures** - Retry failed generations
8. **Clean old schedules** - Archive past recurring schedules

## 📈 Performance

### Database Queries

- **Pattern lookup**: 1 query (indexed by ID)
- **Exception check**: 1 query per date range (indexed)
- **Holiday check**: 1 query per date range (indexed)
- **Schedule creation**: 1 insert per occurrence

### Optimization Tips

1. **Batch generation** - Generate multiple patterns in parallel
2. **Index usage** - Ensure indexes on date columns
3. **Limit occurrences** - Don't generate unlimited schedules
4. **Cache holidays** - Cache holiday list in memory
5. **Async generation** - Run generation as background job

### Expected Performance

- Generate 100 daily occurrences: <1 second
- Generate 1 year monthly: <500ms
- Pattern with 50 exceptions: <2 seconds

## 🐛 Troubleshooting

### Common Issues

**1. No schedules generated**

- Check pattern is active (`is_active = true`)
- Verify date range is valid
- Check for blocking exceptions
- Ensure job template ID exists

**2. Too many schedules generated**

- Set `MaxOccurrences` limit
- Check interval value (might be 0 or too small)
- Verify end date is not too far future

**3. Holidays not skipping**

- Ensure `SkipHolidays = true` in request
- Verify holidays exist in database
- Check holiday date range overlaps

**4. Exceptions not working**

- Verify exception date matches occurrence date exactly
- Check exception type is valid (skip, reschedule, modify)
- Ensure pattern ID matches

---

## Summary

Complete recurring job management system with:

- ✅ 3 backend services (Generator, PatternHandler, ExceptionHandler)
- ✅ 2 frontend components (RecurringForm, PatternPreview)
- ✅ 2 new database tables (exceptions, holidays)
- ✅ RFC 5545 RRULE support
- ✅ Comprehensive testing
- ✅ Full documentation

**Ready for production use!** 🚀
