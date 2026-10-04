# ADR 0014 — Device-code login and "log in with Telegram" share one pending login grant

- Status: accepted
- Date: 2026-10-04

## Context

Two clients cannot type a password into a form we control. A TUI has no browser to bounce through,
and on mobile the person already carries a signed-in Telegram account that
[ADR 0013](0013-linked-identities.md) links to their family-manager user. Both need the same
thing: a session minted on one device because the person approved it somewhere they are already
signed in. ADR 0013 listed both as follow-ups that build on `identities`.

## Decision

**One `login_grants` table in `services/auth` serves both flows.** A grant is a row with two
secrets, a kind (`device`, `telegram`), a ten-minute expiry, and a decision (`approved_at` or
`denied_at` plus the deciding `user_id`), then `consumed_at` once a session is minted from it.

- **`StartDeviceLogin`** (public) returns a `device_code` (256-bit, base64url, only the client
  that started the grant holds it), a `user_code` (`XXXX-XXXX`, 8 letters from a 20-consonant
  alphabet with no vowels or look-alikes, after RFC 8628 §6.1), `expires_at` and
  `interval_seconds` (5). Both codes are stored as SHA-256 hashes and looked up by hash, so no
  comparison of a secret happens in application code. A `telegram` grant also returns
  `telegram_start_payload = "login_" + user_code`, which fits Telegram's 64-character
  `[A-Za-z0-9_-]` start parameter; the client prefixes `https://t.me/<bot>?start=` exactly as it
  does for link tokens.
- **`PollDeviceLogin(device_code)`** (public) answers `PENDING`, `SLOW_DOWN`, `DENIED`,
  `EXPIRED`, or `APPROVED` with the same token pair `Login` returns, minted on a **new refresh
  chain** (ADR 0013's `chains` row is inserted with it). The grant is consumed with a conditional
  `UPDATE … WHERE consumed_at IS NULL`, so it mints at most once; every later poll reads
  `EXPIRED`. An unknown device code also reads `EXPIRED` rather than an error — swept rows and
  forged codes look the same to the client. Polling faster than the interval (with one second of
  jitter allowed) returns `SLOW_DOWN` and a longer interval, as RFC 8628 §3.5 does; the gap is
  enforced by a conditional `UPDATE` on `last_polled_at`.
- **`ApproveDeviceLogin(user_code)` / `DenyDeviceLogin(user_code)`** (authenticated) record the
  caller as the decider. Approving is idempotent for the user who approved; anyone else sees
  `NotFound`, the same answer as for an unknown or expired code. Wrong codes count against a
  per-user throttle, so an authenticated account cannot enumerate the 20⁸ code space. That
  throttle is its own instance, separate from `Login`'s per-email throttle: they share no
  keyspace, so a `Login` attempt with the email `device-login:<uuid>` cannot lock a user out of
  approving, and wrong codes cannot lock an email out of `Login`.
- **`StartDeviceLogin` is limited twice.** Per client address, a fixed window
  (`AUTH_DEVICE_LOGIN_PER_IP`, default 10 per `AUTH_DEVICE_LOGIN_WINDOW`, default 10 minutes)
  answers `ResourceExhausted` past the limit. The address is the right-most `X-Forwarded-For`
  entry — the hop Caddy appends (and, with no `trusted_proxies` configured, the only one it
  forwards) — falling back to the TCP peer. `auth` publishes no port, so only Caddy can set the
  header. Globally, when `AUTH_DEVICE_LOGIN_MAX_ACTIVE` (default 5000) grants are pending —
  neither consumed nor denied, and unexpired — further starts answer `ResourceExhausted` before
  any row is written, which bounds the table no matter how many addresses an attacker holds.
  The limiter is in memory and per process, like the login throttle.

**Telegram approves as the linked user, never as the service.** The app starts a `telegram`
grant and opens `t.me/<bot>?start=login_<code>`. Any bot's `/start login_<code>` resolves the
Telegram user through the telegram service's session store (one session per Telegram user, ADR
0013) and calls `ApproveDeviceLogin` with that user's own access token — the same path `/unlink`
takes. An unlinked Telegram account gets a localized "connect Telegram first" reply and nothing
reaches auth.

