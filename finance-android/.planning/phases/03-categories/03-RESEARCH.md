# Phase 3: Categories - Research

**Researched:** 2026-03-15
**Domain:** CRUD categories with visual identity (icon + color), Go backend + Flutter frontend
**Confidence:** HIGH

## Summary

Phase 3 adds a complete category management system -- users create, edit, reorder, and delete expense categories, each with a name, icon, and color. The backend needs a new `categories` table with a `sort_order` column for drag-to-reorder, plus standard CRUD endpoints behind auth middleware. The Flutter side needs a new `categories` feature following the established feature-first structure (data/domain/presentation layers) with Riverpod StateNotifier for state management.

The complexity centers on three UI elements: (1) an icon picker bottom sheet with 30-50 curated Material Icons, (2) a color picker bottom sheet with 12-16 preset colors, and (3) drag-to-reorder using Flutter's built-in `ReorderableListView`. None of these require external packages -- Material Design's built-in `Icons` class and Flutter's `showModalBottomSheet` with `ReorderableListView` cover all needs.

**Primary recommendation:** Follow the exact patterns from Phase 2 (DB interface for testability, sqlc queries, Riverpod StateNotifier, feature-first folders). Use `ReorderableListView` for drag-to-reorder and `showModalBottomSheet` for pickers -- no external packages needed.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Preset icon grid: 30-50 curated Material Icons relevant to expense categories (food, transport, shopping, health, home, etc.)
- Preset color palette: 12-16 curated colors that look good together and ensure visual harmony
- Both icon and color are required when creating a category -- no defaults/fallbacks
- Picker appears as a bottom sheet that slides up from below the create/edit form
- Categories managed via Settings > Categories (dedicated list screen)
- Tap a category to edit -- opens same form as create, pre-filled with current values
- Swipe left to reveal delete button, tap to confirm with dialog
- On deletion with existing expenses: force user to reassign expenses to another category before completing deletion
- No limit on number of categories
- Category list (Settings): icon in a colored circle + category name per row
- Categories elsewhere in app: colored chip (pill) with icon + name -- compact, works inline
- Manual drag-to-reorder for category list -- user controls sort order, persisted on server
- Start with empty category list (no pre-populated defaults)
- One-time prompt on first visit: "Get started quickly?" with ~6 starter suggestions
- User can "Add these" (bulk create all) or "Skip" -- prompt never shown again
- Starter categories are fully editable/deletable after creation
- Follows flat design from Phase 1 -- no shadows, no elevation

### Claude's Discretion
- Exact starter category icons and colors
- Icon grid layout and grouping (by category like "food", "transport", etc. or flat grid)
- Color palette specific hex values
- Create/edit form layout details
- Drag handle style for reorder
- Animation and transition details
- Backend schema design (categories table structure)
- API endpoint design

### Deferred Ideas (OUT OF SCOPE)
None
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CAT-01 | User can create custom expense categories | CRUD endpoints + CategoryRepository + create form with name/icon/color fields |
| CAT-02 | User can edit category name | Same form as create, pre-filled; PUT endpoint updates all fields |
| CAT-03 | User can delete categories | DELETE endpoint + swipe-to-delete UI + reassignment dialog (future-proofing for expenses) |
| CAT-04 | User can assign an icon to each category | Icon picker bottom sheet with curated Material Icons grid |
| CAT-05 | User can assign a color to each category | Color picker bottom sheet with 12-16 preset color circles |
</phase_requirements>

## Standard Stack

### Core (already in project)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| flutter_riverpod | 2.6.1 | State management | Already used for auth, StateNotifier pattern established |
| go_router | 14.0.0 | Routing | Already used, add `/settings/categories` route |
| dio | 5.4.0 | HTTP client | Already configured with auth interceptor |
| gin | 1.12.0 | Go HTTP framework | Already used for auth endpoints |
| sqlc | 2.x | Go SQL code generation | Already configured, generates type-safe queries |
| goose | - | DB migrations | Already used for users/tokens migrations |
| pgx/v5 | 5.8.0 | PostgreSQL driver | Already used |

