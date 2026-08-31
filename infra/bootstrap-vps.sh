#!/usr/bin/env bash
#
# One-time bootstrap for the family-manager production VPS.
#
# Target (measured, not assumed): Ubuntu 26.04 LTS, x86_64, 1 vCPU, 1.6 GiB RAM, 20 GB disk,
# zero swap, Docker not installed, root SSH.
#
# Run once, as root, on a fresh box:
#   scp infra/bootstrap-vps.sh root@79.108.160.103:/root/
#   ssh root@79.108.160.103 'bash /root/bootstrap-vps.sh'
#
# Every step is idempotent: a second run must change nothing and must not fail. That matters
# more than it looks — this script is the thing an operator reaches for when they are not sure
# whether the box was ever bootstrapped, and a script that is unsafe to re-run gets guessed at
# instead of run.
#
# This script does NOT deploy anything. It prepares the host; docker-compose.prod.yml, the
# Caddyfile, .env and the signing key are copied in afterwards (see the summary it prints).

set -euo pipefail

DEPLOY_DIR=/opt/family-manager
SWAPFILE=/swapfile
SWAP_SIZE_MB=2048

log()  { printf '\n\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[warn]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m[fail]\033[0m %s\n' "$*" >&2; exit 1; }

[[ ${EUID} -eq 0 ]] || die "run this as root"
[[ -r /etc/os-release ]] || die "no /etc/os-release; this script only targets Ubuntu"
# shellcheck disable=SC1091
. /etc/os-release
[[ ${ID:-} == "ubuntu" ]] || warn "expected Ubuntu, found '${ID:-unknown}' — continuing, but the apt bits may not apply"

export DEBIAN_FRONTEND=noninteractive

# ---------------------------------------------------------------------------------------------
# 1. Firewall — FIRST, and in an order that cannot strand the operator.
# ---------------------------------------------------------------------------------------------
# Order is the whole point here. `ufw enable` applies the default deny-incoming policy
# immediately, so if 22 is not already allowed when that line runs, the SSH session executing
# this script is killed mid-bootstrap and the box is unreachable — a fresh reinstall, not a
# fix, because there is no console on most cheap VPS plans. So: add the allow rules while the
# firewall is still inactive (rules are stored, not enforced), and only then enable.
#
# `ufw --force enable` is required because a bare `enable` prompts "may disrupt existing ssh
# connections" and this script is not interactive. The prompt is the only thing being skipped;
# the rule for 22 is already in place by then.
#
# Also worth knowing: Docker publishes ports by writing its own iptables DOCKER chain, which is
# traversed before ufw's filtering. ufw here protects *host* listeners (sshd, anything installed
# later); it does not gate published container ports. That is fine for us — Caddy is the only
# container that publishes anything, and it publishes exactly 80/443, which we want open. It is
# also why the production overlay must not publish Postgres or NATS: ufw would not save us.
log "Configuring ufw"
if ! command -v ufw >/dev/null 2>&1; then
  apt-get update -qq
  apt-get install -y -qq ufw
fi

# No `ufw reset` for idempotence: reset *disables* the firewall before rebuilding it, so a
# re-run would leave the box briefly unfiltered, and it drops a backup file in /etc/ufw every
# time. `ufw allow` and `ufw default` are already idempotent — a duplicate rule is skipped.
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp   comment 'ssh — must exist before enable'
ufw allow 80/tcp   comment 'http, and ACME http-01 for Caddy'
ufw allow 443/tcp  comment 'https'
ufw --force enable
ufw status verbose

# ---------------------------------------------------------------------------------------------
# 2. Swap — 2 GiB, because the box ships with none.
# ---------------------------------------------------------------------------------------------
# 1.6 GiB of RAM against a ~1.18 GiB container budget leaves no headroom for page cache, apt, or
# a `docker compose pull` unpacking layers. Without swap the kernel's only response to a spike is
# the OOM killer, and it will pick Postgres often enough to matter.
#
# swappiness=10, not the default 60 and not 1:
#   - 60 makes the kernel trade away page cache eagerly; on a box where Postgres leans on the
#     host page cache (shared_buffers is only 128MB) that is a direct hit to query latency.
#   - 1 is the usual "database host" advice, but that advice assumes RAM headroom exists. Here it
#     effectively means "do not use the swap we just created", which puts us back with the OOM
#     killer. 10 keeps genuinely cold anonymous pages (idle Go goroutine stacks, apt leftovers)
#     evictable without churning the cache.
# vfs_cache_pressure=50 for the same reason: keep dentry/inode caches around a bit longer.
log "Configuring swap"
if swapon --show --noheadings | grep -q .; then
  warn "swap already active, leaving it alone:"
  swapon --show
