# Phase 5: Expense Management - Research

**Researched:** 2026-03-16
**Domain:** Flutter expense CRUD + Go REST API update/delete
**Confidence:** HIGH

## Summary

Phase 5 adds edit and delete capabilities to existing expenses. The codebase already has a complete CRUD pattern established in Phase 3 (categories) that this phase mirrors almost exactly: PUT/DELETE endpoints on the Go backend, repository+notifier methods in Flutter, reuse of the existing form screen with an optional parameter for edit mode, Dismissible swipe-to-delete on list rows, and confirmation dialogs before destructive actions.

The implementation is straightforward because every pattern needed already exists in the codebase. The Go backend needs two new sqlc queries (UpdateExpense, DeleteExpense), two new ExpenseDB interface methods, two new handler methods, and two new routes. The Flutter side needs updateExpense/deleteExpense in ExpenseRepository and ExpenseNotifier, an edit-mode parameter on ExpenseFormScreen, tap-to-edit on history rows, and Dismissible wrapping on history rows.

**Primary recommendation:** Follow the Phase 3 category edit/delete pattern exactly -- same handler structure, same DB interface pattern, same Dismissible+confirmDismiss pattern, same form-reuse-with-optional-parameter pattern.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Tap an expense row in the History tab to open the edit form directly (same pattern as Phase 3 tap-to-edit categories)
- History tab is the only entry point for editing -- no separate detail screen
- Expense rows show: colored category chip on left, amount on right, date below. Note shown as secondary text only if present
- Reuse the same ExpenseFormScreen from Phase 4, pre-filled with existing values
- AppBar says "Edit Expense" instead of "New Expense"
- Save calls PUT instead of POST
- Same pop + SnackBar ("Expense updated") after saving edits
- Two paths to delete: swipe-to-delete on history row AND a delete button inside the edit form
- Swipe-left pattern matches Phase 3 category deletion for consistency
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

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| EXP-03 | User can edit any field of an existing expense | Reuse ExpenseFormScreen with optional Expense parameter for edit mode; PUT /api/v1/expenses/:id endpoint; updateExpense in repository+notifier |
| EXP-04 | User can delete an expense with confirmation | Dismissible swipe-to-delete + delete button in edit form; DELETE /api/v1/expenses/:id endpoint; confirmation dialog with amount display; deleteExpense in repository+notifier |
</phase_requirements>

## Standard Stack

### Core (already in project)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Flutter | 3.x | UI framework | Already in use |
| Riverpod | 2.6.1 | State management | Project standard (StateNotifier pattern) |
| go_router | latest | Navigation/routing | Already used for all routes |
| Dio | latest | HTTP client | Already used in all repositories |
| Gin | latest | Go HTTP framework | Already used for all endpoints |
| sqlc | 1.30.0 | Go SQL code gen | Already used for all queries |
| mocktail | latest | Flutter test mocking | Already used in widget tests |

### No new dependencies needed
This phase adds no new libraries. All required functionality is available through existing project dependencies.

## Architecture Patterns

### Recommended Project Structure (files to modify/create)
```
server/
  internal/
    db/queries/expenses.sql         # ADD: UpdateExpense, DeleteExpense queries
    db/sqlc/expenses.sql.go         # REGENERATE: sqlc generate
    handler/expense.go              # ADD: Update(), Delete() methods + request struct
    handler/expense_db.go           # ADD: UpdateExpense(), DeleteExpense() implementations
    handler/expense_test.go         # ADD: Update/Delete test cases
    router/router.go                # ADD: PUT /:id, DELETE /:id routes
app/lib/
  features/expenses/
    data/expense_repository.dart    # ADD: updateExpense(), deleteExpense()
    presentation/expense_form_screen.dart  # MODIFY: add optional Expense param for edit mode
  features/history/
    presentation/history_screen.dart  # MODIFY: add tap-to-edit, swipe-to-delete, category chip
  providers/expense_provider.dart   # ADD: updateExpense(), deleteExpense() methods
  core/router/app_router.dart       # ADD: /expenses/:id/edit route
app/test/
  features/expenses/presentation/expense_form_screen_test.dart  # ADD: edit mode tests
  features/history/presentation/history_screen_test.dart         # NEW: history interaction tests
```

