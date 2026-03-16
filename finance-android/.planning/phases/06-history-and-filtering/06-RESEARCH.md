# Phase 6: History and Filtering - Research

**Researched:** 2026-03-16
**Domain:** Server-side filtering (PostgreSQL/sqlc/Go) + Flutter filter UI (Riverpod/Material 3)
**Confidence:** HIGH

## Summary

Phase 6 adds date range and category filtering to the existing expense history list. The history list (sorted by date, tap-to-edit, swipe-to-delete) already exists from Phase 5. This phase extends the server's `GetExpensesByUser` query with optional filter parameters, updates the Go handler to parse them from query params, and builds a Flutter filter bar with two FilterChip widgets that trigger server-side filtered requests.

The main technical challenge is sqlc's handling of optional WHERE clauses. The established pattern uses `sqlc.narg()` to generate nullable parameters combined with `OR sqlc.narg('x') IS NULL` guards. The project's sqlc.yaml uses default `query_parameter_limit` (struct-based), so the known "redeclared in this block" bug does not apply. The existing `idx_expenses_user_date` composite index already covers date range filtering efficiently.

**Primary recommendation:** Write a new sqlc query `GetExpensesByUserFiltered` using `sqlc.narg()` for optional `date_from`, `date_to`, and `category_id` parameters. Keep the existing unfiltered query for backward compatibility. On the Flutter side, create a separate `filterStateProvider` (StateNotifier) that holds current filter values, and have HistoryScreen watch it to trigger filtered `loadExpenses()` calls.

<user_constraints>

## User Constraints (from CONTEXT.md)

### Locked Decisions
- Persistent top bar with two filter chips: Date and Category
- Filter bar is pinned -- stays visible while the expense list scrolls underneath
- Inactive chips show default labels ("All Dates", "All Categories")
- Active chips show the current filter value (e.g., "This Month" or "Food")
- Tapping an active chip opens its picker with a "Clear" option to reset that filter
- Each filter is cleared independently -- no separate "Clear All" button
- Tapping the Date chip opens a popup/bottom sheet with preset options and a custom range option
- Presets: Today, This Week, This Month, Last Month
- "Custom Range" option opens Flutter's showDateRangePicker for arbitrary start/end dates
- Active chip text shows preset name for presets (e.g., "This Month") or short date range for custom ("Mar 1 -- Mar 15")
- Filtering happens server-side -- backend ListExpenses query adds optional date_from, date_to, and category_id query parameters
- Existing pagination (limit/offset) works with filters applied
- Flutter repository passes filter params to the API call

### Claude's Discretion
- Category filter picker style: **bottom sheet with list** (decided in UI-SPEC)
- Category filter select mode: **single-select** (decided in UI-SPEC)
- Empty filter results message and illustration: **icon + two-line text** (decided in UI-SPEC)
- Filter bar layout: **Column with Expanded list** (decided in UI-SPEC)
- Date preset calculation: **calendar-based** (decided in UI-SPEC)
- Filter state management: **separate filterStateProvider** (decided in UI-SPEC)
- Backend query construction: research recommends `sqlc.narg()` pattern
- Loading indicator: **CircularProgressIndicator replacing list area** (decided in UI-SPEC)

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope

</user_constraints>

<phase_requirements>

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| HIST-01 | User can view expense history sorted by date (newest first) | Already implemented in Phase 5. Existing `GetExpensesByUser` query uses `ORDER BY expense_date DESC, created_at DESC`. No changes needed for this requirement. |
| HIST-02 | User can filter expenses by date range | New `GetExpensesByUserFiltered` sqlc query with `sqlc.narg('date_from')` and `sqlc.narg('date_to')` optional params. Go handler parses `date_from` and `date_to` query params. Flutter DateFilterChip + preset bottom sheet + showDateRangePicker. |
| HIST-03 | User can filter expenses by category | Same filtered query adds `sqlc.narg('category_id')` optional param. Go handler parses `category_id` query param. Flutter CategoryFilterChip + category picker bottom sheet. |

