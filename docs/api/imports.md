# Imports (Jobber, Housecall Pro, CSV)

T193. New tenants bring their customers, sites, jobs and open invoices over from
Jobber, Housecall Pro or any spreadsheet. The web page is **Settings → Import data**
(`/settings/import`, linked from onboarding step 3 and the customers list).
Code: `backend/internal/imports` (parsing, column mapping, validation — pure),
`backend/internal/services/import_mapped.go` (writing rows) and
`import_service.go` (the job runner); fixtures in `backend/internal/imports/testdata`.

The older customer-template upload (`POST /customers/import`, ServicePro's own
CSV layout) still works; see [import-export.md](../features/import-export.md).

## Endpoints

All under `/api/v1`, signed in, tenant-scoped. `{entity}` is `customers`,
`sites`, `jobs` or `invoices`; each entity's routes check its own permission.

| Method | Endpoint                         | Permission (customers / sites / jobs / invoices)                            | Description                                   |
| ------ | -------------------------------- | --------------------------------------------------------------------------- | --------------------------------------------- |
| POST   | `/imports/{entity}/preview`      | `customers.create` / `customers.update` / `jobs.create` / `invoices.create` | Map and validate a file; saves nothing        |
| POST   | `/imports/{entity}`              | same                                                                        | Start the import (first chunk runs inline)    |
| POST   | `/imports/{entity}/{job_id}/run` | same                                                                        | Run the next chunk of a pending import        |
| GET    | `/imports/{entity}/{job_id}`     | `customers.list` / `customers.list` / `jobs.list` / `invoices.list`         | Progress, counts and the first 100 row errors |
| GET    | `/imports/{entity}`              | same                                                                        | The 20 most recent imports of the entity      |

Upload form (preview and start, `multipart/form-data`):

| Field     | Required | Description                                                                                          |
| --------- | -------- | ---------------------------------------------------------------------------------------------------- |
| `file`    | yes      | CSV (comma, semicolon or tab; UTF-8 with or without BOM, else read as Latin-1) or XLSX (first sheet) |
| `source`  | no       | `jobber`, `housecall_pro` or `generic` (default)                                                     |
| `mapping` | no       | JSON object `{"<field key>": "<column header>"}`; replaces the automatic mapping                     |

The preview returns the headers, the target fields (`key`, `label`, `required`,
`group` = "at least one of"), the mapping used, `missing` (what must still be
mapped), counts (`valid_rows`, `skip_rows`, `error_rows`), the first 20 rows as
they will be saved and the first 100 row problems. `POST /imports/{entity}`
answers 400 `missing_columns` until `missing` is empty. Exact shapes: Swagger UI
(`/api/docs`, Imports tag).

Errors: 404 unknown entity, 400 `invalid_source` / `invalid_mapping` /
`invalid_file` / `missing_columns`, 413 `file_too_large` (size or rows).

## Limits (security audit F-23)

- **10 MiB per file**, enforced with `http.MaxBytesReader` before the body is
  buffered. Housecall Pro's jobs report has 47 columns (~400 bytes a row): 10 MiB
  is ~25,000 jobs, years of history for a 1–10 tech company. Jobber splits client
  exports into files of ~1,500 rows. Cloud Run's own cap is 32 MiB.
- **50,000 data rows per file**; reading stops at the first row over the cap.
- **XLSX**: at most 64 MiB unzipped (16 MiB per XML part), against zip bombs;
  200 columns at most.
- The parsed rows are stored as the job's JSONB payload (≈20 MB at the cap) and
  deleted when the job finishes.

Formula injection: imported text is stored as it is (it is data); every CSV/XLSX
export escapes cells starting with `= + - @ tab CR` (`pkg/csv.SafeCell`, F-13),
so a hostile cell like `=HYPERLINK(…)` from an old system is inert when exported
again (fixture `jobber_clients.csv` row 6).

## How an import runs

1. `POST /imports/{entity}` stores the mapped rows as a pending `import_jobs` row
   (source, entity) and claims it. Rows are imported in **chunks**: batches of 25
   rows, each saving the row cursor (`processed_rows`), the counts and that batch's
   row errors in one transaction. A chunk stops starting batches after 20 s inside
   an API request (Cloud Run ends requests at 60 s) or 40 s in
   `POST /internal/jobs/imports`, and puts the job back to `pending`.
