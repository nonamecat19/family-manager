# Phase 4: Expense Entry - Research

**Researched:** 2026-03-15
**Domain:** Expense data entry (Go backend + Flutter form UI)
**Confidence:** HIGH

## Summary

Phase 4 adds the core expense logging flow: a full-screen form reached via the existing FAB, backed by a new `expenses` table and CRUD API. The implementation follows the exact patterns established in Phase 2 (auth) and Phase 3 (categories): Go handler + DB interface + sqlc for the backend, Riverpod StateNotifier + Repository + feature-first folders for Flutter.

The key technical decisions are: (1) storing amounts as BIGINT cents in PostgreSQL with integer math throughout the backend, (2) using the `intl` package's `NumberFormat.currency()` for locale-aware display formatting on the client, and (3) reusing the CategoryChip widget in a horizontal scrollable row for fast category selection. The "under 3 seconds" speed target is achievable through auto-focus on the amount field, defaulting to today's date, and a single-tap category selection -- no navigation to secondary screens required.

**Primary recommendation:** Split into two plans -- Plan 1 for backend (migration + sqlc queries + handler + tests), Plan 2 for Flutter (model + repository + provider + form screen + routing + tests). This mirrors the Phase 3 split that worked well.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- FAB opens a full-screen "New Expense" form (not bottom sheet or modal)
- Field order: Amount (with numpad auto-focus) -> Category -> Note (optional) -> Date
- Category picker: horizontal scrollable row of CategoryChips (reuse Phase 3 widget) -- tap to select, no extra screen
- Date defaults to "Today", tap to open date picker to change
- Note field is optional -- can be left empty for quick entry

### Claude's Discretion
- Amount input UX: calculator-style keypad vs standard text field, decimal handling, currency symbol placement
- Speed optimizations: smart defaults, recent categories first, auto-focus behavior
- After-save behavior: whether to stay on form for batch entry, navigate to history, or show brief confirmation
- Save button style and placement
- Form validation approach (inline errors vs disabled save button)
- How "quick entry" flow differs from full entry (if at all)
- Backend expenses table schema and API design

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| EXP-01 | User can log an expense with amount, category, optional note, and date | Full-screen form with all four fields; POST /api/v1/expenses endpoint; expenses table with amount_cents, category_id, note, expense_date |
| EXP-02 | Expense entry is optimized for speed (under 3 seconds for quick entry) | Auto-focus numpad on amount field, single-tap CategoryChip selection, date defaults to today, note is optional -- skip-to-save flow |
| EXP-05 | Amounts stored as integer cents, displayed with locale formatting | BIGINT cents column in PostgreSQL, int in Go/Dart, `intl` package NumberFormat.currency() for display |
</phase_requirements>

## Standard Stack

### Core (already in project)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Gin | 1.12.0 | Go HTTP framework | Already used for auth + category handlers |
| pgx/v5 | 5.8.0 | PostgreSQL driver | Already used, connection pool in place |
| sqlc | 2 (config) | SQL-to-Go codegen | Already used for all DB queries |
| goose | - | Migrations | Already used for 3 migrations |
| flutter_riverpod | 2.6.1 | State management | Already used for auth + categories |
| dio | 5.4.0 | HTTP client | Already used with auth interceptor |
| go_router | 14.0.0 | Navigation | Already used with ShellRoute pattern |

### New Dependencies
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| intl | ^0.20.0 | NumberFormat.currency() for locale-aware amount display | Amount formatting throughout the app |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| intl NumberFormat | Manual string formatting | intl handles locale edge cases (comma vs period, symbol placement) -- don't hand-roll |
| Standard TextField for amount | Custom calculator keypad widget | Standard TextField with `keyboardType: TextInputType.numberWithOptions(decimal: true)` is simpler and meets the speed target; calculator keypad is overengineering for v1 |

**Installation (Flutter):**
```bash
cd app && flutter pub add intl
```

**No new Go dependencies needed** -- all backend requirements covered by existing stack.

## Architecture Patterns

### Recommended Project Structure

**Backend additions:**
```
server/
├── internal/db/migrations/00004_expenses.sql   # New migration
├── internal/db/queries/expenses.sql            # sqlc queries
├── internal/db/sqlc/                           # Regenerated by sqlc
├── internal/handler/expense.go                 # ExpenseHandler + ExpenseDB interface
├── internal/handler/expense_db.go              # PgExpenseDB implementation
└── internal/handler/expense_test.go            # Mock-based handler tests
```

**Flutter additions:**
```
app/lib/features/expenses/
├── data/
│   ├── models/expense.dart                     # Expense model
│   └── expense_repository.dart                 # API calls
├── domain/
│   └── expense_state.dart                      # Sealed state class
└── presentation/
    └── expense_form_screen.dart                # Full-screen form
```

