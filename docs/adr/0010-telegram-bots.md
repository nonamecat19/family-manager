# ADR 0010 — One Go service hosting one Telegram bot per app

- Status: accepted; linking amended by [ADR 0013](0013-linked-identities.md)
- Date: 2026-09-22

## Context

The apps in this repo are used from a phone, and half of what a family does with them is a
sentence long: log 250 on groceries, ask what is for dinner, capture a thought before it is
gone. Opening an Expo app to type one line is more ceremony than the line is worth, and
Telegram is already open on every phone in the family.

Telegram's unit of identity is the bot, and a bot is the thing a person talks to. Blending
recipes, finance and notes into a single bot would give one chat with a flat namespace of
commands, one `/help` listing everything, and one set of notifications nobody can mute
selectively. So: one bot per app — but one bot per app does not imply one process per app.

## Decision

**One Go service, `services/telegram`, hosting every bot.** Each bot is a token plus a command
table built over that app's existing Connect client; the service holds the shared parts — the
Bot API client, the update loop, the session store, the command router.

**Transport is configured, not chosen once.** `TELEGRAM_MODE=polling` runs a `getUpdates` loop
per bot with the offset persisted in Postgres (`bot_offsets`), which is what a laptop with no
public URL can do. `TELEGRAM_MODE=webhook` registers `https://<public>/tg/<bot>` per bot and
verifies Telegram's `X-Telegram-Bot-Api-Secret-Token` on every request, which is what the VPS
behind Caddy should do. The router and command tables never learn which is in use.

**A chat acts as a user, never as the service.** `services/auth` gained two additive RPCs:
`CreateLinkToken` (authenticated; the app mints a one-time, 10-minute token) and
`RedeemLinkToken` (public; the bot exchanges the token plus the Telegram user id for an
ordinary access/refresh pair). The app turns the token into `t.me/<bot>?start=<token>`, so the
person taps a link instead of typing a password into a chat. The bot then calls the domain
services over ConnectRPC with that user's JWT — the same path `packages/api` takes, the same
interceptors, the same family scoping. No impersonation header, no service-wide privilege.

**Stored sessions are encrypted at rest.** `telegram_links` holds the access and refresh tokens
sealed with AES-256-GCM under `TELEGRAM_TOKEN_KEY`; the refresh token is rotated through the
existing `Refresh` RPC and re-sealed. Losing the key revokes sessions, it does not lose data.

**The Bot API client is hand written** (`internal/telegram`, ~200 lines: `getUpdates`,
`sendMessage`, `setWebhook`, `deleteWebhook`, `answerCallbackQuery`). It adds no dependency to
the stack table, and the surface actually used here is small enough that a library would mostly
add types we would translate anyway.

## Alternatives

- **A process per bot.** Four containers that differ by a token and a command table, each with
  its own pool, migrations and deployment. The bots share more than they differ.
- **One bot for everything.** One chat, one command namespace, one notification stream. Cheaper
  to run, worse to use, and it makes muting finance without muting recipes impossible.
- **`go-telegram-bot-api` or `telebot`.** Either would work. Neither is a bad choice; both are
  more surface than the five methods here need, and the stack table stays shorter without them.
- **Bot holds a service-wide token and impersonates users.** One credential that can read every
  family's data, and a new authorization path in every service. The link-token flow reuses the
  auth model already in place instead of adding a second one.
- **Password login in chat.** Telegram keeps message history; a password typed into a chat is a
  password stored on Telegram's servers.

## Consequences

- `libs/proto/auth/v1` grew two RPCs. Additive, `buf breaking` stays green, and the mobile apps
  need a "Connect Telegram" affordance to call `CreateLinkToken`.
- The service needs a database of its own (`telegram`) for links and poll offsets.
- The bots are button-driven: every panel is an inline keyboard, callbacks edit the panel in
  place rather than adding messages, and `setMyCommands` fills Telegram's native command menu.
  Text is HTML, and everything interpolated from a service goes through an escaper.
- Guided flows (pick a category, then send an amount) need memory between two updates, so
  `chat_states` holds one short-lived row per (bot, telegram user): a kind, a small JSON payload
  and a 10-minute expiry, swept in the background and cleared whenever a command runs. This is
  the one piece of conversation state; panels themselves stay derived from the services.
- Every panel is rendered through the bot i18n catalog and the user's shared language setting
  ([ADR 0011](0011-user-settings.md)); `/settings` lives in the framework, so all bots share it.
