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
  owned by the caller. Registering a token another user holds moves it only when the request
  carries the same non-empty `device_id` the token was stored with (the same install changed
  account); anything else is `PermissionDenied`, so knowing a token is not enough to take it.
  Clients send a random per-install id: 16 bytes from `expo-crypto`'s `getRandomBytes`, minted
  once and kept in secure storage (`getInstallId` in `@fm/auth`). Without a secure random source
  the client sends an empty id rather than a weak one, which only ever fails closed. Only Expo
  tokens (`ExponentPushToken[…]`, `ExpoPushToken[…]`) are accepted.
- **Sign-out.** The client unregisters its token from a before-sign-out hook in `@fm/auth`
  (`registerBeforeSignOut`), which runs while the access token is still valid and before the
  refresh token is revoked. The hook first waits for any registration still in flight (token
  lookup, install id, `RegisterPushToken`) and blocks new ones, then unregisters what was
  registered. Sign-out waits for it at most 3 seconds and never fails because of it.
- **Preferences are a mute list, not a matrix.** Each entry is a domain (`finance`) or a topic
  key (`finance.budget.exceeded`); the default is everything on. `SetPreferences` replaces the
  list. The catalogue is served by `GetPreferences` so an app renders toggles without hardcoding
  them.
- **Consumer.** One durable JetStream consumer per subject, named
  `notifications-<domain>-<entity>-<verb>`, through `libs/go/events`. Today:
  `family.member.joined` (tell the other members), `family.member.removed` (tell the removed
  member, only when an admin removed them), `finance.budget.exceeded` (every member),
  `recipes.recipe.created` (every member but the author). `finance.budget.exceeded` is
  computed by finance over shared-visibility accounts only, and is not published when the
  triggering transaction is on a private account — the amounts in it go to every member.
  Consumers send `InProgress` every 10 seconds while a handler runs
  (`events.SubscribeOptions.Heartbeat`), so a slow delivery is not redelivered underneath
  itself. Events nobody should be woken for —
  transaction edits, account changes, recipe updates — are deliberately not consumed.
- **Idempotency without an event id on the wire.** Event payloads carry no id, so the service
  keys `processed_events` on `sha256(subject ∥ payload)`; every payload carries `occurred_at`,
  so two real events never collide. A delivery first claims the event (insert with
  `status = 'claimed'`, `ON CONFLICT DO NOTHING`); a concurrent delivery that finds a fresh
  claim is nacked and retried, and a claim older than 60 seconds (past the 45-second handler
  timeout) is taken over. If the handler fails before Expo accepted anything the claim is
  released and redelivery retries; once Expo accepted any batch the event is marked `done` and
  never retried.
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
  tokens whose receipt says `DeviceNotRegistered`. Either way a token is deleted only if it was
  not registered again after the send (`updated_at` earlier than the ticket). Tickets with no receipt after 24 hours are
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

- At-most-once once Expo accepted a batch: if the first batch is accepted and the second
  fails, the second is not retried (the alternative re-sends the first). Families fit in one
  batch, so in practice this is all-or-nothing.
- **Legacy tokens.** Tokens registered before the install id existed are stored with
  `device_id = ''`. Such a token can never move to another user: the owner's next registration
  stamps the install id onto it, and until then a different account on the same phone gets
  `PermissionDenied` and receives nothing on that token. It clears itself once the original
  owner opens the updated app, signs out (which unregisters it), or Expo reports it dead.
- **Rollout order.** Migration `000002_event_claims` adds `status` and `claimed_at` to
  `processed_events`; the service applies its migrations at startup, so the new binary brings
  its own schema. Existing rows default to `done`, so already-processed events stay
  processed. Deploy notifications before (or with) the client release: an old client sends no
  `device_id` and keeps working for its own account. Rolling back the binary needs the down
  migration, which first deletes in-flight `claimed` rows (they would otherwise read as
  processed to the old code) and then drops the columns.
- **Clocks.** Claim staleness and the "registered again since the send" check compare database
  timestamps only (`NOW()` in SQL, the ticket's `created_at`, a `NOW()` read before each send),
  so clock skew between the service and Postgres cannot shorten a claim or delete a fresh
  token.
- **Known limitation: a session revoked server-side cannot unregister its token.** The
  before-sign-out hook covers a user signing out on the device. A session revoked from
  elsewhere (another device, an admin, refresh-token reuse detection) leaves the device's
  token registered, and it keeps receiving the family's pushes until the app next signs in or
  Expo reports the token dead. Follow-up: `services/auth` publishes an `auth.session.revoked`
  event; notifications stores the session identifier with each token at registration and
  deletes the tokens of a revoked session. Not implemented here.
- Message text is English, fixed in the service. Localising needs the user's locale
  (`family.GetUserSettings`) and a catalogue; a follow-up, not a blocker.
- Notes publishes no events yet, so the notes app receives only family notices. A
  `notes.share.granted` payload in `libs/proto/notes/v1` would be the next topic.
- `processed_events` grows by one row per consumed event and is pruned after 60 days, past the
  stream's 30-day retention, so a replay can never re-notify.
- A new container and a new database (`notifications`) in compose and on the VPS; public only
  over Caddy at its own subdomain like the other services.