### Supporting (no new dependencies needed)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Material Icons (built-in) | Flutter SDK | Icon picker source | `Icons.restaurant`, `Icons.directions_car`, etc. |
| ReorderableListView (built-in) | Flutter SDK | Drag-to-reorder | Category list in settings |
| showModalBottomSheet (built-in) | Flutter SDK | Icon/color picker container | Bottom sheet pickers |
| Dismissible (built-in) | Flutter SDK | Swipe-to-delete | Category list items |
| shared_preferences | 2.2.0 | Persist "first visit" flag | Already in pubspec, use for starter prompt dismissal |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| ReorderableListView | flutter_reorderable_list package | Built-in is sufficient, no extra dependency |
| Material Icons grid | flutter_iconpicker package | Package is overkill -- we need a curated subset, not full picker |
| Preset color grid | flutter_colorpicker package | Package adds color wheel/sliders we don't need -- preset grid is trivial |

**Installation:**
```bash
# No new packages needed -- everything is already in pubspec.yaml or Flutter SDK
```

## Architecture Patterns

### Recommended Project Structure
```
app/lib/features/categories/
  data/
    category_repository.dart    # API calls via Dio
    models/
      category.dart             # Category model with fromJson/toJson
  domain/
    category_state.dart         # Sealed class: Initial/Loading/Loaded/Error
  presentation/
    categories_screen.dart      # Settings > Categories list with reorder
    category_form_screen.dart   # Create/edit form (shared)
    widgets/
      category_chip.dart        # Reusable colored pill (icon + name)
      category_tile.dart        # List tile with colored circle + name
      icon_picker_sheet.dart    # Bottom sheet with icon grid
      color_picker_sheet.dart   # Bottom sheet with color circles

app/lib/providers/
  category_provider.dart        # StateNotifier + provider

server/internal/
  handler/
    category.go                 # CategoryHandler with CRUD methods
    category_db.go              # PgCategoryDB implementing CategoryDB interface
    category_test.go            # Tests with mock CategoryDB
  db/
    migrations/
      00003_categories.sql      # Categories table migration
    queries/
      categories.sql            # sqlc queries for CRUD + reorder
```

### Pattern 1: DB Interface for Testability (from Phase 2)
**What:** Define a `CategoryDB` interface in the handler package, implement with `PgCategoryDB` wrapping sqlc queries
**When to use:** All handler tests -- mock the interface, no real DB needed
**Example:**
```go
// Source: Established pattern from handler/auth.go + auth_db.go

type CategoryDB interface {
    CreateCategory(userID, name, icon, color string, sortOrder int) (Category, error)
    GetCategoriesByUser(userID string) ([]Category, error)
    GetCategoryByID(id, userID string) (Category, error)
    UpdateCategory(id, userID, name, icon, color string) error
    DeleteCategory(id, userID string) error
    UpdateCategorySortOrder(id, userID string, sortOrder int) error
    BulkCreateCategories(userID string, categories []BulkCategory) ([]Category, error)
}
```

### Pattern 2: Riverpod StateNotifier (from Phase 2)
**What:** Sealed state class + StateNotifier + StateNotifierProvider
**When to use:** Category list state management
**Example:**
```dart
// Source: Established pattern from providers/auth_provider.dart

sealed class CategoryState {
  const CategoryState();
}
class CategoryInitial extends CategoryState { const CategoryInitial(); }
class CategoryLoading extends CategoryState { const CategoryLoading(); }
class CategoryLoaded extends CategoryState {
  const CategoryLoaded(this.categories);
  final List<Category> categories;
}
class CategoryError extends CategoryState {
  const CategoryError(this.message);
  final String message;
}

class CategoryNotifier extends StateNotifier<CategoryState> {
  CategoryNotifier(this._repository) : super(const CategoryInitial());
  final CategoryRepository _repository;
  // CRUD methods...
}
```

### Pattern 3: Feature-first folder structure (from Phase 1/2)
**What:** `features/categories/data/`, `features/categories/domain/`, `features/categories/presentation/`
**When to use:** Always -- this is the project convention

