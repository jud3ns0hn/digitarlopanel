#!/usr/bin/env bash
#
# DigitarloPanel one-command installer.
#
# Run on a fresh VPS (Ubuntu/Debian or RHEL/Rocky/CentOS/Alma) as root. By
# default it DOWNLOADS a prebuilt, self-contained binary (no Go/Node, no build)
# and installs it as a systemd service in seconds — this is the recommended path
# and cannot exhaust memory on small VPSes.
#
# Quick start:
#   curl -fsSL https://raw.githubusercontent.com/jud3ns0hn/digitarlopanel/claude/server-admin-program-h026hq/scripts/bootstrap.sh | sudo bash
#
# Build from source instead of downloading (needs more RAM; adds swap if low):
#   ... | sudo DP_BUILD=1 bash
#
# Tunables (environment variables):
#   DP_PORT=8088     panel port
#   DP_BRANCH=...    git branch for downloads / source build
#   DP_REPO=<url>    git URL (source build)
#   DP_BUILD=1       force building from source
#   DP_SRC=/opt/...  where to clone for a source build
#
set -euo pipefail

BRANCH="${DP_BRANCH:-claude/server-admin-program-h026hq}"
REPO_SLUG="${DP_REPO_SLUG:-jud3ns0hn/digitarlopanel}"
REPO_URL="${DP_REPO:-https://github.com/${REPO_SLUG}.git}"
# Use the refs/heads form so branch names containing a slash resolve correctly.
RAW_BASE="${DP_RAW_BASE:-https://raw.githubusercontent.com/${REPO_SLUG}/refs/heads/${BRANCH}}"
PORT="${DP_PORT:-8088}"
SRC_DIR="${DP_SRC:-/opt/digitarlopanel-src}"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/digitarlopanel"
DATA_DIR="/var/lib/digitarlopanel"
SERVICE="/etc/systemd/system/digitarlopanel.service"
GO_MIN="1.25.0"
NODE_MAJOR="20"

log()  { printf '\033[1;36m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[!]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m[x]\033[0m %s\n' "$*" >&2; exit 1; }

[[ $EUID -eq 0 ]] || die "Please run as root (sudo)."

# --- detect distro family ---------------------------------------------------
FAMILY="unknown"
if [[ -r /etc/os-release ]]; then
  . /etc/os-release
  case "${ID:-} ${ID_LIKE:-}" in
    *debian*|*ubuntu*) FAMILY="debian" ;;
    *rhel*|*fedora*|*centos*|*rocky*|*almalinux*) FAMILY="rhel" ;;
  esac
fi
[[ "$FAMILY" != "unknown" ]] || die "Unsupported distribution (need Debian/Ubuntu or RHEL/Rocky/CentOS/Alma)."

# --- detect CPU arch --------------------------------------------------------
case "$(uname -m)" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) die "Unsupported CPU architecture: $(uname -m)" ;;
esac
log "Distribution: $FAMILY ($ARCH)"

pkg_install() {
  if [[ "$FAMILY" == "debian" ]]; then
    DEBIAN_FRONTEND=noninteractive apt-get install -y "$@"
  elif command -v dnf >/dev/null 2>&1; then dnf install -y "$@"; else yum install -y "$@"; fi
}

install_service() {
  local binary="$1"
  log "Installing binary to $INSTALL_DIR/digitarlopanel"
  install -m 0755 "$binary" "$INSTALL_DIR/digitarlopanel"
  mkdir -p "$CONFIG_DIR" "$DATA_DIR"
  chmod 0750 "$CONFIG_DIR" "$DATA_DIR"

  log "Writing systemd unit"
  cat > "$SERVICE" <<EOF
[Unit]
Description=DigitarloPanel server administration
After=network.target

[Service]
Type=simple
ExecStart=$INSTALL_DIR/digitarlopanel -config $CONFIG_DIR/config.json -listen :$PORT
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload
  systemctl enable --now digitarlopanel
}

