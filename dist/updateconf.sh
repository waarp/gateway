#!/usr/bin/env bash

exec >/tmp/gw.log 2>&1

CONFIG_PACKAGE="$1"
CONFIG_FILENAME="$(basename "$CONFIG_PACKAGE")"
INST_NAME="${CONFIG_FILENAME%%-*}"
TMP_DIR=$(mktemp -d)


trap 'rm -rf "$TMP_DIR"' exit

CURDIR=$(cd "$(dirname "$0")" && pwd)

cd "$TMP_DIR" || exit 2
unzip "$1"
"$CURDIR/../bin/waarp-gatewayd" import -v -c "$CURDIR/../etc/waarp-gatewayd.ini" -s "${INST_NAME}.json" 2>&1
cat "${INST_NAME}.json" > /tmp/gw.json
cp "$1" /tmp
cd || exit 2
