#!/bin/bash
# Install only a loopback-bound, authenticated personal instance.
# The HTTPS reverse proxy is configured separately after owner review.
set -euo pipefail
[[ "$(id -u)" = 0 ]] || { echo 'Run as root on the intended server.' >&2; exit 1; }
[[ "$(uname -m)" = x86_64 ]] || { echo 'This bundle requires x86_64.' >&2; exit 1; }
cd "$(dirname "$0")"
sha256sum --check SHA256SUMS.txt
if systemctl cat zhile-personal.service >/dev/null 2>&1 || [[ -e /opt/zhile-personal/zhile ]]; then
    echo 'An installation already exists; use an explicit upgrade procedure.' >&2
    exit 1
fi
if ss -ltn | grep -q ':37910 '; then echo 'Port 37910 is already in use.' >&2; exit 1; fi
getent passwd zhile-personal >/dev/null || useradd --system --home-dir /var/lib/zhile-personal --shell /sbin/nologin zhile-personal
install -d -m 755 /opt/zhile-personal
install -d -m 700 -o zhile-personal -g zhile-personal /var/lib/zhile-personal
install -m 755 zhile /opt/zhile-personal/zhile
install -m 644 LICENSE NOTICE /opt/zhile-personal/
cat >/etc/systemd/system/zhile-personal.service <<'UNIT'
[Unit]
Description=Zhile private music space
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=zhile-personal
Group=zhile-personal
WorkingDirectory=/var/lib/zhile-personal
Environment=MUSIC_DL_CONFIG_DB=/var/lib/zhile-personal/data/settings.db
Environment=MUSIC_DL_COOKIE_FILE=/var/lib/zhile-personal/data/cookies.json
ExecStart=/opt/zhile-personal/zhile web --host 127.0.0.1 --port 37910 --private --secure-cookies --no-browser
Restart=on-failure
RestartSec=5
UMask=0077
NoNewPrivileges=true
PrivateTmp=true
ProtectHome=true
ProtectSystem=full
ReadWriteDirectories=/var/lib/zhile-personal

[Install]
WantedBy=multi-user.target
UNIT
systemctl daemon-reload
systemctl enable --now zhile-personal.service
systemctl is-active zhile-personal.service
