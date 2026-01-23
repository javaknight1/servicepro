# Recurring Jobs

Automated schedule generation with pattern management, exceptions, and holiday calendar integration.

## Overview

The recurring jobs system enables:

- Automated schedule generation from patterns
- Daily, weekly, monthly, and yearly recurrence
- Exception handling (skip, reschedule, modify)
- Holiday calendar integration
- RFC 5545 (iCalendar) compatible patterns

## Recurrence Types

### Daily

Repeats every N days.

```json
{
  "recurrence_type": "daily",
  "interval": 1,
  "start_date": "2024-01-01",
  "time_start": "09:00",
  "time_end": "17:00"
}
```

**RRULE:** `FREQ=DAILY;INTERVAL=1`

### Weekly

Repeats on specific days of the week.

```json
{
  "recurrence_type": "weekly",
  "interval": 1,
  "days_of_week": [1, 3, 5],
  "start_date": "2024-01-01",
  "time_start": "09:00",
  "time_end": "17:00"
}
```

**Days:** 0=Sunday, 1=Monday, ..., 6=Saturday

**RRULE:** `FREQ=WEEKLY;INTERVAL=1;BYDAY=MO,WE,FR`

### Monthly

Repeats on a specific day of the month.

```json
{
  "recurrence_type": "monthly",
  "interval": 1,
  "day_of_month": 15,
  "start_date": "2024-01-01",
  "time_start": "10:00",
  "time_end": "11:00"
}
```

**RRULE:** `FREQ=MONTHLY;INTERVAL=1;BYMONTHDAY=15`

### Yearly

Repeats on a specific date each year.

```json
{
  "recurrence_type": "yearly",
  "interval": 1,
  "month_of_year": 12,
  "day_of_month": 25,
  "start_date": "2024-01-01",
  "time_start": "08:00",
  "time_end": "09:00"
}
```

**RRULE:** `FREQ=YEARLY;INTERVAL=1;BYMONTH=12;BYMONTHDAY=25`

## API Endpoints

### Patterns

| Method | Endpoint                           | Description        |
| ------ | ---------------------------------- | ------------------ |
| GET    | `/recurring/patterns`              | List patterns      |
| GET    | `/recurring/patterns/:id`          | Get pattern        |
| POST   | `/recurring/patterns`              | Create pattern     |
| PUT    | `/recurring/patterns/:id`          | Update pattern     |
| DELETE | `/recurring/patterns/:id`          | Delete pattern     |
| POST   | `/recurring/patterns/:id/generate` | Generate schedules |
| POST   | `/recurring/patterns/:id/preview`  | Preview schedules  |

### Exceptions

| Method | Endpoint                    | Description      |
| ------ | --------------------------- | ---------------- |
| GET    | `/recurring/exceptions`     | List exceptions  |
| POST   | `/recurring/exceptions`     | Create exception |
| PUT    | `/recurring/exceptions/:id` | Update exception |
| DELETE | `/recurring/exceptions/:id` | Delete exception |

### Holidays

| Method | Endpoint                              | Description        |
| ------ | ------------------------------------- | ------------------ |
| GET    | `/recurring/holidays`                 | List holidays      |
| POST   | `/recurring/holidays`                 | Create holiday     |
| POST   | `/recurring/holidays/import/us/:year` | Import US holidays |

## Create Pattern

```http
POST /api/v1/recurring/patterns
```

### Request Body

```json
{
  "title": "Weekly Maintenance",
  "description": "Regular HVAC inspection",
  "recurrence_type": "weekly",
  "interval": 1,
  "days_of_week": [1, 3, 5],
  "start_date": "2024-01-01",
  "end_date": "2024-12-31",
  "time_start": "09:00",
  "time_end": "12:00",
  "duration_minutes": 180,
  "assigned_tech_ids": ["tech-uuid"],
  "location": "Main Building",
  "is_active": true
}
```

### End Conditions

- **No end date:** Pattern continues indefinitely
- **End date:** Pattern stops on specific date
- **Occurrences:** Pattern stops after N occurrences

## Generate Schedules

```http
POST /api/v1/recurring/patterns/:id/generate
```

### Request Body

```json
{
  "start_date": "2024-01-01",
  "end_date": "2024-03-31",
  "skip_holidays": true,
  "skip_weekends": false,
  "max_occurrences": 100
}
```

### Response

```json
{
  "generated_count": 26,
  "skipped_count": 3,
  "skipped_dates": [
    {"date": "2024-01-01", "reason": "New Year's Day"},
    {"date": "2024-01-15", "reason": "Martin Luther King Jr. Day"},
    {"date": "2024-02-19", "reason": "Presidents' Day"}
  ],
  "schedules": [...]
}
```

## Preview Schedules

Preview without creating actual schedules.

```http
POST /api/v1/recurring/patterns/:id/preview
```

Returns the same format as generate, but schedules are not saved.

## Exceptions

### Exception Types

