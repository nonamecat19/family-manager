---
phase: 04-expense-entry
plan: 01
subsystem: api
tags: [go, gin, sqlc, postgresql, expenses, rest-api]

requires:
  - phase: 03-categories
    provides: "categories table, CategoryDB/CategoryHandler pattern"
  - phase: 01-foundation
    provides: "Go+Gin server, sqlc+goose workflow, pgx/v5 pool"
  - phase: 02-authentication
    provides: "AuthMiddleware, user_id context injection, protected routes"
provides:
  - "POST /api/v1/expenses endpoint for creating expenses"
  - "GET /api/v1/expenses endpoint for listing user expenses with pagination"
  - "expenses table with BIGINT amount_cents, FK to categories"
  - "ExpenseDB interface and PgExpenseDB for testability"
affects: [04-expense-entry, 05-dashboard]

tech-stack:
  added: []
  patterns: ["ExpenseDB interface mirroring CategoryDB for handler testability"]

key-files:
  created:
    - server/internal/db/migrations/00004_expenses.sql
    - server/internal/db/queries/expenses.sql
    - server/internal/handler/expense.go
    - server/internal/handler/expense_db.go
    - server/internal/handler/expense_test.go
  modified:
    - server/internal/router/router.go
    - server/cmd/api/main.go
    - server/internal/db/sqlc/models.go
    - server/internal/db/sqlc/querier.go
    - server/internal/db/sqlc/expenses.sql.go

key-decisions:
  - "ExpenseDB interface with MockExpense struct mirroring CategoryDB pattern exactly"
  - "amount_cents as int64 (BIGINT) -- integer cents, never floating-point"
  - "expense_date defaults to today if omitted in request"
  - "FK constraint on category_id with ON DELETE RESTRICT to prevent orphaned expenses"

patterns-established:
  - "ExpenseDB interface: same testability pattern as CategoryDB and AuthDB"
  - "Date-only field: stored as DATE, parsed/formatted as 2006-01-02 string in JSON"

requirements-completed: [EXP-01, EXP-05]

duration: 3min
completed: 2026-03-15
---

# Phase 4 Plan 1: Expense Entry API Summary

**Go expense API with POST/GET endpoints, BIGINT amount_cents, expenses table with FK to categories, and 10 handler unit tests**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-15T21:40:14Z
- **Completed:** 2026-03-15T21:43:00Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments
- Expenses table migration with BIGINT amount_cents, DATE expense_date, FK to categories (ON DELETE RESTRICT)
- POST /api/v1/expenses with full validation (category_id required, amount_cents > 0, date format, default date)
- GET /api/v1/expenses with limit/offset pagination (default limit=50)
- 10 handler unit tests covering success, validation, defaults, and pagination

## Task Commits

Each task was committed atomically:

1. **Task 1: Expenses migration, sqlc queries, ExpenseDB interface, and PgExpenseDB** - `f50cef5` (feat)
2. **Task 2 RED: Failing expense handler tests** - `e8a6852` (test)
3. **Task 2 GREEN: ExpenseHandler Create/List endpoints, route wiring, and tests** - `8ab3c69` (feat)

## Files Created/Modified
- `server/internal/db/migrations/00004_expenses.sql` - Expenses table with indexes
- `server/internal/db/queries/expenses.sql` - CreateExpense and GetExpensesByUser queries
- `server/internal/db/sqlc/expenses.sql.go` - sqlc-generated Go code for expense queries
- `server/internal/db/sqlc/models.go` - Updated with Expense model
- `server/internal/db/sqlc/querier.go` - Updated with expense query methods
- `server/internal/handler/expense.go` - ExpenseDB interface, ExpenseHandler with Create/List
- `server/internal/handler/expense_db.go` - PgExpenseDB implementing ExpenseDB via sqlc
- `server/internal/handler/expense_test.go` - 10 handler unit tests with mockExpenseDB
- `server/internal/router/router.go` - Expense routes in protected group
- `server/cmd/api/main.go` - PgExpenseDB wired into router.Setup

## Decisions Made
- ExpenseDB interface with MockExpense struct mirroring CategoryDB pattern exactly
- amount_cents stored as BIGINT (int64), never floating-point -- follows EXP-05 integer cents decision
- expense_date defaults to today when omitted from request body
- FK constraint ON DELETE RESTRICT on category_id to prevent deleting categories with expenses

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Expense API ready for Flutter expense form (Plan 02)
- POST/GET endpoints wired in protected group with auth middleware
- Migration ready to apply with goose

---
*Phase: 04-expense-entry*
*Completed: 2026-03-15*
