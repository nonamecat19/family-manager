# ADR 0011 — Per-user settings owned by services/family, locale on the token

- Status: accepted
- Date: 2026-09-22

## Context

Language was a per-device, per-app value: each Expo app called `createI18n` with its own
`storageKey` (`fm.finance.locale`, `fm.recipes.locale`) and kept the choice in expo-secure-store.
Pick Ukrainian in the finance app and the recipes app still opens in whatever the phone's locale
suggested; the Telegram bots had no way to know at all. A family where one person reads English
and another Ukrainian has no shared place to say so.

## Decision

**`services/family` owns per-user settings**, in a `user_settings` table keyed by `user_id`,
with three additive RPCs on `family.v1`:

- `GetUserSettings(user_id)` — **internal only**, blocked on the public mux like
  `GetUserMembership`, so `services/auth` can read settings over the hop it already makes.
- `GetMySettings()` / `UpdateMySettings(locale, timezone)` — public, authenticated, and
  deliberately **not** membership-gated.

That last point is the sharp edge of this decision. Every other procedure in family asks
`requireMember` first, because family owns household data. Settings are not household data:
a person picks a language before they have a family, and a person who leaves a family still has
one. So these three procedures authenticate the caller and go no further, and `user_settings`
has no `family_id` column to tempt a future reader into scoping it.

**The locale also rides on the access token.** `services/auth` already calls family at mint
time for `family_id`; it now asks for the locale in the same place and stamps a `locale` claim.
Services and bots read it for free, `libs/go/auth.Claims` and `@fm/auth`'s `AccessClaims` both
carry it, and the RPC stays the source of truth for anything that must be current.

**Consumers.** The bots resolve locale from the session's token claim and render every panel
through `services/telegram/internal/i18n` (typed keys, `uk` and `en`, parity enforced by a test).
`/settings` is a framework-level command, so every bot gets the same language picker. On a
change the bot writes through family, then expires its cached access token so the next command
re-mints one carrying the new claim. The Expo apps read `GetMySettings` through
`packages/api` and hand it to `I18nProvider` as `remoteLocale`; the device value remains the
fallback for an unauthenticated app, and picking a language in the app writes it back.

## Alternatives

- **`services/auth`.** The obvious home: it owns `users`, and the token is minted there, so no
  extra hop. Rejected in favour of keeping auth strictly about identity and credentials.
- **A new `services/settings`.** A clean domain boundary, and the right answer if preferences
  grow into notification rules, themes and privacy. Today it would be a whole service — module,
  database, migrations, compose entry, deployment — for one column.
- **`map<string, string> settings` instead of typed fields.** Extending needs no contract
  change, but nothing validates a key and every consumer parses strings. Typed fields keep
  `buf breaking` meaningful.
- **Leave it on the device.** Nothing to build, and the promise ("my language follows me") is
  simply not kept: a new phone, a second app, or a bot chat starts over.

## Consequences

- `auth`'s dependency on `family` deepens: it now asks for settings as well as membership at
  mint time. Both lookups fail soft — a settings outage mints a token without a locale claim
  rather than blocking sign-in, and consumers fall back to `uk`.
- A locale change is visible to other services only after the next token refresh (≤15 min).
  Anything needing it immediately calls `GetMySettings`, which is what the bots and apps do.
- `timezone` is stored but nothing reads it yet; `services/finance` still takes its timezone
  from configuration.
- `apps/notes` keeps its static string module and is not wired to the shared setting.