### Anti-Patterns to Avoid
- **Storing icon as IconData codePoint in DB:** Store icon name as string (e.g., "restaurant") -- codePoints change between Flutter versions. Map string to IconData on the client side.
- **Storing color as Color object:** Store as hex string (e.g., "#FF5722") in DB -- universal, parseable on any client.
- **Separate create and edit screens:** Use ONE form screen that accepts an optional Category for edit mode, empty for create mode. Reduces duplication.
- **Client-side sort order management:** Persist sort_order on server -- the client sends the full reordered list after a drag operation. Otherwise sort order is lost on reinstall.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Drag-to-reorder list | Custom gesture detector + animation | `ReorderableListView.builder` | Built-in handles all drag UX, callbacks, and animations |
| Swipe-to-delete | Custom horizontal gesture detection | `Dismissible` widget | Built-in handles swipe direction, threshold, background reveal |
| Bottom sheet | Custom overlay management | `showModalBottomSheet` | Built-in handles backdrop, drag-to-dismiss, animations |
| UUID generation (server) | Custom ID generation | PostgreSQL `gen_random_uuid()` | Already used in users table, battle-tested |

**Key insight:** Flutter's built-in widgets (ReorderableListView, Dismissible, showModalBottomSheet) handle all the complex gesture/animation work. No external packages are needed for this phase.

## Common Pitfalls

### Pitfall 1: Icon Name Mapping Fragility
**What goes wrong:** Storing Flutter IconData codePoints in the database, which break across Flutter versions.
**Why it happens:** Developers serialize Icons.restaurant.codePoint to an integer and store it.
**How to avoid:** Store a string key (e.g., "restaurant", "directions_car") and maintain a `Map<String, IconData>` on the client. The map is the single source of truth for available icons.
**Warning signs:** Integer icon codes in the database.

### Pitfall 2: Sort Order Gaps After Deletion
**What goes wrong:** After deleting a category, sort_order values have gaps (e.g., 0, 1, 3, 4). Not technically broken but makes insert-at-position logic harder.
**Why it happens:** Only deleting the row without recompacting sort_order values.
**How to avoid:** Accept gaps -- they don't matter. Sort by sort_order ascending and the ordering is correct regardless of gaps. Only update sort_order values when the user explicitly reorders.
**Warning signs:** Complex re-indexing logic on every delete.

### Pitfall 3: Race Condition on Reorder
**What goes wrong:** Multiple rapid reorder operations overwrite each other.
**Why it happens:** Each reorder sends the full list of (id, sort_order) pairs to the server, but if two requests overlap, the second may use stale data.
**How to avoid:** Debounce reorder API calls (wait 300-500ms after last drag before sending). Update local state immediately (optimistic) but only send one final server request.
**Warning signs:** Category order flickering after multiple rapid reorders.

### Pitfall 4: Deletion Without Expense Reassignment
**What goes wrong:** Foreign key constraint violation when deleting a category that has expenses.
**Why it happens:** Phase 3 builds categories; Phase 4 adds expenses referencing them. The deletion flow must be designed now to handle this future constraint.
**How to avoid:** The API should check if expenses reference this category. If so, return a 409 Conflict with the count of affected expenses. The client shows a reassignment dialog. The delete endpoint accepts an optional `reassign_to` category ID. For Phase 3 (no expenses yet), the check will always pass, but the plumbing must exist.
**Warning signs:** No `reassign_to` parameter on delete endpoint.

### Pitfall 5: Bottom Sheet State Loss
**What goes wrong:** User selects an icon in the bottom sheet, sheet closes, selection is lost.
**Why it happens:** Bottom sheet state is not communicated back to the parent form.
**How to avoid:** Pass selected value back via `Navigator.pop(context, selectedIcon)` and await the result in the parent. Or use a callback parameter.
**Warning signs:** Using global state or providers for ephemeral picker state.

## Code Examples

