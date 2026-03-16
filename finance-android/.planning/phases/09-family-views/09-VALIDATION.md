---
phase: 9
slug: family-views
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-16
---

# Phase 9 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + httptest (Go), flutter_test + mocktail (Flutter) |
| **Config file** | none — standard Go/Flutter test setup |
| **Quick run command** | `cd server && go test ./internal/handler/ -run TestFamilyView -v` / `cd app && flutter test test/features/family/` |
| **Full suite command** | `cd server && go test ./...` / `cd app && flutter test` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run quick run for affected layer (Go or Flutter)
- **After every plan wave:** Run full suite for both Go and Flutter
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 09-01-01 | 01 | 1 | FAM-03 | unit | `cd server && go test ./internal/handler/ -run TestFamilyFeed -v` | ❌ W0 | ⬜ pending |
| 09-01-01 | 01 | 1 | FAM-03 | unit | `cd server && go test ./internal/handler/ -run TestFamilyFeedNoFamily -v` | ❌ W0 | ⬜ pending |
| 09-01-01 | 01 | 1 | FAM-04 | unit | `cd server && go test ./internal/handler/ -run TestFamilySummary -v` | ❌ W0 | ⬜ pending |
| 09-02-01 | 02 | 2 | FAM-03 | widget | `cd app && flutter test test/features/family/presentation/family_feed_screen_test.dart` | ❌ W0 | ⬜ pending |
| 09-02-01 | 02 | 2 | FAM-04 | widget | `cd app && flutter test test/features/family/presentation/family_summary_screen_test.dart` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `server/internal/handler/family_view_test.go` — stubs for FAM-03, FAM-04 (Go handler tests with mockFamilyViewDB)
- [ ] `app/test/features/family/presentation/family_feed_screen_test.dart` — stubs for FAM-03 (family feed UI)
- [ ] `app/test/features/family/presentation/family_summary_screen_test.dart` — stubs for FAM-04 (family summary UI)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Family views update when any member logs/edits/deletes an expense | FAM-03, FAM-04 | Requires multi-user concurrent interaction | 1. Log in as User A (family member), 2. Log in as User B (same family) in second session, 3. User A adds expense, 4. User B refreshes family feed — new expense visible |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
