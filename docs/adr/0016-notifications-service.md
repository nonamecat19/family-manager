# ADR 0016 — `services/notifications` turns domain events into Expo pushes

- Status: accepted
- Date: 2026-10-04
- Builds on: [0003](0003-nats-jetstream.md) (events), [0005](0005-auth.md) (auth),
  [0008](0008-rpc-interceptors.md) (interceptors)

## Context

[ADR 0003](0003-nats-jetstream.md) was written with a notifier in mind — "`notifications` must
react to things that happen in `family`, `shopping` and `recipes` without those services knowing
it exists" — but the only notifier was the Flutter `notifications-android` app, now retired.
Finance, recipes and family publish to JetStream and nothing tells a human. The three Expo apps
(notes, finance, recipes) each have their own Expo push token per device and nowhere to send it.

## Decision

A Go service, `services/notifications`, owning `notifications.v1` and its own database.

- **Contract.** `NotificationsService` with four authenticated rpcs:
  `RegisterPushToken(token, platform, app, device_id)`, `UnregisterPushToken(token)`,
  `GetPreferences()` → `muted` + the `topics` catalogue, `SetPreferences(muted)`. A token is
  owned by the caller; registering a token another user held moves it (the device changed
  account). Only Expo tokens (`ExponentPushToken[…]`, `ExpoPushToken[…]`) are accepted.
- **Preferences are a mute list, not a matrix.** Each entry is a domain (`finance`) or a topic
  key (`finance.budget.exceeded`); the default is everything on. `SetPreferences` replaces the
  list. The catalogue is served by `GetPreferences` so an app renders toggles without hardcoding
  them.
- **Consumer.** One durable JetStream consumer per subject, named
  `notifications-<domain>-<entity>-<verb>`, through `libs/go/events`. Today:
  `family.member.joined` (tell the other members), `family.member.removed` (tell the removed
  member, only when an admin removed them), `finance.budget.exceeded` (every member),
  `recipes.recipe.created` (every member but the author). Events nobody should be woken for —
  transaction edits, account changes, recipe updates — are deliberately not consumed.
- **Idempotency without an event id on the wire.** Event payloads carry no id, so the service
  keys `processed_events` on `sha256(subject ∥ payload)`; every payload carries `occurred_at`,
  so two real events never collide. An event is marked processed only after delivery succeeds.
- **Audience.** Tokens carry the `family_id` from the caller's token at registration; joined and
  removed events keep it current. Before a family-wide send, each candidate is re-checked with
  the family service's internal `GetUserMembership` (gRPC on `family:9090`, via `sdk/go`) and a
  stale `family_id` is corrected. If family is unreachable the event is nacked and retried with
  the bus's backoff rather than sent to a guessed audience.
- **Routing.** A domain event goes to that domain's app (`finance.*` → the finance app's
  tokens). A family event goes once per device, preferring finance, then recipes, then notes,
  so a phone with all three apps is not buzzed three times.
- **Expo.** `POST https://exp.host/--/api/v2/push/send` in batches of at most 100. A ticket with
  `DeviceNotRegistered` deletes its token immediately; successful tickets are stored and a
  background loop fetches receipts after 15 minutes (`/getReceipts`, up to 1000 ids) and deletes
  tokens whose receipt says `DeviceNotRegistered`. Tickets with no receipt after 24 hours are
  dropped — Expo no longer has them. `NOTIFICATIONS_EXPO_ACCESS_TOKEN` is sent when set, for
  projects with enhanced push security. The sender is an interface; tests use a fake.
- **The bus is required.** Unlike the publishers, which log "events disabled" and keep serving,
  this service exits if NATS is unreachable: consuming is its job, and a green healthcheck over a
  dead consumer would hide the outage. It calls `EnsureStream` for each domain it reads so start
  order against the publishers does not matter.

## Alternatives

- **A projection of family membership** (as `services/finance/internal/projection` does). It
  only knows members whose join event is still inside the stream's 30-day window; families older
  than that would be silently empty. The membership check is one internal call per recipient on
  families of a handful of people.
- **A new `ListFamilyMembers` internal rpc on family.** Cleaner audience query, but a contract
  change on another service to save a loop over a few users.
- **FCM/APNs directly.** Two credentials, two payload formats, and the apps already use
  `expo-notifications`. Expo is a single HTTP API in front of both.
- **Per-event-kind on/off matrix per app.** Larger surface for the same outcome; the mute list
  expresses it and stays one table.

## Consequences

- At-least-once, not exactly-once: if Expo accepts the first batch and the second fails, the
  redelivery re-sends the first. Accepted for audiences of a family.
- Message text is English, fixed in the service. Localising needs the user's locale
  (`family.GetUserSettings`) and a catalogue; a follow-up, not a blocker.
- Notes publishes no events yet, so the notes app receives only family notices. A
  `notes.share.granted` payload in `libs/proto/notes/v1` would be the next topic.
- `processed_events` grows by one row per consumed event and is pruned after 60 days, past the
  stream's 30-day retention, so a replay can never re-notify.
- A new container and a new database (`notifications`) in compose and on the VPS; public only
  over Caddy at its own subdomain like the other services.