### Categories Table Migration
```sql
-- Source: Follows pattern from 00002_users.sql

-- +goose Up
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    icon TEXT NOT NULL,          -- Material Icon name string, e.g. "restaurant"
    color TEXT NOT NULL,         -- Hex color string, e.g. "#FF5722"
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_categories_user_id ON categories(user_id);
CREATE UNIQUE INDEX idx_categories_user_sort ON categories(user_id, sort_order);

-- +goose Down
DROP TABLE IF EXISTS categories;
```

### sqlc Queries
```sql
-- Source: Follows pattern from queries/users.sql

-- name: CreateCategory :one
INSERT INTO categories (user_id, name, icon, color, sort_order)
VALUES ($1, $2, $3, $4, (SELECT COALESCE(MAX(sort_order), -1) + 1 FROM categories WHERE user_id = $1))
RETURNING id, user_id, name, icon, color, sort_order, created_at, updated_at;

-- name: GetCategoriesByUser :many
SELECT id, user_id, name, icon, color, sort_order, created_at, updated_at
FROM categories
WHERE user_id = $1
ORDER BY sort_order ASC;

-- name: GetCategoryByID :one
SELECT id, user_id, name, icon, color, sort_order, created_at, updated_at
FROM categories
WHERE id = $1 AND user_id = $2;

-- name: UpdateCategory :exec
UPDATE categories
SET name = $3, icon = $4, color = $5, updated_at = NOW()
WHERE id = $1 AND user_id = $2;

-- name: DeleteCategory :exec
DELETE FROM categories WHERE id = $1 AND user_id = $2;

-- name: UpdateCategorySortOrder :exec
UPDATE categories SET sort_order = $3, updated_at = NOW()
WHERE id = $1 AND user_id = $2;

-- name: CountExpensesByCategory :one
SELECT COUNT(*) FROM expenses WHERE category_id = $1;
```

### REST API Endpoints
```
POST   /api/v1/categories           -- Create category
GET    /api/v1/categories           -- List user's categories
PUT    /api/v1/categories/:id       -- Update category
DELETE /api/v1/categories/:id       -- Delete category (query param: ?reassign_to=uuid)
PUT    /api/v1/categories/reorder   -- Batch update sort_order
POST   /api/v1/categories/bulk      -- Bulk create (for starter categories)
```

### Category Model (Flutter)
```dart
// Source: Follows pattern from auth_response.dart

class Category {
  const Category({
    required this.id,
    required this.name,
    required this.icon,
    required this.color,
    required this.sortOrder,
  });

  factory Category.fromJson(Map<String, dynamic> json) {
    return Category(
      id: json['id'] as String,
      name: json['name'] as String,
      icon: json['icon'] as String,
      color: json['color'] as String,
      sortOrder: json['sort_order'] as int,
    );
  }

  final String id;
  final String name;
  final String icon;   // Material Icon name, e.g. "restaurant"
  final String color;  // Hex string, e.g. "#FF5722"
  final int sortOrder;

  Map<String, dynamic> toJson() => {
    'name': name,
    'icon': icon,
    'color': color,
  };
}
```

### Icon Name to IconData Mapping
```dart
/// Maps icon name strings (stored in DB) to Flutter IconData.
///
/// This is the single source of truth for available category icons.
const Map<String, IconData> categoryIcons = {
  // Food & Drink
  'restaurant': Icons.restaurant,
  'local_cafe': Icons.local_cafe,
  'local_bar': Icons.local_bar,
  'local_grocery_store': Icons.local_grocery_store,
  // Transport
  'directions_car': Icons.directions_car,
  'directions_bus': Icons.directions_bus,
  'local_gas_station': Icons.local_gas_station,
  'flight': Icons.flight,
  // Shopping
  'shopping_cart': Icons.shopping_cart,
  'shopping_bag': Icons.shopping_bag,
  'storefront': Icons.storefront,
  // Home
  'home': Icons.home,
  'electrical_services': Icons.electrical_services,
  'plumbing': Icons.plumbing,
  // Health
  'local_hospital': Icons.local_hospital,
  'fitness_center': Icons.fitness_center,
  'medication': Icons.medication,
  // Entertainment
  'movie': Icons.movie,
  'sports_esports': Icons.sports_esports,
  'music_note': Icons.music_note,
  // Education
  'school': Icons.school,
  'menu_book': Icons.menu_book,
  // Personal
  'checkroom': Icons.checkroom,
  'content_cut': Icons.content_cut,
  // Finance
  'savings': Icons.savings,
  'account_balance': Icons.account_balance,
  'credit_card': Icons.credit_card,
  // Misc
  'pets': Icons.pets,
  'child_care': Icons.child_care,
  'card_giftcard': Icons.card_giftcard,
  'phone_android': Icons.phone_android,
  'wifi': Icons.wifi,
  'more_horiz': Icons.more_horiz,
};
```

