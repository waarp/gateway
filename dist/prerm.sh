#!/bin/sh
#
# Runs before the package files are removed, on both dpkg and rpm.

set -e

# Only on a real uninstall. dpkg calls this with "upgrade" and rpm with "1"
# during an upgrade, and stopping the service there would turn a plain version
# bump into an unannounced outage: nothing in this package ever starts it
# again.
case "${1:-}" in
    remove | 0) ;;
    *) exit 0 ;;
esac

if command -v systemctl >/dev/null 2>&1; then
    systemctl stop waarp-gatewayd.service >/dev/null 2>&1 || :
    systemctl stop waarp-gateway-get-remote.timer >/dev/null 2>&1 || :
fi
