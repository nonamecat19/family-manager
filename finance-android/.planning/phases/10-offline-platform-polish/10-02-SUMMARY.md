---
phase: 10-offline-platform-polish
plan: 02
subsystem: ui
tags: [offline, sqflite, connectivity, sync, material-banner, riverpod]

# Dependency graph
requires:
  - phase: 10-offline-platform-polish plan 01
    provides: LocalDatabase, LocalExpenseSource, LocalCategorySource, SyncService, connectivityProvider
provides:
  - Local-first expense creation (write to sqflite, then background sync)
  - Offline banner UI on all shell screens with live pending count
  - Pending-sync dot indicator on unsynced expense tiles
  - Category caching for offline form support
  - Connectivity-triggered auto-sync with SnackBar feedback
affects: []

# Tech tracking
tech-stack:
  added: [sqflite_common_ffi_web]
  patterns: [local-first write, offline banner via ConsumerWidget, unsyncedCountProvider auto-dispose]

key-files:
  created:
    - app/lib/shared/widgets/offline_banner.dart
    - app/test/shared/widgets/offline_banner_test.dart
  modified:
    - app/lib/features/expenses/data/models/expense.dart
    - app/lib/providers/expense_provider.dart
    - app/lib/providers/category_provider.dart
    - app/lib/main.dart
    - app/lib/shared/widgets/app_scaffold.dart
    - app/lib/features/history/presentation/history_screen.dart
    - app/test/shared/widgets/app_scaffold_test.dart
    - app/test/core/router/app_router_test.dart

key-decisions:
  - "unsyncedCountProvider as FutureProvider.autoDispose for live pending count in offline banner"
  - "ref.listen on connectivityProvider for sync trigger (wasOffline && isOnline transition)"
  - "ref.invalidate(unsyncedCountProvider) after sync to refresh banner count"

patterns-established:
  - "Local-first write: insertExpense then background syncPendingExpenses"
  - "Offline banner pattern: ConsumerWidget reads connectivityProvider, collapses to SizedBox.shrink when online"

requirements-completed: [PLAT-02]

# Metrics
duration: 4min
completed: 2026-03-16
---

# Phase 10 Plan 2: Offline App Wiring Summary

**Local-first expense creation with offline banner, pending-sync dots, category caching, and connectivity-triggered sync with SnackBar feedback**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-16T15:29:05Z
- **Completed:** 2026-03-16T15:33:55Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments
- Expense model gains `synced` field and `copyWith` for offline tracking
- ExpenseNotifier writes to local sqflite first, then syncs in background via SyncService
- CategoryNotifier caches categories on server fetch, falls back to cache when offline
- main.dart initializes sqflite (with web factory), creates LocalExpenseSource/LocalCategorySource, overrides providers
- OfflineBanner shows "You're offline" + pending count on all shell screens
- AppScaffold triggers sync on connectivity restore with success/error SnackBars
- History expense tiles show 8px orange dot for unsynced items
- All 116 tests pass (10 new widget tests + 106 existing)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add synced field, local-first providers, DB init** - `7c6ed9c` (feat)
2. **Task 2: Offline banner, pending-sync dots, sync SnackBars, tests** - `4d1718b` (feat)
3. **Fix: Router test provider overrides** - `f039598` (fix, Rule 3 auto-fix)

## Files Created/Modified
- `app/lib/shared/widgets/offline_banner.dart` - MaterialBanner widget showing offline status and pending count
- `app/lib/features/expenses/data/models/expense.dart` - Added synced field and copyWith
- `app/lib/providers/expense_provider.dart` - Local-first writes with LocalExpenseSource and background SyncService
- `app/lib/providers/category_provider.dart` - Cache categories on load, fallback to cache offline
- `app/lib/main.dart` - Async DB init, provider overrides, web sqflite factory
- `app/lib/shared/widgets/app_scaffold.dart` - ConsumerWidget with unsyncedCountProvider, sync trigger, SnackBars
- `app/lib/features/history/presentation/history_screen.dart` - Pending-sync orange dot on _ExpenseTile
- `app/test/shared/widgets/offline_banner_test.dart` - 5 widget tests for OfflineBanner
- `app/test/shared/widgets/app_scaffold_test.dart` - 5 widget tests with provider overrides
- `app/test/core/router/app_router_test.dart` - Added offline provider overrides

## Decisions Made
- unsyncedCountProvider as FutureProvider.autoDispose for live pending count in offline banner
- ref.listen on connectivityProvider for sync trigger (wasOffline && isOnline transition)
- ref.invalidate(unsyncedCountProvider) after sync to refresh banner count

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Router test needed offline provider overrides**
- **Found during:** Full test suite verification after Task 2
- **Issue:** AppScaffold conversion to ConsumerWidget caused HistoryScreen to read categoryStateProvider, which depends on localCategorySourceProvider (throws UnimplementedError without override)
- **Fix:** Added FakeCategoryNotifier, connectivityProvider, and unsyncedCountProvider overrides to router test
- **Files modified:** app/test/core/router/app_router_test.dart
- **Verification:** All 116 tests pass
- **Committed in:** f039598

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Auto-fix necessary to maintain test suite integrity. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 10 complete: all offline infrastructure wired into the app
- App is now offline-capable: expenses created locally, sync on reconnect
- All 10 phases of the project are complete

---
*Phase: 10-offline-platform-polish*
*Completed: 2026-03-16*
