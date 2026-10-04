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