2. The import page calls `POST /imports/{entity}/{job_id}/run` until the job is
   `completed` or `failed`; if the page is closed, the scheduled imports job
   (every 15 minutes) continues it.
3. A job stuck in `processing` for 5 minutes (instance stopped mid-chunk) goes
   back to `pending` and resumes from its cursor — safe because every row is
   idempotent (below) — up to 3 attempts, then fails with the rows imported so
   far kept.

Each row is written in its own transaction under the job's tenant (taken from the
job row, never the caller). Counts: `successful_rows` (created), `skipped_rows`
(already there, matched to an existing customer, or deliberately not imported such
as paid invoices), `failed_rows` (with the reason in the job's errors).

### Re-importing is safe

`import_external_refs` (migration 000022, tenant-owned with RLS) records which row
each imported record became, under `(tenant, source, entity, external ID)`. A row
claims its key with `INSERT … ON CONFLICT DO NOTHING` before writing, so the same
file imported twice (or two overlapping imports) creates each record once. The
external ID is the old system's ID when the file has one, else a natural key:

| Entity    | External ID                                                                                   |
| --------- | --------------------------------------------------------------------------------------------- |
| customers | Jobber J-ID / Housecall Pro customer ID, else the email address                               |
| jobs      | job number (Jobber `Job #`, Housecall Pro `Invoice`), else a hash of customer + title + start |
| invoices  | invoice number                                                                                |
| sites     | none: a site is matched on its customer + street (+ ZIP)                                      |

A record deleted in ServicePro after its import is imported again by the next run.

### Matching rows to customers

Sites, jobs and invoices name their customer. It is found by the old system's
customer ID (customers imported earlier from the same source), else by email,
else by exact company or contact name (several customers with that name → the
row fails and asks for an email column). Unknown customer → the row fails with
"import customers first". Import in order: customers, sites, jobs, invoices.

### What each entity becomes

- **Customers**: name (first/last, else the full name split, else the company with
  a "Main Contact" placeholder person), email, phone (+ a second phone), billing
  address (the service address when there is none), type (commercial when there is
  a company name, unless mapped), status (active unless the value says
  lead/prospect → prospect or inactive/archived → inactive), notes + tags. The
  service address becomes the customer's first **site** (named from the property
  name). An existing customer with the same email is linked, not duplicated.
  The customers table is US-only (database checks): a 10-digit US phone, a US state
  and a 5 or 5+4 digit ZIP are required, so other rows fail in the preview; phone
  extensions and foreign second numbers go to the notes.
- **Sites**: name (default: the street), address (US states and Canadian provinces),
  access notes, site contact.
