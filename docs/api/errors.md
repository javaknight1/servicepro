# API errors

Every error response from the ServicePro API uses one JSON envelope (T223), and every response
(success or error) carries an `X-Request-ID` header.

```json
{
  "error": "cannot_accept",
  "message": "This quote can no longer be accepted",
  "request_id": "3f6c1b9e-2a4d-4e7b-9a51-0c8d2f1e6b7a",
  "fields": { "email": "email must be a valid email address" },
  "details": ["row 3: email is required"]
}
```

| Key          | Always? | Meaning                                                                                     |
| ------------ | ------- | ------------------------------------------------------------------------------------------- |
| `error`      | yes     | Stable `snake_case` machine code. Clients may branch on it.                                 |
| `message`    | yes     | Human-readable, safe to show to the user. Never contains SQL, hosts or other internals.     |
| `request_id` | yes\*   | Same value as the `X-Request-ID` header; appears on the server's log lines for the request. |
| `fields`     | no      | Validation errors only: request field (JSON name) → problem.                               |
| `details`    | no      | Extra messages, e.g. per-row CSV import errors.                                             |

\* Present on every response written by the API's handlers and middleware. A gateway or proxy error
(e.g. a 502 from the edge) may not use the envelope; clients fall back to the status code there.

The top-level `error`/`message` keys are the shape the API already used before T223, so existing
clients keep working; T223 added `request_id` and `fields` and made `error` always a code.

## Request IDs

- The API keeps an inbound `X-Request-ID` if it is 8–128 characters of `[A-Za-z0-9._:-]`;
  otherwise it generates a UUID. The ID is returned in the `X-Request-ID` response header
  (exposed to browsers by CORS) and in `request_id` of error bodies.
- Every structured log line written with the request's context (`logging.*(c.Request.Context(), …)`)
  carries `request_id`, as does the access log line and Sentry panic reports.
- The web app shows it as "Reference: <id>" on generic error messages so a user's report can be
  matched to a log line.

## Status codes

| Status | When                                                                                            | Typical codes                                          |
| ------ | ----------------------------------------------------------------------------------------------- | ------------------------------------------------------ |
| 400    | Malformed body, failed validation, bad path/query parameter                                     | `validation_failed`, `malformed_json`, `invalid_id`    |
| 401    | Not authenticated (missing/expired session)                                                     | `unauthorized`                                         |
| 403    | Authenticated but not allowed (permission, role, not a member of the tenant)                   | `forbidden`, `tenant_access_denied`, `insufficient_role` |
| 404    | Not found — **including another tenant's rows** (cross-tenant reads are indistinguishable)      | `not_found`, `invitation_not_found`                    |
| 409    | The resource's current state doesn't allow it: invalid status transition, duplicate, race lost  | `conflict`, `cannot_accept`, `cannot_send`             |
| 422    | Well-formed but breaks a business rule                                                          | `unprocessable`                                        |
| 429    | Rate limited                                                                                    | `rate_limited`                                         |
| 500    | Anything unexpected (database, network, bug). Generic message; details only in the server log. | `internal_error`                                       |
| 503    | Dependency unavailable / not ready                                                              | `service_unavailable`                                  |

A state conflict is **409, not 403**: 403 means "you may never do this", 409 means "not in the
current state". An infrastructure failure is **500**, never reported as a domain refusal.

## Backend usage

- Domain failures are typed errors from `backend/internal/errors` (import as `apperrors`):
  `apperrors.Conflict("cannot_accept", "…")`, `NotFound`, `Forbidden`, `BadRequest`,
  `Validation(msg, fields)`, `Unprocessable`. Services return them (or wrap them with `%w`);
  `WithCause` attaches the underlying error for logs only, `WithMessage` refines the text.
- Handlers and middleware write every error through `backend/internal/api/apierror`:
  - `apierror.Respond(c, err, "Failed to accept quote")` — a domain error keeps its
    status/code/message; validation/JSON errors become 400; anything else is logged with the
    request ID and answered `500 internal_error` with the given safe message.
  - `apierror.BindError(c, err)` after `ShouldBind*` — 400 with `fields` (JSON field names).
  - `apierror.Write(c, status, code, message)` / `apierror.WriteResponse(c, status, resp)` for
    explicit responses. A legacy sentence in `error` is normalised: it becomes `message` and the
    code is derived from the status.
- Never put `err.Error()` in a response. Swag annotations use `models.ErrorResponse` for
  `@Failure`.

## Frontend usage

`frontend/src/utils/error.ts`: `getErrorMessage()` returns `message` (falling back to the code),
`getApiErrorInfo()` returns `{status, code, message, requestId, fields}`, and
`getErrorMessageWithReference()` appends "Reference: <id>" for 5xx/generic errors. The request ID is
read from the body or, for endpoints not yet using the envelope, the `X-Request-ID` header.