</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| sqlc | 1.30.0 | SQL query codegen for Go | Already used project-wide. `sqlc.narg()` macro enables nullable params for optional filtering. |
| pgx/v5 | existing | PostgreSQL driver | Already used. `pgtype.Date` and `pgtype.UUID` for nullable filter params. |
| Gin | existing | HTTP framework | Already used. `c.Query()` for optional query param parsing. |
| Flutter SDK | existing | `FilterChip`, `showDateRangePicker`, `showModalBottomSheet` | All needed widgets are built into Material 3. No external packages required. |
| Riverpod 2.6.1 | existing | State management | New `filterStateProvider` follows existing `StateNotifier` pattern. |
| intl | existing | Date formatting | `DateFormat.MMMd()` for chip labels, `DateFormat('yyyy-MM-dd')` for API params. |

### Supporting
No new dependencies needed. All functionality uses existing project libraries.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| sqlc.narg() optional params | Raw SQL / pgx direct queries | Loses sqlc type safety and codegen benefits. Not worth it for 3 optional params. |
| Separate filterStateProvider | Extending ExpenseNotifier with filter fields | Mixes UI filter state with data state. Separate provider is cleaner, matches UI-SPEC decision. |
| Server-side filtering | Client-side filtering | Doesn't scale with large datasets. User decision locks server-side. |

## Architecture Patterns

### Recommended Project Structure
```
server/internal/db/queries/expenses.sql     # Add GetExpensesByUserFiltered query
server/internal/db/sqlc/                    # Regenerated by sqlc generate
server/internal/handler/expense.go          # Extend List() with filter query params
server/internal/handler/expense_db.go       # Update ExpenseDB interface + PgExpenseDB
server/internal/handler/expense_test.go     # Add filtered list tests

app/lib/features/history/domain/            # NEW: filter_state.dart
app/lib/features/history/presentation/      # Modify history_screen.dart, add filter widgets
app/lib/providers/expense_provider.dart     # Extend loadExpenses() with filter params
app/lib/features/expenses/data/expense_repository.dart  # Extend getExpenses() with filter params
```

### Pattern 1: sqlc.narg() for Optional WHERE Clauses
**What:** Use `sqlc.narg('param_name')` to generate nullable Go parameters, combined with `OR sqlc.narg('param_name') IS NULL` guards to skip filtering when NULL.
**When to use:** Any sqlc query needing optional filter parameters.
**Example:**
```sql
-- name: GetExpensesByUserFiltered :many
SELECT id, user_id, category_id, amount_cents, note, expense_date, created_at, updated_at
FROM expenses
WHERE user_id = $1
  AND (expense_date >= sqlc.narg('date_from') OR sqlc.narg('date_from') IS NULL)
  AND (expense_date <= sqlc.narg('date_to') OR sqlc.narg('date_to') IS NULL)
  AND (category_id = sqlc.narg('category_id') OR sqlc.narg('category_id') IS NULL)
ORDER BY expense_date DESC, created_at DESC
LIMIT $2 OFFSET $3;
```

This generates a struct with nullable fields:
```go
type GetExpensesByUserFilteredParams struct {
    UserID     pgtype.UUID
    DateFrom   pgtype.Date  // nullable
    DateTo     pgtype.Date  // nullable
    CategoryID pgtype.UUID  // nullable
    Limit      int32
    Offset     int32
}
```

