#!/usr/bin/env bash
#
# DigitarloPanel one-command bootstrap.
#
# Run on a fresh VPS (Ubuntu/Debian or RHEL/Rocky/CentOS/Alma) as root. It
# installs all build dependencies, builds the self-contained binary from source,
# installs it as a systemd service, opens the firewall and prints the generated
# admin password.
#
# Quick start (clones the repo automatically):
#   curl -fsSL https://raw.githubusercontent.com/jud3ns0hn/digitarlopanel/claude/server-admin-program-h026hq/scripts/bootstrap.sh | sudo bash
#
# Or from a local checkout:
#   sudo bash scripts/bootstrap.sh
#
# Tunables (environment variables):
#   DP_PORT=8088           panel port
#   DP_BRANCH=...          git branch to build
#   DP_REPO=<url>          git URL to clone
#   DP_SRC=/opt/...        where to clone/build
#
set -euo pipefail

REPO_URL="${DP_REPO:-https://github.com/jud3ns0hn/digitarlopanel.git}"
BRANCH="${DP_BRANCH:-claude/server-admin-program-h026hq}"
PORT="${DP_PORT:-8088}"
SRC_DIR="${DP_SRC:-/opt/digitarlopanel-src}"
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
log "Detected distribution family: $FAMILY"

# --- package helpers --------------------------------------------------------
pkg_install() {
  if [[ "$FAMILY" == "debian" ]]; then
    DEBIAN_FRONTEND=noninteractive apt-get install -y "$@"
  else
    if command -v dnf >/dev/null 2>&1; then dnf install -y "$@"; else yum install -y "$@"; fi
  fi
}

log "Refreshing package metadata"
if [[ "$FAMILY" == "debian" ]]; then
  DEBIAN_FRONTEND=noninteractive apt-get update -y
fi

log "Installing base build tools"
if [[ "$FAMILY" == "debian" ]]; then
  pkg_install ca-certificates curl git tar gzip build-essential
else
  pkg_install ca-certificates curl git tar gzip gcc gcc-c++ make
fi

# --- version compare (a >= b) ----------------------------------------------
version_ge() { [[ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -n1)" == "$2" ]]; }

# --- ensure a recent Go (distro packages are too old for go.mod) ------------
ensure_go() {
  if command -v go >/dev/null 2>&1; then
    local cur; cur="$(go version | awk '{print $3}' | sed 's/^go//')"
    if version_ge "$cur" "$GO_MIN"; then
      log "Go $cur already present (>= $GO_MIN)"; return
    fi
    warn "Go $cur is older than $GO_MIN — installing a newer toolchain"
  fi
  local goarch
  case "$(uname -m)" in
    x86_64|amd64) goarch="amd64" ;;
    aarch64|arm64) goarch="arm64" ;;
    *) die "Unsupported CPU architecture for Go: $(uname -m)" ;;
  esac
  local gover
  gover="$(curl -fsSL 'https://go.dev/VERSION?m=text' | head -n1 || true)"
  [[ "$gover" == go* ]] || gover="go1.25.4"
  log "Installing $gover ($goarch) from go.dev"
  curl -fsSL "https://go.dev/dl/${gover}.linux-${goarch}.tar.gz" -o /tmp/go.tar.gz
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tar.gz
  rm -f /tmp/go.tar.gz
  export PATH="/usr/local/go/bin:$PATH"
  # Persist for future shells.
  echo 'export PATH=/usr/local/go/bin:$PATH' > /etc/profile.d/go.sh
  go version
}

# --- ensure a modern Node.js (for the Vite build) ---------------------------
ensure_node() {
  if command -v node >/dev/null 2>&1; then
    local major; major="$(node -v | sed 's/^v//' | cut -d. -f1)"
    if [[ "${major:-0}" -ge 18 ]]; then
      log "Node.js $(node -v) already present"; return
    fi
    warn "Node.js $(node -v) is too old — installing Node $NODE_MAJOR"
  fi
  log "Installing Node.js $NODE_MAJOR via NodeSource"
  if [[ "$FAMILY" == "debian" ]]; then
    curl -fsSL "https://deb.nodesource.com/setup_${NODE_MAJOR}.x" | bash -
    pkg_install nodejs
  else
    curl -fsSL "https://rpm.nodesource.com/setup_${NODE_MAJOR}.x" | bash -
    pkg_install nodejs
  fi
  node -v
}

ensure_go
ensure_node
export PATH="/usr/local/go/bin:$PATH"

# --- locate source (use local checkout if present, else clone) --------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd || true)"
if [[ -n "$SCRIPT_DIR" && -f "$SCRIPT_DIR/../backend/go.mod" ]]; then
  REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
  log "Building from local checkout: $REPO_ROOT"
else
  if [[ -d "$SRC_DIR/.git" ]]; then
    log "Updating existing checkout in $SRC_DIR"
    git -C "$SRC_DIR" fetch --depth 1 origin "$BRANCH"
    git -C "$SRC_DIR" checkout "$BRANCH"
    git -C "$SRC_DIR" reset --hard "origin/$BRANCH"
  else
    log "Cloning $REPO_URL ($BRANCH) into $SRC_DIR"
    git clone --depth 1 --branch "$BRANCH" "$REPO_URL" "$SRC_DIR"
  fi
  REPO_ROOT="$SRC_DIR"
fi

# --- build ------------------------------------------------------------------
log "Building frontend and backend (this can take a few minutes)"
make -C "$REPO_ROOT" build

BINARY="$REPO_ROOT/digitarlopanel"
[[ -x "$BINARY" ]] || die "Build did not produce a binary at $BINARY"

# --- install as systemd service ---------------------------------------------
log "Installing binary and systemd service"
DP_PORT="$PORT" bash "$REPO_ROOT/scripts/install.sh" "$BINARY"

# --- open firewall (best effort) --------------------------------------------
if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q "Status: active"; then
  log "Opening port $PORT in ufw"
  ufw allow "${PORT}/tcp" || warn "ufw rule failed"
elif command -v firewall-cmd >/dev/null 2>&1 && firewall-cmd --state >/dev/null 2>&1; then
  log "Opening port $PORT in firewalld"
  firewall-cmd --permanent --add-port="${PORT}/tcp" >/dev/null 2>&1 || warn "firewalld rule failed"
  firewall-cmd --reload >/dev/null 2>&1 || true
fi

# --- summary ----------------------------------------------------------------
IP="$(curl -fsSL --max-time 5 https://api.ipify.org 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}')"
sleep 1
PW="$(journalctl -u digitarlopanel --no-pager 2>/dev/null | grep -m1 'Password:' | awk '{print $2}' || true)"

cat <<EOF

============================================================
 DigitarloPanel is installed and running.

   URL:       http://${IP:-<server-ip>}:${PORT}
   Username:  admin
   Password:  ${PW:-<see command below>}

 If the password is not shown above, run:
   journalctl -u digitarlopanel --no-pager | grep -A2 'first run'

 Manage the service:
   systemctl status digitarlopanel
   systemctl restart digitarlopanel

 Change the admin password after your first login.
============================================================
EOF