**Provider:**
```
app/lib/providers/expense_provider.dart         # ExpenseNotifier + provider
```

### Pattern 1: ExpenseDB Interface (Backend)
**What:** Abstract database operations behind an interface for testability.
**When to use:** Every handler. Established pattern from AuthDB and CategoryDB.
**Example:**
```go
// Source: Mirrors existing CategoryDB pattern in handler/category.go
type ExpenseDB interface {
    CreateExpense(userID, categoryID string, amountCents int64, note string, expenseDate time.Time) (MockExpense, error)
    GetExpensesByUser(userID string, limit, offset int) ([]MockExpense, error)
}
```

### Pattern 2: Integer Cents Throughout
**What:** Store, transmit, and compute all amounts as integer cents. Only convert to display format at the UI layer.
**When to use:** Every place money appears.
**Example:**
```dart
// Store: amountCents = 123456 (represents $1,234.56)
// Display: NumberFormat.currency(locale: 'en_US', symbol: '$').format(amountCents / 100)
// Input: parse user's "1234.56" -> (1234.56 * 100).round() -> 123456
```

### Pattern 3: Full-Screen Form Route (Outside ShellRoute)
**What:** Expense form is a GoRoute outside the ShellRoute, like category form. No bottom nav, own AppBar.
**When to use:** Any form that should feel like a dedicated screen.
**Example:**
```dart
// Source: Mirrors existing /settings/categories/new pattern in app_router.dart
GoRoute(
  path: '/expenses/new',
  builder: (context, state) => const ExpenseFormScreen(),
),
```

### Pattern 4: ConsumerStatefulWidget Form
**What:** Use ConsumerStatefulWidget (not ConsumerWidget) for forms that need TextEditingControllers and local state.
**When to use:** All form screens.
**Example:** See existing `CategoryFormScreen` -- same pattern for `ExpenseFormScreen`.

### Anti-Patterns to Avoid
- **Floating-point money:** Never use `double` or `NUMERIC` for amounts. BIGINT cents only. Floating-point leads to rounding bugs (0.1 + 0.2 != 0.3).
- **Formatting in backend:** Don't send formatted strings from the API. Send integer cents, format on client.
- **Nested navigation for category pick:** Don't navigate to a separate screen for category selection. Horizontal chip row with tap-to-select meets the speed target.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Currency formatting | Custom string manipulation | `intl` NumberFormat.currency() | Handles locale (commas, periods, symbol position), thousands separators, decimal places |
| Date picker | Custom date selector widget | `showDatePicker()` (Flutter built-in) | Material date picker is standard, accessible, and free |
| UUID generation | Custom ID logic | `gen_random_uuid()` (PostgreSQL) | Server-side UUID generation, same as users/categories tables |
| Input validation for amounts | Complex regex | Parse with `double.tryParse()`, multiply by 100, check range | Simple and reliable |

**Key insight:** The expense form is straightforward CRUD. The only "interesting" part is cents conversion and the speed-optimized UX. Don't over-engineer.

## Common Pitfalls

### Pitfall 1: Floating-Point Cents Conversion
**What goes wrong:** `(1.1 * 100).toInt()` returns 109 in some cases due to floating-point representation.
**Why it happens:** IEEE 754 double precision can't represent all decimals exactly.
**How to avoid:** Use `.round()` not `.toInt()` when converting: `(double.parse(input) * 100).round()`.
**Warning signs:** Off-by-one-cent errors in displayed amounts.

### Pitfall 2: Foreign Key to Non-Existent Category
**What goes wrong:** User submits expense with a category_id that was deleted.
**Why it happens:** Race condition between category deletion and expense creation.
**How to avoid:** PostgreSQL FOREIGN KEY constraint will reject the insert. Handle the constraint violation error in the handler and return a clear 400 error.
**Warning signs:** 500 errors on expense creation.

### Pitfall 3: Date Timezone Confusion
**What goes wrong:** Expense logged on March 15 in user's timezone shows as March 14 or 16.
**Why it happens:** Mixing DATE and TIMESTAMPTZ, or converting between timezones incorrectly.
**How to avoid:** Use `DATE` type in PostgreSQL for expense_date (not TIMESTAMPTZ). Transmit as `"2026-03-15"` string (ISO 8601 date only). Parse in Flutter as `DateTime.parse()` which gives midnight UTC -- fine for date-only values.
**Warning signs:** Expenses appearing on wrong dates.

### Pitfall 4: Category Chip Row Not Loading
**What goes wrong:** Expense form opens but category chips are empty.
**Why it happens:** Category provider hasn't loaded yet, or expense form doesn't trigger category load.
**How to avoid:** Expense form screen should watch `categoryStateProvider`. If categories are in `CategoryInitial` state, trigger `loadCategories()` in initState or use a ref.listen.
**Warning signs:** Empty category row on first form open.