### Recommended Color Palette
```dart
/// Curated category color palette -- 16 colors with good contrast and harmony.
const List<Color> categoryColors = [
  Color(0xFFEF5350), // Red
  Color(0xFFEC407A), // Pink
  Color(0xFFAB47BC), // Purple
  Color(0xFF7E57C2), // Deep Purple
  Color(0xFF5C6BC0), // Indigo
  Color(0xFF42A5F5), // Blue
  Color(0xFF29B6F6), // Light Blue
  Color(0xFF26C6DA), // Cyan
  Color(0xFF26A69A), // Teal
  Color(0xFF66BB6A), // Green
  Color(0xFF9CCC65), // Light Green
  Color(0xFFD4E157), // Lime
  Color(0xFFFFCA28), // Amber
  Color(0xFFFFA726), // Orange
  Color(0xFFFF7043), // Deep Orange
  Color(0xFF8D6E63), // Brown
];
```

### Reusable Category Chip Widget
```dart
// Source: User decision -- "colored chip (pill) with icon + name"

class CategoryChip extends StatelessWidget {
  const CategoryChip({required this.category, super.key});

  final Category category;

  @override
  Widget build(BuildContext context) {
    final color = Color(int.parse(category.color.replaceFirst('#', '0xFF')));
    final iconData = categoryIcons[category.icon] ?? Icons.category;

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: color.withOpacity(0.15),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(iconData, size: 16, color: color),
          const SizedBox(width: 4),
          Text(
            category.name,
            style: TextStyle(color: color, fontWeight: FontWeight.w500),
          ),
        ],
      ),
    );
  }
}
```

