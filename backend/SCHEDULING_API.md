# Scheduling API Documentation

Complete scheduling system for ServicePro with job scheduling, conflict detection, and recurring schedules.

## 📦 Created Files

### Backend Files

#### Models (`internal/models/`)

- **schedule.go** - Schedule models, validations, and request/response types

#### Repository (`internal/repository/`)

- **schedule_repository.go** - Data access layer for schedules

#### Services (`internal/services/`)

- **schedule_service.go** - Business logic for scheduling

#### Handlers (`internal/api/handlers/`)

- **schedule_handler.go** - HTTP handlers for schedule endpoints

#### Migrations (`migrations/`)

- **010_create_scheduling_tables.sql** - Database schema
- **010_create_scheduling_tables_rollback.sql** - Rollback script

## 🗄️ Database Schema

### Tables

#### 1. schedules

Primary table for job schedules

```sql
- id (UUID, PK)
- job_id (UUID, FK to jobs)
- title (VARCHAR 200)
- description (TEXT)
- start_time (TIMESTAMP)
- end_time (TIMESTAMP)
- all_day (BOOLEAN)
- recurrence_type (VARCHAR 50)
- recurring_schedule_id (UUID, FK)
- assigned_tech_ids (UUID[])
- location (VARCHAR 200)
- notes (TEXT)
- is_confirmed (BOOLEAN)
- is_cancelled (BOOLEAN)
- color (VARCHAR 7)
- created_by, updated_by (UUID)
- timestamps
```

#### 2. recurring_schedules

Recurring schedule patterns

```sql
- id (UUID, PK)
- title, description
- recurrence_type (daily, weekly, monthly, yearly)
- recurrence_rule (RFC 5545 RRULE)
- start_date, end_date
- interval (INT)
- days_of_week (INT[])
- day_of_month, month_of_year (INT)
- time_start, time_end (TIME)
- assigned_tech_ids (UUID[])
- is_active (BOOLEAN)
```

#### 3. schedule_conflicts

Detected scheduling conflicts

```sql
- id (UUID, PK)
- schedule_id_1, schedule_id_2 (UUID, FK)
- conflict_type (VARCHAR 50)
- severity (VARCHAR 50)
- description (TEXT)
- is_resolved (BOOLEAN)
- resolved_by (UUID)
- resolution_notes (TEXT)
```

### Indexes

- Time range queries: `idx_schedules_time_range`
- Technician lookups: `idx_schedules_assigned_techs` (GIN)
- Conflict detection: `idx_schedule_conflicts_unique`
- Active recurring: `idx_recurring_schedules_active`

### Constraints

