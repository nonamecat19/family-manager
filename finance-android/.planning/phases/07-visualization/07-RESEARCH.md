# Phase 7: Visualization - Research

**Researched:** 2026-03-16
**Domain:** Flutter charting (fl_chart), server-side aggregation, Riverpod state management
**Confidence:** HIGH

## Summary

Phase 7 adds visual spending summaries to the existing Charts tab (currently a placeholder). The work spans two domains: (1) a new server-side aggregation endpoint that computes category totals and time-series data in PostgreSQL, and (2) Flutter UI using fl_chart to render pie and bar charts plus a monthly summary list.

The existing architecture already has a `/charts` route in the ShellRoute with a placeholder `ChartsScreen`. The expense table has an index on `(user_id, expense_date)` which supports efficient date-range aggregation. Categories carry `color` and `icon` fields that map directly to chart section styling.

**Primary recommendation:** Add a single `GET /api/v1/expenses/summary` endpoint returning both category breakdown and daily totals for a month, then build three chart widgets (pie, bar, summary list) in the Flutter `charts` feature using fl_chart 1.2.0.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| VIS-01 | User can view pie chart of spending by category | fl_chart PieChart widget + server category aggregation endpoint |
| VIS-02 | User can view bar chart of spending over time (weekly/monthly) | fl_chart BarChart widget + server daily-totals aggregation |
| VIS-03 | User can view monthly summary (total spent, category breakdown) | Summary list widget consuming same aggregation data |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| fl_chart | ^1.2.0 | Pie chart, bar chart rendering | Most popular Flutter charting library (3.5k+ GitHub stars), actively maintained, pure Flutter (no platform channels), supports touch interaction and animation |
| flutter_riverpod | 2.6.1 | State management for chart data | Already in use project-wide |
| intl | 0.20.2 | Month/date formatting for axis labels | Already in use project-wide |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| (none new) | - | - | All supporting libraries already in pubspec.yaml |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| fl_chart | syncfusion_flutter_charts | Syncfusion is more feature-rich but has license requirements; fl_chart is MIT and sufficient for pie+bar |
| fl_chart | charts_flutter (Google) | Discontinued/unmaintained; fl_chart is the community standard |
| Server aggregation | Client-side computation | Client-side requires fetching ALL expenses (unbounded); server aggregation is O(1) response size and leverages PostgreSQL GROUP BY |

**Installation:**
```bash
cd app && flutter pub add fl_chart
```

## Architecture Patterns

### Recommended Project Structure
```
app/lib/features/charts/
├── data/
│   └── chart_repository.dart       # API calls to /expenses/summary
├── domain/
│   └── chart_state.dart            # ChartData model + sealed state
├── presentation/
│   ├── charts_screen.dart          # Tab host with month selector
│   ├── widgets/
│   │   ├── spending_pie_chart.dart  # PieChart widget (VIS-01)
│   │   ├── spending_bar_chart.dart  # BarChart widget (VIS-02)
│   │   └── monthly_summary.dart    # Summary list widget (VIS-03)
app/lib/providers/
│   └── chart_provider.dart         # ChartNotifier + provider

server/internal/handler/
│   └── summary.go                  # SummaryHandler + SummaryDB interface
server/internal/db/queries/
│   └── summary.sql                 # Aggregation queries
```

### Pattern 1: Server-Side Aggregation Endpoint
**What:** A single `GET /api/v1/expenses/summary?month=2026-03` endpoint that returns pre-computed category breakdown and daily totals.
**When to use:** Always for chart data. Never fetch raw expenses and aggregate client-side.
**Why:** PostgreSQL GROUP BY is efficient on the existing `idx_expenses_user_date` index. Response payload is bounded (max categories + max 31 days).