### Pattern 2: Separate Filter State Provider (Riverpod)
**What:** A dedicated StateNotifier for filter state, separate from expense data state.
**When to use:** When UI filter controls need independent state from the data they filter.
**Example:**
```dart
class FilterState {
  const FilterState({this.dateFrom, this.dateTo, this.categoryId, this.presetLabel});
  final DateTime? dateFrom;
  final DateTime? dateTo;
  final String? categoryId;
  final String? presetLabel; // "Today", "This Month", etc. or null for custom
}

class FilterNotifier extends StateNotifier<FilterState> {
  FilterNotifier() : super(const FilterState());

  void setDatePreset(String label, DateTime from, DateTime to) {
    state = FilterState(dateFrom: from, dateTo: to, categoryId: state.categoryId, presetLabel: label);
  }

  void setCustomDateRange(DateTime from, DateTime to) {
    state = FilterState(dateFrom: from, dateTo: to, categoryId: state.categoryId);
  }

  void setCategory(String? categoryId) {
    state = FilterState(dateFrom: state.dateFrom, dateTo: state.dateTo, categoryId: categoryId, presetLabel: state.presetLabel);
  }

  void clearDate() {
    state = FilterState(categoryId: state.categoryId);
  }

  void clearCategory() {
    state = FilterState(dateFrom: state.dateFrom, dateTo: state.dateTo, presetLabel: state.presetLabel);
  }

  bool get hasActiveFilters => state.dateFrom != null || state.categoryId != null;
}

final filterStateProvider = StateNotifierProvider<FilterNotifier, FilterState>(
  (ref) => FilterNotifier(),
);
```

### Pattern 3: Go Handler Query Param Parsing for Optional Filters
**What:** Extend the existing `List()` handler to parse optional `date_from`, `date_to`, `category_id` query params, following the existing `limit`/`offset` parsing pattern.
**Example:**
```go
func (h *ExpenseHandler) List(c *gin.Context) {
    userID := c.GetString("user_id")
    limit, offset := 50, 0
    // ... existing limit/offset parsing ...

    // Parse optional filter params
    var dateFrom, dateTo *time.Time
    if df := c.Query("date_from"); df != "" {
        if t, err := time.Parse("2006-01-02", df); err == nil {
            dateFrom = &t
        }
    }
    if dt := c.Query("date_to"); dt != "" {
        if t, err := time.Parse("2006-01-02", dt); err == nil {
            dateTo = &t
        }
    }
    categoryID := c.Query("category_id") // empty string = no filter

    expenses, err := h.db.GetExpensesByUserFiltered(userID, limit, offset, dateFrom, dateTo, categoryID)
    // ...
}
```

### Pattern 4: HistoryScreen Watches Filter State to Trigger Reload
**What:** HistoryScreen uses `ref.listen` on `filterStateProvider` to call `loadExpenses()` with current filter values whenever filters change.
**Example:**
```dart
ref.listen(filterStateProvider, (prev, next) {
  ref.read(expenseStateProvider.notifier).loadExpenses(
    dateFrom: next.dateFrom?.toIso8601String().substring(0, 10),
    dateTo: next.dateTo?.toIso8601String().substring(0, 10),
    categoryId: next.categoryId,
  );
});
```

### Anti-Patterns to Avoid
- **Client-side filtering:** Do NOT fetch all expenses and filter in Dart. The user explicitly chose server-side filtering. This won't scale.
- **Modifying the existing GetExpensesByUser query:** Keep it as-is for backward compatibility. Add a new filtered query instead. Other code paths may depend on the original.
- **Storing filter state inside ExpenseNotifier:** Mixes concerns. Filter state is UI state; expense state is data state. Keep them separate.
- **Using FilterChip.onSelected for both open-picker and clear-filter:** Use `onSelected` to open the picker, and `onDeleted` (trailing X) to clear an active filter directly.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Date range picker | Custom calendar widget | `showDateRangePicker()` from Flutter SDK | Material 3 compliant, accessible, locale-aware, battle-tested |
| Filter chip with selected state | Custom Container with GestureDetector | `FilterChip` from Material 3 | Built-in selected/unselected styling, checkmark, onDeleted X icon |
| Modal bottom sheet | Custom overlay or dialog | `showModalBottomSheet()` | Handles dismiss on tap-outside, drag-to-close, safe area insets |
| Date preset calculation | Manual DateTime arithmetic | Standard DateTime constructor + extension methods | `DateTime(year, month + 1, 0)` for last day of month is idiomatic Dart |
| Optional SQL filtering | Dynamic string concatenation in Go | sqlc.narg() with IS NULL guards | Type-safe, sqlc-generated, no SQL injection risk |

