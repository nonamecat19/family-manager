# Phase 9: Family Views - Research

**Researched:** 2026-03-16
**Domain:** Family expense aggregation (server queries + Flutter UI for combined feed and summary dashboard)
**Confidence:** HIGH

## Summary

Phase 9 adds two read-only views on top of the family group infrastructure built in Phase 8: a combined expense feed (FAM-03) and a summary dashboard with per-person and per-category totals (FAM-04). No new database tables are needed -- the existing `expenses`, `family_members`, `users`, and `categories` tables already contain all the data. The work is new SQL queries that JOIN expenses across family members, new server endpoints, and two new Flutter screens.

The combined feed query joins expenses with family_members to get all expenses from users in the same family, plus joins users to get the spender's email (the "who spent what" requirement). The summary query aggregates totals per-person and per-category across the family. Both queries filter by the requesting user's family_id, resolved through the family_members table.

**Primary recommendation:** Split into two plans -- Plan 1 for server-side (new sqlc queries in a family_expenses.sql file, FamilyViewDB interface, FamilyViewHandler, tests) and Plan 2 for Flutter UI (family feed screen, family summary screen, navigation integration with bottom nav or family screen).

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| FAM-03 | Family members can view combined expense feed showing who spent what | GetFamilyExpenses query joining expenses+family_members+users, FamilyViewHandler.FamilyFeed endpoint, FamilyFeedScreen in Flutter |
| FAM-04 | Family members can view summary dashboard with per-person and per-category totals | GetFamilyMemberTotals + GetFamilyCategoryTotals queries, FamilyViewHandler.FamilySummary endpoint, FamilySummaryScreen in Flutter |
</phase_requirements>

## Standard Stack

### Core (Already in Project -- Zero New Dependencies)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Gin | 1.12.0 | HTTP handlers for family view endpoints | Already used for all server routes |
| pgx/v5 | 5.8.0 | PostgreSQL driver | Already used project-wide |
| sqlc | v2 config | Query codegen for family view queries | Already used for all DB operations |
| go_router | 14.0.0 | Flutter routing for family view screens | Already used, supports path params |
| flutter_riverpod | 2.6.1 | State management for family view state | Already used project-wide |
| dio | 5.4.0 | HTTP client for family view API calls | Already used project-wide |
| intl | (existing) | Currency formatting (Expense.formatCents) | Already used for expense display |
| fl_chart | (existing) | Pie/bar charts if reused for family summary | Already used in charts feature |

### No New Dependencies Required
This phase requires zero new packages. All functionality uses existing stack. The queries are new SQL; the handler follows the established DB interface pattern; the Flutter UI reuses existing widgets (ExpenseTile pattern, MonthlySummary pattern).

## Architecture Patterns

### Recommended Project Structure

**Server (new files):**
```
server/internal/
├── db/
│   └── queries/
│       └── family_expenses.sql         # New queries for family expense views
├── handler/
│   ├── family_view.go                  # FamilyViewDB interface + FamilyViewHandler
│   ├── family_view_db.go              # PgFamilyViewDB implementation
│   └── family_view_test.go            # Handler tests with mock FamilyViewDB
└── router/
    └── router.go                       # Add family view routes (modify)
```

**Flutter (new files):**
```
app/lib/features/family/
├── data/
│   ├── family_view_repository.dart     # API calls to family view endpoints
│   └── models/
│       ├── family_expense.dart         # FamilyExpense model (expense + spender email)
│       └── family_summary.dart         # FamilySummary model (per-person + per-category)
├── domain/
│   ├── family_feed_notifier.dart       # StateNotifier for family feed
│   └── family_summary_notifier.dart    # StateNotifier for family summary
└── presentation/
    ├── family_feed_screen.dart         # Combined expense feed
    └── family_summary_screen.dart      # Summary dashboard
```

