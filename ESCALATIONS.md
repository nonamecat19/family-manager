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

## E3 — 0003/p1: scaffold apps/tasks (resolved)

Unit p1 creates `apps/tasks` and adds its native dependencies
(`@react-native-community/datetimepicker`, `expo-auth-session`, `expo-web-browser`, `expo-crypto`,
`expo-notifications`) in one change to `pnpm-lock.yaml`, which gate-check treats as a
workspace-wide dependency change (STOP). Every app unit (p2-p9) depends on it.

To unblock: approve here ("p1: approved") and run `/escalations`; the run then scaffolds the app
from `apps/finance` as described in the unit notes. Or scaffold it yourself and mark p1 done.

RESOLVED: 2026-10-06 — approved by the human ("use recommended options"). p1 unblocked with gate auto. The STOP on `pnpm-lock.yaml` is authorized for p1 only. It also adds `@react-native-google-signin/google-signin` (E4).

## E4 — 0003/h1: Google OAuth client for the tasks service (decided; credential still open)

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

DECIDED: 2026-10-06 — option 1, Android client for the app plus a Web client for the server. No option was marked recommended; the run chose 1 because it needs no public host and Android is the verified target. Recorded in the notes of u20, p7 and h1. Still open: the human creates both clients and fills `.env` (h1).

## E5 — 0003/h2: BotFather token for the tasks bot (open, credential)

Create the bot with @BotFather and add `TELEGRAM_TASKS_TOKEN=<id>:<secret>` to the repo-root
`.env` (never committed). Nothing depends on it: the e2e suite uses the tgemu fake token.

## E6 — 0003/u25: register services/tasks in go.work (resolved)

`services/tasks` builds and tests with `GOWORK=off` (its go.mod carries the same replace block as
finance). Adding `use ./services/tasks` to `go.work` makes `just check-go`, `just lint-go` and the
knowledge graph pick it up; gate-check treats any go.work edit as a workspace-wide change.

To unblock: approve here ("u25: approved") and run `/escalations`, or add the line yourself and
run `go work sync`. Units u06 (Dockerfile) and the final g1 verify depend on it.

RESOLVED: 2026-10-06 — approved by the human ("use recommended options"). u25 unblocked with gate auto. The STOP on `go.work` / `go.work.sum` is authorized for u25 only.

## E7 — 0003/u07: tasks schema ships without a down migration (resolved)

gate-check stops any `DROP`, including a `.down.sql` that only drops the tables the same unit
creates. The migration runner never executes down files (`libs/go/database/migrate.go`
`LoadMigrations` skips them), so u07 ships `000001_init.up.sql` alone; rolling back a fresh
`tasks` database is `DROP DATABASE tasks`. If you want down files kept for parity with finance,
answer "u07: add down" and the drop list will be added as a human-approved commit.

RESOLVED: 2026-10-06 — no answer requested a down file, so u07 stays as shipped (`000001_init.up.sql` only).

## E8 — 0003/u12, u13: `master` does not build, and handler acceptance commands cannot pass (resolved)

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

RESOLVED: 2026-10-06 — option 1 (recommended). `./internal/grpc/...` is removed from the acceptance of u12, u13, u14, u17 and u20, and `internal/grpc/*` from their file lists. u12's recorded commit is corrected to `344d108`. The u12 shortfalls go to a new unit, u12b (depends on u13), and u14, u15, u17, u20, u22 and u24 now depend on u12b. u13, u14, u15, u17 and u20 are unblocked. u13 starts from its stash.

## E9 — 0003/u21: calendar push failed verification twice (resolved)

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