open_firewall() {
  if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q "Status: active"; then
    log "Opening port $PORT in ufw"; ufw allow "${PORT}/tcp" || warn "ufw rule failed"
  elif command -v firewall-cmd >/dev/null 2>&1 && firewall-cmd --state >/dev/null 2>&1; then
    log "Opening port $PORT in firewalld"
    firewall-cmd --permanent --add-port="${PORT}/tcp" >/dev/null 2>&1 || warn "firewalld rule failed"
    firewall-cmd --reload >/dev/null 2>&1 || true
  elif command -v iptables >/dev/null 2>&1; then
    # Oracle Cloud / some cloud Ubuntu images ship a raw-iptables firewall with a
    # default REJECT rule and no ufw/firewalld. Insert an ACCEPT before the first
    # REJECT/DROP so the panel port is reachable, then persist it.
    if ! iptables -C INPUT -p tcp --dport "$PORT" -j ACCEPT 2>/dev/null; then
      local pos
      pos="$(iptables -L INPUT --line-numbers -n 2>/dev/null | awk '/REJECT|DROP/{print $1; exit}')"
      if [[ -n "$pos" ]]; then
        log "Opening port $PORT in iptables (before rule $pos)"
        iptables -I INPUT "$pos" -p tcp --dport "$PORT" -j ACCEPT || warn "iptables rule failed"
      else
        log "Appending iptables ACCEPT for port $PORT"
        iptables -A INPUT -p tcp --dport "$PORT" -j ACCEPT || warn "iptables rule failed"
      fi
      # Persist across reboots (best effort).
      if command -v netfilter-persistent >/dev/null 2>&1; then
        netfilter-persistent save >/dev/null 2>&1 || true
      elif [[ -d /etc/iptables ]]; then
        iptables-save > /etc/iptables/rules.v4 2>/dev/null || true
      fi
    fi
  fi
}

print_summary() {
  local ip pw
  ip="$(curl -fsSL --max-time 5 https://api.ipify.org 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}')"
  sleep 1
  pw="$(journalctl -u digitarlopanel --no-pager 2>/dev/null | grep -m1 'Password:' | awk '{print $2}' || true)"
  cat <<EOF

============================================================
 DigitarloPanel is installed and running.

   URL:       http://${ip:-<server-ip>}:${PORT}
   Username:  admin
   Password:  ${pw:-<see command below>}

 If the password is not shown, run:
   journalctl -u digitarlopanel --no-pager | grep -A2 'first run'

 Manage the service:
   systemctl status digitarlopanel
   systemctl restart digitarlopanel
============================================================
EOF
}

# ============================================================================
#  FAST PATH: download a prebuilt binary (default)
# ============================================================================
# fetch_dist <remote-name> <output-path>: download dist/<remote-name>. Uses the
# GitHub API with a token when DP_TOKEN is set (works for PRIVATE repos), else
# the public raw URL.
fetch_dist() {
  local name="$1" out="$2"
  if [[ -n "${DP_TOKEN:-}" ]]; then
    curl -fSL --retry 3 \
      -H "Authorization: Bearer ${DP_TOKEN}" \
      -H "Accept: application/vnd.github.raw" \
      -o "$out" \
      "https://api.github.com/repos/${REPO_SLUG}/contents/dist/${name}?ref=${BRANCH}"
  else
    curl -fSL --retry 3 -o "$out" "${RAW_BASE}/dist/${name}"
  fi
}