- **Jobs**: title (else the first line of the description), description, job type
  (`maintenance` unless the value says installation/repair/inspection/emergency),
  status (cancelled; completed for complete/closed/archived/invoiced/paid or a
  completion date; in progress; else scheduled when it has a date, completed when
  an undated status is in the past, new otherwise), schedule (tenant time zone; a
  date alone starts at 08:00; an end time alone is on the start's day), the site
  at its address (created when missing; the customer's only site when the row has
  no address), PO number, notes. The old job number is kept when it is free.
  Job amounts are not imported (invoices carry the money).
- **Open invoices**: only invoices with a **balance due** are imported (paid,
  draft, void, cancelled and bad-debt rows are skipped). Each becomes a `sent`
  invoice with the old number, issue and due dates and **one line for the unpaid
  balance** ("Balance carried over from Jobber invoice 1003"); the original total
  is in the notes. Payments made in the old system are not re-created, so
  ServicePro's totals and payments start from what is still owed. Amounts are
  `shopspring/decimal` throughout. Overdue imported invoices are picked up by the
  overdue marking and payment reminders like any other sent invoice.

Dates are read as `YYYY-MM-DD`, US `MM/DD/YYYY` (both vendors export US dates for
US accounts), `Jan 2, 2006`, with or without a time, or Excel serial numbers.

## Export formats

Neither vendor publishes its export column lists, so the mappers match headers
**flexibly**: headers are compared lower-cased with only letters and digits
(`Main Phone #s` = `main_phone_s`), the source's own aliases are tried first, then
generic ones, and the preview shows (and lets the user change) the result. Aliases
are in `backend/internal/imports/aliases.go`. Assumed layouts:

### Jobber

- **Clients** (Clients → Export clients, CSV, emailed; "up to 1,500 rows" per
  file, larger lists arrive as several files): `J-ID`, `Display Name`,
  `Company Name`, `Title`, `First Name`, `Last Name`, `Main/Mobile/Home/Work/Fax/Other Phone #s`,
  `E-mails` (may list several), `Tags`, `Billing Street 1/2`, `Billing City`,
  `Billing State`, `Billing Country`, `Billing Zip code`, `Service Property Name`,
  `Service Street 1/2`, `Service City`, `Service State`, `Service Country`,
  `Service Zip code`, `Lead Source`, `Created Date`, `Archived`.
  Confirmed: CSV export with property addresses, contact info and custom fields,
  emailed, split above ~1,500 rows ([Jobber: Export client information](https://help.getjobber.com/en/articles/export-client-information/));
  `J-ID` is the client's ID ([Jobber community](https://community.getjobber.com/discussions/operations-forum/best-jobber-automations/208/replies/1110)).
  The remaining header names are assumptions from Jobber's client-report column
  naming.
- **Properties** (report CSV): `Client name`, `Client email`, `Property name`,
  `Street 1/2`, `City`, `State`/`Province`, `Zip code`/`Postal code` — assumed.
- **Jobs** (report CSV): `Job #`, `Client name`, `Client email`, `Title`,
  `Instructions`, `Job status`, `Service street/city/province/ZIP`,
  `Scheduled start date`, `Closed date`, `Total $` — assumed. Jobber's
  `Job type` (one-off / recurring) is not a ServicePro job type and is ignored.
- **Invoices** (report CSV): `Invoice #`, `Client name`, `Client email`, `Subject`,
  `Status`, `Issued date`, `Due date`, `Total $`, `Balance $`, `Paid $` — assumed.

### Housecall Pro

- **Customers** (Customers → More → Download customer list; admins only, emailed):
  `First Name`, `Last Name`, `Mobile Number`, `Email`, `Company`, `Job Title`,
  `Work Number`, `Tags`, `Notes`, `ID`, then one block per address
  (`Street Line 1`, `Street Line 2`, `City`, `State`, `Postal Code`, `Billing?`,
  `Notes`), repeated for each extra address. The first block is used (repeated
  headers are read as `City (2)` etc.).
- **Jobs data report** (47 columns): `Invoice`, `HCP Id`, `Created At`, `Date`,
  `End Time`, …, `Customer`, `First Name`, `Last Name`, `Email`, `Company`,
  `Mobile Phone`, `Home Phone`, `Customer Tags`, `Address`, `Street`,
  `Street Line 2`, `City`, `State`, `Zip`, `Description`, `Line Items`, `Amount`,
  …, `Paid Amount`, `Due`, …, `Notes`, `Employee`, `Job Status`, `Finished`,
  `Payment`, `Invoice Sent`, ….
- **Invoices**: Housecall Pro has no invoice export; the jobs report carries each
  job's invoice number, amount, paid amount and amount due, so **open invoices are
  imported from the jobs report** (`Invoice`, `Amount`, `Paid Amount`, `Due`,
  issue date `Invoice Sent`; a job whose invoice was never sent is skipped).

Sources: [ServiceTitan: Export your Housecall Pro data](https://help.servicetitan.com/docs/export-your-housecall-pro-data)
(column lists of the customers and jobs reports), [Housecall Pro: Export customer list](https://help.housecallpro.com/en/articles/2847823-export-customer-list).

### Generic CSV / Excel

Any file with a header row. Common headers map automatically (including
ServicePro's own customer template); everything else is matched by the user in
the "Match your columns" step (the `mapping` form field).

## Not imported (yet)

Quotes/estimates, line items, payments and payment history, recurring visit
schedules (create service agreements instead), job assignments/technicians,
custom fields, attachments, and Jobber/Housecall Pro API connections (file
exports only). Non-US customer addresses (the customers table is US-only).