**Key insight:** Every component needed for this phase exists in Flutter SDK or sqlc. The only custom code is the filter state management (simple StateNotifier) and wiring the filter UI to the API call.

## Common Pitfalls

### Pitfall 1: sqlc.narg() Type Mismatch for Date Columns
**What goes wrong:** Using `sqlc.narg('date_from')` against a `DATE` column may generate `pgtype.Timestamp` instead of `pgtype.Date` if sqlc can't infer the column type from the `OR ... IS NULL` pattern.
**Why it happens:** sqlc infers types from context. When the parameter appears in both `>=` comparison and `IS NULL` check, inference can be ambiguous.
**How to avoid:** Explicitly cast in SQL: `expense_date >= sqlc.narg('date_from')::DATE` to force `pgtype.Date` generation.
**Warning signs:** sqlc generate produces `pgtype.Timestamp` for date filter params instead of `pgtype.Date`.

### Pitfall 2: showDateRangePicker Returns Null on Cancel
**What goes wrong:** Not handling the null return when user cancels the date range picker.
**Why it happens:** `showDateRangePicker()` returns `Future<DateTimeRange?>` -- nullable.
**How to avoid:** Always check for null: `final range = await showDateRangePicker(...); if (range == null) return;`
**Warning signs:** Null reference errors when user cancels date picker.

### Pitfall 3: Filter State Not Triggering Reload
**What goes wrong:** Changing filter values but the expense list doesn't update.
**Why it happens:** Using `ref.read` instead of `ref.watch`/`ref.listen` for filter state, so changes aren't observed.
**How to avoid:** Use `ref.listen(filterStateProvider, ...)` in initState or build to trigger `loadExpenses()` on filter changes.
**Warning signs:** Filter chips update their labels but the list shows stale data.

### Pitfall 4: Date Boundary Off-by-One
**What goes wrong:** "This Month" filter misses expenses on the last day of the month or includes the first day of the next month.
**Why it happens:** Using `<` instead of `<=` for date_to, or incorrect last-day-of-month calculation.
**How to avoid:** Use inclusive `>=` and `<=` in SQL. Calculate last day of month with `DateTime(year, month + 1, 0)` in Dart. Pass the date_to as the last day (inclusive), not the first day of the next period.
**Warning signs:** Expenses on month boundaries appear/disappear unexpectedly.

### Pitfall 5: ExpenseDB Interface Breaking Change
**What goes wrong:** Changing `GetExpensesByUser` signature breaks mock implementations in tests.
**Why it happens:** Adding filter params to the existing method signature.
**How to avoid:** Add a NEW method `GetExpensesByUserFiltered` to the `ExpenseDB` interface. Update the mock in tests to implement it. Keep existing method for non-filtered paths if needed, or replace all call sites.
**Warning signs:** Test compilation failures after interface change.

### Pitfall 6: Empty UUID String Sent as Category Filter
**What goes wrong:** Sending `category_id=""` (empty string) to the API, which the Go handler may try to parse as UUID and fail.
**Why it happens:** Flutter sends empty string instead of omitting the param entirely.
**How to avoid:** In the repository, only include `category_id` in queryParameters when it's non-null. In the Go handler, only parse `category_id` when the query param is non-empty.
**Warning signs:** 400 errors when no category filter is active.

## Code Examples

### SQL: Filtered Expense Query with sqlc.narg()
```sql
-- Source: sqlc docs + project pattern
-- name: GetExpensesByUserFiltered :many
SELECT id, user_id, category_id, amount_cents, note, expense_date, created_at, updated_at
FROM expenses
WHERE user_id = $1
  AND (expense_date >= sqlc.narg('date_from') OR sqlc.narg('date_from') IS NULL)
  AND (expense_date <= sqlc.narg('date_to') OR sqlc.narg('date_to') IS NULL)
  AND (category_id = sqlc.narg('category_id') OR sqlc.narg('category_id') IS NULL)
ORDER BY expense_date DESC, created_at DESC
LIMIT $2 OFFSET $3;
```

