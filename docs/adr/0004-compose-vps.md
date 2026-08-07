# ADR 0004 — Docker Compose on a VPS for deployment

- Status: accepted
- Date: 2026-08-06

## Context

A family-scale application with a handful of Go services and mobile clients. Traffic is small
and bounded; operational time is the scarce resource, not compute.

## Decision

`infra/` holds a production Compose overlay on top of the root `docker-compose.yml`, plus Caddy
as the TLS-terminating reverse proxy in front of each service's Connect port. Deploy is
`docker compose pull && docker compose up -d` over SSH, images built in CI and pushed to a
registry. Postgres, MinIO and NATS run as containers with named volumes and a scheduled
`pg_dump` + MinIO mirror to off-host storage.

## Alternatives

- **Kubernetes/Helm** — real rollout primitives, but the control plane costs more attention than
  the whole application at this size.
- **Fly.io / Railway** — least ops, managed Postgres, but per-service config diverges from the
  local compose file and NATS/MinIO become paid add-ons or extra machines.

## Consequences

- Dev and prod share one topology; a service that runs under `just up` runs in prod.
- No rolling deploys: a service restart is a brief outage. Acceptable for mobile clients that
  retry; not acceptable during a migration, which is why migrations get a human gate.
- **Backups are the load-bearing part of this decision.** A VPS with no tested restore is worse
  than a managed platform. Restore drill belongs in the release checklist.
- Migration path if it outgrows this: the Compose services map 1:1 to Deployments; contracts and
  events do not change.
