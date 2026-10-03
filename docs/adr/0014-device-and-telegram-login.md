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
  per-user throttle, so an authenticated account cannot enumerate the 20⁸ code space.

**Telegram approves as the linked user, never as the service.** The app starts a `telegram`
grant and opens `t.me/<bot>?start=login_<code>`. Any bot's `/start login_<code>` resolves the
Telegram user through the telegram service's session store (one session per Telegram user, ADR
0013) and calls `ApproveDeviceLogin` with that user's own access token — the same path `/unlink`
takes. An unlinked Telegram account gets a localized "connect Telegram first" reply and nothing
reaches auth.

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
- Migration `services/auth` 000006 adds `login_grants`. The hourly sweeper deletes grants an
  hour past their expiry.
- `StartDeviceLogin` and `PollDeviceLogin` join the public procedure list in
  `services/auth/cmd/server/main.go` and in `packages/api`'s client.
- `StartDeviceLogin` is unauthenticated and writes a row; nothing but the reverse proxy limits
  how fast it can be called. Rate limiting it per source address is a follow-up.
- `packages/api` exports `useStartDeviceLogin`, `usePollDeviceLogin`, `useApproveDeviceLogin`,
  `useDenyDeviceLogin` and `useTelegramLogin`; the login screens and an "approve a device" screen
  in the apps consume them.