### Go: Nullable pgtype.Date Construction
```go
// Source: pgx/v5 pgtype docs + project expense_db.go pattern
func dateToPgDate(t *time.Time) pgtype.Date {
    if t == nil {
        return pgtype.Date{Valid: false}
    }
    return pgtype.Date{Time: *t, Valid: true}
}

func stringToNullableUUID(s string) pgtype.UUID {
    if s == "" {
        return pgtype.UUID{Valid: false}
    }
    return stringToUUID(s) // existing helper
}
```

### Flutter: Date Preset Calculation (Calendar-Based)
```dart
// Source: Dart DateTime docs + UI-SPEC decision
({DateTime from, DateTime to}) calculateDatePreset(String preset) {
  final now = DateTime.now();
  return switch (preset) {
    'Today' => (
      from: DateTime(now.year, now.month, now.day),
      to: DateTime(now.year, now.month, now.day),
    ),
    'This Week' => (
      // Monday = 1, so subtract (weekday - 1) to get Monday
      from: DateTime(now.year, now.month, now.day - (now.weekday - 1)),
      to: DateTime(now.year, now.month, now.day + (7 - now.weekday)),
    ),
    'This Month' => (
      from: DateTime(now.year, now.month, 1),
      to: DateTime(now.year, now.month + 1, 0), // last day of current month
    ),
    'Last Month' => (
      from: DateTime(now.year, now.month - 1, 1),
      to: DateTime(now.year, now.month, 0), // last day of previous month
    ),
    _ => throw ArgumentError('Unknown preset: $preset'),
  };
}
```

### Flutter: showDateRangePicker Integration
```dart
// Source: Flutter SDK docs
Future<void> _openCustomDateRange(BuildContext context, WidgetRef ref) async {
  final now = DateTime.now();
  final range = await showDateRangePicker(
    context: context,
    firstDate: DateTime(2020),
    lastDate: now,
    currentDate: now,
  );
  if (range == null) return; // user cancelled
  ref.read(filterStateProvider.notifier).setCustomDateRange(
    range.start,
    range.end,
  );
}
```

### Flutter: FilterChip with Active/Inactive States
```dart
// Source: Flutter Material 3 FilterChip docs
FilterChip(
  label: Text(isActive ? activeLabel : 'All Dates'),
  selected: isActive,
  onSelected: (_) => _openDatePicker(context),
  onDeleted: isActive ? () => ref.read(filterStateProvider.notifier).clearDate() : null,
)
```

