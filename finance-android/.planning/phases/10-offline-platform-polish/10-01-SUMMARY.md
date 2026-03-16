---
phase: 10-offline-platform-polish
plan: 01
subsystem: database
tags: [sqflite, connectivity_plus, offline-first, sync-queue, sqlite]

requires:
  - phase: 04-expense-entry
    provides: Expense model and ExpenseRepository for server sync
  - phase: 03-categories
    provides: Category model for local category cache

provides:
  - LocalDatabase with local_expenses and cached_categories tables
  - LocalExpenseSource for offline expense CRUD with sync status tracking
  - LocalCategorySource for category caching
  - ConnectivityProvider StreamProvider for online/offline detection
  - SyncService with mutex-guarded background sync engine
  - SyncStatus enum and SyncResult class
  - localExpenseSourceProvider and localCategorySourceProvider placeholders

affects: [10-02-offline-platform-polish]

tech-stack:
  added: [sqflite ^2.4.2, connectivity_plus ^7.0.0, uuid ^4.0.0, path_provider ^2.1.0, sqflite_common_ffi_web ^1.1.0, sqflite_common_ffi ^2.3.0]
  patterns: [sync-queue with synced flag, in-memory sqflite for unit tests, mutex for concurrent sync prevention]

key-files:
  created:
    - app/lib/core/database/local_database.dart
    - app/lib/core/database/local_expense_source.dart
    - app/lib/core/database/local_category_source.dart
    - app/lib/core/network/connectivity_provider.dart
    - app/lib/features/sync/domain/sync_state.dart
    - app/lib/features/sync/data/sync_service.dart
    - app/test/core/database/local_expense_source_test.dart
    - app/test/core/database/local_category_source_test.dart
    - app/test/core/network/connectivity_provider_test.dart
    - app/test/features/sync/data/sync_service_test.dart
  modified:
    - app/pubspec.yaml

key-decisions:
  - "sqflite_common_ffi for in-memory SQLite in unit tests -- avoids platform channel mocking"
  - "localExpenseSourceProvider throws UnimplementedError until overridden after DB init in main.dart"
  - "SyncService stops on first DioException rather than retrying -- avoids hammering broken connection"

patterns-established:
  - "In-memory sqflite testing: sqfliteFfiInit() + databaseFactoryFfi + inMemoryDatabasePath in setUp"
  - "Sync queue pattern: synced INTEGER flag (0/1) with server_id populated after POST"
  - "Mutex guard: bool _syncing flag to prevent concurrent sync runs"

requirements-completed: [PLAT-02]

duration: 4min
completed: 2026-03-16
---

# Phase 10 Plan 01: Offline Data Infrastructure Summary

**Local SQLite database with expense sync queue, category cache, connectivity detection, and mutex-guarded sync service -- 10 production+test files, 110 tests green**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-16T15:22:24Z
- **Completed:** 2026-03-16T15:26:22Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments
- Local SQLite database with local_expenses (sync queue) and cached_categories tables
- LocalExpenseSource with insert, getUnsynced (FIFO), markSynced, getAllExpenses, getUnsyncedCount
- LocalCategorySource with cacheCategories (replace-all) and getCachedCategories (sorted)
- ConnectivityProvider as StreamProvider<bool> detecting wifi/mobile/ethernet
- SyncService processing pending expenses with mutex guard and DioException handling
- 14 new unit tests all passing, 110 total tests green with no regressions

## Task Commits

Each task was committed atomically:

1. **Task 1: Add dependencies and create local database + expense/category sources with tests** - `0b0897f` (feat)
2. **Task 2: Create connectivity provider and sync service with tests** - `840fcfa` (feat)

## Files Created/Modified
- `app/pubspec.yaml` - Added sqflite, connectivity_plus, uuid, path_provider, sqflite_common_ffi dependencies
- `app/lib/core/database/local_database.dart` - SQLite database singleton with two tables and test injection
- `app/lib/core/database/local_expense_source.dart` - Local CRUD for expenses with sync status tracking
- `app/lib/core/database/local_category_source.dart` - Category cache with replace-all and sorted retrieval
- `app/lib/core/network/connectivity_provider.dart` - Riverpod StreamProvider for connectivity state
- `app/lib/features/sync/domain/sync_state.dart` - SyncStatus enum and SyncResult class
- `app/lib/features/sync/data/sync_service.dart` - Background sync engine with mutex and provider definitions
- `app/test/core/database/local_expense_source_test.dart` - 6 tests for expense source operations
- `app/test/core/database/local_category_source_test.dart` - 3 tests for category cache operations
- `app/test/core/network/connectivity_provider_test.dart` - 1 type verification test
- `app/test/features/sync/data/sync_service_test.dart` - 4 tests for sync service behavior

## Decisions Made
- Used sqflite_common_ffi for in-memory SQLite in unit tests to avoid platform channel complexity
- localExpenseSourceProvider and localCategorySourceProvider throw UnimplementedError until overridden after DB init in main.dart (Plan 02 will wire these up)
- SyncService stops processing on first DioException rather than retrying remaining items, to avoid hammering a broken connection

## Deviations from Plan

None - plan executed exactly as written.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All offline data infrastructure is ready for Plan 02 to wire into existing UI and notifiers
- Plan 02 will override the local source providers in main.dart after database initialization
- Plan 02 will modify ExpenseNotifier for local-first writes and add the offline banner UI

---
*Phase: 10-offline-platform-polish*
*Completed: 2026-03-16*
