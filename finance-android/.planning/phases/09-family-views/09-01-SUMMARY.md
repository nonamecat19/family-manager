---
phase: 09-family-views
plan: 01
subsystem: api
tags: [go, gin, sqlc, postgresql, family-expenses, rest-api]

# Dependency graph
requires:
  - phase: 08-family-groups
    provides: FamilyDB interface, family_members table, families table
provides:
  - GET /api/v1/families/me/expenses endpoint (paginated family expense feed)
  - GET /api/v1/families/me/summary endpoint (per-person and per-category totals)
  - FamilyViewDB interface for mock-based handler testing
  - PgFamilyViewDB implementation wrapping sqlc queries
affects: [09-family-views plan 02 (Flutter UI consumes these endpoints)]

# Tech tracking
tech-stack:
  added: []
  patterns: [FamilyViewDB interface pattern mirroring SummaryDB for family expense aggregation]

key-files:
  created:
    - server/internal/db/queries/family_expenses.sql
    - server/internal/db/sqlc/family_expenses.sql.go
    - server/internal/handler/family_view.go
    - server/internal/handler/family_view_db.go
    - server/internal/handler/family_view_test.go
  modified:
    - server/internal/router/router.go
    - server/cmd/api/main.go

key-decisions:
  - "FamilyViewDB as separate interface from FamilyDB to keep view queries decoupled from CRUD operations"
  - "Total cents computed by summing member totals server-side (same pattern as SummaryHandler)"

patterns-established:
  - "FamilyViewDB interface: separate read-model interface for family expense aggregation queries"

requirements-completed: [FAM-03, FAM-04]

# Metrics
duration: 3min
completed: 2026-03-16
---

# Phase 9 Plan 1: Family Views Backend Summary

**Go backend with sqlc queries for family expense feed and summary endpoints, FamilyViewDB interface, and 7 handler tests**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-16T12:07:53Z
- **Completed:** 2026-03-16T12:11:15Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Three sqlc queries (GetFamilyExpenses, GetFamilyMemberTotals, GetFamilyCategoryTotals) joining expenses across family members
- FamilyViewDB interface with PgFamilyViewDB implementation enabling mock-based testing
- FamilyViewHandler with FamilyFeed (paginated feed) and FamilySummary (monthly breakdown) endpoints
- 7 handler tests covering success, 404, 400, and empty-state cases

## Task Commits

Each task was committed atomically:

1. **Task 1: sqlc queries, FamilyViewDB interface, handler, and PgFamilyViewDB** - `eccb5cd` (feat)
2. **Task 2: Route wiring, main.go integration, and handler tests** - `d0f6703` (feat)

## Files Created/Modified
- `server/internal/db/queries/family_expenses.sql` - Three sqlc queries for family expense aggregation
- `server/internal/db/sqlc/family_expenses.sql.go` - Generated sqlc code
- `server/internal/handler/family_view.go` - FamilyViewDB interface, FamilyViewHandler with FamilyFeed and FamilySummary
- `server/internal/handler/family_view_db.go` - PgFamilyViewDB wrapping sqlc queries
- `server/internal/handler/family_view_test.go` - 7 handler tests with mockFamilyViewDB
- `server/internal/router/router.go` - Added familyViewDB param and /me/expenses + /me/summary routes
- `server/cmd/api/main.go` - Instantiate PgFamilyViewDB and pass to router

## Decisions Made
- FamilyViewDB as separate interface from FamilyDB to keep view queries decoupled from CRUD operations
- Total cents computed by summing member totals server-side (same pattern as SummaryHandler)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Both endpoints ready for Flutter UI consumption in Plan 02
- FamilyViewDB interface enables isolated testing in UI integration tests

---
*Phase: 09-family-views*
*Completed: 2026-03-16*