### Pattern 1: Edit Mode via Optional Constructor Parameter
**What:** CategoryFormScreen already does this -- accepts an optional `Category?` parameter. Null = create mode, non-null = edit mode.
**When to use:** For ExpenseFormScreen edit mode.
**Example (from existing category_form_screen.dart):**
```dart
class ExpenseFormScreen extends ConsumerStatefulWidget {
  const ExpenseFormScreen({this.expense, super.key});
  final Expense? expense;
  // ...
  bool get _isEditing => widget.expense != null;
}
```
Pre-fill fields in initState when expense is non-null. AppBar title switches on `_isEditing`. Submit calls updateExpense or createExpense accordingly.

### Pattern 2: Swipe-to-Delete with Confirmation (Dismissible)
**What:** Wrap list items in Dismissible with confirmDismiss callback showing AlertDialog.
**When to use:** For history row swipe-to-delete.
**Example (from existing categories_screen.dart):**
```dart
Dismissible(
  key: ValueKey(expense.id),
  direction: DismissDirection.endToStart,
  background: Container(
    alignment: Alignment.centerRight,
    padding: const EdgeInsets.only(right: 16),
    color: Colors.red,
    child: const Icon(Icons.delete, color: Colors.white),
  ),
  confirmDismiss: (_) => _confirmDelete(context, expense),
  onDismissed: (_) {
    ref.read(expenseStateProvider.notifier).deleteExpense(expense.id);
  },
  child: _ExpenseTile(expense: expense, onTap: () => /* navigate to edit */),
)
```

### Pattern 3: Go Handler Update/Delete with :execrows
**What:** Use sqlc `:execrows` for UPDATE/DELETE to detect not-found via rows affected == 0.
**When to use:** For expense PUT and DELETE endpoints.
**Example (from existing category pattern):**
```go
// In expense.go handler:
func (h *ExpenseHandler) Update(c *gin.Context) {
    id := c.Param("id")
    userID := c.GetString("user_id")
    // ... bind request, validate ...
    err := h.db.UpdateExpense(id, userID, req.CategoryID, req.AmountCents, req.Note, expenseDate)
    if err != nil {
        if errors.Is(err, ErrExpenseNotFound) {
            c.JSON(http.StatusNotFound, gin.H{"error": "Expense not found"})
            return
        }
        // ...
    }
}
```

### Pattern 4: Notifier Update-in-Place for Instant UI
**What:** After a successful API update, replace the expense in the local list rather than reloading. After delete, remove from the local list.
**When to use:** For responsive UX after edit/delete operations.
**Example:**
```dart
// In ExpenseNotifier:
Future<bool> updateExpense({required String id, ...}) async {
  try {
    final updated = await _repository.updateExpense(id: id, ...);
    final currentExpenses = switch (state) {
      ExpenseLoaded(:final expenses) => expenses,
      _ => <Expense>[],
    };
    state = ExpenseLoaded([
      for (final e in currentExpenses)
        if (e.id == id) updated else e,
    ]);
    return true;
  } on Exception catch (e) {
    state = ExpenseError(e.toString());
    return false;
  }
}

Future<bool> deleteExpense(String id) async {
  try {
    await _repository.deleteExpense(id);
    final currentExpenses = switch (state) {
      ExpenseLoaded(:final expenses) => expenses,
      _ => <Expense>[],
    };
    state = ExpenseLoaded(
      currentExpenses.where((e) => e.id != id).toList(),
    );
    return true;
  } on Exception catch (e) {
    state = ExpenseError(e.toString());
    return false;
  }
}
```

### Pattern 5: Route with Extra for Edit Navigation
**What:** Pass the Expense object via GoRouter `extra` parameter (same as category edit).
**When to use:** For navigating to edit screen with expense data.
**Example (from existing app_router.dart):**
```dart
// Route definition:
GoRoute(
  path: '/expenses/edit',
  builder: (context, state) => ExpenseFormScreen(
    expense: state.extra as Expense?,
  ),
),

// Navigation from history screen:
onTap: () => context.push('/expenses/edit', extra: expense),
```

### Anti-Patterns to Avoid
- **Don't fetch single expense from API for edit:** The expense data is already in the list state. Pass it via route extra, don't make a GET /expenses/:id call.
- **Don't reload the full list after edit/delete:** Update the local state in-place for instant UI feedback (the category pattern reloads, but expense lists can be large -- prefer in-place updates).
- **Don't use a separate detail screen:** User decision specifies tap goes directly to edit form, no intermediate detail view.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Swipe-to-delete gesture | Custom GestureDetector | Flutter Dismissible widget | Already used in Phase 3, handles animation/physics |
| Confirmation dialog | Custom overlay | showDialog + AlertDialog | Standard Flutter pattern, already used in Phase 3 |
| Date formatting | Manual string formatting | intl DateFormat | Already imported and used in expense model |
| Amount display | Manual formatting | Expense.formatCents() | Already exists with locale-aware formatting |

