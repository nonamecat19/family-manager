---
phase: 03-categories
plan: 01
subsystem: api
tags: [go, gin, sqlc, postgresql, rest, categories]

requires:
  - phase: 02-authentication
    provides: "AuthDB interface pattern, auth middleware, router protected group"
  - phase: 01-foundation
    provides: "Go+Gin server, sqlc+goose workflow, pgx/v5 connection pool"
provides:
  - "Categories table with user_id FK, name, icon, color, sort_order"
  - "CategoryDB interface with mock-based testability"
  - "6 REST endpoints: create, list, update, delete, reorder, bulk-create"
  - "PgCategoryDB wrapping sqlc-generated queries"
affects: [03-categories-plan-02, 04-expenses]

tech-stack:
  added: []
  patterns: [CategoryDB interface pattern mirroring AuthDB, execrows for not-found detection]

key-files:
  created:
    - server/internal/db/migrations/00003_categories.sql
    - server/internal/db/queries/categories.sql
    - server/internal/handler/category.go
    - server/internal/handler/category_db.go
    - server/internal/handler/category_test.go
  modified:
    - server/internal/router/router.go
    - server/cmd/api/main.go
    - server/internal/db/sqlc/models.go
    - server/internal/db/sqlc/querier.go
    - server/internal/db/sqlc/categories.sql.go

key-decisions:
  - "Used :execrows for UpdateCategory and DeleteCategory to detect not-found (0 rows affected)"
  - "No unique index on (user_id, sort_order) -- gaps acceptable, avoids constraint violations during batch reorder"
  - "Skipped CountExpensesByCategory query since expenses table does not exist yet (Phase 4)"

patterns-established:
  - "CategoryDB interface: same mock-based testing pattern as AuthDB"
  - "Reorder endpoint: PUT /categories/reorder registered before PUT /categories/:id to avoid route conflict"

requirements-completed: [CAT-01, CAT-02, CAT-03]

duration: 3min
completed: 2026-03-15
---

# Phase 3 Plan 1: Categories API Summary

**Go backend CRUD API with 6 REST endpoints for expense categories including reorder and bulk-create, following AuthDB interface pattern**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-15T20:53:33Z
- **Completed:** 2026-03-15T20:56:48Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments
- Categories table migration with user_id FK, auto-incrementing sort_order via subquery
- CategoryDB interface + PgCategoryDB implementation following established AuthDB pattern
- 6 REST endpoints behind auth middleware: create, list, update, delete, reorder, bulk-create
- 9 unit tests with mockCategoryDB, full test suite green (25 tests total)

## Task Commits

Each task was committed atomically:

1. **Task 1: Categories migration, sqlc queries, CategoryDB interface, and PgCategoryDB** - `ada761e` (feat)
2. **Task 2: CategoryHandler CRUD implementation with unit tests** - `f0de326` (feat)

## Files Created/Modified
- `server/internal/db/migrations/00003_categories.sql` - Categories table with goose Up/Down
- `server/internal/db/queries/categories.sql` - 6 sqlc queries for category CRUD + reorder
- `server/internal/db/sqlc/categories.sql.go` - sqlc-generated Go code
- `server/internal/db/sqlc/models.go` - Category model added by sqlc
- `server/internal/db/sqlc/querier.go` - Querier interface updated by sqlc
- `server/internal/handler/category.go` - CategoryDB interface, MockCategory, CategoryHandler with 6 methods
- `server/internal/handler/category_db.go` - PgCategoryDB wrapping sqlc queries
- `server/internal/handler/category_test.go` - 9 unit tests with mockCategoryDB
- `server/internal/router/router.go` - Category routes registered under protected group
- `server/cmd/api/main.go` - PgCategoryDB wired into router

## Decisions Made
- Used :execrows instead of :exec for UpdateCategory and DeleteCategory to detect not-found via rows affected count
- No unique index on (user_id, sort_order) to avoid constraint violations during batch reorder operations
- Skipped CountExpensesByCategory query -- expenses table does not exist until Phase 4
- Delete endpoint accepts optional reassign_to query param as no-op placeholder for Phase 4

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Changed sqlc query type from :exec to :execrows for Update/Delete**
- **Found during:** Task 1 (query design)
- **Issue:** :exec discards rows affected count, making it impossible to detect not-found for 404 responses
- **Fix:** Changed UpdateCategory and DeleteCategory queries to :execrows, PgCategoryDB checks rowsAffected == 0
- **Files modified:** server/internal/db/queries/categories.sql, server/internal/handler/category_db.go
- **Verification:** TestUpdateCategory_NotFound and TestDeleteCategory_NotFound pass with 404
- **Committed in:** ada761e (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug fix)
**Impact on plan:** Essential for correctness of 404 responses. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Categories API complete, ready for Flutter UI consumption (Plan 02)
- Migration must be applied to database before testing: `cd server && goose -dir internal/db/migrations postgres "$DATABASE_URL" up`
- Phase 4 (expenses) will add CountExpensesByCategory and wire reassign_to delete logic

---
*Phase: 03-categories*
*Completed: 2026-03-15*
