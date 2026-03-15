---
phase: 05-expense-management
plan: 01
subsystem: api
tags: [go, gin, sqlc, rest, expenses, crud]

requires:
  - phase: 04-expense-entry
    provides: "Expense model, CreateExpense/GetExpensesByUser endpoints, ExpenseDB interface pattern"
provides:
  - "PUT /api/v1/expenses/:id endpoint with full validation"
  - "DELETE /api/v1/expenses/:id endpoint with 204 response"
  - "UpdateExpense and DeleteExpense sqlc queries with user_id filtering"
  - "7 new unit tests for update/delete handlers"
affects: [05-02-flutter-expense-management-ui]

tech-stack:
  added: []
  patterns: [":one for UPDATE RETURNING, :execrows for DELETE with rows-affected check"]

key-files:
  created: []
  modified:
    - server/internal/db/queries/expenses.sql
    - server/internal/handler/expense.go
    - server/internal/handler/expense_db.go
    - server/internal/handler/expense_test.go
    - server/internal/router/router.go

key-decisions:
  - "UpdateExpense uses :one (RETURNING) to return full updated expense JSON, matching Create pattern"
  - "DeleteExpense uses :execrows to detect not-found via rows affected == 0, matching category pattern"

patterns-established:
  - "ErrExpenseNotFound sentinel error pattern matching ErrCategoryNotFound"
  - "Update handler validates same fields as Create (category_id, amount_cents, expense_date format)"

requirements-completed: [EXP-03, EXP-04]

duration: 3min
completed: 2026-03-15
---

# Phase 5 Plan 1: Expense Update/Delete API Summary

**PUT and DELETE expense endpoints with input validation, user_id filtering, and 7 unit tests**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-15T22:31:58Z
- **Completed:** 2026-03-15T22:35:00Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- PUT /api/v1/expenses/:id with category_id, amount_cents, expense_date validation and full JSON response
- DELETE /api/v1/expenses/:id returning 204 on success, 404 for not-found/wrong-user
- 7 new unit tests covering success, not-found, and validation error cases
- All existing tests pass with no regressions

## Task Commits

Each task was committed atomically:

1. **Task 1: Add UpdateExpense/DeleteExpense queries, interface, handlers, routes** - `026c23a` (feat)
2. **Task 2: Add unit tests for Update and Delete handlers** - `007faf5` (test)

## Files Created/Modified
- `server/internal/db/queries/expenses.sql` - Added UpdateExpense (:one) and DeleteExpense (:execrows) queries
- `server/internal/db/sqlc/expenses.sql.go` - Generated sqlc code for new queries
- `server/internal/db/sqlc/querier.go` - Updated interface with new query methods
- `server/internal/handler/expense.go` - ErrExpenseNotFound, Update/Delete handlers, extended ExpenseDB interface
- `server/internal/handler/expense_db.go` - PgExpenseDB UpdateExpense (pgx.ErrNoRows) and DeleteExpense (rows affected)
- `server/internal/handler/expense_test.go` - 7 new tests, extended mock with UpdateExpense/DeleteExpense
- `server/internal/router/router.go` - Registered PUT /:id and DELETE /:id routes

## Decisions Made
- UpdateExpense uses `:one` with RETURNING to send full updated expense JSON (matching Create pattern)
- DeleteExpense uses `:execrows` to detect not-found via rows affected == 0 (matching category Delete pattern)
- Update handler validates same fields as Create for consistency

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Update and Delete API endpoints ready for Flutter UI consumption in plan 05-02
- All CRUD operations now available: Create, List, Update, Delete

---
*Phase: 05-expense-management*
*Completed: 2026-03-15*