**API Response Shape:**
```json
{
  "month": "2026-03",
  "total_cents": 125000,
  "by_category": [
    {
      "category_id": "uuid",
      "category_name": "Food",
      "category_color": "#FF7043",
      "category_icon": "restaurant",
      "total_cents": 45000,
      "count": 12
    }
  ],
  "by_date": [
    { "date": "2026-03-01", "total_cents": 5000 },
    { "date": "2026-03-02", "total_cents": 12000 }
  ]
}
```

### Pattern 2: SummaryDB Interface (Matches Existing Pattern)
**What:** A `SummaryDB` interface for testability, matching `AuthDB`, `CategoryDB`, `ExpenseDB` patterns.
**When to use:** Always -- this project uses interface-based handler testing.

```go
type CategoryTotal struct {
    CategoryID    string
    CategoryName  string
    CategoryColor string
    CategoryIcon  string
    TotalCents    int64
    Count         int
}

type DateTotal struct {
    Date       time.Time
    TotalCents int64
}

type SummaryDB interface {
    GetCategoryTotals(userID string, dateFrom, dateTo time.Time) ([]CategoryTotal, error)
    GetDailyTotals(userID string, dateFrom, dateTo time.Time) ([]DateTotal, error)
}
```

### Pattern 3: ChartNotifier (Matches Existing StateNotifier Pattern)
**What:** A `ChartNotifier extends StateNotifier<ChartState>` following `ExpenseNotifier` and `CategoryNotifier` patterns.
**When to use:** For managing chart data loading with month selection.

```dart
class ChartNotifier extends StateNotifier<ChartState> {
  ChartNotifier(this._repository) : super(const ChartInitial());
  final ChartRepository _repository;

  Future<void> loadSummary({required String month}) async {
    state = const ChartLoading();
    try {
      final data = await _repository.getSummary(month: month);
      state = ChartLoaded(data);
    } on Exception catch (e) {
      state = ChartError(e.toString());
    }
  }
}
```

### Pattern 4: Month Selector for Chart Navigation
**What:** A simple left/right arrow + month label header to navigate between months.
**When to use:** Charts need a month context. Default to current month, allow navigating to previous months.

### Anti-Patterns to Avoid
- **Fetching all expenses client-side for aggregation:** Unbounded data, slow on large datasets, wastes bandwidth
- **Separate API calls per chart:** One summary endpoint serves all three visualizations (pie, bar, summary)
- **Hardcoded chart colors:** Categories already have user-assigned colors -- use them directly
- **Rebuilding chart on every expense change:** Use the month-based summary endpoint; charts reload when month changes or when user navigates to the Charts tab

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Pie chart rendering | Custom painter with arc calculations | fl_chart PieChart | Touch handling, animation, label positioning are complex |
| Bar chart rendering | Custom painter with axis/gridlines | fl_chart BarChart | Axis labels, tooltips, responsive sizing |
| Date aggregation | Client-side loops over expense lists | PostgreSQL GROUP BY | Efficient, bounded response, works with any data volume |
| Color parsing (hex to Color) | New utility | Existing `parseHexColor` in `category_colors.dart` | Already used throughout the app |

**Key insight:** fl_chart handles all the hard rendering problems (label collision, touch zones, animation interpolation). The real work in this phase is the data pipeline: SQL aggregation -> API endpoint -> repository -> provider -> chart widgets.

## Common Pitfalls

### Pitfall 1: PieChart with Zero Data
**What goes wrong:** PieChart throws or renders empty when all sections have value 0 or the list is empty.
**Why it happens:** New users or months with no expenses.
**How to avoid:** Show an empty state message ("No spending data for this month") instead of rendering PieChart with empty sections.
**Warning signs:** White/blank chart area with no explanation.

### Pitfall 2: Bar Chart Date Gaps
**What goes wrong:** Bar chart skips dates with no expenses, creating misleading gaps.
**Why it happens:** The SQL query only returns rows for dates that have expenses.
**How to avoid:** Client-side, fill in missing dates with 0 values for the full month range before passing to BarChart.
**Warning signs:** Bar chart shows fewer bars than expected days.

