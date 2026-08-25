# Deployment

Production is Docker Compose on a single VPS behind Caddy, images built in CI and pulled by the
box (ADR [0004](../docs/adr/0004-compose-vps.md)). Object storage is Cloudflare R2, not MinIO
(ADR [0007](../docs/adr/0007-object-storage.md)).

The box is small and that shapes every decision here: **1 vCPU, 1.6 GiB RAM, 20 GB disk**. It
cannot build a Go image — `go build` there either OOMs or takes long enough that a deploy stops
being a deploy. CI builds; the VPS only pulls.

## What is in this directory

| file | runs where | purpose |
|---|---|---|
| `bootstrap-vps.sh` | VPS, once, as root | Docker, 2 GiB swap, ufw, deploy dir, weekly image prune |
| `docker-compose.prod.yml` | VPS | the production stack — self-contained, not an overlay |
| `Caddyfile` | VPS | TLS termination and one vhost per service |
| `nats.conf` | VPS | JetStream store limits — they have no command-line equivalent |
| `.env.example` | template | copied to `/opt/family-manager/.env` and filled in there |
| `deploy.sh` | VPS, every deploy | pull, up, health-check, roll back on failure |
| `wizards/r2-setup.sh` | your workstation | walks Cloudflare's dashboard, validates the credentials |

`.github/workflows/deploy.yml` is the other half: it builds the four images, pushes them to
GHCR, and SSHes in to run `deploy.sh`.

## Topology

One subdomain per service, which is the same shape as development — each app already holds a
per-service base URL (`packages/api/src/client.ts`), so nothing in the app changes between a
laptop and production except the hostnames.

```
                    :443
  auth.nonamecat.pp.ua    ─┐
  family.nonamecat.pp.ua  ─┤                      ┌─ auth     :8080
  finance.nonamecat.pp.ua ─┼─→ caddy (TLS) ──────→┼─ family   :8080  (+ :9090 internal gRPC)
  recipes.nonamecat.pp.ua ─┘                      ├─ finance  :8080
                                                  └─ recipes  :8080
                                                       │
                                    postgres ──────────┤
                                    nats ──────────────┘
```

Caddy is the only container that publishes a host port. Postgres, NATS and all four services
are reachable only on the compose network — `ufw` would not save us if they were published,
because Docker writes its own iptables chain ahead of ufw's.

`family:9090` is deliberately never proxied. It serves the procedures that take a user id as an
argument instead of reading it from a token; exposing it would be an unauthenticated
impersonation API.

## First deploy

Steps 1 and 2 are yours; the rest is mechanical.

### 1. Point DNS at the box

All four A records must resolve **before** the first start. Caddy issues certificates over the
ACME HTTP-01 challenge, so a hostname that does not yet resolve to this box cannot get one, and
repeated failed issuance counts against Let's Encrypt rate limits.

```
auth.nonamecat.pp.ua      A  79.108.160.103
family.nonamecat.pp.ua    A  79.108.160.103
finance.nonamecat.pp.ua   A  79.108.160.103
recipes.nonamecat.pp.ua   A  79.108.160.103
```

They currently point at `135.181.41.169`. Verify with `getent ahostsv4 auth.nonamecat.pp.ua`
(`dig` needs the `dnsutils` package and is not installed by default on either end).

If you use R2's custom domain for images, that hostname needs a record too — Cloudflare creates
it for you when the bucket is in a zone you control.

### 2. Provision R2

```sh
./infra/wizards/r2-setup.sh
```

It walks the dashboard, collects the account id, bucket, scoped token and public domain, then
does a real upload-and-fetch round trip before declaring success. It prints the env block to
paste into `.env`.

### 3. Bootstrap the box

```sh
scp infra/bootstrap-vps.sh root@79.108.160.103:/root/
ssh root@79.108.160.103 'bash /root/bootstrap-vps.sh'
```

Idempotent — safe to re-run when you are not sure whether a box was ever bootstrapped.

### 4. Copy the runtime files

```sh
scp infra/docker-compose.prod.yml infra/Caddyfile infra/nats.conf infra/deploy.sh \
    root@79.108.160.103:/opt/family-manager/
ssh root@79.108.160.103 'chmod 0755 /opt/family-manager/deploy.sh'

# Only this one file. init-postgres.sql and init-finance.sql are leftovers from the
# pre-split layout; init-finance.sql would create a role with a publicly-known password.
scp postgres/init/init-services.sql root@79.108.160.103:/opt/family-manager/postgres-init/
```

### 5. Install the signing key

Generate it once and keep the only other copy somewhere safe. Losing it invalidates every token
ever issued; leaking it lets anyone mint tokens for every family in the system.

```sh
# Into infra/secrets/, not the working directory. `*.pem` is gitignored repo-wide, but that
# rule is the backstop — generating a production key inside a git worktree at all is the habit
# worth not forming.
openssl ecparam -name prime256v1 -genkey -noout -out infra/secrets/auth-signing-key.pem
scp infra/secrets/auth-signing-key.pem root@79.108.160.103:/opt/family-manager/secrets/
ssh root@79.108.160.103 'chown 65532:65532 /opt/family-manager/secrets/auth-signing-key.pem &&
                         chmod 0400        /opt/family-manager/secrets/auth-signing-key.pem'
```

