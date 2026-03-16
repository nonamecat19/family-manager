---
phase: 8
slug: family-groups
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-16
---

# Phase 8 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + httptest (server), flutter_test + mocktail (app) |
| **Config file** | None needed (standard Go/Flutter test setup) |
| **Quick run command** | `cd server && go test ./internal/handler/ -run TestFamily -v` / `cd app && flutter test test/features/family/` |
| **Full suite command** | `cd server && go test ./... && cd ../app && flutter test` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run quick run for affected layer (Go or Flutter)
- **After every plan wave:** Run `cd server && go test ./... && cd ../app && flutter test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 08-01-01 | 01 | 1 | FAM-01 | unit (Go) | `cd server && go test ./internal/handler/ -run TestCreateFamily -v` | ❌ W0 | ⬜ pending |
| 08-01-02 | 01 | 1 | FAM-02 | unit (Go) | `cd server && go test ./internal/handler/ -run TestCreateInvitation -v` | ❌ W0 | ⬜ pending |
| 08-01-03 | 01 | 1 | FAM-02 | unit (Go) | `cd server && go test ./internal/handler/ -run TestAcceptInvitation -v` | ❌ W0 | ⬜ pending |
| 08-01-04 | 01 | 1 | FAM-01 | unit (Go) | `cd server && go test ./internal/handler/ -run TestLeaveFamily -v` | ❌ W0 | ⬜ pending |
| 08-01-05 | 01 | 1 | FAM-02 | unit (Go) | `cd server && go test ./internal/handler/ -run TestRevokeInvitation -v` | ❌ W0 | ⬜ pending |
| 08-02-01 | 02 | 2 | FAM-01 | widget | `cd app && flutter test test/features/family/presentation/family_screen_test.dart` | ❌ W0 | ⬜ pending |
| 08-02-02 | 02 | 2 | FAM-02 | widget | `cd app && flutter test test/features/family/presentation/accept_invite_screen_test.dart` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `server/internal/handler/family_test.go` — stubs for FAM-01, FAM-02 (Go handler tests)
- [ ] `app/test/features/family/presentation/family_screen_test.dart` — stubs for FAM-01 (family management UI)
- [ ] `app/test/features/family/presentation/accept_invite_screen_test.dart` — stubs for FAM-02 (invitation acceptance UI)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Deep link opens app and navigates to invite acceptance | FAM-02 | Requires device/emulator with URL scheme registered | Generate invite link, open in browser/notes app, verify app launches to accept screen |
| Invite link copied to clipboard | FAM-02 | Clipboard interaction requires platform integration | Create invite, tap copy, paste elsewhere to verify link format |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