elif [[ -e ${SWAPFILE} ]]; then
  warn "${SWAPFILE} exists but is not active — not touching it; inspect it by hand"
else
  # fallocate leaves a sparse/extent-mapped file that some filesystems refuse to swapon; dd
  # writes it out for real. 2 GiB of writes on a 20 GB disk is a few seconds, once, ever.
  dd if=/dev/zero of=${SWAPFILE} bs=1M count=${SWAP_SIZE_MB} status=none
  chmod 0600 ${SWAPFILE}
  mkswap ${SWAPFILE} >/dev/null
  swapon ${SWAPFILE}
  log "swap enabled:"
  swapon --show
fi

if ! grep -qE "^[^#]*[[:space:]]${SWAPFILE}[[:space:]]|^${SWAPFILE}[[:space:]]" /etc/fstab; then
  printf '%s none swap sw 0 0\n' "${SWAPFILE}" >>/etc/fstab
fi

cat >/etc/sysctl.d/99-family-manager.conf <<'EOF'
# See infra/bootstrap-vps.sh for the reasoning. Low-but-nonzero swappiness: use the swapfile for
# cold anonymous pages, but do not trade away the page cache Postgres depends on.
vm.swappiness = 10
vm.vfs_cache_pressure = 50
EOF
sysctl --quiet --load /etc/sysctl.d/99-family-manager.conf

# ---------------------------------------------------------------------------------------------
# 3. Docker Engine + compose v2, from Docker's own apt repository.
# ---------------------------------------------------------------------------------------------
# NOT the Ubuntu `docker.io` package: it ships the engine without the compose v2 plugin, and the
# whole deploy is `docker compose pull && docker compose up -d`. The `docker-compose` (v1, Python)
# package is not a substitute — the production overlay uses v2 syntax and `docker compose -f a -f b`
# merge semantics.
log "Installing Docker Engine"
apt-get update -qq
apt-get install -y -qq ca-certificates curl gnupg

install -m 0755 -d /etc/apt/keyrings
if [[ ! -s /etc/apt/keyrings/docker.asc ]]; then
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
fi
chmod a+r /etc/apt/keyrings/docker.asc

# Docker publishes one suite per Ubuntu codename, and it lands *after* the Ubuntu release, not
# with it. On a brand-new 26.04 box `$(lsb_release -cs)` very likely has no suite yet, and the
# failure mode if we write it blindly is an apt error about a missing Release file several steps
# later — obscure, and easy to misread as a network problem. So probe first and fall back to the
# newest suite Docker actually publishes.
#
# Falling back is safe: the packages are statically-ish built against a much older glibc than
# any of these releases carry, and the fallback is what Docker's own install script effectively
# does. It is still a temporary state — re-run this script once Docker ships the real suite so
# the box tracks its own release.
CODENAME="$(. /etc/os-release && echo "${UBUNTU_CODENAME:-${VERSION_CODENAME:-}}")"
[[ -n ${CODENAME} ]] || die "cannot determine the Ubuntu codename; set CODENAME by hand"

docker_suite_exists() {
  curl -fsSL --max-time 15 -o /dev/null "https://download.docker.com/linux/ubuntu/dists/$1/Release"
}

DOCKER_SUITE=""
for candidate in "${CODENAME}" questing plucky oracular noble; do
  if docker_suite_exists "${candidate}"; then
    DOCKER_SUITE="${candidate}"
    break
  fi
done

if [[ -z ${DOCKER_SUITE} ]]; then
  die "Docker publishes no apt suite we recognise, and download.docker.com may be unreachable.
     Check connectivity, then either wait for Docker to publish '${CODENAME}' or set
     DOCKER_SUITE by hand to a suite listed at https://download.docker.com/linux/ubuntu/dists/"
fi
if [[ ${DOCKER_SUITE} != "${CODENAME}" ]]; then
  warn "Docker has no '${CODENAME}' suite yet; using '${DOCKER_SUITE}' packages instead."
  warn "Re-run this script after Docker publishes '${CODENAME}' to move the box onto its own suite."
fi

echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu ${DOCKER_SUITE} stable" \
  >/etc/apt/sources.list.d/docker.list