RESOLVED: 2026-10-06 — recommendations applied. (1) calsync owns the calendar switch through `Pusher.SwitchCalendar` (ErrForbidden and ErrNotFound count as gone), u20 calls it, and u20 now depends on u21. (2) Inserts use deterministic client-supplied event IDs from a new unit, u18b, which u21 depends on, and ErrConflict leads to an update. The item lock is taken before the item is read, and the sweep locks per link. A single tasks replica is recorded in u24. (3) `ListOrphanCalendarLinks` also matches tasks without a deadline (`google.sql` added to u21's files). u21 is unblocked with its attempts reset, and starts from its stash.


## E10 — 0004 (home-screen widgets): human gates (open)

The run continues with the units that are not gated (shared kit reads/writes, strings, justfile
recipes, ADR). These seven units wait for you:

- **u01: scaffold `packages/widgets` and add the native dependencies.** The packages are
  react-native-android-widget, @bacons/apple-targets, expo-background-task and expo-task-manager,
  for finance, notes and recipes. This is one `pnpm-lock.yaml` change (STOP). Answer "u01:
  approved". Everything in the widget kit (u02-u06) and every app's renderers wait on it.
- **u08: let the iOS widget extension read the session.** This moves the session in `@fm/auth`
  into an App Group keychain access group. It is security-sensitive, so it gets your review.
  Widgets never refresh tokens: after the 15-minute access token expires, iOS widget actions say
  "Open app to sign in". Answer "u08: approved" or name changes. f07, n07 and t09 (the iOS App
  Intents) wait on it.
- **r07: the recipes basket has no server state.** It lives in `useState` in
  `apps/recipes/components/basket.tsx`, and `recipes.v1` has no basket rpc. Pick one:
  - (a) *(recommended)* store the basket on the device (AsyncStorage plus the widget snapshot) and
    let the widget tick items on the device;
  - (b) a server-side basket, which needs a new brief;
  - (c) drop the basket widget and keep only "today's meal plan".
- **t00 and t01: tasks widgets.** They wait until brief 0003 has `apps/tasks` (p1-p3, a5).
  Mark t00 done afterwards. t01 is a second `pnpm-lock.yaml` change, unless the widget dependencies
  are folded into 0003/p1.
- **c01: a CI macOS job that compiles the Swift widget targets** (`.github/workflows/verify.yml`,
  STOP). Locally, `check-ios-widgets` only proves that prebuild works.
- **h01: Apple Developer team id and App Group ids.** These are credentials, needed only for
  signed iOS builds.

Not blocked, but assumed: neither platform lets a widget take typed input. So finance "quick add"
logs a template from the widget, and a custom amount opens the app's add screen.

## E11 — 0005 (bank notification capture): human gates (open)

- **gw: `go.work`.** Registers `services/capture` (STOP). Until then, every capture unit tests
  with `GOWORK=off`. Answer "gw: approved".
- **hk: `CAPTURE_SEAL_KEY`.** Add a base64 32-byte key to the repo-root `.env`, which is never
  committed: `openssl rand -base64 32`.
- **rt: local runtime wiring.** Compose service on host :8089, the capture DB in postgres init,
  and `.env.example` (STOP, deploy surface). Run it after 0003's u26, which edits the same files.
- **hr: review data at rest before the first deploy.** Read ADR 0018 (written by unit d1) and
  accept it: AES-GCM sealing of the raw text, owner-only visibility, raw text purged 90 days after
  a confirm or dismiss.
- **hp: prod deploy surface.** Caddy host `capture.<domain>` and the prod compose (`infra/**`).
- **hr2: real bank samples.** The parser fixtures are synthesised from known formats. Replace them
  with real notifications (redact the card digits) and confirm the package ids for PUMB, Sense,
  A-Bank and Raiffeisen.

## E12 — `just lint-go` panics on go1.27 everywhere (open, informational, predates these runs)

golangci-lint 2.12.2 with staticcheck v0.7.0 panics on go1.27
(`fact_purity: package "poll" ... not *buildir.IR`, rc=3). It panics on a clean export of HEAD as
well, so no unit caused it. `just verify` does not run `lint-go`, so the runs are not blocked. With
`--disable staticcheck,unused`, every service lints clean except tasks: its findings are queued as
0003/u27 (gcal) and u12b (handler gofmt).

To fix: upgrade golangci-lint to a release built against go1.27, wherever CI and local installs
pin it. That is a toolchain change outside any unit.

## E13 — 0005/d1: ADR 0018 (capture data at rest) failed verification twice (open, stop-the-line)

The run stopped because two verifiers each found a blocker in d1. Units that were already
running finish their own verification; no new unit starts. The ADR draft is saved in `git stash`
as "autopilot 0005/d1: ADR 0018 attempt 2". After review 1, the plan gained the deletion
tombstone (s2, s3, h2, h4), a purge of unparsed notifications (s3, h5), a phone queue encrypted
with an Android Keystore key (n1), and a no-logging rule in the service (h2). Those notes are
committed.

**Decisions needed (recommendations first):**
1. **Members can't clear learned prefill rules.** `prefill_rules` keep merchant and counterparty
   names with no expiry, and no rpc removes them. *Recommended:* add an additive contract unit
   (`ListPrefillRules` / `DeletePrefillRule` / `ClearPrefillRules`), a handler unit and a section
   in the capture settings screen. Alternative: never learn from transfers (counterparty names)
   and disclose "kept until the member disables capture".
2. **No-logging rule on the device.** *Recommended:* add it to n1, n2, a4 and f2, plus a JS test
   for a4/f2 that no console output contains the fixture text. This is mechanical; say "ok".
3. **A member leaves or is removed from the family.** Their capture rows stay under the old
   family id, out of reach, with clear fields kept forever. *Recommended:* add a consumer of
   `family.member.removed` that deletes all of that member's capture rows. This is the same
   pattern 0003/u24 uses.
4. **The purge rules contradict each other.** h5 says pending suggestions are never purged; the
   ADR says pending text is purged after 365 days. *Recommended:* the ADR rule — 365 days for
   pending, and 90 days after receipt for unparsed — with tests for both.
5. **Backups.** The database dumps hold every clear field (merchant and counterparty names,
   balances, card hints) unencrypted. `CAPTURE_SEAL_KEY` will sit in both `.env` and
   `infra/secrets`, and both are archived next to the dumps. *Decide:* accept this and disclose
   it in the ADR, or encrypt backups / keep the key out of the archive (an infra change, a human
   gate).
6. **Minor (recommended yes):**
   - bind the row id and field name into the AES-GCM additional data (s4);
   - list `allowed_packages`, `parser_id` and the tombstone key as stored in clear and kept
     indefinitely;
   - prune tombstones after a year.

Answer each point (or "use recommended"), then `/escalations`. The d1 stash is reapplied and
the ADR rewritten against the updated plan.
