# Phase 4: Expense Entry - Context

**Gathered:** 2026-03-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Fast manual expense logging. Users can log an expense with amount, category, optional note, and date. Quick entry (amount + category only) should take under 3 seconds. Amounts stored as integer cents, displayed with locale formatting. No editing or deleting expenses in this phase — that's Phase 5.

</domain>

<decisions>
## Implementation Decisions

### Entry form layout
- FAB opens a full-screen "New Expense" form (not bottom sheet or modal)
- Field order: Amount (with numpad auto-focus) → Category → Note (optional) → Date
- Category picker: horizontal scrollable row of CategoryChips (reuse Phase 3 widget) — tap to select, no extra screen
- Date defaults to "Today", tap to open date picker to change
- Note field is optional — can be left empty for quick entry

### Claude's Discretion
- Amount input UX: calculator-style keypad vs standard text field, decimal handling, currency symbol placement
- Speed optimizations: smart defaults, recent categories first, auto-focus behavior
- After-save behavior: whether to stay on form for batch entry, navigate to history, or show brief confirmation
- Save button style and placement
- Form validation approach (inline errors vs disabled save button)
- How "quick entry" flow differs from full entry (if at all)
- Backend expenses table schema and API design

</decisions>

<specifics>
## Specific Ideas

- The FAB is the primary action in the entire app — expense entry should feel instant and frictionless
- CategoryChip row reuses the exact widget from Phase 3, keeping visual consistency
- "Under 3 seconds" target means: tap FAB, type amount, tap category chip, tap Save — done

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `app/lib/features/categories/presentation/widgets/category_chip.dart`: CategoryChip widget — reuse directly in expense form for category selection
- `app/lib/providers/category_provider.dart`: categoryStateProvider — fetch user's categories for the picker
- `app/lib/core/network/api_client.dart`: Dio with auth interceptor — use for expense API calls
- `server/internal/handler/category.go`: CategoryHandler pattern — follow same structure for ExpenseHandler

### Established Patterns
- Go: handler + DB interface + sqlc pattern (auth, categories)
- Flutter: Riverpod StateNotifier + Repository + Feature-first folders
- Flutter: GoRouter routes outside ShellRoute for full-screen forms (auth screens, category form)

### Integration Points
- FAB in AppScaffold already exists — wire onPressed to navigate to /expenses/new
- Categories API already returns user's categories — expense form fetches these
- History tab (currently empty placeholder) will show expenses after this phase

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 04-expense-entry*
*Context gathered: 2026-03-15*
