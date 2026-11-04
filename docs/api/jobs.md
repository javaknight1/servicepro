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

| Method | Endpoint                   | Description                                                  |
| ------ | -------------------------- | ------------------------------------------------------------ |
| POST   | `/jobs/:id/transition`     | Change status (`models.StatusChangeRequest`: `to_status`, optional `reason`, `notes`) |
| GET    | `/jobs/:id/status-history` | Status history                                               |

Job statuses: `new`, `scheduled`, `en_route`, `in_progress`, `on_hold`,
`completed`, `invoiced`, `paid`, `cancelled`. The one state machine is
`backend/internal/models/job_status.go`; every job response carries
`allowed_transitions` (the statuses `/transition` accepts next), so clients
read that rather than hard-coding the rules. The per-action endpoints
(`/start`, `/complete`, `/cancel`, `/hold`, `/resume`) and
`/allowed-transitions` were removed in T267.

### Assignments, materials, notes

| Method | Endpoint                         | Description                                   |
| ------ | -------------------------------- | --------------------------------------------- |
| POST   | `/jobs/:id/assign`               | Assign a technician (`technician_id`, `role`) |
| DELETE | `/jobs/:id/assign/:technicianId` | Unassign a technician                         |
| POST   | `/jobs/:id/materials`            | Add a material                                |
| POST   | `/jobs/:id/notes`                | Add a note (`note`, `is_internal`)            |

### Photos and files (T174)

| Method | Endpoint                                     | Permission    | Description                                  |
| ------ | -------------------------------------------- | ------------- | -------------------------------------------- |
| GET    | `/jobs/:id/attachments`                      | `jobs.read`   | List, oldest first, with presigned links     |
| POST   | `/jobs/:id/attachments`                      | `jobs.update` | Upload (multipart `file`, optional `caption`) |
| GET    | `/jobs/:id/attachments/:attachmentId/file`   | `jobs.read`   | 302 to a presigned link to the stored file   |
| GET    | `/jobs/:id/attachments/:attachmentId/thumb`  | `jobs.read`   | 302 to the photo's thumbnail (files: 404)    |
| PATCH  | `/jobs/:id/attachments/:attachmentId`        | `jobs.update` | Set or clear the caption (`caption`)         |
| DELETE | `/jobs/:id/attachments/:attachmentId`        | `jobs.update` | Delete the attachment and its stored objects |

Only registered when object storage is configured. Uploads go through the API
(not presigned PUTs) because the server has to see the bytes anyway:

- **Size**: at most 15 MB (`413 file_too_large`; the body is capped before the
  multipart form is parsed). A job holds at most 100 (`409 attachment_limit`).
- **Type**: sniffed from the bytes, never the declared type or the name. JPEG,
  PNG and WebP photos, and PDF files; anything else is
  `415 unsupported_file_type`.
- **Photos**: the declared dimensions are checked with `image.DecodeConfig`
  before decoding (security audit F-19): over 12000 px a side or 40 megapixels
  is `400 image_too_large`, undecodable bytes `400 invalid_image`. The photo
  is then turned upright from its EXIF orientation, scaled to 2048 px on the
  longest side and re-encoded as JPEG, which strips EXIF (GPS, device data);
  a 400 px JPEG thumbnail is stored next to it. At most two photos decode at
  once per instance.
- **Files** (PDF) are stored as uploaded and have no thumbnail.

Objects live in the private bucket under
`job-attachments/{tenant}/{job}/{attachment}.jpg|.pdf` (and `_thumb.jpg`).
Responses carry `url` / `thumb_url` (presigned, about an hour; refetch the list
for fresh ones) and `file_path` / `thumb_path` (API paths that redirect to a
fresh link, for invoices and service reports).

Technicians (no `jobs.view_all`) see and add attachments only on jobs assigned
to them — other jobs are `404 job_not_found`, the same row filter as
`GET /jobs/:id` — and may caption or delete only what they uploaded
(`403 attachment_not_yours`). Office staff can change any. Another tenant's
attachments are 404.

### Customer sign-off and service report (T176)

| Method | Endpoint                         | Permission    | Description                                             |
| ------ | -------------------------------- | ------------- | ------------------------------------------------------- |
| GET    | `/jobs/:id/signoff`              | `jobs.read`   | The sign-off (`signoff: null` when unsigned) and `suggested_recipients` |
| POST   | `/jobs/:id/signoff`              | `jobs.update` | Record the signature (`signer_name`, optional `signer_title`, `signer_email`, `signature`) |
| GET    | `/jobs/:id/signoff/signature`    | `jobs.read`   | 302 to a presigned link to the signature PNG            |
| GET    | `/jobs/:id/signoff/report`       | `jobs.read`   | 302 to the service report PDF                           |
| POST   | `/jobs/:id/signoff/email`        | `jobs.update` | Email the report (`{"resend": true}` to send it again)  |

Only registered when object storage is configured.

- **Signing**: the customer signs on the technician's phone; `signature` is a
  PNG data URL (`canvas.toDataURL("image/png")`) or bare base64. The type is
  sniffed from the bytes (PNG only, else `415 unsupported_signature_type`), at
  most 512 KB (`413 signature_too_large`; the body is capped at 800 KB before
  parsing), and the declared size is checked with `image.DecodeConfig` before
  decoding: 100 x 40 to 3000 x 3000 px (`400 signature_dimensions`). A canvas
  without ink is `400 signature_empty`. The PNG is flattened onto white,
  scaled to 1200 px wide and re-encoded. The signed time, IP
  (`middleware.ClientIP`), user agent and the signed-in user are recorded.
- **When**: the visit must be started or done (`in_progress`, `on_hold`,
  `completed`, `invoiced`, `paid`), else `409 job_not_started`.
- **One per job**: signing again replaces the sign-off (and deletes the old
  signature) until the report has been emailed; after that the sign-off is
  the record the customer received and is locked (`409 signoff_locked`).
- **Report**: a PDF with the job, customer, site and PO, scheduled and
  on-site times in the tenant's time zone (actual start/end, else the status
  history), the crew, work done, completion and customer-visible notes,
  materials with quantities (never prices), up to 12 photo thumbnails and the
  signature with signer, title and time. Before it is emailed,
  `GET .../report` builds it from the job as it is now; after, it returns the
  stored copy that was sent.
- **Recipients**: the customer's email and the site contact's (skipped when
  the customer has `do_not_email`), plus the signer's email if given,
  deduplicated. None is `409 no_recipients`.
- **Email is idempotent**: the send is claimed with a conditional `UPDATE`
  before anything goes out, so a retried or doubled request sends at most
  once; a repeated request after a successful send returns `sent: false`. A
  send in flight is `409 email_in_progress` (a claim older than 10 minutes is
  taken over), a provider failure is `502 email_failed` and releases the
  claim. `{"resend": true}` sends the stored report again. The email comes
  from "<Business> via ServicePro" with replies to the business, the PDF
  attached.

Objects live in the private bucket under
`job-signoffs/{tenant}/{job}/{uuid}_signature.png` and
`job-signoffs/{tenant}/{job}/{signoff}_report.pdf`. Technicians sign, read
and email only on jobs assigned to them; other jobs and other tenants' jobs
are `404 job_not_found`.

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
