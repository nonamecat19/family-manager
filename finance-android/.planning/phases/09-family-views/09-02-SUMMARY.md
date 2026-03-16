---
phase: 09-family-views
plan: 02
subsystem: ui
tags: [flutter, riverpod, statenotifier, family-views, widget-tests]

# Dependency graph
requires:
  - phase: 09-family-views
    provides: GET /api/v1/families/me/expenses and GET /api/v1/families/me/summary endpoints
provides:
  - FamilyFeedScreen with month navigation, spender email, and pull-to-refresh
  - FamilySummaryScreen with per-person and per-category dashboard
  - FamilyViewRepository for both family view endpoints
  - FamilyFeedNotifier and FamilySummaryNotifier with sealed states
  - FamilyScreen navigation tiles for family spending views
  - Route registration for /settings/family/expenses and /settings/family/summary
affects: [phase 10 if family views need enhancement]

# Tech tracking
tech-stack:
  added: []
  patterns: [FamilyViewRepository separate from FamilyRepository for view-only endpoints, FakeFamilyFeedNotifier/FakeFamilySummaryNotifier for widget test mocking]

key-files:
  created:
    - app/lib/features/family/data/models/family_expense.dart
    - app/lib/features/family/data/models/family_summary.dart
    - app/lib/features/family/data/family_view_repository.dart
    - app/lib/features/family/domain/family_feed_notifier.dart
    - app/lib/features/family/domain/family_summary_notifier.dart
    - app/lib/features/family/presentation/family_feed_screen.dart
    - app/lib/features/family/presentation/family_summary_screen.dart
    - app/test/features/family/presentation/family_feed_screen_test.dart
    - app/test/features/family/presentation/family_summary_screen_test.dart
  modified:
    - app/lib/features/family/presentation/family_screen.dart
    - app/lib/core/router/app_router.dart
    - app/test/features/family/presentation/family_screen_test.dart

key-decisions:
  - "FamilyViewRepository separate from FamilyRepository to keep view-only endpoints decoupled from CRUD"
  - "Client-side month filtering for feed (server endpoint is paginated without month param)"
  - "Reuse categoryIcons map and parseHexColor from categories feature for icon/color resolution"

patterns-established:
  - "FamilyViewRepository: separate read-model repository for family view endpoints"
  - "Month selector pattern reused from ChartsScreen for consistent UX"

requirements-completed: [FAM-03, FAM-04]

# Metrics
duration: 5min
completed: 2026-03-16
---

# Phase 9 Plan 2: Family Views Flutter UI Summary

**Flutter screens for family expense feed and summary dashboard with month navigation, spender email display, per-person/per-category breakdowns, and 7 widget tests**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-16T12:13:44Z
- **Completed:** 2026-03-16T12:18:22Z
- **Tasks:** 2
- **Files modified:** 12

## Accomplishments
- FamilyExpense and FamilySummary models with fromJson, FamilyViewRepository calling both endpoints
- FamilyFeedNotifier and FamilySummaryNotifier with sealed state (Initial/Loading/Loaded/Error) and Fake variants for testing
- FamilyFeedScreen showing combined family expenses with spender email, category icon, month navigation, empty state, and pull-to-refresh
- FamilySummaryScreen with family total, per-person breakdown with CircleAvatar, per-category breakdown with color chips
- FamilyScreen navigation ListTiles ("Family Expenses" and "Family Summary") inserted before admin section
- 7 new widget tests across feed and summary screens, plus 2 existing test fixes for scrolling

## Task Commits

Each task was committed atomically:

1. **Task 1: Data layer (models, repository, notifiers)** - `0214b06` (feat)
2. **Task 2: Screens, navigation, routes, and widget tests** - `31cdee4` (feat)

## Files Created/Modified
- `app/lib/features/family/data/models/family_expense.dart` - FamilyExpense model with fromJson (includes userEmail)
- `app/lib/features/family/data/models/family_summary.dart` - FamilySummary, MemberTotal, FamilyCategoryTotal models
- `app/lib/features/family/data/family_view_repository.dart` - Repository for /families/me/expenses and /families/me/summary
- `app/lib/features/family/domain/family_feed_notifier.dart` - FamilyFeedNotifier with sealed state and Fake for tests
- `app/lib/features/family/domain/family_summary_notifier.dart` - FamilySummaryNotifier with sealed state and Fake for tests
- `app/lib/features/family/presentation/family_feed_screen.dart` - Family expense feed with month selector and expense tiles
- `app/lib/features/family/presentation/family_summary_screen.dart` - Family summary dashboard with per-person and per-category
- `app/lib/features/family/presentation/family_screen.dart` - Added Family Spending navigation section
- `app/lib/core/router/app_router.dart` - Added /settings/family/expenses and /settings/family/summary routes
- `app/test/features/family/presentation/family_feed_screen_test.dart` - 4 widget tests for feed screen states
- `app/test/features/family/presentation/family_summary_screen_test.dart` - 3 widget tests for summary screen states
- `app/test/features/family/presentation/family_screen_test.dart` - Fixed 2 existing tests for scrolling after nav additions

## Decisions Made
- FamilyViewRepository separate from FamilyRepository to keep view-only endpoints decoupled from CRUD
- Client-side month filtering for feed (server endpoint is paginated without month param)
- Reuse categoryIcons map and parseHexColor from categories feature for icon/color resolution

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed existing family_screen_test scrolling after navigation additions**
- **Found during:** Task 2 (screens and tests)
- **Issue:** Adding navigation ListTiles to FamilyScreen pushed "Copy Invite Link" and "Leave Family" below viewport in tests
- **Fix:** Added scrollUntilVisible() calls in 2 existing test cases
- **Files modified:** app/test/features/family/presentation/family_screen_test.dart
- **Verification:** All 16 family tests pass
- **Committed in:** 31cdee4 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Necessary fix for test correctness after adding navigation widgets. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Both family view screens complete and accessible from FamilyScreen
- FAM-03 (combined expense feed) and FAM-04 (summary dashboard) requirements fulfilled
- Phase 9 complete -- all 2 plans done

---
*Phase: 09-family-views*
*Completed: 2026-03-16*