## Common Pitfalls

### Pitfall 1: User ID Scoping on Update/Delete
**What goes wrong:** Update/Delete SQL queries that don't filter by user_id allow users to modify other users' expenses.
**Why it happens:** Forgetting the WHERE user_id = $2 clause.
**How to avoid:** Always include `AND user_id = $2` in UPDATE/DELETE queries (matches category pattern).
**Warning signs:** Tests pass with single-user mock but fail in multi-user scenarios.

### Pitfall 2: Dismissible Key Uniqueness
**What goes wrong:** Dismissible animation glitches or wrong item deleted.
**Why it happens:** Using index instead of expense.id as the key.
**How to avoid:** Always use `ValueKey(expense.id)` as the Dismissible key.

### Pitfall 3: SnackBar After Pop
**What goes wrong:** SnackBar not visible because context is gone after Navigator.pop.
**Why it happens:** Showing SnackBar on a context that's already been popped.
**How to avoid:** Pop first, then show SnackBar. The SnackBar should be shown on the parent (history) screen's context. Use the pattern from Phase 4: `Navigator.pop(context)` then `ScaffoldMessenger.of(context).showSnackBar(...)` -- this works because ScaffoldMessenger is above the route in the widget tree.

### Pitfall 4: Amount Pre-fill Formatting
**What goes wrong:** Edit form shows "1250" instead of "12.50" for a 1250 cents expense.
**Why it happens:** Pre-filling with raw amountCents instead of formatted dollars.
**How to avoid:** Convert cents to dollars string: `(expense.amountCents / 100).toStringAsFixed(2)` for the text controller initial value. Don't use formatCents() as that includes the $ symbol which would conflict with the prefix.

### Pitfall 5: Category Chip Resolution in History Row
**What goes wrong:** History row shows category ID instead of name/color/icon.
**Why it happens:** Expense model only stores categoryId, not the full category object.
**How to avoid:** Look up category from the loaded categories state using the expense's categoryId. If category not found (deleted), show a fallback chip.

### Pitfall 6: PUT Response Should Return Updated Expense
**What goes wrong:** Client doesn't get the updated expense back after PUT, has to reconstruct it.
**Why it happens:** Category Update returns just `{"message": "Category updated"}` -- but expenses need the full object back for local state update.
**How to avoid:** Have the expense PUT endpoint return the full updated expense JSON (use `:one` query that returns the row, or query after update).

## Code Examples

### sqlc Queries for Update and Delete
```sql
-- name: UpdateExpense :one
UPDATE expenses
SET category_id = $3, amount_cents = $4, note = $5, expense_date = $6, updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, category_id, amount_cents, note, expense_date, created_at, updated_at;

-- name: DeleteExpense :execrows
DELETE FROM expenses
WHERE id = $1 AND user_id = $2;
```

Using `:one` for UpdateExpense (not `:execrows`) so the handler can return the full updated expense to the client. Using `:execrows` for DeleteExpense to detect not-found via rows affected == 0 (matching category pattern).

### ExpenseDB Interface Extension
```go
type ExpenseDB interface {
    CreateExpense(userID, categoryID string, amountCents int64, note string, expenseDate time.Time) (MockExpense, error)
    GetExpensesByUser(userID string, limit, offset int) ([]MockExpense, error)
    UpdateExpense(id, userID, categoryID string, amountCents int64, note string, expenseDate time.Time) (MockExpense, error)
    DeleteExpense(id, userID string) error
}
```

### Confirmation Dialog with Amount Display
```dart
Future<bool> _confirmDelete(BuildContext context, Expense expense) async {
  final result = await showDialog<bool>(
    context: context,
    builder: (context) => AlertDialog(
      title: Text('Delete this ${Expense.formatCents(expense.amountCents)} expense?'),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context, false),
          child: const Text('Cancel'),
        ),
        TextButton(
          onPressed: () => Navigator.pop(context, true),
          child: const Text('Delete'),
        ),
      ],
    ),
  );
  return result ?? false;
}
```

## State of the Art

No changes from previous phases. All patterns are stable and well-established in the codebase.

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| N/A | Using existing Phase 3/4 patterns | Phase 3 established | None -- direct reuse |

## Discretion Recommendations

