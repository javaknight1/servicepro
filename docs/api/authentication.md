# Authentication API

Registration, login, sessions, password reset and email verification.

> **Two sign-in paths today.** The SPA signs in with the API's **own JWTs in
> httpOnly cookies** (below). The API also accepts **Neon Auth (Neon's managed
> Better Auth) JWTs** as `Authorization: Bearer` tokens for users linked to a
> Neon identity (`users.auth_user_id`), verified against Neon's JWKS when
> `NEON_AUTH_BASE_URL` is set. The SPA moves to Neon Auth in T158, after which
> the in-house password/JWT code is retired. Swagger UI (`/api/docs`) is the
> source of truth for request/response shapes.

## Endpoints

| Method | Endpoint                            | Description                                                        |
| ------ | ----------------------------------- | ------------------------------------------------------------------ |
| POST   | `/api/v1/auth/register`             | Register (invite code or invitation needed in `invite_only` mode)  |
| POST   | `/api/v1/auth/login`                | Log in; sets the access + refresh cookies                          |
| POST   | `/api/v1/auth/refresh`              | Rotate tokens from the refresh cookie                              |
| POST   | `/api/v1/auth/logout`               | Revoke all of the user's tokens and clear the cookies              |
| POST   | `/api/v1/auth/reset-request`        | Email a password-reset link (rate limited)                         |
| POST   | `/api/v1/auth/reset-password`       | Set a new password with the reset token                            |
| GET    | `/api/v1/auth/verify?token=…`       | Verify an email address (link from the verification email)        |
| POST   | `/api/v1/auth/resend-verification`  | Resend the verification email                                      |
| POST   | `/api/v1/auth/neon/provision`       | Link or create the ServicePro user for a Neon Auth sign-in         |
| GET    | `/api/v1/users/me`                  | The signed-in user                                                 |
| PATCH  | `/api/v1/users/me/profile`          | Update first/last name                                             |

## Register

```http
POST /api/v1/auth/register
```

```json
{
  "email": "user@example.com",
  "password": "SecurePass123!",
  "invite_code": "optional",
  "invitation_token": "optional"
}
```

Passwords need at least 8 characters with an uppercase letter, a lowercase
letter, a number and a special character. `REGISTRATION_MODE` is `open` in
development and `invite_only` elsewhere: without a valid `invite_code` or an
`invitation_token` for that email, registration returns **403**
`registration_closed` (or `invalid_invite_code` / `invalid_invitation`). An
existing email is **409**. Success is **201** with `id`, `email`, `role`,
`createdAt`; the user must verify their email before logging in.

## Login

```http
POST /api/v1/auth/login
X-Requested-With: XMLHttpRequest
```

```json
{ "email": "user@example.com", "password": "SecurePass123!" }
```

Response **200**:

```json
{ "message": "Login successful", "expiresIn": 3600 }
```

The tokens are **not** in the body: they're set as httpOnly cookies
(`access_token`, `refresh_token` by default; `COOKIE_ACCESS_TOKEN_NAME` /
`COOKIE_REFRESH_TOKEN_NAME`). Access tokens last 1 hour, refresh tokens 7 days.

Errors: unknown email, wrong password and locked accounts all return **401**
`invalid_credentials` (no account enumeration); `email_not_verified` (**403**)
is only revealed after a correct password; repeated failures are throttled
(**429**, with backoff per email and IP).

## Refresh and Logout

`POST /auth/refresh` reads the refresh cookie and sets new cookies (**401**
`invalid_token` when missing/expired/revoked). `POST /auth/logout` bumps the
user's token version — invalidating every access and refresh token issued so
far — and clears the cookies.

## Password Reset

```http
POST /api/v1/auth/reset-request
{ "email": "user@example.com" }
```

Always answers **200** with a generic message (no enumeration); rate limited.
The emailed token is single-use and expires after 1 hour.

```http
POST /api/v1/auth/reset-password
{ "token": "token-from-email", "newPassword": "NewStrongP@ssw0rd" }
```

**401** for an invalid/expired token, **400** for a weak password.

## Email Verification

The verification email links to the app, which calls:

```http
GET /api/v1/auth/verify?token=<token>
```

Tokens expire after 24 hours. `POST /auth/resend-verification` with
`{"email": "…"}` sends a new one.

## Neon Auth Provisioning

After a Neon Auth sign-in, a user the API doesn't know yet gets **401**
`account_not_linked`. The client then calls:

```http
POST /api/v1/auth/neon/provision
Authorization: Bearer <Neon Auth JWT>

{ "invite_code": "optional", "invitation_token": "optional" }
```

It links the existing user with the same **verified** email, or creates one
when an invitation or invite code admits it. Idempotent. **403**
`email_not_verified` / `registration_closed` / `invalid_invite_code` /
`invalid_invitation`, **409** `identity_conflict`, **404** when Neon Auth isn't
enabled.

## Making Authenticated Requests

- **Browser (SPA):** cookies are sent automatically on same-origin `/api`
  calls. Every `POST`/`PUT`/`PATCH`/`DELETE` carrying an auth cookie must send
  `X-Requested-With: XMLHttpRequest` or it gets **403** `csrf_check_failed`
  (the axios client in `frontend/src/services/api.ts` does this).
- **Scripts / API clients:** `Authorization: Bearer <token>` also works (an
  API-issued access token, or a Neon Auth JWT for a linked user). Bearer
  requests need no CSRF header.
- **Tenant:** tenant-scoped routes use the caller's membership. With several
  tenants, send `X-Tenant-ID: <uuid>` to pick one — it only selects among
  tenants the caller belongs to and never grants access.

```bash
curl http://localhost:8080/api/v1/customers \
  -H "Authorization: Bearer <access token>" \
  -H "X-Tenant-ID: <tenant uuid>"
```

## Security Notes

- bcrypt (cost 12); password hashes are never returned
- Account lockout and login throttling; generic login errors
- Email verification required before login
- Tokens revoked server-side on logout (token version)
- Rate limits: [README.md](./README.md#rate-limiting)

## Configuration

See `backend/.env.example`: `JWT_SECRET`, cookie settings (`COOKIE_*`),
`REGISTRATION_MODE`, `NEON_AUTH_BASE_URL`, `FRONTEND_URL` (links in emails),
and the email provider (Resend in production — T159; Mailpit/SMTP locally).
Redis is optional; without `REDIS_URL` tokens and rate limits use an in-memory,
per-instance store.

## Database

`users` (email, `password_hash`, verification and lockout fields,
`auth_user_id` for the Neon Auth link — migration `000002`). Tenancy lives in
`tenants` / `tenant_users`, roles in `roles` / `user_roles`; see
[DATABASE.md](../architecture/DATABASE.md).