download_install() {
  command -v curl >/dev/null 2>&1 || pkg_install curl ca-certificates
  command -v gzip >/dev/null 2>&1 || pkg_install gzip

  local tmp; tmp="$(mktemp -d)"
  log "Downloading prebuilt binary ($ARCH)${DP_TOKEN:+ via GitHub API (private)}"
  if ! fetch_dist "digitarlopanel-linux-${ARCH}.gz" "$tmp/dp.gz"; then
    rm -rf "$tmp"; return 1
  fi

  # Verify checksum if the manifest is reachable (best effort).
  if fetch_dist "SHA256SUMS" "$tmp/SHA256SUMS" 2>/dev/null; then
    local want got
    want="$(grep "digitarlopanel-linux-${ARCH}.gz" "$tmp/SHA256SUMS" | awk '{print $1}')"
    got="$(sha256sum "$tmp/dp.gz" | awk '{print $1}')"
    if [[ -n "$want" && "$want" != "$got" ]]; then
      rm -rf "$tmp"; die "Checksum mismatch for downloaded binary — aborting."
    fi
    log "Checksum verified"
  fi

  gzip -d "$tmp/dp.gz"
  chmod +x "$tmp/dp"
  install_service "$tmp/dp"
  rm -rf "$tmp"
}

# ============================================================================
#  PRIVATE-REPO PATH: clone via SSH deploy key / token, install prebuilt binary
# ============================================================================
# Set DP_SSH=1 to clone over SSH (deploy key). Point DP_SSH_KEY at the key file
# if it is not in root's ~/.ssh (e.g. DP_SSH_KEY=/home/ubuntu/.ssh/deploy_key).
clone_install() {
  command -v git >/dev/null 2>&1 || pkg_install git
  command -v gzip >/dev/null 2>&1 || pkg_install gzip
  export GIT_TERMINAL_PROMPT=0   # never hang on an interactive credential prompt

  local url
  if [[ "${DP_SSH:-0}" == "1" ]]; then
    url="git@github.com:${REPO_SLUG}.git"
    [[ -n "${DP_SSH_KEY:-}" ]] && export GIT_SSH_COMMAND="ssh -i ${DP_SSH_KEY} -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new"
  elif [[ -n "${DP_TOKEN:-}" ]]; then
    url="https://x-access-token:${DP_TOKEN}@github.com/${REPO_SLUG}.git"
  else
    return 1   # no private credentials available
  fi

  local tmp; tmp="$(mktemp -d)"
  log "Cloning ${REPO_SLUG} (${BRANCH}) for prebuilt binary"
  if ! git clone --depth 1 --branch "$BRANCH" "$url" "$tmp/repo" 2>/dev/null; then
    rm -rf "$tmp"; return 1
  fi
  local gz="$tmp/repo/dist/digitarlopanel-linux-${ARCH}.gz"
  if [[ -f "$gz" ]]; then
    gzip -dc "$gz" > "$tmp/dp" && chmod +x "$tmp/dp"
    install_service "$tmp/dp"; rm -rf "$tmp"; return 0
  fi
  rm -rf "$tmp"; return 1
}