- End time must be after start time
- Valid recurrence types
- Color format validation (hex #RRGGBB)
- Different schedules in conflicts

### Triggers

- **detect_schedule_conflicts**: Automatically detects technician overlap
- **validate_schedule_times**: Validates time ranges
- **update_updated_at**: Auto-updates timestamps

## 🔌 API Endpoints

### Base URL: `/api/v1/schedules`

### 1. Create Schedule

```http
POST /api/v1/schedules
Authorization: Bearer <token>
Content-Type: application/json

{
  "job_id": "uuid",
  "title": "Install HVAC System",
  "description": "Install new HVAC unit",
  "start_time": "2024-01-15T09:00:00Z",
  "end_time": "2024-01-15T12:00:00Z",
  "all_day": false,
  "recurrence_type": "none",
  "assigned_tech_ids": ["tech-uuid"],
  "location": "123 Main St",
  "notes": "Customer prefers morning",
  "color": "#3B82F6",
  "reminders_enabled": true
}

Response: 201 Created
{
  "id": "schedule-uuid",
  "job_id": "uuid",
  "job_number": "JOB-001",
  "title": "Install HVAC System",
  "start_time": "2024-01-15T09:00:00Z",
  "end_time": "2024-01-15T12:00:00Z",
  "duration": 180,
  "recurrence_type": "none",
  "assigned_tech_ids": ["tech-uuid"],
  "location": "123 Main St",
  "is_confirmed": false,
  "is_cancelled": false,
  "color": "#3B82F6",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### 2. Get Schedule

```http
GET /api/v1/schedules/:id
Authorization: Bearer <token>

Response: 200 OK
{
  // Same as create response
}
```

### 3. Update Schedule

```http
PUT /api/v1/schedules/:id
Authorization: Bearer <token>
Content-Type: application/json

{
  // Same fields as create
}

Response: 200 OK
{
  // Updated schedule
}
```

### 4. Delete Schedule

```http
DELETE /api/v1/schedules/:id
Authorization: Bearer <token>

Response: 204 No Content
```

### 5. List Schedules

```http
GET /api/v1/schedules?start_date=2024-01-01&end_date=2024-01-31&page=1&page_size=50
Authorization: Bearer <token>

Query Parameters:
- start_date (string, RFC3339)
- end_date (string, RFC3339)
- tech_id (uuid)
- confirmed (boolean)
- cancelled (boolean)
- page (int, default: 1)
- page_size (int, default: 50, max: 100)

Response: 200 OK
{
  "schedules": [ /* array of schedules */ ],
  "total": 100,
  "page": 1,
  "page_size": 50
}
```

### 6. Get Technician Schedule

```http
GET /api/v1/schedules/technician/:tech_id?start_date=2024-01-01&end_date=2024-01-07
Authorization: Bearer <token>

Response: 200 OK
[
  // Array of schedules for the technician
]
```

## 💡 Features

### 1. Automatic Conflict Detection

When a schedule is created or updated, the system automatically:

- Detects technician overlap conflicts
- Creates conflict records in `schedule_conflicts` table
- Logs conflicts for resolution
- Runs asynchronously to not block the response

### 2. Recurring Schedules

Support for recurring schedules following RFC 5545:

- Daily, Weekly, Monthly, Yearly patterns
- Custom intervals (e.g., every 2 weeks)
- Specific days of week
- End date or max occurrences
- RRULE format support

### 3. Validation

- End time after start time
- Valid recurrence types
- Color format (hex)
- Required fields
- Time range constraints (max 1 year in past)

### 4. Soft Deletes

All deletions are soft deletes using `deleted_at` timestamp.

## 🔒 Security & Middleware

### Authentication

All endpoints require Bearer token authentication:

```http
Authorization: Bearer <jwt-token>
```

### Authorization

Based on user roles (implemented in existing middleware):

- Admin: Full access
- Manager: Full access
- Technician: Own schedules only
- Customer: View only

### Rate Limiting

Standard rate limiting from existing middleware applies.

### Validation

- Request body validation using Gin binding
- Field-level validation in models
- Business logic validation in service layer

## 🧪 Testing

### Unit Tests

Create `schedule_service_test.go`:

```go
func TestCreateSchedule(t *testing.T) {
    // Test successful creation
    // Test validation errors
    // Test conflict detection
}

func TestUpdateSchedule(t *testing.T) {
    // Test successful update
    // Test not found
    // Test invalid time range
}

func TestListSchedules(t *testing.T) {
    // Test pagination
    // Test filters
    // Test empty results
}
```

### Integration Tests

Create `schedule_handler_test.go`:

```go
func TestScheduleEndpoints(t *testing.T) {
    // Setup test database
    // Test complete CRUD flow
    // Test conflict scenarios
    // Test authorization
}
```

### Load Testing

Using `hey` or `ab`:

```bash
# Load test list endpoint
hey -n 10000 -c 100 -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/schedules

# Expected performance:
# - < 100ms p50
# - < 200ms p95
# - < 500ms p99
# - > 1000 req/s throughput
```

## 📊 Performance Optimizations

### Database

1. **Indexes**:

   - GIN indexes for array columns (assigned_tech_ids)
   - Composite indexes for common queries
   - Partial indexes for active records

2. **Queries**:
   - Efficient date range queries
   - Array containment for technician lookups
   - Preloading related data

### Application

1. **Async Operations**:

   - Conflict detection runs asynchronously
   - Doesn't block schedule creation/update response

2. **Caching** (Future):

   - Redis cache for frequently accessed schedules
   - Cache invalidation on updates

3. **Connection Pooling**:
   - GORM handles connection pooling
   - Configured in database package

## 🔄 Integration with Calendar Frontend

### Calendar Event Format

The API responses map directly to the calendar component:

```typescript
// API Response
{
  "id": "uuid",
  "job_id": "uuid",
  "job_number": "JOB-001",
  "title": "Install HVAC",
  "start_time": "2024-01-15T09:00:00Z",
  "end_time": "2024-01-15T12:00:00Z",
  "duration": 180,
  "assigned_tech_ids": ["tech-uuid"],
  "location": "123 Main St",
  "color": "#3B82F6"
}

// Maps to Calendar JobEvent
const event: JobEvent = {
  id: schedule.id,
  title: schedule.title,
  start: new Date(schedule.start_time),
  end: new Date(schedule.end_time),
  jobNumber: schedule.job_number,
  status: job.status, // from job relation
  location: schedule.location,
  assignedTo: schedule.assigned_tech_ids,
  resource: schedule
};
```

### API Client Example

```typescript
import { useQuery, useMutation } from '@tanstack/react-query';

// Fetch schedules for calendar
const { data: schedules } = useQuery({
  queryKey: ['schedules', startDate, endDate],
  queryFn: async () => {
    const res = await fetch(
      `/api/v1/schedules?start_date=${startDate}&end_date=${endDate}`,
      { headers: { Authorization: `Bearer ${token}` } }
    );
    return res.json();
  },
});

// Update schedule (drag-drop)
const updateSchedule = useMutation({
  mutationFn: async (data: UpdateScheduleData) => {
    const res = await fetch(`/api/v1/schedules/${data.id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(data),
    });
    return res.json();
  },
});
```

## 📝 Next Steps

### 1. Add Routing

Add to `internal/api/routes/routes.go`:

```go
func SetupScheduleRoutes(router *gin.RouterGroup, handler *handlers.ScheduleHandler) {
    schedules := router.Group("/schedules")
    schedules.Use(middleware.AuthMiddleware())
    {
        schedules.POST("", handler.CreateSchedule)
        schedules.GET("", handler.ListSchedules)
        schedules.GET("/:id", handler.GetSchedule)
        schedules.PUT("/:id", handler.UpdateSchedule)
        schedules.DELETE("/:id", handler.DeleteSchedule)
        schedules.GET("/technician/:tech_id", handler.GetTechnicianSchedule)
    }
}
```

### 2. Run Migrations

```bash
# Apply migration
psql -U username -d servicepro -f migrations/010_create_scheduling_tables.sql

# Or using migration tool
migrate -path migrations -database "postgres://..." up
```

### 3. Initialize in Main

Add to `cmd/api/main.go`:

```go
// Initialize repository
scheduleRepo := repository.NewScheduleRepository(db)

// Initialize service
scheduleService := services.NewScheduleService(scheduleRepo)

// Initialize handler
scheduleHandler := handlers.NewScheduleHandler(scheduleService)

// Setup routes
routes.SetupScheduleRoutes(api, scheduleHandler)
```

### 4. Create Tests

```bash
# Unit tests
go test ./internal/services/schedule_service_test.go -v

# Integration tests
go test ./internal/api/handlers/schedule_handler_test.go -v

# All tests
go test ./... -v

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 🐛 Troubleshooting

### Common Issues

1. **Migration fails**: Check PostgreSQL version (need >= 12 for array functions)
2. **Conflicts not detecting**: Verify trigger is created
3. **Performance slow**: Check indexes are created
4. **Auth fails**: Ensure middleware is properly configured

## 📚 References

- [RFC 5545 - iCalendar](https://tools.ietf.org/html/rfc5545) - Recurrence rules
- [GORM Documentation](https://gorm.io) - ORM usage
- [Gin Documentation](https://gin-gonic.com) - HTTP framework
- [PostgreSQL Arrays](https://www.postgresql.org/docs/current/arrays.html) - Array operations

---

## Summary

Complete scheduling API with:

- ✅ CRUD operations
- ✅ Conflict detection
- ✅ Recurring schedules
- ✅ Technician scheduling
- ✅ Database migrations
- ✅ Comprehensive validation
- ✅ API documentation
- ✅ Performance optimization
- ✅ Calendar frontend integration

Ready for integration and testing! 🚀