### Pitfall 3: Integer Cents Display on Charts
**What goes wrong:** Chart labels show raw cents (e.g., "125000") instead of formatted currency ("$1,250.00").
**Why it happens:** Forgetting to use `Expense.formatCents()` for chart labels.
**How to avoid:** Always format through `Expense.formatCents()` for any user-visible amount. For axis labels, use abbreviated format ($1.2K).
**Warning signs:** Numbers in thousands/millions on chart labels.

### Pitfall 4: Chart Doesn't Reflect Latest Data
**What goes wrong:** User adds expense on History tab, switches to Charts tab, sees stale data.
**Why it happens:** Chart state is loaded once and cached.
**How to avoid:** Reload chart data when the Charts tab becomes active. Use `ref.listen` on expense state changes or reload on tab activation.
**Warning signs:** Success criteria #4 fails ("Charts update to reflect the current data").

### Pitfall 5: SQL Aggregation Ignoring Category Metadata
**What goes wrong:** Summary endpoint returns only category_id and total_cents, requiring a second API call to get names/colors.
**Why it happens:** Simple GROUP BY on expenses table without JOIN.
**How to avoid:** JOIN categories table in the aggregation query to include name, color, icon in the response.
**Warning signs:** Extra API round-trip or client-side category lookup needed.

## Code Examples

### SQL: Category Totals with JOIN
```sql
-- name: GetCategoryTotals :many
SELECT
    e.category_id,
    c.name AS category_name,
    c.color AS category_color,
    c.icon AS category_icon,
    SUM(e.amount_cents)::BIGINT AS total_cents,
    COUNT(*)::INT AS expense_count
FROM expenses e
JOIN categories c ON c.id = e.category_id
WHERE e.user_id = $1
  AND e.expense_date >= $2
  AND e.expense_date <= $3
GROUP BY e.category_id, c.name, c.color, c.icon
ORDER BY total_cents DESC;
```

### SQL: Daily Totals
```sql
-- name: GetDailyTotals :many
SELECT
    expense_date AS date,
    SUM(amount_cents)::BIGINT AS total_cents
FROM expenses
WHERE user_id = $1
  AND expense_date >= $2
  AND expense_date <= $3
GROUP BY expense_date
ORDER BY expense_date ASC;
```

### Flutter: PieChart with Category Colors
```dart
// Source: fl_chart official documentation
PieChart(
  PieChartData(
    sections: data.byCategory.map((cat) {
      return PieChartSectionData(
        value: cat.totalCents.toDouble(),
        color: parseHexColor(cat.color),
        title: cat.name,
        radius: 100,
        titleStyle: const TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.bold,
          color: Colors.white,
        ),
      );
    }).toList(),
    centerSpaceRadius: 40,
    sectionsSpace: 2,
  ),
)
```

### Flutter: BarChart with Daily Totals
```dart
// Source: fl_chart official documentation
BarChart(
  BarChartData(
    barGroups: dailyTotals.asMap().entries.map((entry) {
      return BarChartGroupData(
        x: entry.key,
        barRods: [
          BarChartRodData(
            toY: entry.value.totalCents / 100,
            color: Theme.of(context).colorScheme.primary,
            width: 12,
            borderRadius: const BorderRadius.vertical(
              top: Radius.circular(4),
            ),
          ),
        ],
      );
    }).toList(),
    titlesData: FlTitlesData(
      bottomTitles: AxisTitles(
        sideTitles: SideTitles(
          showTitles: true,
          getTitlesWidget: (value, meta) {
            final day = value.toInt() + 1;
            // Show every 5th day to avoid crowding
            if (day % 5 != 1 && day != daysInMonth) {
              return const SizedBox.shrink();
            }
            return Text('$day', style: const TextStyle(fontSize: 10));
          },
        ),
      ),
    ),
  ),
)
```

