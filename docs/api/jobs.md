# Jobs API

Jobs (work orders): create, schedule, assign technicians, move through the job
status machine, add materials and notes, and check scheduling conflicts. The
OpenAPI spec (Swagger UI at `/api/docs`) is the source of truth for shapes.

There is no `/schedules` resource and no recurring-job API — recurring service
agreements are planned (T167; see
[recurring-jobs.md](../features/recurring-jobs.md)).

## Endpoints

All under `/api/v1`, authenticated and tenant-scoped.

### Jobs

| Method | Endpoint                         | Description                                     |
| ------ | -------------------------------- | ----------------------------------------------- |
| GET    | `/jobs`                          | List jobs with filters                          |
| POST   | `/jobs`                          | Create a job                                    |
| GET    | `/jobs/my`                       | Jobs assigned to the caller                     |
| GET    | `/jobs/scheduled?start=…&end=…`  | Jobs scheduled in a range (RFC 3339) — calendar |
| GET    | `/jobs/overdue`                  | Overdue jobs                                    |
| GET    | `/jobs/stats`                    | Job statistics                                  |
| GET    | `/jobs/technicians/:id/workload` | A technician's workload                         |
| GET    | `/jobs/number/:jobNumber`        | Get a job by number                             |
| GET    | `/jobs/:id`                      | Get a job                                       |
| PUT    | `/jobs/:id`                      | Update a job                                    |
| DELETE | `/jobs/:id`                      | Delete a job                                    |
| GET    | `/customers/:id/jobs`            | A customer's jobs                               |

### Status

| Method | Endpoint                        | Description                                       |
| ------ | ------------------------------- | ------------------------------------------------- |
| POST   | `/jobs/:id/start`               | Start the job                                     |
| POST   | `/jobs/:id/complete`            | Complete (`{"completion_notes": "…"}` optional)   |
| POST   | `/jobs/:id/cancel`              | Cancel (`{"reason": "…"}` optional)               |
| POST   | `/jobs/:id/hold`                | Put on hold (`{"reason": "…"}` optional)          |
| POST   | `/jobs/:id/resume`              | Resume from `on_hold`                             |
| POST   | `/jobs/:id/transition`          | Generic transition (`models.StatusChangeRequest`) |
| GET    | `/jobs/:id/allowed-transitions` | Transitions allowed from the current status       |
| GET    | `/jobs/:id/status-history`      | Status history                                    |

Job statuses: `new`, `scheduled`, `en_route`, `in_progress`, `on_hold`,
`completed`, `invoiced`, `paid`, `cancelled`. The state machine lives in the
job service; ask `allowed-transitions` rather than hard-coding it.

### Assignments, materials, notes

| Method | Endpoint                         | Description                                   |
| ------ | -------------------------------- | --------------------------------------------- |
| POST   | `/jobs/:id/assign`               | Assign a technician (`technician_id`, `role`) |
| DELETE | `/jobs/:id/assign/:technicianId` | Unassign a technician                         |
| POST   | `/jobs/:id/materials`            | Add a material                                |
| POST   | `/jobs/:id/notes`                | Add a note (`note`, `is_internal`)            |

### Conflicts

| Method | Endpoint           | Description                              |
| ------ | ------------------ | ---------------------------------------- |
| POST   | `/conflicts/check` | Check a time slot for scheduling clashes |

## Create Job

```http
POST /api/v1/jobs
```

```json
{
  "customer_id": "uuid",
  "title": "Nightly janitorial — Building A",
  "description": "Lobby, restrooms, 3rd floor offices",
  "job_type": "maintenance",
  "priority": "normal",
  "scheduled_start_at": 1761951600,
  "scheduled_end_at": 1761966000,
  "estimated_duration": 240,
  "service_address": {
    "street": "…",
    "city": "…",
    "state": "OH",
    "zip": "43215"
  },
  "assignments": [{ "user_id": "uuid", "role": "lead" }]
}
```

Required: `customer_id`, `title`, `job_type` (`installation`, `maintenance`,
`repair`, `inspection`, `emergency`). `priority` is `low`, `normal`, `high` or
`urgent`. Scheduled times are integer epoch timestamps; `estimated_duration` is
in minutes. Full shape: Swagger (`models.CreateJobRequest`).

## Check Conflicts

```http
POST /api/v1/conflicts/check
```

```json
{
  "job_id": "uuid",
  "start_time": 1761951600,
  "end_time": 1761966000,
  "assigned_tech_ids": ["uuid"],
  "location": "optional"
}
```

Response:

```json
{
  "has_conflicts": true,
  "conflicts": [
    {
      "conflict_type": "technician_overlap",
      "severity": "high",
      "description": "…",
      "technician_id": "uuid",
      "conflicting_with": { "id": "uuid", "title": "…" }
    }
  ],
  "suggestions": []
}
```

The tenant comes from the auth context, never from the body.

## Invoicing a Job

Create an invoice with the job's `job_id` (`POST /api/v1/invoices`). A job can
be invoiced only once; cancelling that invoice frees it. See
[invoices.md](./invoices.md).
