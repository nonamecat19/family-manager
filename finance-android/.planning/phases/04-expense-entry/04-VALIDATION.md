---
phase: 4
slug: expense-entry
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-15
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Go)** | Go testing + httptest (standard library) |
| **Framework (Flutter)** | flutter_test + mocktail 1.0.0 |
| **Config file** | None needed (standard Go test + Flutter test) |
| **Quick run (Go)** | `cd server && go test ./internal/handler/ -run TestExpense -v` |
| **Quick run (Flutter)** | `cd app && flutter test test/features/expenses/` |
| **Full suite (Go)** | `cd server && go test ./...` |
| **Full suite (Flutter)** | `cd app && flutter test` |
| **Estimated runtime** | ~15 seconds (Go) + ~20 seconds (Flutter) |

---

## Sampling Rate

- **After every task commit:** Run quick command for affected layer
- **After every plan wave:** Run full suite for both Go and Flutter
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 35 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 1 | EXP-01 | unit (Go) | `cd server && go test ./internal/handler/ -run TestCreateExpense_Success -v` | ❌ W0 | ⬜ pending |
| 04-01-02 | 01 | 1 | EXP-01 | unit (Go) | `cd server && go test ./internal/handler/ -run TestCreateExpense_MissingFields -v` | ❌ W0 | ⬜ pending |
| 04-01-03 | 01 | 1 | EXP-05 | unit (Go) | `cd server && go test ./internal/handler/ -run TestCreateExpense_AmountCents -v` | ❌ W0 | ⬜ pending |
| 04-01-04 | 01 | 1 | EXP-05 | unit (Go) | `cd server && go test ./internal/handler/ -run TestCreateExpense_InvalidAmount -v` | ❌ W0 | ⬜ pending |
| 04-02-01 | 02 | 2 | EXP-01 | widget (Flutter) | `cd app && flutter test test/features/expenses/presentation/expense_form_screen_test.dart` | ❌ W0 | ⬜ pending |
| 04-02-02 | 02 | 2 | EXP-02 | widget (Flutter) | included in form screen test | ❌ W0 | ⬜ pending |
| 04-02-03 | 02 | 2 | EXP-02 | widget (Flutter) | included in form screen test | ❌ W0 | ⬜ pending |
| 04-02-04 | 02 | 2 | EXP-05 | unit (Dart) | `cd app && flutter test test/features/expenses/data/models/expense_test.dart` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `server/internal/handler/expense_test.go` — stubs for EXP-01, EXP-05 (handler tests)
- [ ] `app/test/features/expenses/presentation/expense_form_screen_test.dart` — covers EXP-01, EXP-02
- [ ] `app/test/features/expenses/data/models/expense_test.dart` — covers EXP-05 (cents formatting)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Quick expense under 3 seconds | EXP-02 | Timing is subjective/device-dependent | Open app → tap FAB → enter amount → tap category → tap save. Stopwatch from FAB tap to save confirmation. |
| Expense appears immediately in app | EXP-01 | Requires visual confirmation of UI update | After saving expense, verify it appears in the list without pull-to-refresh |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 35s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
