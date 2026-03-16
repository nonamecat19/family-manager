---
phase: 10
slug: offline-platform-polish
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-16
---

# Phase 10 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | flutter_test + mocktail ^1.0.0 |
| **Config file** | app/pubspec.yaml (dev_dependencies) |
| **Quick run command** | `cd app && flutter test test/features/sync/` |
| **Full suite command** | `cd app && flutter test` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd app && flutter test test/features/sync/`
- **After every plan wave:** Run `cd app && flutter test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 10-01-01 | 01 | 1 | PLAT-02a | unit | `cd app && flutter test test/features/sync/data/local_expense_source_test.dart` | ❌ W0 | ⬜ pending |
| 10-01-02 | 01 | 1 | PLAT-02b | unit | `cd app && flutter test test/features/sync/data/sync_service_test.dart` | ❌ W0 | ⬜ pending |
| 10-01-03 | 01 | 1 | PLAT-02c | unit | `cd app && flutter test test/features/sync/data/sync_service_test.dart` | ❌ W0 | ⬜ pending |
| 10-01-04 | 01 | 1 | PLAT-02d | unit | `cd app && flutter test test/core/network/connectivity_provider_test.dart` | ❌ W0 | ⬜ pending |
| 10-02-01 | 02 | 2 | PLAT-02e | widget | `cd app && flutter test test/shared/widgets/offline_banner_test.dart` | ❌ W0 | ⬜ pending |
| 10-02-02 | 02 | 2 | PLAT-02f | unit | `cd app && flutter test test/features/categories/data/local_category_source_test.dart` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `test/features/sync/data/local_expense_source_test.dart` — stubs for PLAT-02a
- [ ] `test/features/sync/data/sync_service_test.dart` — stubs for PLAT-02b, PLAT-02c
- [ ] `test/core/network/connectivity_provider_test.dart` — stubs for PLAT-02d
- [ ] `test/shared/widgets/offline_banner_test.dart` — stubs for PLAT-02e
- [ ] `test/features/categories/data/local_category_source_test.dart` — stubs for PLAT-02f
- [ ] sqflite test setup: use `sqflite_common_ffi` for unit tests (in-memory SQLite)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Cross-platform visual consistency | PLAT-02 SC4 | Visual rendering differences across platforms | Run app on Android, iOS, and web; verify layout and styling are consistent |
| Real network disconnect/reconnect sync | PLAT-02 SC2 | Requires actual network state change | Toggle airplane mode, create expense, re-enable, verify sync |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
