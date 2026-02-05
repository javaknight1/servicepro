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

**IMPORTANT: Never use `any` type.** Always use proper TypeScript types:

- For unknown error types in catch blocks, use `unknown` and the `getErrorMessage()` utility from `/frontend/src/utils/error.ts`
- For third-party library callbacks, create typed interfaces (see examples in `/frontend/src/components/calendar/types.ts`)
- For test mocks, create proper mock interfaces instead of using `as any`
- If you must cast an incomplete mock object, use `as unknown as ProperType` instead of `as any`

## Permissions

When adding new features that require permissions:

1. Add constants to `/backend/internal/permissions/constants.go`
2. Update `AllPermissions()` function
3. Update `DefaultRolePermissions()` for appropriate roles
4. Update `PermissionsByResource()` map
5. Add permission grants to `001_schema.sql` seed data

## TODO Task Management

The `/TODO.md` file uses an indexed task system. Each task has a unique ID (e.g., T001, T002).

When implementing a task from TODO.md:

1. Look up the task by its ID in the Task Index table
2. Find the detailed description in the appropriate priority section (P0, P1, P2, P3)
3. After completing, mark it complete in both the Sprint Roadmap and the detailed section
4. Move completed items to the "Verified Complete" section with a brief description

When adding new tasks to TODO.md:

1. Assign the next available ID (T019, T020, etc.)
2. Add to the Task Index table with priority, category, and brief description
3. Add to the appropriate Sprint Roadmap section with the ID
4. Add detailed description in the appropriate priority section (P0/P1/P2/P3)

---

## Environment Variables

**IMPORTANT: Always update both `.env` files when adding new environment variables.**

When adding new environment variables to the backend:

1. Add to `/backend/.env.example`:
   - Include a descriptive comment explaining the variable
   - Use placeholder/example values for sensitive data (e.g., `sk_test_xxxx`, `your-api-key-here`)
   - Use actual default values for non-sensitive configuration

2. Add to `/backend/.env`:
   - For sensitive values that need to be obtained elsewhere, add a `# TODO:` comment
   - For non-sensitive defaults, use the same values as `.env.example`
   - Example: `STRIPE_API_KEY=  # TODO: Get from Stripe Dashboard`

3. Update `/backend/config/config.go`:
   - Add the field to the appropriate config struct
   - Add the `getEnv()` call in `Load()` with a sensible default

This ensures developers can quickly set up their environment by copying `.env.example` to `.env`.

---

## Makefile

**IMPORTANT: Use the single root Makefile for all build tasks.**

All make targets should be defined in the root `/Makefile`. Do NOT create separate Makefiles in subdirectories (e.g., `/backend/Makefile` or `/frontend/Makefile`).

Key targets:

- `make swagger` - Generate Swagger documentation from Go annotations
- `make generate-api` - Generate TypeScript API types from Swagger
- `make docs` - Run the full API documentation pipeline (swagger + generate-api)
- `make help` - List all available targets

When adding new backend or frontend build tasks, add them to the root Makefile following the existing patterns for `backend-*` and `frontend-*` targets.

---

## API Documentation (OpenAPI/Swagger)

**IMPORTANT: Keep API documentation in sync with code changes.**

When modifying backend API endpoints (handlers), always update the Swagger annotations:

1. **Adding a new endpoint**: Add full swagger annotations above the handler function:

   ```go
   // @Summary      Short description
   // @Description  Longer description
   // @Tags         ResourceName
   // @Accept       json
   // @Produce      json
   // @Security     BearerAuth
   // @Param        id   path      string  true  "Resource ID"
   // @Param        body body      models.RequestType  true  "Request body"
   // @Success      200  {object}  models.ResponseType
   // @Failure      400  {object}  models.ErrorResponse
   // @Router       /resource/{id} [get]
   ```

2. **Modifying an endpoint**: Update the annotations to reflect changes to:
   - Request/response types
   - Path or query parameters
   - Status codes
   - Security requirements

3. **Removing an endpoint**: Remove the handler and its annotations

4. **After making changes**: Regenerate the documentation:
   ```bash
   make docs
   ```
   This regenerates both the Swagger JSON and the frontend TypeScript types.

The generated files are:

- `backend/docs/swagger.json` - OpenAPI specification
- `frontend/src/types/api.generated.ts` - TypeScript types for API responses