### Starter Categories (Claude's Discretion)
```dart
/// Default starter categories offered on first visit.
const List<Map<String, String>> starterCategories = [
  {'name': 'Food', 'icon': 'restaurant', 'color': '#FF7043'},
  {'name': 'Transport', 'icon': 'directions_car', 'color': '#42A5F5'},
  {'name': 'Housing', 'icon': 'home', 'color': '#66BB6A'},
  {'name': 'Shopping', 'icon': 'shopping_cart', 'color': '#AB47BC'},
  {'name': 'Coffee', 'icon': 'local_cafe', 'color': '#8D6E63'},
  {'name': 'Entertainment', 'icon': 'movie', 'color': '#FFCA28'},
];
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `ListView` + manual drag detection | `ReorderableListView.builder` | Flutter 1.22+ | Built-in reorder with proper accessibility |
| `Scaffold.bottomSheet` | `showModalBottomSheet` with `DraggableScrollableSheet` | Flutter 3.x | Better bottom sheet UX with drag-to-dismiss |
| Riverpod 3.x codegen providers | Riverpod 2.6.1 StateNotifier (project constraint) | N/A | Dart 3.7.2 doesn't support Riverpod 3.x |

**Deprecated/outdated:**
- `ChangeNotifier` pattern: Project uses `StateNotifier` (sealed class pattern from auth)
- Riverpod 1.x `StateProvider`: Replaced by `StateNotifierProvider` in this project

## Open Questions

1. **Unique sort_order constraint under concurrent reorder**
   - What we know: A UNIQUE index on (user_id, sort_order) prevents duplicate positions, but batch reorder updates may temporarily violate the constraint mid-transaction.
   - What's unclear: Whether sqlc-generated code handles this within a transaction.
   - Recommendation: Drop the unique constraint on sort_order (gaps are fine), or use a transaction that sets all sort_orders to negative values first, then to positive final values. Simpler approach: just remove the unique index -- sort_order is advisory, not a hard constraint.

2. **Expense reassignment at deletion time (Phase 3 vs Phase 4)**
   - What we know: The user wants forced reassignment when expenses exist. Phase 3 has no expenses table yet.
   - What's unclear: How much of the reassignment plumbing to build now.
   - Recommendation: Build the API parameter (`reassign_to`) now but skip the actual reassignment logic. The delete endpoint checks for expenses (returns 0 in Phase 3). Phase 4 will add the actual reassignment query.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | `go test` (standard library) |
| Framework (Flutter) | `flutter_test` + `mocktail` 1.0.0 |
| Config file | None needed (both use convention) |
| Quick run (Go) | `cd server && go test ./internal/handler/ -run Category -v` |
| Quick run (Flutter) | `cd app && flutter test test/features/categories/` |
| Full suite (Go) | `cd server && go test ./...` |
| Full suite (Flutter) | `cd app && flutter test` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CAT-01 | Create category with name/icon/color | unit (handler + widget) | `go test ./internal/handler/ -run TestCreateCategory -v` | No -- Wave 0 |
| CAT-02 | Edit category name (and other fields) | unit (handler + widget) | `go test ./internal/handler/ -run TestUpdateCategory -v` | No -- Wave 0 |
| CAT-03 | Delete category with confirmation | unit (handler + widget) | `go test ./internal/handler/ -run TestDeleteCategory -v` | No -- Wave 0 |
| CAT-04 | Assign icon to category | unit (widget) | `flutter test test/features/categories/presentation/category_form_screen_test.dart` | No -- Wave 0 |
| CAT-05 | Assign color to category | unit (widget) | `flutter test test/features/categories/presentation/category_form_screen_test.dart` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `cd server && go test ./internal/handler/ -run Category -v` and `cd app && flutter test test/features/categories/`
- **Per wave merge:** `cd server && go test ./...` and `cd app && flutter test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `server/internal/handler/category_test.go` -- covers CAT-01, CAT-02, CAT-03 (handler CRUD tests with mockDB)
- [ ] `app/test/features/categories/presentation/categories_screen_test.dart` -- covers CAT-01, CAT-03 (list/delete UI)
- [ ] `app/test/features/categories/presentation/category_form_screen_test.dart` -- covers CAT-01, CAT-02, CAT-04, CAT-05 (form with pickers)

## Sources

### Primary (HIGH confidence)
- Project codebase: `server/internal/handler/auth.go`, `auth_db.go`, `auth_test.go` -- established Go patterns
- Project codebase: `app/lib/providers/auth_provider.dart` -- established Riverpod StateNotifier pattern
- Project codebase: `server/internal/db/migrations/00002_users.sql` -- established migration pattern
- Project codebase: `server/sqlc.yaml` -- sqlc configuration (pgx/v5, emit_json_tags)
- Project codebase: `app/lib/core/router/app_router.dart` -- go_router ShellRoute pattern
- Project codebase: `app/lib/core/theme/app_theme.dart` -- zero-elevation flat design

### Secondary (MEDIUM confidence)
- Flutter SDK documentation: ReorderableListView, Dismissible, showModalBottomSheet -- well-established stable APIs
- Material Icons class -- stable icon names across Flutter versions

### Tertiary (LOW confidence)
- None -- all patterns derive from existing project code and stable Flutter/Go SDK features

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies, everything already in project
- Architecture: HIGH -- follows exact patterns from Phase 2 (DB interface, StateNotifier, feature-first)
- Pitfalls: HIGH -- common patterns well understood, verified against project codebase
- UI patterns: MEDIUM -- icon/color picker layout is discretionary, but the technical approach (bottom sheet + grid) is straightforward

**Research date:** 2026-03-15
**Valid until:** 2026-04-15 (stable -- no fast-moving dependencies)