### Pattern 1: FamilyViewDB Interface (Mirrors SummaryDB)
**What:** Separate DB interface for family view read operations, distinct from the FamilyDB interface used for group management
**When to use:** All family view handlers
**Why separate:** The FamilyDB interface handles CRUD for groups/invitations. Family views are read-only aggregation queries -- different concern, different mock in tests.
**Example:**
```go
// FamilyExpense represents a single expense in the family feed.
type FamilyExpense struct {
    ID          string
    UserID      string
    UserEmail   string
    CategoryID  string
    CategoryName  string
    CategoryColor string
    CategoryIcon  string
    AmountCents int64
    Note        string
    ExpenseDate time.Time
    CreatedAt   time.Time
}

// FamilyMemberTotal represents one member's total spending.
type FamilyMemberTotal struct {
    UserID     string
    UserEmail  string
    TotalCents int64
    Count      int
}

// FamilyCategoryTotal represents one category's total across the family.
type FamilyCategoryTotal struct {
    CategoryID    string
    CategoryName  string
    CategoryColor string
    CategoryIcon  string
    TotalCents    int64
    Count         int
}

// FamilyViewDB abstracts database operations for family expense views.
type FamilyViewDB interface {
    GetFamilyExpenses(familyID string, limit, offset int) ([]FamilyExpense, error)
    GetFamilyMemberTotals(familyID string, dateFrom, dateTo time.Time) ([]FamilyMemberTotal, error)
    GetFamilyCategoryTotals(familyID string, dateFrom, dateTo time.Time) ([]FamilyCategoryTotal, error)
}
```

### Pattern 2: Family Membership Check via FamilyDB
**What:** The FamilyViewHandler needs to verify the requesting user belongs to a family before serving data. Reuse the existing FamilyDB.GetFamilyByUserID method.
**When to use:** Every family view endpoint
**Implementation:** The FamilyViewHandler takes both a FamilyDB (for membership lookup) and FamilyViewDB (for aggregation queries). This avoids duplicating the membership check logic.
```go
type FamilyViewHandler struct {
    familyDB FamilyDB
    viewDB   FamilyViewDB
}

func (h *FamilyViewHandler) FamilyFeed(c *gin.Context) {
    userID := c.GetString("user_id")
    family, err := h.familyDB.GetFamilyByUserID(userID)
    if err != nil {
        // Not in a family -> 404
    }
    expenses, err := h.viewDB.GetFamilyExpenses(family.ID, limit, offset)
    // ...
}
```

### Pattern 3: Reuse Existing Expense Display Patterns
**What:** The family feed is essentially the history screen with an extra "who" column. Reuse the `_ExpenseTile` layout pattern with an added subtitle showing the spender's email.
**When to use:** FamilyFeedScreen
**Details:** The FamilyExpense model extends the Expense concept with a `userEmail` field. The tile shows the spender's email (or a truncated version) as additional context.

### Pattern 4: Reuse Summary Dashboard Pattern
**What:** The family summary mirrors the existing MonthlySummary widget but adds a per-person breakdown section above the per-category breakdown.
**When to use:** FamilySummaryScreen
**Details:** The screen shows: (1) total family spending for the month, (2) per-person totals (ListTiles with email and amount), (3) per-category totals (same as MonthlySummary pattern).

### Anti-Patterns to Avoid
- **Making N+1 queries (one per family member):** Use a single SQL query that joins family_members to expenses. Never loop over members and query each one's expenses separately.
- **Duplicating the family membership check in SQL:** The handler checks membership via FamilyDB, then passes the familyID to FamilyViewDB. The SQL query uses familyID directly -- it does not re-verify membership.
- **Building family views into the existing expense/summary endpoints:** Keep family views as separate endpoints and handlers. The personal expense endpoints remain untouched. Family views are additive.
- **Adding a "family" tab that's always visible:** The family feed/summary should only be accessible when the user is in a family. Either integrate into the family management screen or add conditional navigation.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cross-user expense aggregation | Multiple queries + client-side merge | Single SQL JOIN query | DB handles aggregation efficiently; avoids N+1 |
| Per-person totals | Client-side groupBy on raw expenses | SQL GROUP BY user_id | Server computes totals, sends minimal data |
| Per-category totals across family | Client-side reduce | SQL GROUP BY category_id with SUM | Mirrors existing GetCategoryTotals pattern |
| Currency formatting | Custom format function | Reuse Expense.formatCents | Already handles locale-aware formatting |
| Category icon/color resolution | Re-fetch categories on client | Include in SQL JOIN (category name, color, icon) | Same pattern as GetCategoryTotals |

**Key insight:** The SQL queries do the heavy lifting. The server returns pre-aggregated data. The Flutter UI renders it. No new computation patterns are needed.

## Common Pitfalls