apt-get update -qq
apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

systemctl enable --now docker
docker --version
docker compose version

# ---------------------------------------------------------------------------------------------
# 4. Docker daemon config — bounded logs, because 20 GB fills fast.
# ---------------------------------------------------------------------------------------------
# json-file capped at 10m x 3 per container. This is the *default* for anything started on this
# host; the production compose file sets the same limits per service explicitly. That duplication
# is deliberate and not redundant: the compose-level `logging:` block wins wherever it is set, so
# the file documents the cap next to the service it applies to and survives being run on a host
# that was not bootstrapped by this script. Where both exist they agree — if you change one,
# change the other, or a service will quietly get a different cap than its neighbours.
#
# `live-restore` keeps containers running across a dockerd restart (an apt upgrade of docker-ce,
# for instance), which on a single-node box is the difference between a package update and an
# outage. It is incompatible with swarm mode; we do not use swarm.
log "Configuring the Docker daemon"
mkdir -p /etc/docker
cat >/etc/docker/daemon.json <<'EOF'
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  },
  "live-restore": true
}
EOF
systemctl restart docker

# ---------------------------------------------------------------------------------------------
# 5. Deploy directory.
# ---------------------------------------------------------------------------------------------
# The repo is never cloned onto the VPS (it cannot build images here anyway — 1 core, 1.6 GB).
# This directory holds only what compose needs at runtime.
#
# secrets/ is 0700 root — no other host user can even traverse into it. Anyone who reads the
# ES256 signing key can mint tokens for every family in the system.
#
# The key file itself is 0400 owned by 65532:65532, NOT 0600 root:root. 65532 is the `nonroot`
# uid the distroless base image runs the service as. Compose bind-mounts a file-backed secret
# with its host ownership intact and silently ignores the long-syntax `uid`/`gid`/`mode`
# overrides (it warns that they "will be ignored"), and a top-level `mode:` makes it refuse to
# validate the file at all. So host ownership is the only lever there is: 0600 root:root arrives
# in the container as an unreadable file and auth dies at boot unable to read its signing key.
#
# 0600 root:root looks tighter and is the thing someone will "correct" this to. It is not
# tighter — 0400/65532 is read-only where 0600 is read-write, and the 0700 root directory above
# means no other host user reaches either one. (infra/secrets/README.md suggests 0444 for local
# dev and says not to do that in a real deployment; this is stricter than both.)
log "Creating ${DEPLOY_DIR}"
mkdir -p "${DEPLOY_DIR}/secrets" "${DEPLOY_DIR}/postgres-init"
chmod 0755 "${DEPLOY_DIR}" "${DEPLOY_DIR}/postgres-init"
chown root:root "${DEPLOY_DIR}" "${DEPLOY_DIR}/postgres-init" "${DEPLOY_DIR}/secrets"
chmod 0700 "${DEPLOY_DIR}/secrets"
# Repair a key copied in by an earlier run or an earlier operator. deploy.sh re-checks this
# before every start; this is the freshly-bootstrapped-box path, not the only line of defence.
find "${DEPLOY_DIR}/secrets" -type f -exec chown 65532:65532 {} + -exec chmod 0400 {} +

# ---------------------------------------------------------------------------------------------
# 6. Weekly image prune.
# ---------------------------------------------------------------------------------------------
# Every deploy pulls a new SHA-tagged image per service. Four services, a few deploys a week, and
# the old layers stay on disk forever until something reaps them — on 20 GB that is weeks, not
# years, and a full disk takes Postgres down with it.
#
# `docker image prune`, NEVER `docker system prune --volumes`. The pgdata and natsdata volumes are
# the only copy of the family's data on this box. `--volumes` would delete any volume no container
# currently references, and a compose file that is momentarily down during a deploy is exactly
# that situation. There is no undo. If you edit this unit, do not add --volumes.
#
# `until=168h` keeps the last week of images so a rollback to the previous SHA still has something
# to start; -a is what actually reclaims untagged parent layers.
log "Installing the weekly image prune timer"
cat >/etc/systemd/system/docker-image-prune.service <<'EOF'
[Unit]
Description=Prune unused Docker images (never volumes)
Documentation=file:///opt/family-manager
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
# Images only. Volumes hold pgdata/natsdata — pruning them destroys the database.
ExecStart=/usr/bin/docker image prune -af --filter until=168h
ExecStart=/usr/bin/docker builder prune -af --filter until=168h
EOF