### Flutter: Repository with Optional Filter Params
```dart
// Source: project expense_repository.dart pattern
Future<List<Expense>> getExpenses({
  int limit = 50,
  int offset = 0,
  String? dateFrom,
  String? dateTo,
  String? categoryId,
}) async {
  final params = <String, dynamic>{'limit': limit, 'offset': offset};
  if (dateFrom != null) params['date_from'] = dateFrom;
  if (dateTo != null) params['date_to'] = dateTo;
  if (categoryId != null) params['category_id'] = categoryId;

  final response = await _dio.get<List<dynamic>>(
    '/expenses',
    queryParameters: params,
  );
  return response.data!
      .cast<Map<String, dynamic>>()
      .map(Expense.fromJson)
      .toList();
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Dynamic SQL string building in Go | sqlc.narg() with IS NULL guards | sqlc 1.20+ | Type-safe optional params without dynamic SQL |
| Client-side filtering | Server-side with query params | Project decision | Scales with data growth, consistent pagination |
| Custom date picker packages | Flutter built-in showDateRangePicker | Flutter 3.x+ | Material 3 support built-in, no external dep |

## Open Questions

1. **sqlc.narg() type inference for DATE column**
   - What we know: sqlc.narg() should infer `pgtype.Date` from the `expense_date` column comparison
   - What's unclear: Whether the `OR ... IS NULL` pattern confuses type inference, requiring explicit `::DATE` cast
   - Recommendation: Try without cast first during `sqlc generate`. Add `::DATE` cast if it generates wrong type.

2. **Whether to replace GetExpensesByUser or add GetExpensesByUserFiltered**
   - What we know: Adding a new query is safer. The filtered query with all-null params behaves identically to the unfiltered query.
   - What's unclear: Whether any code paths other than List() handler use GetExpensesByUser
   - Recommendation: Add new query. Update List() handler to use it. Keep old query in case other code uses it, or remove if no other callers.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | Go testing + httptest (built-in) |
| Framework (Flutter) | flutter_test (built-in) |
| Config file | None needed -- both use built-in test runners |
| Quick run command (Go) | `cd server && go test ./internal/handler/ -run TestList -v` |
| Quick run command (Flutter) | `cd app && flutter test test/features/history/` |
| Full suite command | `cd server && go test ./... && cd ../app && flutter test` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| HIST-01 | Expense history sorted by date (newest first) | unit (Go) + widget (Flutter) | `cd server && go test ./internal/handler/ -run TestList -v` | Partial (Go list test exists, Flutter history test exists) |
| HIST-02 | Filter expenses by date range | unit (Go) + widget (Flutter) | `cd server && go test ./internal/handler/ -run TestListFiltered -v` | No -- Wave 0 |
| HIST-03 | Filter expenses by category | unit (Go) + widget (Flutter) | `cd server && go test ./internal/handler/ -run TestListFiltered -v` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `cd server && go test ./internal/handler/ -run TestList -v` + `cd app && flutter test test/features/history/`
- **Per wave merge:** `cd server && go test ./... && cd ../app && flutter test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `server/internal/handler/expense_test.go` -- add `TestListExpenses_WithDateFilter`, `TestListExpenses_WithCategoryFilter`, `TestListExpenses_WithCombinedFilters` (mock must implement new filtered method)
- [ ] `app/test/features/history/presentation/history_screen_test.dart` -- add filter bar visibility test, filter chip interaction tests, empty filter state test
- [ ] `FakeExpenseNotifier.loadExpenses()` signature must be updated to accept filter params

## Sources

### Primary (HIGH confidence)
- Project source code: `server/internal/db/queries/expenses.sql`, `server/internal/handler/expense.go`, `app/lib/features/history/presentation/history_screen.dart` -- current implementation
- [sqlc 1.30.0 docs - Macros (sqlc.narg)](https://docs.sqlc.dev/en/stable/reference/macros.html) -- nullable parameter generation
- [sqlc docs - Named Parameters](https://docs.sqlc.dev/en/stable/howto/named_parameters.html) -- sqlc.narg() usage pattern
- [Flutter API - showDateRangePicker](https://api.flutter.dev/flutter/material/showDateRangePicker.html) -- date range picker API
- `06-UI-SPEC.md` -- all UI decisions, component inventory, interaction contracts
- `06-CONTEXT.md` -- locked decisions and discretion areas

### Secondary (MEDIUM confidence)
- [sqlc GitHub Issue #200](https://github.com/kyleconroy/sqlc/issues/200) -- optional WHERE parameter patterns and limitations
- [sqlc GitHub Issue #4191](https://github.com/sqlc-dev/sqlc/issues/4191) -- query_parameter_limit interaction with sqlc.narg (not applicable to this project since we use default struct params)

### Tertiary (LOW confidence)
- None -- all findings verified against official sources or project code

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all libraries already in project, no new dependencies
- Architecture: HIGH - extends established patterns (DB interface, sqlc queries, Riverpod StateNotifier, repository)
- Pitfalls: HIGH - based on actual sqlc docs, Flutter SDK behavior, and project code patterns

**Research date:** 2026-03-16
**Valid until:** 2026-04-16 (stable domain, no fast-moving dependencies)