# ============================================================================
#  SOURCE BUILD (fallback / DP_BUILD=1): adds swap, builds, installs
# ============================================================================
version_ge() { [[ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -n1)" == "$2" ]]; }

ensure_swap() {
  # The frontend build is memory hungry; on < 2 GB RAM add a temporary swapfile
  # so the build cannot OOM-kill the box (which can also drop your SSH session).
  local mem_kb swap_kb
  mem_kb="$(awk '/MemTotal/{print $2}' /proc/meminfo)"
  swap_kb="$(awk '/SwapTotal/{print $2}' /proc/meminfo)"
  if (( mem_kb < 2100000 && swap_kb < 1000000 )); then
    if [[ ! -f /swapfile ]]; then
      log "Low RAM detected — creating a 2G swapfile to protect the build"
      if fallocate -l 2G /swapfile 2>/dev/null || dd if=/dev/zero of=/swapfile bs=1M count=2048 status=none; then
        chmod 600 /swapfile && mkswap /swapfile >/dev/null && swapon /swapfile || warn "swap setup failed"
      fi
    fi
  fi
}

ensure_go() {
  if command -v go >/dev/null 2>&1; then
    local cur; cur="$(go version | awk '{print $3}' | sed 's/^go//')"
    version_ge "$cur" "$GO_MIN" && { log "Go $cur present"; return; }
  fi
  local gover; gover="$(curl -fsSL 'https://go.dev/VERSION?m=text' | head -n1 || true)"
  [[ "$gover" == go* ]] || gover="go1.25.4"
  log "Installing $gover ($ARCH)"
  curl -fsSL "https://go.dev/dl/${gover}.linux-${ARCH}.tar.gz" -o /tmp/go.tar.gz
  rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go.tar.gz && rm -f /tmp/go.tar.gz
  export PATH="/usr/local/go/bin:$PATH"
  echo 'export PATH=/usr/local/go/bin:$PATH' > /etc/profile.d/go.sh
}

ensure_node() {
  if command -v node >/dev/null 2>&1; then
    local major; major="$(node -v | sed 's/^v//' | cut -d. -f1)"
    (( ${major:-0} >= 18 )) && { log "Node.js $(node -v) present"; return; }
  fi
  log "Installing Node.js $NODE_MAJOR via NodeSource"
  if [[ "$FAMILY" == "debian" ]]; then
    curl -fsSL "https://deb.nodesource.com/setup_${NODE_MAJOR}.x" | bash - ; pkg_install nodejs
  else
    curl -fsSL "https://rpm.nodesource.com/setup_${NODE_MAJOR}.x" | bash - ; pkg_install nodejs
  fi
}

source_build() {
  log "Building from source"
  if [[ "$FAMILY" == "debian" ]]; then
    DEBIAN_FRONTEND=noninteractive apt-get update -y
    pkg_install ca-certificates curl git tar gzip build-essential
  else
    pkg_install ca-certificates curl git tar gzip gcc gcc-c++ make
  fi
  ensure_swap
  ensure_go
  ensure_node
  export PATH="/usr/local/go/bin:$PATH"

  export GIT_TERMINAL_PROMPT=0   # fail fast instead of prompting for credentials
  local clone_url="$REPO_URL"
  if [[ "${DP_SSH:-0}" == "1" ]]; then
    clone_url="git@github.com:${REPO_SLUG}.git"
    [[ -n "${DP_SSH_KEY:-}" ]] && export GIT_SSH_COMMAND="ssh -i ${DP_SSH_KEY} -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new"
  elif [[ -n "${DP_TOKEN:-}" ]]; then
    clone_url="https://x-access-token:${DP_TOKEN}@github.com/${REPO_SLUG}.git"
  fi

  local root
  local sd; sd="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd || true)"
  if [[ -n "$sd" && -f "$sd/../backend/go.mod" ]]; then
    root="$(cd "$sd/.." && pwd)"
  else
    if [[ -d "$SRC_DIR/.git" ]]; then
      git -C "$SRC_DIR" fetch --depth 1 origin "$BRANCH"
      git -C "$SRC_DIR" reset --hard "origin/$BRANCH"
    else
      git clone --depth 1 --branch "$BRANCH" "$clone_url" "$SRC_DIR" \
        || die "git clone failed. For a PRIVATE repo, set DP_TOKEN=<github-pat> or DP_SSH=1 (with DP_SSH_KEY), or make the repo public."
    fi
    root="$SRC_DIR"
  fi

  # Cap Node heap and skip type-checking for a lighter, faster production build.
  export NODE_OPTIONS="--max-old-space-size=1024"
  ( cd "$root/frontend" && npm install --no-audit --no-fund && npm run build )
  ( cd "$root/backend" && CGO_ENABLED=0 go build -ldflags "-s -w" -o "$root/digitarlopanel" ./cmd/digitarlopanel )
  [[ -x "$root/digitarlopanel" ]] || die "Build did not produce a binary"
  install_service "$root/digitarlopanel"
}

# --- dispatch ---------------------------------------------------------------
if [[ "${DP_BUILD:-0}" == "1" ]]; then
  source_build
elif download_install; then
  :
elif clone_install; then
  :
else
  warn "Prebuilt install failed — falling back to building from source"
  source_build
fi

open_firewall
print_summary
