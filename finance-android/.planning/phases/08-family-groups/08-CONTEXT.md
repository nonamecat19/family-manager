# Phase 8: Family Groups - Context

**Gathered:** 2026-03-16
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can create family groups and invite others to join. This is the first multi-user collaborative feature. Phase 9 (Family Views) adds shared expense feeds and dashboards — this phase handles only group creation, invitation, and membership management.

</domain>

<decisions>
## Implementation Decisions

### Invitation mechanism
- Invite via shareable link (deep link into app)
- Links expire after 7 days
- Creator can revoke pending invitations before acceptance
- A user can belong to only one family group at a time — joining a new family requires leaving the current one

### Group creation flow
- Only a group name is required to create a family (no description, icon, or other fields)
- Maximum 10 members per group
- Creator is automatically the admin — admin role cannot be transferred
- Admin can remove members from the group (removed member's expenses stay with them)

### Membership visibility
- Family management lives under Settings as a subsection (new "Family" row in the Settings ListView, next to Categories and Log Out)
- Family management screen shows: group name, member list (name/email), pending invitations, and an invite button
- Admin sees remove buttons next to each non-admin member
- "Leave Family" button at the bottom of the family screen with confirmation dialog
- When admin leaves/deletes the group, the group is dissolved — all members are removed
- Expenses always stay with the individual user regardless of group changes

### Claude's Discretion
- Deep link implementation approach (custom URL scheme vs universal links)
- Invite link format and token generation strategy
- Empty state design for users not in any family
- Loading states and error handling for invitation acceptance flow
- Database schema design for families, memberships, and invitations tables

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

No external specs — requirements fully captured in decisions above. Key project files for reference:

### Authentication (invitation auth context)
- `server/internal/handler/auth.go` — Existing auth handler pattern (JWT, middleware)
- `server/internal/middleware/auth.go` — Auth middleware for protected routes

### Database patterns
- `server/internal/db/migrations/00002_users.sql` — Users table schema (UUID PKs, timestamps)
- `server/internal/db/migrations/00004_expenses.sql` — Example of FK relationships and indexes
- `server/sqlc.yaml` — sqlc configuration for query generation

### Flutter patterns
- `app/lib/features/settings/presentation/settings_screen.dart` — Settings screen where Family row will be added
- `app/lib/features/auth/presentation/login_screen.dart` — Auth flow pattern for reference on invitation acceptance

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `SettingsScreen` (ListView with ListTile rows): Add "Family" row following Categories pattern
- Auth handler pattern (`AuthDB` interface, `PgAuthDB` adapter): Replicate for `FamilyDB`/`InviteDB`
- `authStateProvider` / `AuthNotifier`: Pattern for family state management
- Go router navigation: Existing `/settings/categories` nested route pattern for `/settings/family`

### Established Patterns
- Server: handler → DB interface → sqlc queries → goose migrations (consistent across auth, categories, expenses, summary)
- Flutter: feature folder → domain/data/presentation layers, Riverpod StateNotifier
- UUIDs for all primary keys, `TIMESTAMPTZ` for timestamps
- `ON DELETE CASCADE` for dependent records

### Integration Points
- Settings screen: New ListTile for "Family" navigation
- Go router: New `/settings/family` and `/invite/:token` routes
- Server router: New `/api/v1/families` and `/api/v1/invites` route groups under auth middleware
- Users table: Family membership FK (or join table) linking users to families

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 08-family-groups*
*Context gathered: 2026-03-16*
