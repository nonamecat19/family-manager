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

Write a new ADR when a change would displace a row in [../stack.md](../stack.md), add a
container to `docker-compose.yml`, or alter a boundary rule in
[../architecture.md](../architecture.md). Do not write one for library choices confined to a
single service or app.
