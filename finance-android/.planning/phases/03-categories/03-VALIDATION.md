---
phase: 03
slug: categories
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-15
---

# Phase 03 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Go)** | `go test` (standard library) |
| **Framework (Flutter)** | `flutter_test` + `mocktail` 1.0.0 |
| **Config file** | None needed (both use convention) |
| **Quick run (Go)** | `cd server && go test ./internal/handler/ -run Category -v` |
| **Quick run (Flutter)** | `cd app && flutter test test/features/categories/` |
| **Full suite (Go)** | `cd server && go test ./...` |
| **Full suite (Flutter)** | `cd app && flutter test` |
| **Estimated runtime** | ~15 seconds (Go ~5s, Flutter ~10s) |

---

## Sampling Rate

- **After every task commit:** Run quick commands for the relevant stack
- **After every plan wave:** Run full suite for both stacks
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 03-01-01 | 01 | 1 | CAT-01 | unit (handler) | `cd server && go test ./internal/handler/ -run TestCreateCategory -v` | ❌ W0 | ⬜ pending |
| 03-01-02 | 01 | 1 | CAT-02 | unit (handler) | `cd server && go test ./internal/handler/ -run TestUpdateCategory -v` | ❌ W0 | ⬜ pending |
| 03-01-03 | 01 | 1 | CAT-03 | unit (handler) | `cd server && go test ./internal/handler/ -run TestDeleteCategory -v` | ❌ W0 | ⬜ pending |
| 03-02-01 | 02 | 2 | CAT-01, CAT-04, CAT-05 | widget | `cd app && flutter test test/features/categories/presentation/category_form_screen_test.dart` | ❌ W0 | ⬜ pending |
| 03-02-02 | 02 | 2 | CAT-02 | widget | `cd app && flutter test test/features/categories/presentation/category_form_screen_test.dart` | ❌ W0 | ⬜ pending |
| 03-02-03 | 02 | 2 | CAT-03 | widget | `cd app && flutter test test/features/categories/presentation/categories_screen_test.dart` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `server/internal/handler/category_test.go` — stubs for CAT-01, CAT-02, CAT-03 (handler CRUD tests with mockDB)
- [ ] `app/test/features/categories/presentation/categories_screen_test.dart` — stubs for CAT-01, CAT-03 (list/delete UI)
- [ ] `app/test/features/categories/presentation/category_form_screen_test.dart` — stubs for CAT-01, CAT-02, CAT-04, CAT-05 (form with pickers)

*Existing infrastructure covers test framework setup — only test files needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Drag-to-reorder visual feedback | CAT-01 | ReorderableListView animations are visual | Drag a category up/down, verify smooth animation and correct final order |
| Bottom sheet icon/color picker UX | CAT-04, CAT-05 | Visual layout and scroll behavior | Tap icon/color field, verify bottom sheet opens with grid, select item |
| Starter suggestions prompt | CAT-01 | One-time prompt display logic | First visit to Categories, verify prompt appears, tap Add/Skip |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
