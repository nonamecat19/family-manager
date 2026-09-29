# Escalations

Decisions an autonomous run could not make. Written by `/autopilot`, answered by a human,
applied by `/escalations`. Resolved entries stay — they are the record of what was authorized.

Answer inline under the entry, then run `/escalations` to apply and resume.

---

## E1 — Public RedeemLinkToken lets anyone squat a Telegram id (open, blocker, predates brief 0001)

`services/auth/cmd/server/main.go:174` serves `RedeemLinkToken` on the public listener and Caddy
proxies every path on `AUTH_HOST`; `external_id` comes from the caller (`link.go:96-103`). Anyone
can Register → Login → CreateLinkToken → RedeemLinkToken with a victim's Telegram id; the victim's
own link later fails `AlreadyExists` (`identity.go:31`) and they are shut out of linking, the bots
and Telegram login.

Proposed fix: serve `RedeemLinkToken` only on an internal listener Caddy does not proxy (the
`family:9090` pattern) and have `services/telegram` call it there.

Decision (2026-10-04): ship brief 0001 now, fix in a follow-up.

## E2 — Device-login limiter memory and shared-network budget (open, risk, from brief 0001)

- `services/auth/internal/handler/devicelogin.go:58-61` — the per-client entry is created before
  the network limit refuses, and old entries are swept once per window: ~1M distinct /64s across
  a few /48s cost ~107 MiB against a 140 MiB GOMEMLIMIT. Check the network window first and
  hard-cap entries per map.
- `devicelogin.go:60` — the /24 network budget applies always, so one caller on a CGNAT or cloud
  /24 can deny device and Telegram login to that whole /24. Apply it only under pressure.
- `devicelogin.go:379` — the network window count is a loose proxy for pending grants; pin the
  window to the grant TTL or count pending grants per network in SQL.
- `login_grants.sql:53` — drop the unreachable `approver_chain_id IS NULL` consume branch.

Decision (2026-10-04): ship brief 0001 now, fix in a follow-up.

## E3 — 0003/p1: scaffold apps/tasks (open, gate: workspace dependency change)

Unit p1 creates `apps/tasks` and adds its native dependencies
(`@react-native-community/datetimepicker`, `expo-auth-session`, `expo-web-browser`, `expo-crypto`,
`expo-notifications`) in one change to `pnpm-lock.yaml`, which gate-check treats as a
workspace-wide dependency change (STOP). Every app unit (p2-p9) depends on it.

To unblock: approve here ("p1: approved") and run `/escalations`; the run then scaffolds the app
from `apps/finance` as described in the unit notes. Or scaffold it yourself and mark p1 done.

## E4 — 0003/h1: Google OAuth client for the tasks service (open, credential)

Create an OAuth client in Google Cloud (type: Web application; redirect URI: the tasks app's
custom scheme `fmtasks:/oauthredirect` and, for production, `https://tasks.<domain>/oauth/google`),
enable the Google Calendar API, then put the values in the repo-root `.env` (never committed):

    TASKS_GOOGLE_CLIENT_ID=...
    TASKS_GOOGLE_CLIENT_SECRET=...

Nothing depends on this unit: every test uses a fake Google API. Without it the service starts
and "Connect Google Calendar" answers FailedPrecondition.

**Decision needed — OAuth client type.** A Google "Web application" client rejects custom-scheme
redirect URIs (`fmtasks:/oauthredirect`), and new Android clients have custom schemes disabled, so
the brief's default will fail at Google's authorize step. Options:
1. Android client for the app + Web client for the server: the app signs in with Google
   (`serverAuthCode`) and the service exchanges that code with the Web client id/secret
   (redirect URI empty). Needs the app's SHA-1 signing fingerprint in Google Cloud.
2. Web client with an HTTPS redirect on the tasks host (`https://tasks.<domain>/oauth/google`)
   that hands the code back to the app via a deep link. Needs the public host (unit h3).
The service supports both (it sends `client_secret` only when configured, and checks that the
calendar and email scopes were actually granted). The app must request
`access_type=offline&prompt=consent` with scopes `openid email https://www.googleapis.com/auth/calendar`.

## E5 — 0003/h2: BotFather token for the tasks bot (open, credential)

Create the bot with @BotFather and add `TELEGRAM_TASKS_TOKEN=<id>:<secret>` to the repo-root
`.env` (never committed). Nothing depends on it: the e2e suite uses the tgemu fake token.

## E6 — 0003/u25: register services/tasks in go.work (open, gate: workspace change)

`services/tasks` builds and tests with `GOWORK=off` (its go.mod carries the same replace block as
finance). Adding `use ./services/tasks` to `go.work` makes `just check-go`, `just lint-go` and the
knowledge graph pick it up; gate-check treats any go.work edit as a workspace-wide change.

To unblock: approve here ("u25: approved") and run `/escalations`, or add the line yourself and
run `go work sync`. Units u06 (Dockerfile) and the final g1 verify depend on it.

## E7 — 0003/u07: tasks schema ships without a down migration (open, informational)

gate-check stops any `DROP`, including a `.down.sql` that only drops the tables the same unit
creates. The migration runner never executes down files (`libs/go/database/migrate.go`
`LoadMigrations` skips them), so u07 ships `000001_init.up.sql` alone; rolling back a fresh
`tasks` database is `DROP DATABASE tasks`. If you want down files kept for parity with finance,
answer "u07: add down" and the drop list will be added as a human-approved commit.