cat >/etc/systemd/system/docker-image-prune.timer <<'EOF'
[Unit]
Description=Weekly Docker image prune

[Timer]
OnCalendar=Sun 04:00
# The box may be off or the timer may have been installed late; catch the missed run rather than
# waiting another week.
Persistent=true
RandomizedDelaySec=30m

[Install]
WantedBy=timers.target
EOF

systemctl daemon-reload
systemctl enable --now docker-image-prune.timer

# ---------------------------------------------------------------------------------------------
# 6b. Nightly database backup.
# ---------------------------------------------------------------------------------------------
# The unit is installed here so a rebuilt box has it from the start, but it only fires once
# backup.sh and the compose stack are actually in place — the script exits non-zero if postgres
# is not running, which is the correct behaviour for a box that has been bootstrapped but not
# yet deployed.
#
# Local-only, on the same disk as the database. That is an undo button for a bad migration or a
# mistaken DELETE, not disaster recovery: if this disk dies the backups die with it. The header
# of backup.sh says so at more length, and off-box copies remain an open gap.
log "Installing the nightly backup timer"
cat >/etc/systemd/system/family-manager-backup.service <<EOF
[Unit]
Description=Back up the family-manager Postgres databases
Documentation=file://${DEPLOY_DIR}/backup.sh
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
ExecStart=${DEPLOY_DIR}/backup.sh
# The box has one vCPU and 1.6 GiB. pg_dump competing with the API for either is how a backup
# turns into an outage, so it runs at the back of the queue for both.
Nice=10
IOSchedulingClass=idle
EOF

cat >/etc/systemd/system/family-manager-backup.timer <<'EOF'
[Unit]
Description=Nightly family-manager database backup

[Timer]
# 03:20, an hour before the Sunday image prune, so the two never contend for the single vCPU.
OnCalendar=*-*-* 03:20:00
# A missed night (box off, timer installed late) is caught on the next boot rather than skipped.
Persistent=true
RandomizedDelaySec=10m

[Install]
WantedBy=timers.target
EOF

systemctl daemon-reload
systemctl enable --now family-manager-backup.timer

# ---------------------------------------------------------------------------------------------
# 7. Unattended security updates.
# ---------------------------------------------------------------------------------------------
# Enabled, but security-only and with automatic reboots OFF.
#
# Worth spelling out why this is safe on a 1.6 GB box, since "skip it, too heavy" is the tempting
# call: unattended-upgrades runs once a day for a minute or two, and it now has 2 GiB of swap
# under it. The real risk was never memory, it is an unattended *restart* of something
# load-bearing. So:
#   - only the -security origin is enabled; Docker's repo is deliberately not listed, so
#     docker-ce is never upgraded (and dockerd never restarted) behind our back. Upgrade Docker
#     by re-running this script, when you are watching.
#   - Automatic-Reboot "false": a kernel update leaves /var/run/reboot-required for a human. An
#     unattended reboot here is a silent outage with no rolling deploy to hide it.
log "Enabling unattended security updates"
apt-get install -y -qq unattended-upgrades

cat >/etc/apt/apt.conf.d/51family-manager-unattended <<'EOF'
Unattended-Upgrade::Allowed-Origins {
    "${distro_id}:${distro_codename}-security";
    "${distro_id}ESMApps:${distro_codename}-apps-security";
    "${distro_id}ESM:${distro_codename}-infra-security";
};
// Docker's repo is intentionally absent: an unattended docker-ce upgrade restarts dockerd.
Unattended-Upgrade::Automatic-Reboot "false";
Unattended-Upgrade::Remove-Unused-Kernel-Packages "true";
Unattended-Upgrade::Remove-Unused-Dependencies "true";
EOF

cat >/etc/apt/apt.conf.d/20auto-upgrades <<'EOF'
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
EOF

# ---------------------------------------------------------------------------------------------
# What the operator does next.
# ---------------------------------------------------------------------------------------------
cat <<EOF

$(printf '\033[1;32m')Host is bootstrapped.$(printf '\033[0m')

  docker:   $(docker --version)
  compose:  $(docker compose version --short 2>/dev/null || echo unknown)
  swap:     $(free -h | awk '/Swap:/ {print $2" total"}')  (vm.swappiness=$(cat /proc/sys/vm/swappiness))
  firewall: 22, 80, 443 in; everything else denied
  prune:    weekly, images only — never volumes