### Delete Button Placement on Edit Screen
**Recommendation:** AppBar trash icon (IconButton in actions).
**Rationale:** Consistent with common Material Design patterns. Red button at bottom competes visually with the Save button. AppBar trash icon is discoverable but not accidentally tappable. The user also has swipe-to-delete from the list, so the edit-screen delete is a secondary path.

### History Row Layout
**Recommendation:** Custom Row widget (not ListTile) to match the user's specification of "colored category chip on left, amount on right, date below."
**Rationale:** ListTile's leading/title/subtitle/trailing layout doesn't naturally support the specified layout. A custom Row with category chip + Column(amount, date) + optional note gives full control. The existing `_ExpenseTile` already uses ListTile but will need restructuring anyway to add the category chip.

### Edit Mode Detection
**Recommendation:** Optional Expense constructor parameter (matching CategoryFormScreen pattern).
**Rationale:** Simpler than route parameters, avoids extra API call to fetch expense by ID, established pattern in the codebase. The Expense object is passed via GoRouter `extra`.

### Swipe-to-Delete Visual Treatment
**Recommendation:** Red background with white delete icon, matching Phase 3 categories exactly.
**Rationale:** Consistency. Already implemented in categories_screen.dart.

## Open Questions

None. This phase is well-defined and all patterns are already established in the codebase.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Flutter) | flutter_test + mocktail |
| Framework (Go) | testing + httptest |
| Config file | app/pubspec.yaml (Flutter), go test (Go) |
| Quick run command (Flutter) | `cd app && flutter test test/features/expenses/` |
| Quick run command (Go) | `cd server && go test ./internal/handler/ -run Expense` |
| Full suite command | `cd app && flutter test && cd ../server && go test ./...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| EXP-03 | PUT /expenses/:id updates expense | unit (Go) | `cd server && go test ./internal/handler/ -run TestUpdateExpense -x` | Wave 0 |
| EXP-03 | Edit form pre-fills with expense data | widget (Flutter) | `cd app && flutter test test/features/expenses/presentation/expense_form_screen_test.dart` | Exists (needs edit tests) |
| EXP-03 | Save in edit mode calls updateExpense | widget (Flutter) | `cd app && flutter test test/features/expenses/presentation/expense_form_screen_test.dart` | Exists (needs edit tests) |
| EXP-04 | DELETE /expenses/:id removes expense | unit (Go) | `cd server && go test ./internal/handler/ -run TestDeleteExpense -x` | Wave 0 |
| EXP-04 | Swipe-to-delete shows confirmation dialog | widget (Flutter) | `cd app && flutter test test/features/history/presentation/history_screen_test.dart` | Wave 0 |
| EXP-04 | Delete from edit screen pops + shows SnackBar | widget (Flutter) | `cd app && flutter test test/features/expenses/presentation/expense_form_screen_test.dart` | Exists (needs delete tests) |

### Sampling Rate
- **Per task commit:** `cd app && flutter test test/features/expenses/ && cd ../server && go test ./internal/handler/ -run Expense`
- **Per wave merge:** `cd app && flutter test && cd ../server && go test ./...`
- **Phase gate:** Full suite green before /gsd:verify-work

### Wave 0 Gaps
- [ ] `app/test/features/history/presentation/history_screen_test.dart` -- covers EXP-04 swipe-to-delete, EXP-03 tap-to-edit navigation
- [ ] Go handler tests for Update/Delete in `server/internal/handler/expense_test.go` (file exists but needs new test functions)
- [ ] FakeExpenseNotifier needs updateExpense/deleteExpense methods added

## Sources

### Primary (HIGH confidence)
- Existing codebase: `categories_screen.dart` -- swipe-to-delete + confirmation dialog pattern
- Existing codebase: `category_form_screen.dart` -- optional parameter edit mode pattern
- Existing codebase: `category.go` handler -- PUT/DELETE handler pattern with :execrows
- Existing codebase: `categories.sql` -- UpdateCategory/DeleteCategory query patterns
- Existing codebase: `expense_form_screen.dart` -- current create-only form to be extended
- Existing codebase: `history_screen.dart` -- current list to be enhanced with interactions
- Existing codebase: `expense_test.go` -- mock pattern for expense handler tests

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new libraries, all existing
- Architecture: HIGH -- direct reuse of established Phase 3 patterns
- Pitfalls: HIGH -- derived from actual code review of existing patterns

**Research date:** 2026-03-16
**Valid until:** 2026-04-16 (stable patterns, no external dependencies changing)