### Pitfall 5: FAB Accessible When Not Logged In
**What goes wrong:** FAB in AppScaffold navigates to expense form, but AppScaffold is inside ShellRoute which is behind auth redirect -- so this shouldn't happen. But the FAB onPressed currently does nothing.
**How to avoid:** Simply wire `onPressed: () => context.push('/expenses/new')`. The auth redirect will prevent unauthenticated access since ShellRoute is already protected.
**Warning signs:** None expected, but verify in testing.

## Code Examples

### Expenses Migration (00004_expenses.sql)
```sql
-- +goose Up
CREATE TABLE expenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    note TEXT NOT NULL DEFAULT '',
    expense_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_expenses_user_id ON expenses(user_id);
CREATE INDEX idx_expenses_user_date ON expenses(user_id, expense_date DESC);

-- +goose Down
DROP TABLE IF EXISTS expenses;
```

**Key decisions:**
- `BIGINT` not `INTEGER` for amount_cents -- supports up to $92 quadrillion, future-proof
- `ON DELETE RESTRICT` for category_id -- prevent deleting categories that have expenses (Phase 5 will handle reassignment)
- `CHECK (amount_cents > 0)` -- enforce positive amounts at DB level
- `DATE` not `TIMESTAMPTZ` for expense_date -- we only care about the day, not the time
- `idx_expenses_user_date` composite index -- covers the most common query pattern (user's expenses by date)
- `note TEXT NOT NULL DEFAULT ''` -- empty string instead of NULL simplifies Go/Dart handling

### Expense Model (Dart)
```dart
class Expense {
  const Expense({
    required this.id,
    required this.categoryId,
    required this.amountCents,
    required this.note,
    required this.expenseDate,
  });

  factory Expense.fromJson(Map<String, dynamic> json) {
    return Expense(
      id: json['id'] as String,
      categoryId: json['category_id'] as String,
      amountCents: json['amount_cents'] as int,
      note: json['note'] as String,
      expenseDate: DateTime.parse(json['expense_date'] as String),
    );
  }

  final String id;
  final String categoryId;
  final int amountCents;
  final String note;
  final DateTime expenseDate;
}
```

### Amount Input -> Cents Conversion
```dart
/// Parses a user-entered amount string to integer cents.
/// Returns null if the input is invalid or non-positive.
int? parseAmountToCents(String input) {
  final value = double.tryParse(input.replaceAll(',', ''));
  if (value == null || value <= 0) return null;
  return (value * 100).round();
}
```

### Cents -> Display Formatting
```dart
import 'package:intl/intl.dart';

/// Formats integer cents as a locale-aware currency string.
/// Example: 123456 -> "$1,234.56"
String formatCents(int cents) {
  return NumberFormat.currency(locale: 'en_US', symbol: r'$')
      .format(cents / 100);
}
```

### API Endpoint Design
```
POST   /api/v1/expenses          # Create expense
GET    /api/v1/expenses          # List user's expenses (paginated, newest first)
```

**Create request body:**
```json
{
  "category_id": "uuid-string",
  "amount_cents": 123456,
  "note": "Lunch with team",
  "expense_date": "2026-03-15"
}
```

**Create response (201):**
```json
{
  "id": "uuid-string",
  "user_id": "uuid-string",
  "category_id": "uuid-string",
  "amount_cents": 123456,
  "note": "Lunch with team",
  "expense_date": "2026-03-15",
  "created_at": "2026-03-15T12:00:00Z"
}
```

**List response (200):** Array of expense objects, ordered by expense_date DESC, created_at DESC.

### Discretion Recommendations

**Amount input UX:** Use a standard `TextField` with `keyboardType: TextInputType.numberWithOptions(decimal: true)` and a `$` prefix. A calculator keypad is overkill for v1. Auto-focus this field when the form opens to enable immediate typing.

**After-save behavior:** Pop back to the previous screen (history tab) after save. Show a brief `SnackBar` confirmation ("Expense saved"). Don't stay on the form -- batch entry is rare for personal finance; users typically log one expense at a time.

**Save button:** `FilledButton` at the bottom of the form, matching the CategoryFormScreen pattern. Full-width for easy thumb reach.

**Form validation:** Inline errors (shown after first submit attempt, same as CategoryFormScreen pattern with `_submitted` flag). Don't disable the save button -- it hides the reason a user can't submit.

**Quick entry vs full entry:** They are the same flow. "Quick" means: tap FAB -> type amount -> tap category chip -> tap Save. The note and date fields have sensible defaults (empty and today) so they can be skipped. No separate "quick mode" needed.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Store money as DECIMAL/NUMERIC | Store as integer cents | Industry standard | No floating-point bugs, simpler math |
| Bottom sheets for quick entry | Full-screen forms | Material 3 guidance | Better for complex forms, more room |
| Manual date formatting | intl package NumberFormat | Stable since intl 0.17+ | Locale-aware, handles edge cases |

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | Go testing + httptest (standard library) |
| Framework (Flutter) | flutter_test + mocktail 1.0.0 |
| Config file | None needed (standard Go test + Flutter test) |
| Quick run (Go) | `cd server && go test ./internal/handler/ -run TestExpense -v` |
| Quick run (Flutter) | `cd app && flutter test test/features/expenses/` |
| Full suite (Go) | `cd server && go test ./...` |
| Full suite (Flutter) | `cd app && flutter test` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| EXP-01 | Create expense with all fields | unit (Go handler) | `cd server && go test ./internal/handler/ -run TestCreateExpense_Success -v` | Wave 0 |
| EXP-01 | Create expense with missing required fields returns 400 | unit (Go handler) | `cd server && go test ./internal/handler/ -run TestCreateExpense_MissingFields -v` | Wave 0 |
| EXP-01 | Expense form submits and calls API | widget (Flutter) | `cd app && flutter test test/features/expenses/presentation/expense_form_screen_test.dart` | Wave 0 |
| EXP-02 | Form auto-focuses amount field | widget (Flutter) | included in form screen test | Wave 0 |
| EXP-02 | Date defaults to today | widget (Flutter) | included in form screen test | Wave 0 |
| EXP-05 | Amount cents stored and returned as integer | unit (Go handler) | `cd server && go test ./internal/handler/ -run TestCreateExpense_AmountCents -v` | Wave 0 |
| EXP-05 | Negative/zero amounts rejected | unit (Go handler) | `cd server && go test ./internal/handler/ -run TestCreateExpense_InvalidAmount -v` | Wave 0 |
| EXP-05 | formatCents() produces locale string | unit (Dart) | `cd app && flutter test test/features/expenses/data/models/expense_test.dart` | Wave 0 |

### Sampling Rate
- **Per task commit:** Quick run command for affected layer
- **Per wave merge:** Full suite for both Go and Flutter
- **Phase gate:** Full suite green before verify-work

### Wave 0 Gaps
- [ ] `server/internal/handler/expense_test.go` -- covers EXP-01, EXP-05 (handler tests)
- [ ] `app/test/features/expenses/presentation/expense_form_screen_test.dart` -- covers EXP-01, EXP-02
- [ ] `app/test/features/expenses/data/models/expense_test.dart` -- covers EXP-05 (cents formatting)

## Open Questions

1. **Category deletion with existing expenses**
   - What we know: Migration uses `ON DELETE RESTRICT` which prevents deleting categories that have expenses
   - What's unclear: Should Phase 4 update the category delete handler to check for expenses and return a meaningful error? Or defer to Phase 5?
   - Recommendation: Add a check in the category delete handler that returns `409 Conflict` with message "Category has expenses and cannot be deleted" -- minimal effort, prevents confusing 500 errors. But this is technically Phase 5 territory. At minimum, handle the FK constraint error gracefully.

2. **Pagination for expense list**
   - What we know: GET /expenses will need pagination eventually (Phase 6 history)
   - What's unclear: Should Plan 1 add pagination params now or keep it simple?
   - Recommendation: Add limit/offset to the query now (default limit=50). It costs nothing and the planner can decide to expose it or keep it internal.

## Sources

### Primary (HIGH confidence)
- Existing codebase: `server/internal/handler/category.go`, `category_db.go`, `category_test.go` -- established Go handler patterns
- Existing codebase: `app/lib/features/categories/` -- established Flutter feature structure
- Existing codebase: `server/internal/db/migrations/` -- established migration patterns
- Existing codebase: `app/lib/shared/widgets/app_scaffold.dart` -- FAB wiring point
- Existing codebase: `app/lib/core/router/app_router.dart` -- routing patterns

### Secondary (MEDIUM confidence)
- Flutter `intl` package: well-established for NumberFormat.currency(), stable API
- PostgreSQL BIGINT for money: industry standard practice, documented in PostgreSQL docs

### Tertiary (LOW confidence)
- None -- all patterns are established in the existing codebase

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all libraries already in use except `intl` which is well-known
- Architecture: HIGH -- exact patterns exist in Phase 2 and Phase 3 code
- Pitfalls: HIGH -- standard money-handling pitfalls, well-documented
- Validation: HIGH -- test patterns established in category_test.go and Flutter test directory

**Research date:** 2026-03-15
**Valid until:** 2026-04-15 (stable -- no fast-moving dependencies)