Nothing is deployed yet. From your workstation, in the repo:

  1. Point DNS at this box. All four A records must resolve to 79.108.160.103 BEFORE the first
     start, or Caddy's ACME http-01 challenge fails and it will back off for a while:
         auth.nonamecat.pp.ua      A  79.108.160.103
         family.nonamecat.pp.ua    A  79.108.160.103
         finance.nonamecat.pp.ua   A  79.108.160.103
         recipes.nonamecat.pp.ua   A  79.108.160.103
     They currently point at 135.181.41.169. Verify with:
         getent ahostsv4 auth.nonamecat.pp.ua
     (getent is part of libc and is always present; dig would need the dnsutils package.)

  2. Copy the runtime files:
         scp infra/docker-compose.prod.yml infra/Caddyfile infra/nats.conf \\
             infra/deploy.sh infra/backup.sh infra/restore.sh \\
             root@79.108.160.103:${DEPLOY_DIR}/
         ssh root@79.108.160.103 'chmod 0755 ${DEPLOY_DIR}/{deploy,backup,restore}.sh'

     deploy.sh is not optional: the GitHub Actions deploy job SSHes in and runs
     ${DEPLOY_DIR}/deploy.sh <sha>. Skip it and the first automated deploy fails with
     "no such file", after CI has already pushed the images.

     backup.sh is what family-manager-backup.timer (installed above) runs every night.
     The timer is already enabled, so without the script the first 03:20 fires and fails.

     Then exactly ONE init file — not the whole directory:
         scp postgres/init/init-services.sql root@79.108.160.103:${DEPLOY_DIR}/postgres-init/

     Copy only init-services.sql. It creates the auth, family, finance and recipes databases. It is
     the only file in that directory today, and the copy stays a single named file rather than
     a glob: the pre-split leftovers that used to sit beside it (one creating schemas in the
     default database, one creating a role with a hardcoded password) were exactly the kind of
     thing a glob drags onto the box unnoticed. Postgres runs everything in
     /docker-entrypoint-initdb.d on first init, once, and only on an empty volume — so the
     damage is silent and only visible much later.

  3. Install the auth signing key (generate it once, keep the only other copy somewhere safe —
     losing it invalidates every issued token):
         openssl ecparam -name prime256v1 -genkey -noout -out infra/secrets/auth-signing-key.pem
         scp infra/secrets/auth-signing-key.pem root@79.108.160.103:${DEPLOY_DIR}/secrets/
         ssh root@79.108.160.103 'chown 65532:65532 ${DEPLOY_DIR}/secrets/auth-signing-key.pem \\
                                  && chmod 0400 ${DEPLOY_DIR}/secrets/auth-signing-key.pem'

     65532 is the distroless 'nonroot' uid the auth service runs as. Compose mounts the secret
     with host ownership and ignores uid/gid/mode overrides, so 0600 root:root — which looks
     safer — arrives unreadable and auth fails to start. Do not "tighten" it to 0600.

  4. Create ${DEPLOY_DIR}/.env on this box: Postgres credentials, the R2 keys/endpoint/bucket,
     and the image tag to deploy. It holds secrets — chmod 0600 it.

  5. Decide how this box authenticates to GHCR. The four packages are private by default, so
     a plain 'docker compose pull' gets 401 until one of these is true:

       a) Make the four ghcr.io/nonamecat19/family-manager/* packages public. The box then
          needs no registry credentials at all — nothing to store, nothing to rotate. The
          images contain only compiled service binaries; decide whether that is acceptable.

       b) Have the deploy inject a short-lived token per deploy. CI already holds an ephemeral
          GITHUB_TOKEN, so the deploy job can pass one in and log out afterwards, and no
          standing credential lives on the box. This is the intended shape; check with whoever
          owns the CI workflow and deploy.sh before wiring anything by hand.

     Only if neither is available yet, fall back to a long-lived read:packages PAT:
         docker login ghcr.io -u <github-user> --password-stdin
     Understand what that costs: the PAT lands in root's ~/.docker/config.json base64-encoded,
     not encrypted, and nothing rotates it. Treat it as a temporary measure and write down
     that it needs removing, or it will still be there in a year.

     Then bring the stack up:
         cd ${DEPLOY_DIR} && docker compose -f docker-compose.prod.yml up -d

  6. A VPS with no tested restore is worse than a managed platform (ADR 0004). Schedule the
     pg_dump job and do a restore drill before this box holds anything you care about.

EOF
