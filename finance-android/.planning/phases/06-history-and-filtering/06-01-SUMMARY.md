---
phase: 06-history-and-filtering
plan: 01
subsystem: server/expenses
tags: [filtering, query-params, sqlc, backend]
dependency_graph:
  requires: [phase-04-expense-entry, phase-05-expense-edit-delete]
  provides: [filtered-expense-api]
  affects: [flutter-history-screen]
tech_stack:
  added: []
  patterns: [sqlc.narg-optional-params, pgtype.Date-nullable]
key_files:
  created: []
  modified:
    - server/internal/db/queries/expenses.sql
    - server/internal/db/sqlc/expenses.sql.go
    - server/internal/handler/expense.go
    - server/internal/handler/expense_db.go
    - server/internal/handler/expense_test.go
decisions:
  - "sqlc.narg with ::DATE cast for optional date params (pgtype.Date nullable)"
  - "GetExpensesByUserFiltered replaces GetExpensesByUser in List handler for unified path"
  - "dateToPgDate and stringToNullableUUID helpers for clean nil-to-pgtype conversion"
metrics:
  duration: 2min
  completed: "2026-03-16T07:36:27Z"
---

# Phase 6 Plan 1: Server-Side Expense Filtering Summary

Server-side expense filtering with optional date_from, date_to, and category_id query params using sqlc.narg nullable parameters.

## Tasks Completed

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Add filtered sqlc query, update interface and handler | 701368e | expenses.sql, expense.go, expense_db.go |
| 2 | Add unit tests for filtered list endpoint | 7c27a75 | expense_test.go |

## What Was Built

- **GetExpensesByUserFiltered** sqlc query with optional `date_from`, `date_to`, `category_id` via `sqlc.narg()` and `::DATE` casts
- **Updated List handler** parsing `date_from`, `date_to`, `category_id` query parameters from URL
- **Updated ExpenseDB interface** with new `GetExpensesByUserFiltered` method
- **Helper functions**: `dateToPgDate` (nil-safe time.Time to pgtype.Date) and `stringToNullableUUID` (empty string to invalid pgtype.UUID)
- **Four new tests**: date filter, category filter, combined filters, no-filters backward compatibility

## Deviations from Plan

None - plan executed exactly as written.

## Verification

- `sqlc generate` exits 0
- `go build ./...` exits 0
- All 4 new filter tests pass
- All existing tests pass (full suite green)
