# Claude Code Instructions for ServicePro

This file contains project-specific instructions for Claude Code to follow when working on this codebase.

## Database Migrations

**IMPORTANT: Do NOT create separate migration files.**

All database schema changes should be made directly to the main schema file:

- `/backend/migrations/001_schema.sql`

When adding new tables, columns, indexes, or seed data:

1. Add the changes to the appropriate section in `001_schema.sql`
2. Use `ON CONFLICT` clauses for INSERT statements to make them idempotent
3. Follow the existing patterns for table creation, indexes, and triggers

This approach is used during development. When we push to production, we will implement proper incremental migrations.

## Project Structure

### Backend (Go)

- Models: `/backend/internal/models/`
- Repositories: `/backend/internal/repository/`
- Services: `/backend/internal/services/`
- Handlers: `/backend/internal/api/handlers/`
- Routes: `/backend/internal/api/routes/`
- Permissions: `/backend/internal/permissions/`

### Frontend (React/TypeScript)

- Types: `/frontend/src/types/`
- Services: `/frontend/src/services/`
- Stores: `/frontend/src/store/`
- Components: `/frontend/src/components/`
- Pages: `/frontend/src/pages/`
- Routes: `/frontend/src/routes/`

## Coding Conventions

### Backend

- Use GORM for database operations
- Follow existing error handling patterns with custom error types
- Add logging with contextual prefixes (e.g., `[SERVICE-NAME]`)
- Use UUID for primary keys

### Frontend

- Use Zustand for state management
- Use axios-based API services (see `/frontend/src/services/api.ts`)
- Follow existing component patterns in `/frontend/src/components/shared/`
- Use TypeScript strict mode

## Permissions

When adding new features that require permissions:

1. Add constants to `/backend/internal/permissions/constants.go`
2. Update `AllPermissions()` function
3. Update `DefaultRolePermissions()` for appropriate roles
4. Update `PermissionsByResource()` map
5. Add permission grants to `001_schema.sql` seed data