**Only in a private chat, only by the person who asked.** The bot is a member of family groups,
and a `/start login_<code>` posted there would put a "sign me in" button in front of every
linked member: whoever tapped it would approve the poster's device as themselves. So both the
`/start login_…` handler and the button handler refuse unless the chat is private (`chat.type
== "private"` and the chat id equals the sender's Telegram id), and the buttons carry the
Telegram id of the user who sent `/start` (`login:ok:<code>:<telegram id>`, well under
Telegram's 64-byte callback limit). A press from anyone else, or a button without that id, is
answered "out of date" and reaches nothing.

**A session approved from a session dies with it.** Access tokens carry the refresh chain they
were minted on as a `sid` claim. `ApproveDeviceLogin` verifies the bearer token itself
(`token.Signer.ChainOf`) and stores that chain on the grant as `approver_chain_id`; a token with
no `sid` is refused as `Unauthenticated` (only tokens minted before this change, which expire
within one access TTL). `PollDeviceLogin` picks the new session's chain id before consuming and
writes it as `chain_id` in the same conditional `UPDATE` that consumes the grant, which also
requires the approver's chain to exist and be unrevoked, taking `FOR SHARE` on that `chains`
row. `Unlink` — and a relink that replaces an identity's chain — tombstones the identity's chain
and then every chain reachable from it through `approver_chain_id → chain_id` (a recursive
query, so a device approved by a device that Telegram approved goes too). The row lock orders
the two: either the consume commits first and the unlink's lineage query sees its `chain_id`
(the chain is tombstoned before the session is minted, so minting fails), or the tombstone
commits first and the consume matches nothing and reads `EXPIRED`. The Telegram path needs no
special marker: the telegram service always calls with the access token of the identity's own
chain (ADR 0013), so "approved through Telegram" is exactly "`approver_chain_id` is the
identity's `chain_id`". The sweeper keeps a consumed grant past its hour of grace while the
chain it minted still exists, because that row is the only record of the lineage.

**The bot asks before it approves.** The deep link shows a confirmation with the code and two
buttons; only the "sign me in" button calls `ApproveDeviceLogin`. A one-tap deep link that
approved on arrival would let anyone who gets a victim to tap their link sign in as the victim —
the device-code phishing pattern. The prompt says to approve only a sign-in started just now.

## Alternatives

- **A separate `StartTelegramLogin` RPC and table.** The two flows differ only in who approves
  and how the code travels; a second table would duplicate expiry, consumption and throttling.
  A `kind` field on one grant keeps one state machine.
- **A service credential in the telegram service that mints sessions for a Telegram user id.**
  The service-wide impersonation ADR 0010 and ADR 0013 already rejected.
- **Put the deep link in the response.** Auth does not know bot usernames; the apps already do
  (`EXPO_PUBLIC_TELEGRAM_BOT`), and `telegramStartUrl` already validates them.
- **A longer secret in the deep link instead of the user code.** Would remove the human-readable
  code the confirmation prompt shows, and the user code is already single-use, ten-minute and
  throttled.
- **Auto-approve on `/start` without a confirmation.** One fewer tap, one-tap account takeover.

## Consequences

- `libs/proto/auth/v1` grew four RPCs, two enums and eight messages. Additive; `buf breaking`
  stays green.
- Migration `services/auth` 000006 adds `login_grants`, including `approver_chain_id` and
  `chain_id`. The hourly sweeper deletes grants an hour past their expiry unless the chain the
  grant minted still exists.
- Access tokens grow a `sid` claim (the refresh chain id). Verifiers in other services ignore it.
- `StartDeviceLogin` and `PollDeviceLogin` join the public procedure list in
  `services/auth/cmd/server/main.go` and in `packages/api`'s client.
- `StartDeviceLogin` is unauthenticated and writes a row; it is bounded by the per-address
  window and the global pending cap above. Both counters live in one process; running more than
  one `auth` replica multiplies the per-address limit by the replica count (the global cap is in
  Postgres and holds).
- `packages/api` exports `useStartDeviceLogin`, `usePollDeviceLogin`, `useApproveDeviceLogin`,
  `useDenyDeviceLogin` and `useTelegramLogin`; the login screens and an "approve a device" screen
  in the apps consume them.