## E8 — 0003/u12, u13: `master` does not build, and handler acceptance commands cannot pass (open, stop-the-line for services/tasks)

**What is broken.** Commit `344d108` (u12, recorded in the backlog as commit `HEAD`) adds
`services/tasks/internal/handler/birthdays.go`, which is in u13's file list, and that file does not
compile (`birthdays.go:111:63: undefined: loc` plus unused imports). So
`cd services/tasks && GOWORK=off go test ./internal/handler/...` fails on `master`. u12 was marked
done while its acceptance command failed.

**Why the acceptance commands cannot pass.** u12, u13, u14 and u17 all use
`go test -race ./internal/handler/... ./internal/grpc/...`. `services/tasks/internal/grpc` does not
exist, so the command always ends `FAIL ./internal/grpc/... [setup failed]`. AGENTS.md lists an
`internal/grpc` layer, but no service in the repo has one: finance, recipes and family register
their `internal/handler` types directly with `New<X>ServiceHandler`, and tasks does the same
(`cmd/server/main.go`).

**u13 is done, but not committed.** The rewritten `birthdays.go` and a new `birthdays_test.go`
(scoping, validation, 29 February, timezone, reminder fan-out, known-members refresh) pass
`GOWORK=off go test -race ./internal/handler/...`. They are saved in
`git stash` as "autopilot 0003/u13: birthday handlers". Fixes in that work:
- `ListMembers` got the caller's user id as the bearer token. It now gets the request's token.
- The early and day-of birthday reminders shared one `occurrence`, so the UNIQUE key folded them
  into one row. Day-of is now `YYYY-MM-DD` and the early one `YYYY-MM-DD-<N>d`.
- Age comes from `Occurrence.HasAge`, and the next occurrence uses the family's timezone.

**u12 does not meet its own notes** (read in passing, not fixed because u12 is marked done):
`CreateTask` never sets `due_at`, so no task reminders are ever created; assignees are not checked
for membership; there is no `ListMembers` refresh; `tasks.task.assigned` is published inside the
transaction (that belongs to u15); and `touchKnownMember` overwrites `display_name` with the email.

**Decision needed:**
1. *(recommended)* Change the acceptance for u12, u13, u14 and u17 to `./internal/handler/...` only,
   which matches every existing service, and reopen u12 so it meets its notes. Then
   `git stash pop` and commit u13.
2. Keep `internal/grpc` as AGENTS.md describes: reopen u12 to add `internal/grpc/server.go` as a
   Connect adapter, and the later units add their own `grpc/*.go`.

Until this is answered, nothing under `services/tasks/internal/handler` can pass acceptance:
u13, u14, u15, u17, u20, u22 and u24 are stuck.

## E9 — 0003/u21: calendar push failed verification twice (open, stop-the-line)

The run stopped here because two verifiers each found a blocker in u21. The work is saved in
`git stash` as "autopilot 0003/u21: calsync push"
(`services/tasks/internal/calsync/{mapping,push,push_test}.go`, `go test -race` green).

**Review 1 blocker (fixed in the stash):** when a member switched calendars, the old event was
left live with no link, and on a shared calendar a second copy appeared. The stash also contains
fixes for review 1's other findings: an event is deleted when its link cannot be saved, pushes
for one item hold a per-item mutex, timezone errors are no longer hidden, and a lost scope asks
the user to reconnect.

**Review 2 blocker (open):** the fix deletes the old calendar's event before inserting into the
new calendar. If the old calendar has lost write access (`ErrForbidden`), that delete never
succeeds, so every item linked there stops syncing for good. Review 2's other open findings:
- The shared-calendar switch depends on member order (`ORDER BY user_id`) and can leave the shared
  calendar with no event.
- The per-item lock is taken after the item is read, so a stale push can re-create an event for a
  deleted task.
- The delete that undoes an insert reuses a possibly cancelled `ctx`.
- `SweepOrphans` runs without the item lock.

**This needs a design decision:**
1. **u20 and u21 contradict each other.** u20's notes say `SetGoogleCalendar` deletes the user's
   old `calendar_links`, which orphans the old events before calsync ever sees the switch. Pick one
   owner for the calendar switch. Recommended: u20 calls a calsync `SwitchCalendar` that deletes the
   old events, treating `ErrForbidden` and `ErrNotFound` as gone, then drops the links. The push path
   then never has to handle a link whose calendar differs from the connection's.
2. **Duplicate inserts after a timed-out insert.** A request that times out after Google created
   the event can only be made safe with event IDs we choose ourselves (a `gcal` change, u18) or a
   database-level guard such as a pg advisory lock, or an insert-only link that loses the race and
   deletes its own event. Also confirm the push runs in a single tasks replica.
3. **Retrying a delete after the deadline is removed.** If deleting the event fails after a deadline
   is removed, the event stays live. Only a sweep that also matches tasks with `due_on IS NULL`
   (a `google.sql` change) can clean it up.

Once you decide, `git stash pop` the u21 entry and run `/autopilot` again. u22 and u23 wait on
u21.
