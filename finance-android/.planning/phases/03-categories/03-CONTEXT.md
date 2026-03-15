# Phase 3: Categories - Context

**Gathered:** 2026-03-15
**Status:** Ready for planning

<domain>
## Phase Boundary

User-created expense categories with visual identity (icon and color). Users can create, edit, reorder, and delete categories. Categories persist across sessions and display with their assigned icon and color throughout the app. No expense entry in this phase — just the category system.

</domain>

<decisions>
## Implementation Decisions

### Icon & color picker
- Preset icon grid: 30-50 curated Material Icons relevant to expense categories (food, transport, shopping, health, home, etc.)
- Preset color palette: 12-16 curated colors that look good together and ensure visual harmony
- Both icon and color are required when creating a category — no defaults/fallbacks
- Picker appears as a bottom sheet that slides up from below the create/edit form

### Category management
- Categories managed via Settings > Categories (dedicated list screen)
- Tap a category to edit — opens same form as create, pre-filled with current values
- Swipe left to reveal delete button, tap to confirm with dialog
- On deletion with existing expenses: force user to reassign expenses to another category before completing deletion
- No limit on number of categories

### Visual presentation
- Category list (Settings): icon in a colored circle + category name per row
- Categories elsewhere in app (expense history, charts): colored chip (pill) with icon + name — compact, works inline
- Manual drag-to-reorder for category list — user controls sort order, persisted on server

### Default categories
- Start with empty category list (no pre-populated defaults)
- One-time prompt on first visit to Categories screen: "Get started quickly?" with ~6 starter suggestions (Food, Transport, Housing, Shopping, Coffee, Entertainment)
- User can "Add these" (bulk create all) or "Skip" — prompt never shown again
- Starter categories are fully editable/deletable after creation

### Claude's Discretion
- Exact starter category icons and colors
- Icon grid layout and grouping (by category like "food", "transport", etc. or flat grid)
- Color palette specific hex values
- Create/edit form layout details
- Drag handle style for reorder
- Animation and transition details
- Backend schema design (categories table structure)
- API endpoint design

</decisions>

<specifics>
## Specific Ideas

- Follows flat design from Phase 1 — no shadows, no elevation on cards or sheets
- Bottom sheet for pickers should feel native and smooth
- Category colored chips should be reusable — they'll appear in expense history (Phase 6) and charts (Phase 7)
- Deletion confirmation should clearly explain the consequence (expenses will be reassigned)

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `app/lib/core/theme/app_theme.dart`: Zero-elevation flat theme — all new UI follows this
- `app/lib/core/network/api_client.dart`: Dio HTTP client with auth interceptor — use for category API calls
- `app/lib/core/router/app_router.dart`: go_router with ShellRoute — add categories route under settings
- `app/lib/providers/auth_provider.dart`: Riverpod auth state — categories are per-user
- `server/internal/handler/`: Go handler pattern — add category handlers
- `server/internal/db/`: sqlc + goose pattern — add categories migration and queries
- `server/internal/middleware/auth.go`: Auth middleware — protect category endpoints

### Established Patterns
- Go: handler → router registration, sqlc for DB, goose for migrations
- Flutter: Riverpod 2.6.x StateNotifier pattern (see auth_provider.dart)
- Flutter: Feature-first folder structure: `features/{name}/presentation/`
- Flutter: go_router with redirect guards

### Integration Points
- Settings screen already has "Log Out" — add "Categories" row above it
- Auth middleware protects all category endpoints (user_id from JWT)
- Categories will be referenced by expenses in Phase 4

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 03-categories*
*Context gathered: 2026-03-15*
