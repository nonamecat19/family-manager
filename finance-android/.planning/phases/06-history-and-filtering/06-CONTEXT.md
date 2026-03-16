# Phase 6: History and Filtering - Context

**Gathered:** 2026-03-16
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can review their spending history and find specific expenses. The history list already exists from Phase 5 (sorted by date, tap-to-edit, swipe-to-delete). This phase adds date range filtering and category filtering to the existing list. No search by text -- that's v2 (FEAT-V2-01). No charts or summaries -- that's Phase 7.

</domain>

<decisions>
## Implementation Decisions

### Filter UI placement
- Persistent top bar with two filter chips: Date and Category
- Filter bar is pinned -- stays visible while the expense list scrolls underneath
- Inactive chips show default labels ("All Dates", "All Categories")
- Active chips show the current filter value (e.g., "This Month" or "Food")
- Tapping an active chip opens its picker with a "Clear" option to reset that filter
- Each filter is cleared independently -- no separate "Clear All" button

### Date range interaction
- Tapping the Date chip opens a popup/bottom sheet with preset options and a custom range option
- Presets: Today, This Week, This Month, Last Month
- "Custom Range" option opens Flutter's showDateRangePicker for arbitrary start/end dates
- Active chip text shows preset name for presets (e.g., "This Month") or short date range for custom ("Mar 1 -- Mar 15")

### Server-side filtering
- Filtering happens server-side -- backend ListExpenses query adds optional date_from, date_to, and category_id query parameters
- Existing pagination (limit/offset) works with filters applied
- Flutter repository passes filter params to the API call

### Claude's Discretion
- Category filter picker style (horizontal chip row, dropdown, or bottom sheet with category list)
- Whether category filter supports multi-select (multiple categories) or single-select (one at a time)
- Empty filter results message and illustration
- How the filter bar integrates with the existing HistoryScreen layout (Column vs SliverAppBar vs other)
- Date preset calculation logic (calendar-based: "This Week" = Monday-Sunday vs rolling: "last 7 days")
- How filter state is managed in Riverpod (separate provider vs extending ExpenseNotifier)
- Backend query construction for optional filter params (COALESCE, conditional WHERE clauses, or separate queries)
- Loading indicator behavior when switching filters

</decisions>

<specifics>
## Specific Ideas

- Filter chips should feel lightweight -- not modal or heavy. Tap, pick, done.
- The existing history list layout (category chip, amount, note/date) stays exactly the same -- filters just narrow what's shown
- "This Month" will likely be the most-used filter -- it should be the first preset option

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `app/lib/features/history/presentation/history_screen.dart`: Current history screen with ListView.builder, _ExpenseTile, Dismissible, _CategoryChipSmall -- add filter bar above the list
- `app/lib/features/expenses/data/expense_repository.dart`: `getExpenses(limit, offset)` -- extend with optional date_from, date_to, category_id params
- `app/lib/providers/expense_provider.dart`: ExpenseNotifier with loadExpenses() -- extend to accept filter params
- `app/lib/providers/category_provider.dart`: categoryStateProvider -- fetch categories for the category filter picker
- `server/internal/db/queries/expenses.sql`: ListExpenses query with WHERE user_id, ORDER BY expense_date DESC, LIMIT/OFFSET -- add optional filter WHERE clauses
- `server/internal/handler/expense.go`: List() handler parses limit/offset from query params -- add date_from, date_to, category_id parsing

### Established Patterns
- Go: handler + DB interface + sqlc (auth, categories, expenses)
- Flutter: Riverpod StateNotifier + Repository + Feature-first folders
- Flutter: Horizontal chip rows for selection (expense form category picker)
- Server: Query params for pagination (limit, offset) -- extend pattern for filter params

### Integration Points
- HistoryScreen is already a tab in the ShellRoute bottom navigation
- ExpenseNotifier.loadExpenses() is called from HistoryScreen.initState and after mutations
- ListExpenses DB method is called by ExpenseHandler.List
- ExpenseDB interface needs updated ListExpenses signature (or new filtered variant)

</code_context>

<deferred>
## Deferred Ideas

None -- discussion stayed within phase scope

</deferred>

---

*Phase: 06-history-and-filtering*
*Context gathered: 2026-03-16*
