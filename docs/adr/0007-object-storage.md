# ADR 0007 — MinIO in development, Cloudflare R2 in production

- Status: accepted
- Date: 2026-08-24
- Amends: [0004](0004-compose-vps.md) (the VPS shrinks by one container)

## Context

Recipe images are the only user-uploaded files in the repo. Until now `libs/go/storage` talked
to MinIO everywhere, and the production target is a 2 GB / 1 core / 30 GB VPS. MinIO's resident
set is 300–450 MB — roughly a fifth of that box — to serve a family's recipe photos. It is the
single largest consumer of RAM in the deployment and the least justified one.

Object storage is also the one dependency with no failure-coupling to anything else: the
handler already degrades to "uploads error, everything else works" when it is absent, so
moving it off-box costs no availability that the design does not already tolerate.

## Decision

Two providers, selected by an explicit `RECIPES_STORAGE_PROVIDER`:

- **`minio`** — local development, unchanged. `just up` still brings MinIO up; the service
  creates its own bucket and sets the anonymous-read policy, and objects are served at
  `<public>/<bucket>/<key>`.
- **`r2`** — production. Bucket and public hostname are provisioned out of band, the service's
  API token is scoped to Object Read & Write, and objects are served from a bucket-scoped
  custom domain at `<public>/<key>`.

`minio-go` remains the client for both — R2 is S3-compatible — so no dependency is added or
removed. The provider is a config field rather than something inferred from the endpoint,
because the two backends differ in ways an endpoint string does not reveal:

| | MinIO | R2 |
|---|---|---|
| `PutBucketPolicy` | supported | **not implemented at all** |
| `CreateBucket` | service does it at boot | denied to a scoped token |
| region | ignored | must be `auto`; no `GetBucketLocation` |
| addressing | virtual-host | path-style |
| public URL | `<host>/<bucket>/<key>` | `<domain>/<key>` |

`EnsureBucket` is therefore a deliberate no-op on R2. It does not even probe: a scoped token
can be denied `HeadBucket` on a healthy bucket, and a probe would convert a correct deployment
into a boot failure. A genuinely missing bucket surfaces on first upload, which the caller
already degrades on.

## Alternatives

- **MinIO in production too** — one code path, but ~350 MB on a 2 GB box, and it puts the
  durability of every family photo on a single unreplicated VPS disk.
- **Garage** (self-hosted, Rust, ~50 MB) — keeps data on-box and fixes the RAM problem, but the
  backup burden stays ours and the box still has to serve image bytes over a home-grade uplink.
- **R2 in development too** — deletes the dual code path entirely, but makes `just up` require
  network access and live credentials, and every developer shares one mutable bucket. Local dev
  must work on a plane.
- **Filesystem volume + Caddy `file_server`** — cheapest of all, but replaces the S3 interface
  in `libs/go/storage` with a second implementation and gives up off-host durability.

## Consequences

- **~350 MB freed** on the production box, and image bytes no longer traverse the VPS uplink.
- **Dev and prod now differ in one place.** ADR 0004's "a service that runs under `just up`
  runs in prod" no longer holds for storage. The provider split is deliberately narrow and
  totally contained in `libs/go/storage` to keep that divergence auditable.
- **R2's free tier is 10 GB with zero egress fees**, comfortably past what a family cookbook
  will hold. Past it, storage is billed; egress stays free.
- Recipe images move out of the `pg_dump` + volume backup story into Cloudflare's durability.
  The restore drill in the release checklist must cover the bucket, not just the database.
- `image_url` values already stored point at the old host. Existing rows are dev-only today; a
  production cutover after real data exists would need a URL rewrite migration.
- Config vars renamed `RECIPES_MINIO_*` → `RECIPES_STORAGE_*`. A stale local `.env` silently
  disables image upload rather than erroring, because storage is optional by design.
