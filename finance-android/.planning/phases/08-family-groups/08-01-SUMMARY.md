---
phase: 08-family-groups
plan: 01
subsystem: api
tags: [go, gin, sqlc, family-groups, invitations, sha256]

# Dependency graph
requires:
  - phase: 02-authentication
    provides: "AuthDB pattern, JWT middleware, user_id context, SHA-256 token hashing"
  - phase: 01-foundation
    provides: "sqlc/goose workflow, pgx/v5 pool, UUID helpers"
provides:
  - "families, family_members, family_invitations DB tables"
  - "FamilyDB interface with 12 methods"
  - "FamilyHandler with 9 REST endpoints"
  - "PgFamilyDB adapter for sqlc queries"
  - "Family route wiring in router.go"
affects: [08-family-groups]

# Tech tracking
tech-stack:
  added: []
  patterns: [FamilyDB interface pattern mirroring AuthDB/CategoryDB, crypto/rand token generation with SHA-256 hashing]

key-files:
  created:
    - server/internal/db/migrations/00005_families.sql
    - server/internal/db/queries/families.sql
    - server/internal/db/queries/invitations.sql
    - server/internal/handler/family.go
    - server/internal/handler/family_db.go
    - server/internal/handler/family_test.go
  modified:
    - server/internal/router/router.go
    - server/cmd/api/main.go

key-decisions:
  - "FamilyDB interface with MockFamily/MockFamilyMember/MockInvitation types mirroring AuthDB pattern"
  - "crypto/rand 32-byte token with SHA-256 hash storage for invitations (same as refresh token pattern)"
  - "7-day invitation expiry with server-side enforcement"
  - "UNIQUE(user_id) on family_members enforces one-family-per-user at DB level"

patterns-established:
  - "FamilyDB interface: 12 methods covering families, members, invitations -- same testability pattern as AuthDB/CategoryDB"
  - "Invitation token flow: crypto/rand -> hex encode -> SHA-256 hash -> store hash, return raw"

requirements-completed: [FAM-01, FAM-02]

# Metrics
duration: 6min
completed: 2026-03-16
---

# Phase 8 Plan 1: Family Groups Backend Summary

**Go backend for family groups with 3 DB tables, sqlc queries, FamilyHandler with 9 endpoints (CRUD + invitations), and 16 test cases**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-16T09:58:10Z
- **Completed:** 2026-03-16T10:04:28Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments
- Migration with families, family_members (UNIQUE user_id), family_invitations tables with proper FK cascades and indexes
- FamilyHandler with 9 endpoints: create/get/delete family, leave, remove member, create/revoke invitation, get/accept invitation
- 16 handler tests covering success paths and error cases (409 conflict, 403 forbidden, 404 not found, 400 bad request)
- SHA-256 invitation token hashing mirroring refresh token pattern

## Task Commits

Each task was committed atomically:

1. **Task 1: Migration, sqlc queries, and generated code** - `27787fb` (feat)
2. **Task 2: FamilyHandler, PgFamilyDB, route wiring, and tests** - `085839b` (feat)

## Files Created/Modified
- `server/internal/db/migrations/00005_families.sql` - Migration: families, family_members, family_invitations tables
- `server/internal/db/queries/families.sql` - 7 sqlc queries for family CRUD and member management
- `server/internal/db/queries/invitations.sql` - 5 sqlc queries for invitation lifecycle
- `server/internal/db/sqlc/families.sql.go` - Generated Go code for family queries
- `server/internal/db/sqlc/invitations.sql.go` - Generated Go code for invitation queries
- `server/internal/handler/family.go` - FamilyDB interface and FamilyHandler with 9 endpoints
- `server/internal/handler/family_db.go` - PgFamilyDB implementation delegating to sqlc
- `server/internal/handler/family_test.go` - mockFamilyDB and 16 test cases
- `server/internal/router/router.go` - Route wiring for families and invitations groups
- `server/cmd/api/main.go` - PgFamilyDB initialization and router parameter

## Decisions Made
- FamilyDB interface with MockFamily/MockFamilyMember/MockInvitation types mirroring AuthDB pattern for testability
- crypto/rand 32-byte token with SHA-256 hash storage for invitations (same as refresh token pattern)
- 7-day invitation expiry with server-side enforcement via SQL WHERE expires_at > NOW()
- UNIQUE(user_id) on family_members enforces one-family-per-user at DB level
- Admin checks via family.AdminUserID comparison (not separate role lookup)
- Max 10 members enforced server-side before invitation creation and acceptance

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated router.Setup signature and main.go call site**
- **Found during:** Task 2 (route wiring)
- **Issue:** Adding familyDB parameter to router.Setup required updating the call site in main.go
- **Fix:** Added familyDB parameter to Setup function and updated main.go to instantiate PgFamilyDB
- **Files modified:** server/internal/router/router.go, server/cmd/api/main.go
- **Verification:** go build ./... passes
- **Committed in:** 085839b (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary for compilation. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Family backend API complete, ready for Flutter UI integration (Plan 2)
- All 9 endpoints registered and tested
- FamilyDB interface ready for use in Flutter data layer

---
*Phase: 08-family-groups*
*Completed: 2026-03-16*