**The ownership is not cosmetic.** Compose bind-mounts a file-backed secret with the host's
ownership and mode, and ignores every override — `mode:` on a top-level secret is rejected
outright, and the service-level `uid`/`gid`/`mode` are dropped with a warning. A key left
`0600 root:root` is unreadable by uid 65532, the `nonroot` user the distroless image runs as,
so auth exits at boot while the other three services come up fine. That is a deploy that looks
three-quarters successful and has no working login. `deploy.sh` re-checks and repairs this on
every run, but a hand-started stack will not.

### 6. Fill in `.env`

```sh
scp infra/.env.example root@79.108.160.103:/opt/family-manager/.env
ssh root@79.108.160.103 'chmod 0600 /opt/family-manager/.env'
```

Then edit it on the box. Every `CHANGE_ME_` value must be replaced; the R2 block comes from the
wizard.

### 7. Create the GitHub secrets

| secret | value |
|---|---|
| `VPS_HOST` | `79.108.160.103` |
| `VPS_SSH_KEY` | private half of the deploy keypair, OpenSSH format, no passphrase |
| `VPS_SSH_FINGERPRINT` | `ssh-keygen -l -f /etc/ssh/ssh_host_ecdsa_key.pub \| cut -d' ' -f2`, run on the box. **The ECDSA key, not ed25519** — the action is a Go program and `x/crypto/ssh` prefers `ecdsa-sha2-nistp256`, so an ed25519 fingerprint fails with `host key fingerprint mismatch` despite being a valid key for that host. |

Install the public half on the box with `ssh-copy-id`, or append it to
`/root/.ssh/authorized_keys` by hand.

The fingerprint is optional to the action and not optional in practice: without it the runner
accepts whatever host answers on that IP.

### 8. Deploy

Push to `master`, or run the `deploy` workflow manually. To deploy by hand:

```sh
ssh root@79.108.160.103 '/opt/family-manager/deploy.sh <commit-sha>'
```

## Redeploy

Every push to `master` builds the four images, tags them with the commit SHA and `latest`,
pushes to GHCR, then SSHes in and runs `deploy.sh <sha>`. The deploy is serialised — a second
push waits rather than interleaving with the first, because interrupting a deploy between
`pull` and `up -d` leaves the box running a mix of old and new images.

`deploy.sh` pulls first and separately, so a bad tag or an expired credential fails while the
old containers are still serving. After `up -d` it probes `/healthz` on all four services from
inside the Caddy container — the services publish no ports and their distroless images have no
shell, so there is no other vantage point — and rolls back to the previously deployed tag if
any of them does not answer.

## Rollback

Re-run the workflow with `workflow_dispatch` and an older commit SHA as `image_tag`, or on the
box:

```sh
ssh root@79.108.160.103 '/opt/family-manager/deploy.sh <older-sha>'
```

Images are tagged by SHA and the weekly prune keeps a week of them, so a recent rollback needs
no rebuild. Note that **a rollback does not roll back migrations** — services migrate their own
schema at boot and nothing reverses that. A schema change that is not backwards compatible with
the previous image cannot be rolled back this way, which is why migrations carry a human gate.

## Memory budget

The whole point of the tuning. Total 1.6 GiB, plus the 2 GiB swapfile bootstrap adds.

| container | limit | |
|---|---|---|
| postgres | 320m | `shared_buffers=128MB`, `max_connections=50`, `work_mem=4MB` |
| nats | 128m | JetStream capped at 512MB file / 32MB memory store |
| caddy | 96m | |
| auth, family, finance, recipes | 160m each | `GOMEMLIMIT=140MiB`, `GOGC=50` |

≈1.18 GiB committed, leaving ~400 MiB for the OS.

Two settings that look redundant and are not. `mem_limit` rather than
`deploy.resources.limits`: the latter is a swarm key that `docker compose up` ignores outside
swarm, so it would appear to be set and do nothing. And `GOMEMLIMIT` on every Go service: the
Go runtime sizes its heap against the *host's* memory, not the cgroup, so without it a service
under load grows past `mem_limit` and is OOM-killed instead of collecting.

## Backups — the load-bearing part

ADR 0004 is explicit that a VPS with no tested restore is worse than a managed platform, and
this is the part that is **not yet built**. What is needed:

- `pg_dump` of all four databases on a schedule, shipped off-box.
- A restore drill, run before the box holds anything anyone would miss.
- Recipe images live in R2 now, so they are outside the `pg_dump` story — the drill has to cover
  the bucket too, not just the database.

Two volumes are the only copy of anything: `pgdata` and `caddydata` (the issued certificates).
Nothing in `deploy.sh` or the prune timer touches volumes, and nothing added later should —
`docker system prune --volumes` would delete any volume no running container references, which
is exactly the state the box is in mid-deploy.

## Known gaps

- **Backups are not implemented.** See above. This is the highest-value next piece of work.
- **No rolling deploys.** A service restart is a brief outage; mobile clients retry. Not
  acceptable during a migration, which is why migrations get a human gate.
- **Observability** is `docker compose logs` and nothing else. See ADR 0006 for the intent.
- **`init-postgres.sql` and `init-finance.sql`** are pre-split leftovers still in the repo, kept
  because `finance-legacy` has not been retired yet. Neither belongs on the production box.
