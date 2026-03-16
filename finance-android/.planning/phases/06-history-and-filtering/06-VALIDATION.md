---
phase: 6
slug: history-and-filtering
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-16
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Go)** | Go testing + httptest |
| **Framework (Flutter)** | flutter_test |
| **Config file** | None — both use built-in test runners |
| **Quick run command (Go)** | `cd server && go test ./internal/handler/ -run TestList -v` |
| **Quick run command (Flutter)** | `cd app && flutter test test/features/history/` |
| **Full suite command** | `cd server && go test ./... && cd ../app && flutter test` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd server && go test ./internal/handler/ -run TestList -v` + `cd app && flutter test test/features/history/`
- **After every plan wave:** Run `cd server && go test ./... && cd ../app && flutter test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 06-XX-01 | 01 | 1 | HIST-01 | unit (Go) | `cd server && go test ./internal/handler/ -run TestList -v` | ✅ (partial) | ⬜ pending |
| 06-XX-02 | 01 | 1 | HIST-02 | unit (Go) | `cd server && go test ./internal/handler/ -run TestListFiltered -v` | ❌ W0 | ⬜ pending |
| 06-XX-03 | 01 | 1 | HIST-03 | unit (Go) | `cd server && go test ./internal/handler/ -run TestListFiltered -v` | ❌ W0 | ⬜ pending |
| 06-XX-04 | 02 | 2 | HIST-01 | widget (Flutter) | `cd app && flutter test test/features/history/` | ✅ (partial) | ⬜ pending |
| 06-XX-05 | 02 | 2 | HIST-02, HIST-03 | widget (Flutter) | `cd app && flutter test test/features/history/` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `server/internal/handler/expense_test.go` — add TestListExpenses_WithDateFilter, TestListExpenses_WithCategoryFilter, TestListExpenses_WithCombinedFilters stubs
- [ ] `app/test/features/history/presentation/history_screen_test.dart` — add filter bar visibility, chip interaction, empty filter state stubs
- [ ] FakeExpenseNotifier.loadExpenses() signature updated to accept filter params

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Date range picker visual appearance | HIST-02 | Material showDateRangePicker styling not testable in widget tests | Tap date chip → Custom Range → verify picker renders with correct firstDate/lastDate |
| Filter bar pinned during scroll | HIST-01 | Scroll behavior not reliably testable in widget tests | Scroll expense list, verify filter bar stays visible at top |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
