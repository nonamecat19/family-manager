# ADR 0013 — Linked identities live in `services/auth`; one Telegram link covers every bot

- Status: accepted
- Date: 2026-09-26

## Context

[ADR 0010](0010-telegram-bots.md) linked a Telegram account per bot: `telegram_links` was keyed
by `(bot, telegram_user_id)`, so a person who connected the finance bot had to connect the
recipes, notes and family bots again, one link each. `services/auth` only minted the one-time
link token; it never recorded that Telegram user X is user Y, so nothing outside the telegram
service could list or revoke that connection. A TUI and "log in with Telegram" on mobile are
coming, and each would have needed its own answer to the same question.

Telegram's `user.id` is the same for every bot, so the per-bot key bought nothing.

## Decision

**`services/auth` owns linked identities.** A new `identities` table holds `(user_id, provider,
external_id, chain_id)` with `UNIQUE (provider, external_id)`. `RedeemLinkToken` binds the
identity in the same transaction that spends the link token, and records the refresh chain it
mints for that identity.

- **Re-linking the same account** to the same user replaces the recorded chain and revokes the
  previous one — one live session per identity.
- **An account already linked to another user is refused** with `AlreadyExists`; the link token
  is not spent. Moving an account between users means unlinking it first. No silent takeover.
- **`ListIdentities` and `Unlink(provider, external_id)`** are new authenticated RPCs, scoped to
  the caller. `Unlink` deletes the identity and revokes its chain; the holder's next `Refresh`
  fails. Unlinking from the app, a bot, or a future client is the same call.

**Revocation has a row of its own.** A new `chains` table holds one row per refresh chain with
its `revoked_at`. Revoking a chain upserts that row before marking the chain's tokens, so a chain
can be revoked before its first token exists. Minting a refresh token locks the chain row in the
same transaction as the insert and refuses a revoked chain. A new chain (login, link) inserts
its row; a refresh requires the row to exist, so a chain whose row was swept fails closed; a concurrent revoke therefore either
lands first and is seen, or waits for the mint to commit and then covers the new token. Binding
and unlinking an identity take a transaction-scoped advisory lock on `provider:external_id`, so
the chain a relink revokes is always the one that was current.

**`services/telegram` keeps one session per Telegram user**, not per bot. `telegram_links` is
keyed by `telegram_user_id`; linking in any bot unlocks all of them, and `/unlink` in any bot
calls `Unlink` in auth with the user's own access token, then drops the row. Refreshes are
single-flight per Telegram user inside the process — every bot now shares the row, and two
concurrent refreshes with one token would read as a replay and revoke the chain. Writes after a
refresh are conditional on the refresh token that was presented, and `/unlink` deletes only the
row of the user it unlinked, so a stale refresh never overwrites or drops a newer link.

## Alternatives

- **Only re-key `telegram_links` by Telegram user.** Fixes the per-bot linking, but the identity
  stays private to the telegram service: no listing in the app, no revocation from anywhere
  else, and nothing a future "log in with Telegram" could build on.
- **Tag every refresh token with an identity id.** Revocation by identity without a
  `chain_id` on the identity, at the cost of threading the id through rotation. One chain per
  identity is simpler and matches how the telegram service holds exactly one session per user.
- **Revoke by marking `refresh_tokens` only.** An `UPDATE` touches no row for a chain whose
  first token is not written yet, and under READ COMMITTED an insert racing an uncommitted revoke
  lands unrevoked. Both left a live session after an unlink.
- **A service credential that trades a Telegram user id for a user token.** Removes stored
  sessions from the telegram service, but it is the service-wide impersonation ADR 0010 rejected.

## Consequences

- `libs/proto/auth/v1` grew two RPCs and three messages. Additive; `buf breaking` stays green.
- Migration `services/auth` 000005 adds `chains`, backfilled from `refresh_tokens`; chains with
  no tokens left are swept after a day. This also closes the same race for `Logout`, which
  predates this ADR.
- Migration `services/telegram` 000003 truncates `telegram_links`. Sessions linked before it
  carry no identity in auth and could never be revoked by `Unlink`, so everyone links once more.
  The orphaned refresh tokens expire at their 30-day TTL.
- Revocation is as fast as ADR 0005 allows: the telegram service may hold a valid access token
  for up to 15 minutes after an `Unlink` from elsewhere.
- Follow-ups build on `identities`: a "Connected accounts" screen in the apps
  (`ListIdentities` / `Unlink`), log in with Telegram on mobile, and a device-code flow for the
  TUI.
