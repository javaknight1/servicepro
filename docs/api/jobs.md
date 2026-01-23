# Jobs API

REST API for job scheduling, management, and conflict detection.

## Overview

The Jobs API provides:

- Job creation and lifecycle management
- Technician scheduling and assignment
- Conflict detection for overlapping schedules
- Recurring job support
- Calendar integration

## Endpoints

### Jobs

| Method | Endpoint           | Description       |
| ------ | ------------------ | ----------------- |
| GET    | `/jobs`            | List jobs         |
| GET    | `/jobs/:id`        | Get job details   |
| POST   | `/jobs`            | Create job        |
| PUT    | `/jobs/:id`        | Update job        |
| DELETE | `/jobs/:id`        | Delete job        |
| POST   | `/jobs/:id/assign` | Assign technician |

### Schedules

| Method | Endpoint                    | Description               |
| ------ | --------------------------- | ------------------------- |
| GET    | `/schedules`                | List schedules            |
| GET    | `/schedules/:id`            | Get schedule              |
| POST   | `/schedules`                | Create schedule           |
| PUT    | `/schedules/:id`            | Update schedule           |
| DELETE | `/schedules/:id`            | Delete schedule           |
| GET    | `/schedules/technician/:id` | Get technician's schedule |

### Conflicts

| Method | Endpoint           | Description         |
| ------ | ------------------ | ------------------- |
| POST   | `/conflicts/check` | Check for conflicts |

## Create Schedule

Create a new schedule for a job.

```http
POST /api/v1/schedules
```

### Request Body

```json
{
  "job_id": "uuid",
  "title": "Install HVAC System",
  "description": "Install new HVAC unit at customer location",
  "start_time": "2024-01-15T09:00:00Z",
  "end_time": "2024-01-15T12:00:00Z",
  "all_day": false,
  "recurrence_type": "none",
  "assigned_tech_ids": ["tech-uuid-1", "tech-uuid-2"],
  "location": "123 Main St, Springfield, IL 62701",
  "notes": "Customer prefers morning appointments",
  "color": "#3B82F6",
  "reminders_enabled": true
}
```

### Recurrence Types

- `none` - One-time schedule
- `daily` - Repeats daily
- `weekly` - Repeats weekly
- `monthly` - Repeats monthly
- `yearly` - Repeats yearly

### Response (201 Created)

```json
{
  "id": "schedule-uuid",
  "job_id": "job-uuid",
  "job_number": "JOB-001",
  "title": "Install HVAC System",
  "start_time": "2024-01-15T09:00:00Z",
  "end_time": "2024-01-15T12:00:00Z",
  "duration": 180,
  "recurrence_type": "none",
  "assigned_tech_ids": ["tech-uuid"],
  "location": "123 Main St, Springfield, IL 62701",
  "is_confirmed": false,
  "is_cancelled": false,
  "color": "#3B82F6",
  "created_at": "2024-01-01T00:00:00Z"
}
```

## List Schedules

Get schedules with filters.

```http
GET /api/v1/schedules?start_date=2024-01-01&end_date=2024-01-31
```

### Query Parameters

| Parameter  | Type    | Description                            |
| ---------- | ------- | -------------------------------------- |
| start_date | date    | Filter from date (RFC3339)             |
| end_date   | date    | Filter to date (RFC3339)               |
| tech_id    | UUID    | Filter by technician                   |
| confirmed  | boolean | Filter by confirmation status          |
| cancelled  | boolean | Filter by cancellation status          |
| page       | integer | Page number (default: 1)               |
| page_size  | integer | Items per page (default: 50, max: 100) |

### Response

```json
{
  "schedules": [...],
  "total": 100,
  "page": 1,
  "page_size": 50
}
```

## Get Technician Schedule

Get all schedules for a specific technician.

```http
GET /api/v1/schedules/technician/:tech_id?start_date=2024-01-01&end_date=2024-01-07
```

Returns an array of schedules assigned to the technician.

## Check Conflicts

Check for scheduling conflicts before creating/updating a schedule.

```http
POST /api/v1/conflicts/check
```

### Request Body

```json
{
  "job_id": "uuid",
  "start_time": "2024-01-15T09:00:00Z",
  "end_time": "2024-01-15T12:00:00Z",
  "assigned_tech_ids": ["tech-uuid-1", "tech-uuid-2"],
  "location": "123 Main St"
}
```

### Response

```json
{
  "has_conflicts": true,
  "conflicts": [
    {
      "conflict_type": "technician_overlap",
      "severity": "high",
      "description": "Tech-1 is already scheduled for another job",
      "conflicting_with": {
        "schedule_id": "other-schedule-uuid",
        "title": "Repair AC Unit",
        "start_time": "2024-01-15T08:00:00Z",
        "end_time": "2024-01-15T10:00:00Z"
      },
      "technician_id": "tech-uuid-1",
      "time_overlap": {
        "start": "2024-01-15T09:00:00Z",
        "end": "2024-01-15T10:00:00Z",
        "duration_minutes": 60
      }
    }
  ],
  "suggestions": [
    {
      "type": "reschedule",
      "title": "Reschedule to Alternative Time",
      "confidence": 0.85,
      "impact_level": "medium",
      "suggestions": [
        {
          "start_time": "2024-01-15T14:00:00Z",
          "end_time": "2024-01-15T17:00:00Z",
          "reason": "All technicians available"
        }
      ]
    },
    {
      "type": "reassign_technician",
      "title": "Assign Different Technician",
      "confidence": 0.75,
      "impact_level": "low",
      "suggestions": [
        {
          "technician_id": "tech-uuid-3",
          "name": "Jane Smith",
          "availability": "Available during requested time"
        }
      ]
    }
  ]
}
```

