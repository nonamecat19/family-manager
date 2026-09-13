#!/usr/bin/env bash

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

log "Configuring ufw"
if ! command -v ufw >/dev/null 2>&1; then
  apt-get update -qq
  apt-get install -y -qq ufw
fi

ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp   comment 'ssh — must exist before enable'
ufw allow 80/tcp   comment 'http, and ACME http-01 for Caddy'
ufw allow 443/tcp  comment 'https'
ufw --force enable
ufw status verbose

log "Configuring swap"
if swapon --show --noheadings | grep -q .; then
  warn "swap already active, leaving it alone:"
  swapon --show
elif [[ -e ${SWAPFILE} ]]; then
  warn "${SWAPFILE} exists but is not active — not touching it; inspect it by hand"
else
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
vm.swappiness = 10
vm.vfs_cache_pressure = 50
EOF
sysctl --quiet --load /etc/sysctl.d/99-family-manager.conf

log "Installing Docker Engine"
apt-get update -qq
apt-get install -y -qq ca-certificates curl gnupg

install -m 0755 -d /etc/apt/keyrings
if [[ ! -s /etc/apt/keyrings/docker.asc ]]; then
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
fi
chmod a+r /etc/apt/keyrings/docker.asc

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

log "Creating ${DEPLOY_DIR}"
mkdir -p "${DEPLOY_DIR}/secrets" "${DEPLOY_DIR}/postgres-init"
chmod 0755 "${DEPLOY_DIR}" "${DEPLOY_DIR}/postgres-init"
chown root:root "${DEPLOY_DIR}" "${DEPLOY_DIR}/postgres-init" "${DEPLOY_DIR}/secrets"
chmod 0700 "${DEPLOY_DIR}/secrets"
find "${DEPLOY_DIR}/secrets" -type f -exec chown 65532:65532 {} + -exec chmod 0400 {} +

log "Installing the weekly image prune timer"
cat >/etc/systemd/system/docker-image-prune.service <<'EOF'
[Unit]
Description=Prune unused Docker images (never volumes)
Documentation=file:///opt/family-manager
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
ExecStart=/usr/bin/docker image prune -af --filter until=168h
ExecStart=/usr/bin/docker builder prune -af --filter until=168h
EOF

cat >/etc/systemd/system/docker-image-prune.timer <<'EOF'
[Unit]
Description=Weekly Docker image prune

[Timer]
OnCalendar=Sun 04:00
Persistent=true
RandomizedDelaySec=30m

[Install]
WantedBy=timers.target
EOF

systemctl daemon-reload
systemctl enable --now docker-image-prune.timer

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
Nice=10
IOSchedulingClass=idle
EOF

cat >/etc/systemd/system/family-manager-backup.timer <<'EOF'
[Unit]
Description=Nightly family-manager database backup

[Timer]
OnCalendar=*-*-* 03:20:00
Persistent=true
RandomizedDelaySec=10m

[Install]
WantedBy=timers.target
EOF

systemctl daemon-reload
systemctl enable --now family-manager-backup.timer

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
         scp postgres/init/init-services.sh root@79.108.160.103:${DEPLOY_DIR}/postgres-init/

     Copy only init-services.sh. It creates the auth, family, finance, notes and recipes
     databases, and it skips any that already exist, so it is safe to re-run by hand. It is
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