### Pitfall 1: Missing Authorization -- User Sees Other Families' Data
**What goes wrong:** A user crafts a request with a familyID they don't belong to and sees someone else's expenses
**Why it happens:** The handler passes a user-supplied familyID directly to the query
**How to avoid:** NEVER accept familyID from the client. Always resolve it server-side via `FamilyDB.GetFamilyByUserID(userID)` using the JWT-authenticated userID. The familyID comes from the DB, not the request.
**Warning signs:** FamilyID in request body or URL params

### Pitfall 2: N+1 Query for Category Details
**What goes wrong:** The family feed query returns category_id but not name/color/icon, requiring a second query to resolve categories
**Why it happens:** Copying the individual expense query pattern which only returns category_id
**How to avoid:** JOIN categories in the family expenses query to include category_name, category_color, category_icon in each row. This mirrors the GetCategoryTotals pattern which already JOINs categories.
**Warning signs:** Client-side category lookup maps for family expenses

### Pitfall 3: Empty Family Summary When No Expenses in Date Range
**What goes wrong:** Family summary shows empty/broken state when no expenses exist for the selected month
**Why it happens:** No empty-state handling for the summary
**How to avoid:** Return zero total_cents and empty arrays (not null) when no data. Flutter handles empty state with a message like "No family spending this month". Same pattern as existing ChartLoaded with empty byCategory.
**Warning signs:** Null JSON arrays causing parse errors

### Pitfall 4: Stale Family Feed After Member Logs New Expense
**What goes wrong:** User A logs an expense but User B's family feed doesn't update
**Why it happens:** No push mechanism; feed only loads on screen mount
**How to avoid:** Reload family feed data every time the screen is visited (same as existing HistoryScreen pattern which reloads on mount). Add pull-to-refresh. For v1, polling or WebSocket is overkill -- reload-on-visit is sufficient.
**Warning signs:** Users reporting stale data

### Pitfall 5: Performance Degradation with Large Expense Histories
**What goes wrong:** Family with 5+ members and months of expenses results in slow queries
**Why it happens:** No pagination or date filtering on family feed
**How to avoid:** Apply the same pagination pattern (limit/offset) used in the individual expense list. Add optional date filtering (date_from, date_to). Default to the current month or last 30 days for the feed view.
**Warning signs:** Slow response times on family endpoints

## Code Examples

### SQL: GetFamilyExpenses (family_expenses.sql)
```sql
-- name: GetFamilyExpenses :many
SELECT
    e.id,
    e.user_id,
    u.email AS user_email,
    e.category_id,
    c.name AS category_name,
    c.color AS category_color,
    c.icon AS category_icon,
    e.amount_cents,
    e.note,
    e.expense_date,
    e.created_at
FROM expenses e
JOIN family_members fm ON fm.user_id = e.user_id
JOIN users u ON u.id = e.user_id
JOIN categories c ON c.id = e.category_id
WHERE fm.family_id = $1
ORDER BY e.expense_date DESC, e.created_at DESC
LIMIT $2 OFFSET $3;
```

### SQL: GetFamilyMemberTotals (family_expenses.sql)
```sql
-- name: GetFamilyMemberTotals :many
SELECT
    e.user_id,
    u.email AS user_email,
    SUM(e.amount_cents)::BIGINT AS total_cents,
    COUNT(*)::INT AS expense_count
FROM expenses e
JOIN family_members fm ON fm.user_id = e.user_id
JOIN users u ON u.id = e.user_id
WHERE fm.family_id = $1
  AND e.expense_date >= $2
  AND e.expense_date <= $3
GROUP BY e.user_id, u.email
ORDER BY total_cents DESC;
```

### SQL: GetFamilyCategoryTotals (family_expenses.sql)
```sql
-- name: GetFamilyCategoryTotals :many
SELECT
    e.category_id,
    c.name AS category_name,
    c.color AS category_color,
    c.icon AS category_icon,
    SUM(e.amount_cents)::BIGINT AS total_cents,
    COUNT(*)::INT AS expense_count
FROM expenses e
JOIN family_members fm ON fm.user_id = e.user_id
JOIN categories c ON c.id = e.category_id
WHERE fm.family_id = $1
  AND e.expense_date >= $2
  AND e.expense_date <= $3
GROUP BY e.category_id, c.name, c.color, c.icon
ORDER BY total_cents DESC;
```

