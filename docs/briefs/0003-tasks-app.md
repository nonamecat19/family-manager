# 0003 — Tasks app: family tasks, birthdays, Google Calendar and a Telegram bot

## Goal
A family can keep a shared task list and a shared birthday book. A task has a title, notes, a
priority (low / medium / high / urgent), an optional deadline (a date, or a date and time in the
family's timezone), a status (open / done) and zero or more assignees chosen from the family's
members. A birthday is a family-shared contact — member or not — with a name, day and month, an
optional birth year (to show the age) and a "remind me N days before" setting. Each member can
connect their Google account once; from then on the tasks with a deadline and the birthdays appear
as events in a calendar they pick, and edits or deletions made in Google come back into the app
(Notion Calendar shows the same Google calendar). A new Telegram "tasks" bot reminds assignees
before a deadline and on or before a birthday with Done / Snooze buttons, lets a member manage
tasks in chat (/add, /today, /mine, /done), and sends a morning digest of today's tasks, overdue
tasks and upcoming birthdays. All of it is reachable from a new Expo app, `apps/tasks`.

## Non-goals
- No recurring tasks, subtasks, checklists, attachments, comments or task history.
- No assignees outside the family; birthdays are the only place non-members appear.
- No calendar provider other than Google (no iCloud, Outlook, CalDAV, ICS feed, Notion API).
- No Google push notifications (watch channels need a public HTTPS endpoint): Google → app sync
  is incremental polling with sync tokens.
- No web or desktop build of the tasks app; Android is the verified target (iOS must compile).
- No change to auth, to the family contract, or to any other app's screens.
- No production deploy, DNS, BotFather or Google Cloud console work — those are human steps.

## Acceptance
- [ ] `libs/proto/tasks/v1/tasks.proto` exists and `just proto-check` passes (lint + no breaking change).
- [ ] `services/tasks` builds, is registered in `go.work`, and `just check-go` passes with unit
      tests for: family scoping (a member of family A cannot read or change family B's tasks or
      birthdays), assignees restricted to the caller's family members, deadline timezone handling,
      birthday next-occurrence and age (including 29 February), and reminder scheduling.
- [ ] Migrations for tasks, assignees, birthdays, Google connections and calendar event links are
      additive and apply to the local compose database.
- [ ] Google Calendar sync: with a fake Google API in tests, creating, editing, completing and
      deleting a task with a deadline (and creating / editing / deleting a birthday) creates,
      updates and deletes the matching event; an event edited or deleted in Google is pulled back
      on the next incremental sync. OAuth client id / secret are read from config only; with them
      unset the service starts and the connect RPC returns FailedPrecondition with a clear message.
- [ ] `services/telegram` hosts a `tasks` bot (TELEGRAM_TASKS_TOKEN, TELEGRAM_TASKS_ADDR):
      /add, /today, /mine, /done work, reminders and the daily digest are sent at the right time,
      and Done / Snooze buttons work. `just test-telegram` covers the tasks bot through tgemu and
      reports no non-200 Bot API calls.
- [ ] `services/notifications` knows the new topics (task assigned, task due, birthday) with
      human labels in the apps, and pushes them.
- [ ] `apps/tasks` exists (scaffolded like the other apps), passes `just check-ts`, builds a release
      APK with `EXPO_PUBLIC_API_ENV=emulator`, and has screens for: task list (filters: mine /
      all / done, sorted by deadline then priority), create / edit task (priority, deadline date
      and optional time, multi-select assignees), birthdays list and editor, settings (language,
      notifications, Google Calendar connect / disconnect / pick calendar, connected accounts).
- [ ] `.maestro/flows/tasks/` covers register → create family → create a task with two
      assignees and a deadline → see it under "mine" → complete it → add a birthday → see it in
      the list with the age; `just test-android tasks` passes on the emulator.
- [ ] `just verify` passes and the knowledge graph shows `service:tasks`, `app:tasks` and the
      `tasks.v1` contract with their edges.

## Affected nodes
New nodes (do not exist yet): `service:tasks`, `app:tasks`, `proto:libs/proto/tasks/v1/tasks.proto`
with its rpcs and messages, and the tasks tables.

Existing nodes touched, with blast radius from `just impact`:
- `service:telegram` (5 nodes: its migrations and tables `bot_offsets`, `chat_states`,
  `telegram_links`) — a new bot, new config, a reminder / digest scheduler.
- `service:notifications` (10 nodes: its rpcs, tables, migration, the notifications proto) — new
  topics in the catalog and new event subscriptions.
- `service:family` (22 nodes) — read-only use of `ListMembers` / membership over gRPC; no edit.
- `pkg:@fm/api` (3 nodes: `app:finance`, `app:notes`, `app:recipes`) — new tasks query hooks.
- `pkg:@fm/ui` (3 nodes: the three apps) — only if a shared control is missing (date/time picker,
  multi-select); otherwise untouched.
- `sdk/typescript`, `sdk/go` — regenerated from the new proto.
- `go.work`, `docker-compose.services.yml`, `docker-compose.tgemu.yml`, `justfile`.

## Known gates
- `docker-compose.services.yml` — adding the `tasks` service and the tasks bot token/addr to the
  telegram service (STOP: compose is deploy surface).
- `infra/**` — Caddyfile host for `tasks.<domain>`, `infra/docker-compose.prod.yml`, the
  `.env.example` entries (STOP: deploy surface).
- Credentials: the Google OAuth client id / secret and the BotFather token for the tasks bot are
  provisioned by a human (STOP: credential values). Code reads them from config; nothing commits a
  value.
- `docker-compose.tgemu.yml` gets `TELEGRAM_TASKS_TOKEN` with a fake value — compose surface, the
  same gate as above.
- Migrations are additive and new (AUTO, local DB only).

## Open questions
- Which timezone drives "today", the digest hour and reminder times: the family's (finance stores
  one per household) or each member's? Default if unanswered: a per-family timezone stored by the
  tasks service, initialised from the device on first use, with a fixed digest time of 08:00.
- Google OAuth on Android: the app gets an authorization code with PKCE and the service exchanges
  it, holding the refresh token encrypted at rest. Is a "Web application" or "Android" client
  type expected in the human's Google Cloud project? Default: one Web client with the service's
  redirect URI plus the app's custom scheme.
- Should completing a task also delete its Google event, or keep it marked as done ("✓ " title
  prefix)? Default: keep it, prefix the title.
- Snooze duration choices for the Telegram buttons. Default: 1 hour and tomorrow 09:00.
