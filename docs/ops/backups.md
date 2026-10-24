# Database backups

> **Not live yet.** There is no production database (the host is undecided —
> Supabase is the likely choice). Until the `MIGRATION_DATABASE_URL` secret
> exists, the nightly workflow skips itself.

Nightly at 08:00 UTC, `.github/workflows/db-backup.yml` dumps production to the
Cloudflare R2 bucket **`servicepro-backups`**:

```
YYYY/MM/DD/public.dump    schema + data of the public schema (pg_dump -Fc)
```

ServicePro keeps its own `users` table and JWT auth in `public`, so unlike
apps on Supabase Auth there is no separate `auth.*` dump. Run it by hand from
Actions → "Database backup" → Run workflow.

Retention is the bucket's job — the workflow never deletes. Give the bucket a
lifecycle rule (e.g. `expire-30d`). Any provider-side backups (Supabase keeps
none on Free, 7 days on Pro) are separate; this copy exists so a lost or deleted
database project is recoverable.

## One-time setup

1. Cloudflare dashboard → R2 → create bucket `servicepro-backups`; add a
   lifecycle rule deleting objects after 30 days.
2. R2 → Manage API tokens → **Object Read & Write**, scoped to
   `servicepro-backups`.
3. Store the credentials:
   ```bash
   gh secret set R2_ACCESS_KEY_ID
   gh secret set R2_SECRET_ACCESS_KEY
   gh secret set CLOUDFLARE_ACCOUNT_ID       # if not already set for the Worker deploy
   gh secret set MIGRATION_DATABASE_URL      # session pooler / direct URL, port 5432
   ```
4. Run the workflow once by hand and check the step summary lists the dump.
5. Add the bucket to fleet costs (R2 free tier: 10 GB-month, 1M writes).

## Restore

Download (R2 credentials above, or the dashboard):

```bash
export AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=... AWS_DEFAULT_REGION=auto
EP=https://<account-id>.r2.cloudflarestorage.com
aws s3 cp s3://servicepro-backups/2026/10/16/public.dump ./restore/ --endpoint-url "$EP"
```

Use `pg_restore` at (at least) the server's major version, e.g.
`docker run --rm -v "$PWD/restore:/r" postgres:18 pg_restore ...`.

**Into an empty database** (session/direct URL, port 5432):

```bash
pg_restore --no-owner --no-privileges --dbname "$TARGET_URL" restore/public.dump
cd backend && MIGRATION_DATABASE_URL="$TARGET_URL" DATABASE_URL="$TARGET_URL" go run ./cmd/migrate version
# expect the same version the dump was taken at, dirty=false
```

Then point the `DATABASE_URL` / `MIGRATION_DATABASE_URL` secrets at the new
database (see [`cloud-run-setup.md`](./cloud-run-setup.md) §5, §8).

**Inspect without restoring:** `pg_restore --list restore/public.dump`.

**Local drill** (once a quarter — a backup nobody has restored is a guess):

```bash
docker run -d --name sp-restore -e POSTGRES_PASSWORD=x -p 5433:5432 postgres:18
pg_restore --no-owner --no-privileges \
  --dbname "postgresql://postgres:x@localhost:5433/postgres" restore/public.dump
psql "postgresql://postgres:x@localhost:5433/postgres" -c "select count(*) from customers"
docker rm -f sp-restore
```