### Server: API Endpoints
```
GET /api/v1/families/me/expenses           -- Family combined expense feed (paginated)
GET /api/v1/families/me/summary?month=YYYY-MM  -- Family summary dashboard
```

Both endpoints are added to the existing `families` protected route group in router.go.

### Server: FamilyViewHandler
```go
// FamilyFeed handles GET /api/v1/families/me/expenses.
func (h *FamilyViewHandler) FamilyFeed(c *gin.Context) {
    userID := c.GetString("user_id")
    family, err := h.familyDB.GetFamilyByUserID(userID)
    if err != nil {
        if errors.Is(err, ErrFamilyNotFound) {
            c.JSON(http.StatusNotFound, gin.H{"error": "no family"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
        return
    }

    limit, offset := parsePagination(c)
    expenses, err := h.viewDB.GetFamilyExpenses(family.ID, limit, offset)
    // ... serialize and return
}

// FamilySummary handles GET /api/v1/families/me/summary.
func (h *FamilyViewHandler) FamilySummary(c *gin.Context) {
    userID := c.GetString("user_id")
    family, err := h.familyDB.GetFamilyByUserID(userID)
    // ... resolve month, call GetFamilyMemberTotals + GetFamilyCategoryTotals
}
```

### Flutter: FamilyExpense Model
```dart
/// An expense in the family feed, including who logged it.
class FamilyExpense {
  const FamilyExpense({
    required this.id,
    required this.userId,
    required this.userEmail,
    required this.categoryId,
    required this.categoryName,
    required this.categoryColor,
    required this.categoryIcon,
    required this.amountCents,
    required this.note,
    required this.expenseDate,
  });

  factory FamilyExpense.fromJson(Map<String, dynamic> json) {
    return FamilyExpense(
      id: json['id'] as String,
      userId: json['user_id'] as String,
      userEmail: json['user_email'] as String,
      categoryId: json['category_id'] as String,
      categoryName: json['category_name'] as String,
      categoryColor: json['category_color'] as String,
      categoryIcon: json['category_icon'] as String,
      amountCents: json['amount_cents'] as int,
      note: (json['note'] as String?) ?? '',
      expenseDate: DateTime.parse(json['expense_date'] as String),
    );
  }

  final String id;
  final String userId;
  final String userEmail;
  final String categoryId;
  final String categoryName;
  final String categoryColor;
  final String categoryIcon;
  final int amountCents;
  final String note;
  final DateTime expenseDate;
}
```

