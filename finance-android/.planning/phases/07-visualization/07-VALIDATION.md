---
phase: 7
slug: visualization
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-16
---

# Phase 7 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | flutter_test + Go testing |
| **Config file** | `app/analysis_options.yaml` (very_good_analysis) |
| **Quick run command** | `cd app && flutter test test/features/charts/` |
| **Full suite command** | `cd app && flutter test && cd ../server && go test ./...` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd app && flutter test test/features/charts/`
- **After every plan wave:** Run `cd app && flutter test && cd ../server && go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 07-01-01 | 01 | 1 | VIS-01/02/03 | unit (Go) | `cd server && go test ./internal/handler/ -run TestSummary -v` | ❌ W0 | ⬜ pending |
| 07-02-01 | 02 | 2 | VIS-01 | widget | `cd app && flutter test test/features/charts/presentation/widgets/spending_pie_chart_test.dart` | ❌ W0 | ⬜ pending |
| 07-02-02 | 02 | 2 | VIS-02 | widget | `cd app && flutter test test/features/charts/presentation/widgets/spending_bar_chart_test.dart` | ❌ W0 | ⬜ pending |
| 07-02-03 | 02 | 2 | VIS-03 | widget | `cd app && flutter test test/features/charts/presentation/widgets/monthly_summary_test.dart` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `app/test/features/charts/presentation/widgets/spending_pie_chart_test.dart` — stubs for VIS-01
- [ ] `app/test/features/charts/presentation/widgets/spending_bar_chart_test.dart` — stubs for VIS-02
- [ ] `app/test/features/charts/presentation/widgets/monthly_summary_test.dart` — stubs for VIS-03
- [ ] `server/internal/handler/summary_test.go` — stubs for summary API endpoint
- [ ] `fl_chart` dependency: `cd app && flutter pub add fl_chart` — if not yet added

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Chart animations render smoothly | VIS-01, VIS-02 | Visual quality cannot be asserted programmatically | Run app, navigate to Charts tab, verify animations play without jank |
| Charts update after adding expense | VIS-01/02/03 | Requires multi-screen interaction flow | Add expense, switch to Charts tab, verify data reflects new entry |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
