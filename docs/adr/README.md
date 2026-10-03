# Architecture decision records

One file per decision that could reasonably have gone the other way. Format: context →
decision → alternatives → consequences. Numbered, never renumbered, never deleted — a reversed
decision gets a new ADR that supersedes the old one, and the old one's status changes to
`superseded by NNNN`.

| # | decision | status |
|---|---|---|
| [0001](0001-connectrpc.md) | ConnectRPC for app↔service, gRPC between services | accepted |
| [0002](0002-nativewind.md) | NativeWind for `packages/ui` | accepted |
| [0003](0003-nats-jetstream.md) | NATS JetStream as the event backbone | accepted |
| [0004](0004-compose-vps.md) | Docker Compose on a VPS for deployment | accepted |
| [0005](0005-auth.md) | ES256 JWT + rotating refresh tokens, JWKS verification | accepted |
| [0006](0006-observability.md) | OTel traces correlated with slog; CI runs the verify node | accepted |
| [0007](0007-object-storage.md) | MinIO in development, Cloudflare R2 in production | accepted |
| [0008](0008-rpc-interceptors.md) | Shared interceptor chain in `libs/go/rpc`; request ids before traces | accepted |
| [0009](0009-tauri-desktop.md) | Tauri v2 wrapping the Expo web export for the notes desktop app | accepted |
| [0010](0010-telegram-bots.md) | One Go service hosting one Telegram bot per app | accepted |
| [0011](0011-user-settings.md) | Per-user settings owned by `services/family`, locale on the token | accepted |
| [0012](0012-notes-connect-java.md) | `services/notes` serves `notes.v1` over Connect from Spring | accepted |
| [0013](0013-linked-identities.md) | Linked identities live in `services/auth`; one Telegram link covers every bot | accepted |
| [0014](0014-device-and-telegram-login.md) | Device-code login and "log in with Telegram" share one pending login grant | accepted |
| [0015](0015-prometheus-metrics.md) | Prometheus metrics from `rpc.Observe`, scraped off the internal listener | accepted |

Write a new ADR when a change would displace a row in [../stack.md](../stack.md), add a
container to `docker-compose.yml`, or alter a boundary rule in
[../architecture.md](../architecture.md). Do not write one for library choices confined to a
single service or app.