## Conflict Types

| Type               | Description                   | Severity    |
| ------------------ | ----------------------------- | ----------- |
| technician_overlap | Technician double-booked      | High/Medium |
| workload_excess    | Exceeds daily work hours      | High/Medium |
| business_hours     | Outside business hours        | Medium/Low  |
| location_overlap   | Same location, different jobs | Low         |

### Severity Levels

- **High**: Confirmed schedules, >2 hour overlap
- **Medium**: >30 minute overlap, overtime
- **Low**: <30 minute overlap, warnings

## Validation

### Time Range

- End time must be after start time
- Maximum 1 year in the past
- Maximum duration varies by job type

### Technician Assignment

- At least one technician required
- Technicians must exist and be active

### Business Hours

- Default: Monday-Friday, 8 AM - 6 PM
- Weekend scheduling generates warnings
- After-hours scheduling allowed with warning

## Error Responses

### Invalid Time Range (400)

```json
{
  "error": "invalid_time_range",
  "message": "End time must be after start time"
}
```

### Schedule Not Found (404)

```json
{
  "error": "schedule_not_found",
  "message": "Schedule not found"
}
```

## Examples

### Create Schedule

```bash
curl -X POST http://localhost:8080/api/v1/schedules \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "job_id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "Install HVAC System",
    "start_time": "2024-01-15T09:00:00Z",
    "end_time": "2024-01-15T12:00:00Z",
    "assigned_tech_ids": ["tech-uuid"],
    "location": "123 Main St",
    "color": "#3B82F6"
  }'
```

### Check Conflicts Before Scheduling

```bash
curl -X POST http://localhost:8080/api/v1/conflicts/check \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "job_id": "550e8400-e29b-41d4-a716-446655440000",
    "start_time": "2024-01-15T09:00:00Z",
    "end_time": "2024-01-15T12:00:00Z",
    "assigned_tech_ids": ["tech-uuid-1", "tech-uuid-2"]
  }'
```

### Get Weekly Schedule

```bash
curl "http://localhost:8080/api/v1/schedules?start_date=2024-01-15&end_date=2024-01-21" \
  -H "Authorization: Bearer <token>"
```

## Database Schema

### schedules table

| Column                | Type         | Description               |
| --------------------- | ------------ | ------------------------- |
| id                    | UUID         | Primary key               |
| job_id                | UUID         | Foreign key to jobs       |
| title                 | VARCHAR(200) | Schedule title            |
| description           | TEXT         | Description               |
| start_time            | TIMESTAMP    | Start time                |
| end_time              | TIMESTAMP    | End time                  |
| all_day               | BOOLEAN      | All-day event             |
| recurrence_type       | VARCHAR(50)  | Recurrence pattern        |
| recurring_schedule_id | UUID         | Parent recurring schedule |
| assigned_tech_ids     | UUID[]       | Array of technician IDs   |
| location              | VARCHAR(200) | Job location              |
| notes                 | TEXT         | Notes                     |
| is_confirmed          | BOOLEAN      | Confirmation status       |
| is_cancelled          | BOOLEAN      | Cancellation status       |
| color                 | VARCHAR(7)   | Hex color code            |
| created_by            | UUID         | User who created          |
| created_at            | TIMESTAMP    | Creation timestamp        |
| updated_at            | TIMESTAMP    | Last update               |

### schedule_conflicts table

| Column           | Type        | Description          |
| ---------------- | ----------- | -------------------- |
| id               | UUID        | Primary key          |
| schedule_id_1    | UUID        | First schedule       |
| schedule_id_2    | UUID        | Second schedule      |
| conflict_type    | VARCHAR(50) | Type of conflict     |
| severity         | VARCHAR(50) | Severity level       |
| description      | TEXT        | Conflict description |
| is_resolved      | BOOLEAN     | Resolution status    |
| resolved_by      | UUID        | User who resolved    |
| resolution_notes | TEXT        | Resolution notes     |

### Indexes

- Time range: `idx_schedules_time_range`
- Technicians: `idx_schedules_assigned_techs` (GIN)
- Conflicts: `idx_schedule_conflicts_unique`

## Frontend Integration

### Calendar Event Format

```typescript
const event: CalendarEvent = {
  id: schedule.id,
  title: schedule.title,
  start: new Date(schedule.start_time),
  end: new Date(schedule.end_time),
  jobNumber: schedule.job_number,
  location: schedule.location,
  assignedTo: schedule.assigned_tech_ids,
  color: schedule.color,
};
```

### React Query Example

```typescript
const { data: schedules } = useQuery({
  queryKey: ['schedules', startDate, endDate],
  queryFn: () => fetchSchedules(startDate, endDate),
});
```