### Flutter: Navigation Integration
```dart
// Option A: Add to family management screen as action buttons
// In FamilyScreen (when FamilyLoaded state):
ListTile(
  leading: const Icon(Icons.list_alt),
  title: const Text('Family Expenses'),
  subtitle: const Text('See what everyone spent'),
  onTap: () => context.push('/settings/family/expenses'),
),
ListTile(
  leading: const Icon(Icons.bar_chart),
  title: const Text('Family Summary'),
  subtitle: const Text('Totals by person and category'),
  onTap: () => context.push('/settings/family/summary'),
),
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Client-side aggregation | Server-side SQL aggregation | Always preferred | Reduces data transfer, faster rendering |
| Separate API per member | Single JOIN query | Standard SQL pattern | Avoids N+1, single round-trip |

**No deprecated patterns apply.** This phase uses standard SQL aggregation and established project patterns.

## Open Questions

1. **Navigation: Where do family views live?**
   - What we know: Family management is under Settings > Family. The family feed and summary are new views.
   - Options: (A) Add as rows in the FamilyScreen when in FamilyLoaded state, navigate to separate screens. (B) Add a "Family" tab in the bottom nav that conditionally appears when user is in a family.
   - Recommendation: Option A -- keep family views accessible from the existing FamilyScreen. Adding/removing a bottom nav tab based on family state creates UX complexity and routing issues. Two ListTiles ("Family Expenses" and "Family Summary") on the FamilyScreen are simple and discoverable.

2. **Family feed: Show all-time or default to current month?**
   - What we know: The individual expense list shows all expenses (paginated). The summary endpoints use month-based filtering.
   - Recommendation: Default to current month for both feed and summary, with month navigation (matching charts screen pattern). This avoids loading massive datasets and aligns with the summary dashboard UX.

3. **Category ownership: Family members may have different categories**
   - What we know: Categories are user-owned. User A's "Food" category is different from User B's "Food". The JOIN includes category details per-expense.
   - Recommendation: Display each expense with its own category info (name, color, icon) as returned by the server. The per-category summary aggregates by category_id, which means different users' categories appear separately even if named similarly. This is correct behavior -- each user's categories are independent.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | Go testing + httptest (built-in) |
| Framework (Flutter) | flutter_test + mocktail |
| Config file | None needed (standard Go/Flutter test setup) |
| Quick run (Go) | `cd server && go test ./internal/handler/ -run TestFamilyView -v` |
| Quick run (Flutter) | `cd app && flutter test test/features/family/` |
| Full suite (Go) | `cd server && go test ./...` |
| Full suite (Flutter) | `cd app && flutter test` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FAM-03 | Family combined expense feed returns expenses with user email | unit (Go handler) | `cd server && go test ./internal/handler/ -run TestFamilyFeed -v` | No -- Wave 0 |
| FAM-03 | Family feed shows who spent what in UI | widget (Flutter) | `cd app && flutter test test/features/family/presentation/family_feed_screen_test.dart` | No -- Wave 0 |
| FAM-03 | Family feed 404 when user not in family | unit (Go handler) | `cd server && go test ./internal/handler/ -run TestFamilyFeedNoFamily -v` | No -- Wave 0 |
| FAM-04 | Family summary returns per-person totals | unit (Go handler) | `cd server && go test ./internal/handler/ -run TestFamilySummary -v` | No -- Wave 0 |
| FAM-04 | Family summary returns per-category totals | unit (Go handler) | `cd server && go test ./internal/handler/ -run TestFamilySummary -v` | No -- Wave 0 |
| FAM-04 | Family summary dashboard UI shows person and category breakdown | widget (Flutter) | `cd app && flutter test test/features/family/presentation/family_summary_screen_test.dart` | No -- Wave 0 |
| FAM-04 | Family summary month navigation | widget (Flutter) | `cd app && flutter test test/features/family/presentation/family_summary_screen_test.dart` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** Quick run for affected layer (Go or Flutter)
- **Per wave merge:** Full suite for both Go and Flutter
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `server/internal/handler/family_view_test.go` -- covers FAM-03, FAM-04 (Go handler tests with mockFamilyViewDB)
- [ ] `app/test/features/family/presentation/family_feed_screen_test.dart` -- covers FAM-03 (family feed UI)
- [ ] `app/test/features/family/presentation/family_summary_screen_test.dart` -- covers FAM-04 (family summary UI)

## Sources

### Primary (HIGH confidence)
- Project codebase: `server/internal/handler/summary.go` + `summary_db.go` -- established summary handler/DB interface pattern being replicated
- Project codebase: `server/internal/db/queries/summary.sql` -- SQL aggregation pattern with JOINs and GROUP BY
- Project codebase: `server/internal/db/queries/expenses.sql` -- GetExpensesByUserFiltered with pagination and optional filters
- Project codebase: `server/internal/db/migrations/00005_families.sql` -- family_members table schema (JOIN target)
- Project codebase: `server/internal/handler/family.go` -- FamilyDB interface and membership verification pattern
- Project codebase: `app/lib/features/history/presentation/history_screen.dart` -- expense list UI pattern being extended
- Project codebase: `app/lib/features/charts/presentation/widgets/monthly_summary.dart` -- summary display pattern being replicated
- Project codebase: `app/lib/features/charts/domain/chart_state.dart` -- ChartData/CategoryBreakdown models being mirrored
- Project codebase: `app/lib/features/family/domain/family_notifier.dart` -- FamilyNotifier StateNotifier pattern
- Project codebase: `server/internal/router/router.go` -- route registration pattern (family routes group at line 59)

### Secondary (MEDIUM confidence)
- None -- all patterns are already proven in this codebase

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- zero new dependencies, all patterns already established
- Architecture: HIGH -- direct replication of existing summary handler + history screen patterns
- Pitfalls: HIGH -- authorization, N+1, and pagination are well-understood problems with known solutions in this codebase
- SQL queries: HIGH -- standard JOIN + GROUP BY aggregation, mirrors existing GetCategoryTotals

**Research date:** 2026-03-16
**Valid until:** 2026-04-16 (stable -- no fast-moving dependencies)
