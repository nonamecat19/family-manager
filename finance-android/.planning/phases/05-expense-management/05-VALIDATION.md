---
phase: 5
slug: expense-management
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-16
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Flutter)** | flutter_test + mocktail |
| **Framework (Go)** | testing + httptest |
| **Config file** | app/pubspec.yaml (Flutter), go test (Go) |
| **Quick run command (Flutter)** | `cd app && flutter test test/features/expenses/` |
| **Quick run command (Go)** | `cd server && go test ./internal/handler/ -run Expense` |
| **Full suite command** | `cd app && flutter test && cd ../server && go test ./...` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd app && flutter test test/features/expenses/ && cd ../server && go test ./internal/handler/ -run Expense`
- **After every plan wave:** Run `cd app && flutter test && cd ../server && go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 05-XX-01 | 01 | 1 | EXP-03 | unit (Go) | `cd server && go test ./internal/handler/ -run TestUpdateExpense -x` | ❌ W0 | ⬜ pending |
| 05-XX-02 | 01 | 1 | EXP-03 | widget (Flutter) | `cd app && flutter test test/features/expenses/presentation/expense_form_screen_test.dart` | ✅ (needs edit tests) | ⬜ pending |
| 05-XX-03 | 01 | 1 | EXP-03 | widget (Flutter) | `cd app && flutter test test/features/expenses/presentation/expense_form_screen_test.dart` | ✅ (needs edit tests) | ⬜ pending |
| 05-XX-04 | 01 | 1 | EXP-04 | unit (Go) | `cd server && go test ./internal/handler/ -run TestDeleteExpense -x` | ❌ W0 | ⬜ pending |
| 05-XX-05 | 01 | 1 | EXP-04 | widget (Flutter) | `cd app && flutter test test/features/history/presentation/history_screen_test.dart` | ❌ W0 | ⬜ pending |
| 05-XX-06 | 01 | 1 | EXP-04 | widget (Flutter) | `cd app && flutter test test/features/expenses/presentation/expense_form_screen_test.dart` | ✅ (needs delete tests) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `app/test/features/history/presentation/history_screen_test.dart` — stubs for EXP-04 swipe-to-delete, EXP-03 tap-to-edit navigation
- [ ] Go handler tests for Update/Delete in `server/internal/handler/expense_test.go` (file exists but needs new test functions)
- [ ] FakeExpenseNotifier needs updateExpense/deleteExpense methods added

*Existing infrastructure covers Flutter widget tests and Go handler tests. Wave 0 adds test stubs for new behaviors.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Swipe animation visual quality | EXP-04 | Visual behavior not testable in widget tests | Swipe left on expense row, verify red background with delete icon appears smoothly |
| SnackBar-after-pop timing | EXP-03, EXP-04 | Timing-dependent UI behavior | Save/delete from edit screen, verify SnackBar appears after navigation completes |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