| Type       | Description                   |
| ---------- | ----------------------------- |
| skip       | Skip this occurrence entirely |
| reschedule | Move to a different date      |
| modify     | Change schedule details       |

### Create Skip Exception

```http
POST /api/v1/recurring/exceptions
```

```json
{
  "recurring_pattern_id": "pattern-uuid",
  "exception_date": "2024-12-25",
  "exception_type": "skip",
  "reason": "Christmas Day"
}
```

### Create Reschedule Exception

```json
{
  "recurring_pattern_id": "pattern-uuid",
  "exception_date": "2024-07-04",
  "exception_type": "reschedule",
  "reason": "Independence Day",
  "rescheduled_date": "2024-07-05"
}
```

### Create Modify Exception

```json
{
  "recurring_pattern_id": "pattern-uuid",
  "exception_date": "2024-11-28",
  "exception_type": "modify",
  "reason": "Thanksgiving - half day",
  "modifications": {
    "time_end": "12:00",
    "notes": "Half day schedule"
  }
}
```

## Holiday Calendar

### US Holidays (Auto-Import)

```http
POST /api/v1/recurring/holidays/import/us/2024
```

Imports standard US holidays:

- New Year's Day (Jan 1)
- Martin Luther King Jr. Day
- Presidents' Day
- Memorial Day
- Independence Day (Jul 4)
- Labor Day
- Veterans Day (Nov 11)
- Thanksgiving
- Christmas Day (Dec 25)

### Create Custom Holiday

```http
POST /api/v1/recurring/holidays
```

```json
{
  "name": "Company Anniversary",
  "date": "2024-06-15",
  "country": "US",
  "is_recurring": true
}
```

## Use Cases

### Weekly Maintenance Schedule

```json
{
  "title": "HVAC Maintenance",
  "recurrence_type": "weekly",
  "interval": 1,
  "days_of_week": [1, 5],
  "time_start": "08:00",
  "time_end": "12:00",
  "assigned_tech_ids": ["hvac-tech-uuid"]
}
```

### Monthly Inspection

```json
{
  "title": "Monthly Safety Inspection",
  "recurrence_type": "monthly",
  "interval": 1,
  "day_of_month": 15,
  "time_start": "09:00",
  "time_end": "10:00"
}
```

### Quarterly Service

```json
{
  "title": "Quarterly Equipment Service",
  "recurrence_type": "monthly",
  "interval": 3,
  "day_of_month": 1,
  "time_start": "10:00",
  "time_end": "14:00"
}
```

## Database Schema

### recurring_schedules

| Column            | Type         | Description                    |
| ----------------- | ------------ | ------------------------------ |
| id                | UUID         | Primary key                    |
| title             | VARCHAR(200) | Pattern title                  |
| description       | TEXT         | Description                    |
| recurrence_type   | VARCHAR(50)  | daily, weekly, monthly, yearly |
| recurrence_rule   | TEXT         | RFC 5545 RRULE                 |
| start_date        | DATE         | Pattern start                  |
| end_date          | DATE         | Pattern end (optional)         |
| interval          | INTEGER      | Repeat interval                |
| days_of_week      | INTEGER[]    | Days for weekly                |
| day_of_month      | INTEGER      | Day for monthly/yearly         |
| month_of_year     | INTEGER      | Month for yearly               |
| occurrences       | INTEGER      | Max occurrences                |
| time_start        | TIME         | Start time                     |
| time_end          | TIME         | End time                       |
| duration          | INTERVAL     | Duration                       |
| assigned_tech_ids | UUID[]       | Technician IDs                 |
| is_active         | BOOLEAN      | Active status                  |
| created_at        | TIMESTAMP    | Creation time                  |

### schedule_exceptions

| Column               | Type        | Description              |
| -------------------- | ----------- | ------------------------ |
| id                   | UUID        | Primary key              |
| recurring_pattern_id | UUID        | Parent pattern           |
| exception_date       | DATE        | Exception date           |
| exception_type       | VARCHAR(50) | skip, reschedule, modify |
| reason               | TEXT        | Reason for exception     |
| rescheduled_date     | DATE        | New date (if reschedule) |
| modifications        | JSONB       | Modified fields          |
| created_by           | UUID        | Creator                  |
| created_at           | TIMESTAMP   | Creation time            |

### holidays

| Column       | Type         | Description    |
| ------------ | ------------ | -------------- |
| id           | UUID         | Primary key    |
| name         | VARCHAR(200) | Holiday name   |
| date         | DATE         | Holiday date   |
| country      | VARCHAR(2)   | Country code   |
| is_recurring | BOOLEAN      | Repeats yearly |
| created_at   | TIMESTAMP    | Creation time  |

## Best Practices

1. **Generate in advance** - Generate 1-3 months ahead
2. **Use max occurrences** - Prevent unlimited generation
3. **Import holidays annually** - Keep calendar updated
4. **Review exceptions** - Audit regularly
5. **Preview before generating** - Verify pattern works
6. **Use meaningful titles** - Easy identification
