# Phase 8: Family Groups - Research

**Researched:** 2026-03-16
**Domain:** Multi-user group management (server CRUD + invitation system + Flutter UI)
**Confidence:** HIGH

## Summary

Phase 8 introduces the first multi-user collaborative feature: family groups. This requires new database tables (families, family_members, family_invitations), new server API endpoints for group CRUD and invitation management, and a new Flutter feature module with screens for family management and invitation acceptance.

The implementation follows established project patterns exactly: Go handler with FamilyDB/InviteDB interfaces, sqlc-generated queries, goose migrations, and Flutter feature-first architecture with Riverpod StateNotifier. The invitation system uses cryptographically random tokens stored hashed in the database (mirroring the refresh token pattern), shared via deep links that the Flutter app intercepts.

**Primary recommendation:** Split into two plans -- Plan 1 for server-side (migration + handlers + tests) and Plan 2 for Flutter UI (family feature module, settings integration, deep link handling, widget tests).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Invite via shareable link (deep link into app)
- Links expire after 7 days
- Creator can revoke pending invitations before acceptance
- A user can belong to only one family group at a time -- joining a new family requires leaving the current one
- Only a group name is required to create a family (no description, icon, or other fields)
- Maximum 10 members per group
- Creator is automatically the admin -- admin role cannot be transferred
- Admin can remove members from the group (removed member's expenses stay with them)
- Family management lives under Settings as a subsection (new "Family" row in the Settings ListView, next to Categories and Log Out)
- Family management screen shows: group name, member list (name/email), pending invitations, and an invite button
- Admin sees remove buttons next to each non-admin member
- "Leave Family" button at the bottom of the family screen with confirmation dialog
- When admin leaves/deletes the group, the group is dissolved -- all members are removed
- Expenses always stay with the individual user regardless of group changes

### Claude's Discretion
- Deep link implementation approach (custom URL scheme vs universal links)
- Invite link format and token generation strategy
- Empty state design for users not in any family
- Loading states and error handling for invitation acceptance flow
- Database schema design for families, memberships, and invitations tables

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| FAM-01 | User can create a family group | DB schema (families table), CreateFamily handler, CreateFamilyScreen UI, FamilyNotifier state management |
| FAM-02 | User can invite others to join their family | Invitations table, CreateInvite/AcceptInvite handlers, invite token generation, deep link routing, invitation acceptance UI |
</phase_requirements>

## Standard Stack

### Core (Already in Project)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Gin | 1.12.0 | HTTP handlers for family/invite endpoints | Already used for all server routes |
| pgx/v5 | 5.8.0 | PostgreSQL driver | Already used project-wide |
| sqlc | v2 config | Query codegen for family/invite queries | Already used for all DB operations |
| goose | (migrations) | Schema migrations for new tables | Already used for all migrations |
| golang-jwt/v5 | 5.3.1 | JWT for auth middleware (existing) | Already protecting routes |
| google/uuid | 1.6.0 | UUID generation for invite tokens | Already in go.mod |
| golang.org/x/crypto | 0.48.0 | SHA-256 hashing for invite tokens | Already used for refresh tokens |
| go_router | 14.0.0 | Flutter routing + deep link handling | Already used, supports path params |
| flutter_riverpod | 2.6.1 | State management for family state | Already used project-wide |
| dio | 5.4.0 | HTTP client for family API calls | Already used project-wide |

### No New Dependencies Required
This phase requires zero new packages. All functionality is achievable with the existing stack. Deep links are handled natively by go_router (path-based routing) and platform configuration (AndroidManifest.xml / Info.plist).

## Architecture Patterns

### Recommended Project Structure

**Server (new files):**
```
server/internal/
├── db/
│   ├── migrations/
│   │   └── 00005_families.sql          # families, family_members, family_invitations tables
│   └── queries/
│       ├── families.sql                 # family CRUD queries
│       └── invitations.sql              # invitation CRUD queries
├── handler/
│   ├── family.go                        # FamilyDB interface + FamilyHandler + routes
│   ├── family_db.go                     # PgFamilyDB implementation
│   └── family_test.go                   # Handler tests with mock FamilyDB
└── router/
    └── router.go                        # Add family/invite route groups (modify)
```

**Flutter (new files):**
```
app/lib/features/family/
├── data/
│   ├── models/
│   │   ├── family.dart                  # Family model (id, name, created_at)
│   │   ├── family_member.dart           # FamilyMember model (id, email, role)
│   │   └── family_invitation.dart       # Invitation model (id, token, expires_at, status)
│   └── family_repository.dart           # API calls to /families and /invites
├── domain/
│   └── family_state.dart                # Sealed class: FamilyInitial/Loading/Loaded/NoFamily/Error
└── presentation/
    ├── family_screen.dart               # Main family management screen
    ├── create_family_screen.dart         # Simple form with name field
    └── accept_invite_screen.dart         # Invitation acceptance flow
```

### Pattern 1: FamilyDB Interface (Mirrors AuthDB/CategoryDB)
**What:** Abstract DB operations behind an interface for testability
**When to use:** All new handler operations
**Example:**
```go
// FamilyDB abstracts database operations for family groups.
type FamilyDB interface {
    CreateFamily(name, adminUserID string) (MockFamily, error)
    GetFamilyByID(id string) (MockFamily, error)
    GetFamilyByUserID(userID string) (MockFamily, error)
    GetFamilyMembers(familyID string) ([]MockFamilyMember, error)
    RemoveFamilyMember(familyID, userID string) error
    DeleteFamily(familyID string) error
    CreateInvitation(familyID, inviterUserID, tokenHash string, expiresAt time.Time) (MockInvitation, error)
    GetInvitationByTokenHash(tokenHash string) (MockInvitation, error)
    AcceptInvitation(tokenHash, userID string) error
    RevokeInvitation(invitationID, familyID string) error
    GetPendingInvitations(familyID string) ([]MockInvitation, error)
    GetFamilyMemberCount(familyID string) (int, error)
}
```

### Pattern 2: Invitation Token Strategy (Mirrors Refresh Token Pattern)
**What:** Generate crypto-random tokens, store SHA-256 hash in DB, share raw token in link
**When to use:** Creating and accepting invitations
**Details:**
- Generate: `crypto/rand` 32-byte token, hex-encode -> 64 char string
- Store: SHA-256 hash of token in `family_invitations.token_hash`
- Share: Raw token in URL like `financetracker://invite/{raw_token}`
- Accept: Hash incoming token, look up by hash, validate expiry, add member
- This is the exact same pattern used for refresh tokens in Phase 2

### Pattern 3: FamilyNotifier StateNotifier (Mirrors CategoryNotifier)
**What:** Riverpod StateNotifier managing family state
**When to use:** All family state management in Flutter
**Example:**
```dart
sealed class FamilyState {
  const FamilyState();
}
class FamilyInitial extends FamilyState { const FamilyInitial(); }
class FamilyLoading extends FamilyState { const FamilyLoading(); }
class FamilyLoaded extends FamilyState {
  const FamilyLoaded(this.family, this.members, this.invitations);
  final Family family;
  final List<FamilyMember> members;
  final List<FamilyInvitation> invitations;
}
class NoFamily extends FamilyState { const NoFamily(); }
class FamilyError extends FamilyState {
  const FamilyError(this.message);
  final String message;
}
```

### Pattern 4: Deep Link via Custom URL Scheme
**What:** Register `financetracker://` custom URL scheme for invite links
**When to use:** Invitation sharing and acceptance
**Why custom scheme over universal links:** Simpler setup, no domain verification needed, sufficient for invite-only feature. Universal links require Apple/Google domain verification which adds deployment complexity for a v1 app.
**Implementation:**
- Android: Add `<intent-filter>` in `AndroidManifest.xml` for `financetracker://invite/{token}`
- iOS: Add URL scheme in `Info.plist`
- go_router: Add route `/invite/:token` that maps to `AcceptInviteScreen`
- The go_router redirect logic must allow `/invite/:token` for authenticated users (add to non-auth-redirect paths)

### Anti-Patterns to Avoid
- **Adding family_id FK directly to users table:** Use a separate `family_members` join table instead. Direct FK creates migration complexity and makes "one family at a time" constraint harder to enforce cleanly. A join table with UNIQUE(user_id) achieves the same thing more cleanly.
- **Storing raw invite tokens in DB:** Always hash tokens before storage (same as refresh tokens). Raw tokens should only exist in the shareable link.
- **Using ON DELETE CASCADE from families to expenses:** Expenses MUST stay with individual users per locked decisions. There should be no FK relationship between families and expenses.
- **Transferring admin role:** Per locked decisions, admin cannot be transferred. When admin leaves, group dissolves.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Invite token generation | Custom random string | `crypto/rand` + hex encoding | Cryptographic randomness required for security |
| Token hashing | Custom hash | `crypto/sha256` (existing pattern) | Already established in refresh token flow |
| Deep link routing | Manual URL parsing | go_router path parameters | go_router already handles path params and deep links |
| Share dialog | Custom share UI | Flutter `Share.share()` from `share_plus` or simple clipboard copy | Platform share sheet is standard UX... BUT `share_plus` adds a dependency. Recommend clipboard copy with a "Copy invite link" button instead to avoid new deps |
| UUID generation (Go) | Custom | `google/uuid` (already in go.mod) | Already used throughout |

**Key insight:** This phase introduces no novel technical challenges. Every pattern (token generation, hashed storage, DB interface, StateNotifier, route registration) already exists in the codebase. The work is applying established patterns to new domain objects.

## Common Pitfalls

### Pitfall 1: Race Condition on Invitation Acceptance
**What goes wrong:** Two users accept the same invite simultaneously, or a user joins when the family is at max capacity
**Why it happens:** Check-then-act without transaction isolation
**How to avoid:** Use a single SQL transaction for accept-invitation that: (1) locks the invitation row, (2) checks family member count < 10, (3) checks user is not already in a family, (4) inserts member, (5) marks invitation as accepted. Use `SELECT ... FOR UPDATE` on the invitation row.
**Warning signs:** Intermittent duplicate member entries or exceeding 10-member limit

### Pitfall 2: Orphaned Members When Admin Dissolves Group
**What goes wrong:** Admin deletes group but member rows or invitation rows are not cleaned up
**Why it happens:** Missing CASCADE or forgetting to handle all related records
**How to avoid:** Use `ON DELETE CASCADE` on `family_members.family_id` and `family_invitations.family_id` referencing `families.id`. When admin deletes the family, all members and invitations are automatically removed.
**Warning signs:** Foreign key constraint errors or orphaned records

### Pitfall 3: Deep Link Not Reaching App When Not Installed
**What goes wrong:** User clicks invite link but app is not installed, gets an error
**Why it happens:** Custom URL schemes fail silently when no app is registered
**How to avoid:** For v1, this is acceptable. Document that the app must be installed. The invite link format should be copyable so the user can manually navigate to an "Enter invite code" screen as fallback. Add a text field on the create-family/no-family screen for manual token entry.
**Warning signs:** User reports "nothing happens" when clicking link

### Pitfall 4: Stale Family State After Member Changes
**What goes wrong:** Admin removes a member but the removed member's app still shows the family
**Why it happens:** No push notification or polling mechanism
**How to avoid:** Reload family state on every family screen visit (pull-to-refresh or auto-load in initState/build). The FamilyNotifier should call `loadFamily()` when the family screen mounts. If the user has been removed, the API returns 404 and state transitions to `NoFamily`.
**Warning signs:** Removed member still sees old family data

### Pitfall 5: go_router Redirect Blocking Invite Route
**What goes wrong:** Authenticated user clicks invite deep link but gets redirected to /history
**Why it happens:** Current redirect logic sends all authenticated users on non-standard routes to /history
**How to avoid:** The `/invite/:token` route must be outside the ShellRoute (like expense/category routes) and the redirect logic must recognize it as a valid authenticated route. The current redirect only blocks unauthenticated users from non-auth routes, so this should work by default -- but verify.
**Warning signs:** Deep link always lands on history screen instead of invite acceptance

## Code Examples

### Migration: 00005_families.sql
```sql
-- +goose Up
CREATE TABLE families (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    admin_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE family_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id)  -- A user can only be in one family
);

CREATE INDEX idx_family_members_family_id ON family_members(family_id);
CREATE INDEX idx_family_members_user_id ON family_members(user_id);

CREATE TABLE family_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    inviter_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'revoked', 'expired')),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_family_invitations_token_hash ON family_invitations(token_hash);
CREATE INDEX idx_family_invitations_family_id ON family_invitations(family_id);

-- +goose Down
DROP TABLE IF EXISTS family_invitations;
DROP TABLE IF EXISTS family_members;
DROP TABLE IF EXISTS families;
```

### Key Design Notes on Schema:
- `UNIQUE(user_id)` on family_members enforces one-family-at-a-time constraint at DB level
- `admin_user_id` on families is denormalized for quick admin checks (also in family_members with role='admin')
- `ON DELETE CASCADE` from families to members and invitations handles group dissolution
- No FK from families to expenses -- expenses remain user-owned per locked decisions

### sqlc Query Examples
```sql
-- name: CreateFamily :one
INSERT INTO families (name, admin_user_id)
VALUES ($1, $2)
RETURNING id, name, admin_user_id, created_at, updated_at;

-- name: GetFamilyByUserID :one
SELECT f.id, f.name, f.admin_user_id, f.created_at, f.updated_at
FROM families f
JOIN family_members fm ON fm.family_id = f.id
WHERE fm.user_id = $1;

-- name: GetFamilyMembers :many
SELECT fm.id, fm.user_id, u.email, fm.role, fm.joined_at
FROM family_members fm
JOIN users u ON u.id = fm.user_id
WHERE fm.family_id = $1
ORDER BY fm.joined_at;

-- name: AddFamilyMember :one
INSERT INTO family_members (family_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING id, family_id, user_id, role, joined_at;

-- name: RemoveFamilyMember :execrows
DELETE FROM family_members
WHERE family_id = $1 AND user_id = $2 AND role != 'admin';

-- name: DeleteFamily :execrows
DELETE FROM families WHERE id = $1 AND admin_user_id = $2;

-- name: GetFamilyMemberCount :one
SELECT COUNT(*) FROM family_members WHERE family_id = $1;

-- name: CreateInvitation :one
INSERT INTO family_invitations (family_id, inviter_user_id, token_hash, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING id, family_id, inviter_user_id, token_hash, status, expires_at, created_at;

-- name: GetInvitationByTokenHash :one
SELECT fi.id, fi.family_id, fi.inviter_user_id, fi.token_hash, fi.status, fi.expires_at, fi.created_at,
       f.name as family_name
FROM family_invitations fi
JOIN families f ON f.id = fi.family_id
WHERE fi.token_hash = $1 AND fi.status = 'pending' AND fi.expires_at > NOW();

-- name: AcceptInvitation :execrows
UPDATE family_invitations SET status = 'accepted' WHERE id = $1 AND status = 'pending';

-- name: RevokeInvitation :execrows
UPDATE family_invitations SET status = 'revoked'
WHERE id = $1 AND family_id = $2 AND status = 'pending';

-- name: GetPendingInvitations :many
SELECT id, family_id, inviter_user_id, status, expires_at, created_at
FROM family_invitations
WHERE family_id = $1 AND status = 'pending' AND expires_at > NOW()
ORDER BY created_at DESC;
```

### Server API Endpoints
```
POST   /api/v1/families              -- Create family (name in body)
GET    /api/v1/families/me            -- Get current user's family + members + invitations
DELETE /api/v1/families/me            -- Delete family (admin only, dissolves group)
DELETE /api/v1/families/me/members/:userId -- Remove member (admin only)
POST   /api/v1/families/me/leave      -- Leave family (non-admin)
POST   /api/v1/families/me/invitations     -- Create invitation (returns raw token)
DELETE /api/v1/families/me/invitations/:id -- Revoke invitation (admin only)
POST   /api/v1/invitations/accept     -- Accept invitation (token in body)
GET    /api/v1/invitations/:token     -- Get invitation info (for accept screen preview)
```

### Flutter Route Registration
```dart
// Outside ShellRoute (own AppBar, no bottom nav)
GoRoute(
  path: '/settings/family',
  builder: (context, state) => const FamilyScreen(),
),
GoRoute(
  path: '/settings/family/create',
  builder: (context, state) => const CreateFamilyScreen(),
),
GoRoute(
  path: '/invite/:token',
  builder: (context, state) => AcceptInviteScreen(
    token: state.pathParameters['token']!,
  ),
),
```

### Settings Screen Integration
```dart
// In SettingsScreen ListView children, between Categories and Log Out:
ListTile(
  leading: const Icon(Icons.group),
  title: const Text('Family'),
  onTap: () => context.push('/settings/family'),
),
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Universal links for deep linking | Custom URL schemes still valid for app-to-app | Ongoing | Custom scheme is simpler for v1, universal links needed only for web fallback |
| Firebase Dynamic Links | Deprecated June 2025 | 2025 | Do NOT use; custom URL scheme or manual token entry is the right approach |

**Deprecated/outdated:**
- Firebase Dynamic Links: Deprecated. Do not use for invite links.
- `uni_links` Flutter package: Superseded by `app_links`. However, go_router handles deep link routing natively so neither is needed.

## Open Questions

1. **Clipboard vs Share Sheet for invite links**
   - What we know: `share_plus` package would add a new dependency; clipboard copy is zero-dep
   - What's unclear: User preference for share sheet vs copy button
   - Recommendation: Use `Clipboard.setData()` from Flutter services (zero dependencies) with a SnackBar confirmation "Invite link copied!" -- simplest approach, no new deps

2. **Manual token entry fallback**
   - What we know: Custom URL schemes fail when app is not installed; deep links may not work on all platforms (especially web)
   - What's unclear: Whether to add a text field for manual invite code paste
   - Recommendation: Add a "Have an invite code?" text button on the no-family empty state screen that opens a simple text field dialog. Low effort, high resilience.

3. **Invite link format**
   - Recommendation: `financetracker://invite/{hex_token}` where hex_token is 64 characters (32 bytes hex-encoded). Display to user as full URL. For manual entry, accept just the token portion.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | Go testing + httptest (built-in) |
| Framework (Flutter) | flutter_test + mocktail |
| Config file | None needed (standard Go/Flutter test setup) |
| Quick run (Go) | `cd server && go test ./internal/handler/ -run TestFamily -v` |
| Quick run (Flutter) | `cd app && flutter test test/features/family/` |
| Full suite (Go) | `cd server && go test ./...` |
| Full suite (Flutter) | `cd app && flutter test` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FAM-01 | Create family with name | unit (Go handler) | `cd server && go test ./internal/handler/ -run TestCreateFamily -v` | No -- Wave 0 |
| FAM-01 | Create family UI | widget (Flutter) | `cd app && flutter test test/features/family/presentation/family_screen_test.dart` | No -- Wave 0 |
| FAM-02 | Create invitation | unit (Go handler) | `cd server && go test ./internal/handler/ -run TestCreateInvitation -v` | No -- Wave 0 |
| FAM-02 | Accept invitation | unit (Go handler) | `cd server && go test ./internal/handler/ -run TestAcceptInvitation -v` | No -- Wave 0 |
| FAM-02 | Accept invite UI | widget (Flutter) | `cd app && flutter test test/features/family/presentation/accept_invite_screen_test.dart` | No -- Wave 0 |
| FAM-01 | Family member list display | widget (Flutter) | `cd app && flutter test test/features/family/presentation/family_screen_test.dart` | No -- Wave 0 |
| FAM-01 | Leave/dissolve family | unit (Go handler) | `cd server && go test ./internal/handler/ -run TestLeaveFamily -v` | No -- Wave 0 |
| FAM-02 | Revoke invitation | unit (Go handler) | `cd server && go test ./internal/handler/ -run TestRevokeInvitation -v` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** Quick run for affected layer (Go or Flutter)
- **Per wave merge:** Full suite for both Go and Flutter
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `server/internal/handler/family_test.go` -- covers FAM-01, FAM-02 (Go handler tests with mockFamilyDB)
- [ ] `app/test/features/family/presentation/family_screen_test.dart` -- covers FAM-01 (family management UI)
- [ ] `app/test/features/family/presentation/accept_invite_screen_test.dart` -- covers FAM-02 (invitation acceptance UI)

## Sources

### Primary (HIGH confidence)
- Project codebase: `server/internal/handler/auth.go`, `auth_db.go`, `category.go`, `category_db.go` -- established handler/DB interface patterns
- Project codebase: `server/internal/db/migrations/00002_users.sql`, `00004_expenses.sql` -- migration patterns (UUID PKs, TIMESTAMPTZ, CASCADE)
- Project codebase: `server/sqlc.yaml` -- sqlc v2 configuration with pgx/v5
- Project codebase: `app/lib/features/categories/` -- complete Flutter feature module pattern (data/domain/presentation)
- Project codebase: `app/lib/core/router/app_router.dart` -- go_router configuration with auth redirect and nested routes
- Project codebase: `server/internal/router/router.go` -- Gin router setup with protected route groups

### Secondary (MEDIUM confidence)
- go_router deep linking: go_router natively supports path parameters which handles `/invite/:token` routing
- Custom URL scheme: Standard Android/iOS mechanism, well-documented in Flutter docs

### Tertiary (LOW confidence)
- None -- all patterns are already proven in this codebase

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- zero new dependencies, all patterns already established
- Architecture: HIGH -- direct replication of existing handler/DB/StateNotifier patterns
- Pitfalls: HIGH -- race conditions and cascade behavior are well-understood SQL problems
- Deep linking: MEDIUM -- custom URL scheme is straightforward but platform config varies

**Research date:** 2026-03-16
**Valid until:** 2026-04-16 (stable -- no fast-moving dependencies)