### Flutter: ChartData Model
```dart
class ChartData {
  const ChartData({
    required this.month,
    required this.totalCents,
    required this.byCategory,
    required this.byDate,
  });

  factory ChartData.fromJson(Map<String, dynamic> json) {
    return ChartData(
      month: json['month'] as String,
      totalCents: json['total_cents'] as int,
      byCategory: (json['by_category'] as List<dynamic>)
          .cast<Map<String, dynamic>>()
          .map(CategoryBreakdown.fromJson)
          .toList(),
      byDate: (json['by_date'] as List<dynamic>)
          .cast<Map<String, dynamic>>()
          .map(DateBreakdown.fromJson)
          .toList(),
    );
  }

  final String month;
  final int totalCents;
  final List<CategoryBreakdown> byCategory;
  final List<DateBreakdown> byDate;
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| charts_flutter (Google) | fl_chart | 2022+ | charts_flutter is unmaintained; fl_chart is the community standard |
| fl_chart 0.x API | fl_chart 1.x API | 2024 | Some class renames; 1.x is stable API |
| Client-side aggregation | Server-side aggregation | Best practice | Required for scalability with growing expense data |

**Deprecated/outdated:**
- `charts_flutter`: Google's official Flutter charts package is no longer maintained. Use `fl_chart` instead.

## Open Questions

1. **Weekly vs Monthly bar chart granularity**
   - What we know: Requirement says "weekly or monthly". Monthly (daily bars) is simpler.
   - What's unclear: Whether to show toggle or default to one view.
   - Recommendation: Default to monthly view (daily bars for current month). Weekly view can be a stretch goal. Keep API flexible by accepting date range params.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | flutter_test + Go testing |
| Config file | `app/analysis_options.yaml` (very_good_analysis) |
| Quick run command | `cd app && flutter test test/features/charts/` |
| Full suite command | `cd app && flutter test && cd ../server && go test ./...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| VIS-01 | Pie chart renders category sections with correct colors | widget | `cd app && flutter test test/features/charts/presentation/widgets/spending_pie_chart_test.dart -x` | Wave 0 |
| VIS-02 | Bar chart renders daily bars for selected month | widget | `cd app && flutter test test/features/charts/presentation/widgets/spending_bar_chart_test.dart -x` | Wave 0 |
| VIS-03 | Summary shows total and per-category breakdown | widget | `cd app && flutter test test/features/charts/presentation/widgets/monthly_summary_test.dart -x` | Wave 0 |
| VIS-01/02/03 | Summary API returns aggregated data | unit (Go) | `cd server && go test ./internal/handler/ -run TestSummary -v` | Wave 0 |

### Sampling Rate
- **Per task commit:** `cd app && flutter test test/features/charts/`
- **Per wave merge:** `cd app && flutter test && cd ../server && go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `app/test/features/charts/presentation/widgets/spending_pie_chart_test.dart` -- covers VIS-01
- [ ] `app/test/features/charts/presentation/widgets/spending_bar_chart_test.dart` -- covers VIS-02
- [ ] `app/test/features/charts/presentation/widgets/monthly_summary_test.dart` -- covers VIS-03
- [ ] `server/internal/handler/summary_test.go` -- covers summary API endpoint
- [ ] `fl_chart` dependency: `cd app && flutter pub add fl_chart` -- if not yet added

## Sources

### Primary (HIGH confidence)
- pub.dev/packages/fl_chart - version 1.2.0, Dart SDK >=3.6, verified publisher
- Project codebase - existing patterns (StateNotifier, DB interfaces, sqlc, goose migrations)

### Secondary (MEDIUM confidence)
- fl_chart GitHub documentation - PieChart and BarChart widget APIs
- fl_chart examples app (app.flchart.dev)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - fl_chart is the clear standard for Flutter charting; verified compatible with Dart 3.7.2
- Architecture: HIGH - follows established project patterns (DB interface, StateNotifier, repository)
- Pitfalls: HIGH - based on direct code analysis (empty states, date gaps, cents formatting all observable in existing code)

**Research date:** 2026-03-16
**Valid until:** 2026-04-16 (fl_chart 1.x is stable)
