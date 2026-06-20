#!/usr/bin/env bash
#
# DigitarloPanel installer. Installs a prebuilt binary as a systemd service.
# Run as root:  sudo bash scripts/install.sh /path/to/digitarlopanel
#
set -euo pipefail

BINARY_SRC="${1:-./digitarlopanel}"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/digitarlopanel"
DATA_DIR="/var/lib/digitarlopanel"
SERVICE="/etc/systemd/system/digitarlopanel.service"
PORT="${DP_PORT:-8088}"

if [[ $EUID -ne 0 ]]; then
  echo "Please run as root (sudo)." >&2
  exit 1
fi

if [[ ! -f "$BINARY_SRC" ]]; then
  echo "Binary not found: $BINARY_SRC" >&2
  echo "Build it first with 'make build'." >&2
  exit 1
fi

echo "Installing binary to $INSTALL_DIR/digitarlopanel"
install -m 0755 "$BINARY_SRC" "$INSTALL_DIR/digitarlopanel"

mkdir -p "$CONFIG_DIR" "$DATA_DIR"
chmod 0750 "$CONFIG_DIR" "$DATA_DIR"

echo "Writing systemd unit to $SERVICE"
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

echo
echo "DigitarloPanel installed and started on port $PORT."
echo "View the generated admin password with:"
echo "  journalctl -u digitarlopanel --no-pager | grep -A2 'first run'"
