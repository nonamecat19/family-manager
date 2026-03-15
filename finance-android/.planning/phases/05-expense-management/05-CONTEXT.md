# Phase 5: Expense Management - Context

**Gathered:** 2026-03-16
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can correct mistakes by editing or deleting any expense. Edit any field (amount, category, note, date) and delete with confirmation. No new expense creation flows — that's Phase 4 (done). No filtering or search — that's Phase 6.

</domain>

<decisions>
## Implementation Decisions

### Edit entry point
- Tap an expense row in the History tab to open the edit form directly (same pattern as Phase 3 tap-to-edit categories)
- History tab is the only entry point for editing — no separate detail screen
- Expense rows show: colored category chip on left, amount on right, date below. Note shown as secondary text only if present
- Reuse the same ExpenseFormScreen from Phase 4, pre-filled with existing values
- AppBar says "Edit Expense" instead of "New Expense"
- Save calls PUT instead of POST
- Same pop + SnackBar ("Expense updated") after saving edits

### Delete interaction
- Two paths to delete: swipe-to-delete on history row AND a delete button inside the edit form
- Swipe-left pattern matches Phase 3 category deletion for consistency
- Delete button on edit screen (trash icon in AppBar or red button at bottom — Claude's discretion)
- Confirmation dialog before deletion: "Delete this $12.50 expense?" with Cancel/Delete buttons
- After delete: row disappears from list + "Expense deleted" SnackBar
- If deleted from edit screen: pop back to history, then SnackBar

### Claude's Discretion
- Delete button placement on edit screen (AppBar trash icon vs bottom red button)
- Swipe-to-delete visual treatment (background color, icon)
- How the history list row layout is structured (ListTile vs custom)
- How ExpenseFormScreen detects edit mode (optional expense parameter vs route parameter)
- Backend PUT endpoint design and validation
- Backend DELETE endpoint design
- Animation for row removal after delete

</decisions>

<specifics>
## Specific Ideas

- "Tap row to edit" should feel as natural as tapping a category in Phase 3 — no intermediate screens
- Confirmation dialog shows the amount so the user can verify it's the right expense
- The edit form is the exact same form as create, just pre-filled — users shouldn't have to learn a new UI

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `app/lib/features/expenses/presentation/expense_form_screen.dart`: Existing create form — add optional expense parameter for edit mode
- `app/lib/features/expenses/data/models/expense.dart`: Expense model with `formatCents()` — use in history list rows and delete dialog
- `app/lib/features/expenses/data/expense_repository.dart`: ExpenseRepository — add `updateExpense()` and `deleteExpense()` methods
- `app/lib/providers/expense_provider.dart`: ExpenseNotifier — add update/delete actions
- `server/internal/handler/expense.go`: ExpenseHandler — add Update and Delete methods following existing Create/List pattern
- `_SelectableCategoryChip` in expense_form_screen.dart: Category chip widget for the history list rows

### Established Patterns
- Go: handler + DB interface + sqlc (auth, categories, expenses)
- Flutter: Riverpod StateNotifier + Repository + Feature-first folders
- Flutter: Swipe-to-delete with Dismissible widget (Phase 3 categories)
- Flutter: Confirmation dialog before destructive actions (Phase 3 category delete)
- Flutter: Pop + SnackBar after successful mutations (Phase 4 expense save)

### Integration Points
- History tab (`history_screen.dart`) already shows expense list — add tap handler and swipe-to-delete
- ExpenseFormScreen route needs edit variant (e.g., `/expenses/:id/edit`)
- ExpenseDB interface needs UpdateExpense and DeleteExpense methods
- Router needs PUT and DELETE `/api/v1/expenses/:id` endpoints

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 05-expense-management*
*Context gathered: 2026-03-16*
