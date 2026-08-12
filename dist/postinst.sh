#!/bin/sh
#
# Runs after the package files are unpacked, on both dpkg and rpm.

set -e

case "${1:-}" in
    configure | 1 | 2) ;;
    *) exit 0 ;;
esac

CONFDIR=/etc/waarp-gateway
DATADIR=/var/lib/waarp-gateway
LOGDIR=/var/log/waarp-gateway
SHAREDIR=/usr/share/waarp-gateway

#
# Ownership and permissions.
#
# These are also declared in dist/nfpm.yaml, and the redundancy is deliberate:
# dpkg never re-applies the archive's modes or owners on an upgrade -- not even
# to a conffile whose contents it has just replaced -- while rpm resets
# everything it ships. Declaring alone would leave every upgraded Debian host
# on the old, wrong permissions while fresh installs and all RPM hosts got the
# new ones, which is the worst possible way for a defect to be distributed.
#
# dpkg also never resets a directory that already exists, so the directories
# below have to be repaired explicitly: releases up to 0.16.0 left SHAREDIR at
# 0750 root:waarp, which makes the compatibility shims unreachable for anyone
# outside the waarp group -- that is, precisely on the upgraded hosts the shims
# exist to serve.
#

# The daemon writes into CONFDIR as the waarp user: the AES passphrase on first
# start, the cluster override file, and fw.json / get-file.list from the
# UPDATECONF task. It must be group-writable, or a fresh install never starts
# at all -- the AES key is loaded before the database is even created.
#
# The sticky bit is load-bearing: without it, group write on the directory
# would let the daemon account unlink or rename any file here whatever its own
# mode, gatewayd.ini and administrator-supplied keys included. Every file the
# daemon writes it also owns, and it opens them in place, so 01770 costs it
# nothing.
chown root:waarp "$CONFDIR"
chmod 01770 "$CONFDIR"

# Named files only, never "chown -R": a recursive chown here would hand the
# daemon account every TLS or SSH private key the administrator has placed in
# this directory, which is what earlier releases did on every upgrade.
#
# passphrase.aes is in the list although the package does not ship it. It
# encrypts every stored credential and the PGP private keys, and it is created
# by the daemon -- but an administrator who ran an export as root before the
# first start ends up with a root-owned copy the daemon cannot read, and the
# service then fails to start for good.
for name in gatewayd.ini passphrase.aes; do
    if [ -f "$CONFDIR/$name" ]; then
        chown root:waarp "$CONFDIR/$name"
        chmod 0640 "$CONFDIR/$name"
    fi
done

# Read by systemd as root, before it drops to User=waarp, and it carries the
# REST credentials in a URL. The waarp account has no business reading it.
if [ -f /etc/default/waarp-gateway-get-remote ]; then
    chown root:root /etc/default/waarp-gateway-get-remote
    chmod 0640 /etc/default/waarp-gateway-get-remote
fi

chown waarp:waarp "$DATADIR" "$LOGDIR"
chmod 0750 "$DATADIR" "$LOGDIR"

for name in in out tmp db; do
    if [ -d "$DATADIR/$name" ]; then
        chown waarp:waarp "$DATADIR/$name"
        chmod 0750 "$DATADIR/$name"
    fi
done

chown root:root "$SHAREDIR"
chmod 0755 "$SHAREDIR"

# db/ is the daemon's alone, so the package asserts its state on every
# configure rather than only repairing it once. Saying "one-time" here would be
# a lie the code cannot keep: dpkg gives the previous version in $2 but rpm
# gives only an instance count, so there is no portable way to run this on
# upgrades from a specific release, and a stamp file would just be undeclared
# state in a packaged directory.
#
# What it repairs today is what "chmod -R 774 /var/lib/waarp-gateway" left
# behind in releases up to 0.16.0: database files that are group-writable and
# executable. Neither packager will ever correct those, since neither ships
# them.
#
# Scoped to db/ on purpose. Files under in/, out/ and tmp/ belong to the
# administrator's own workflows and are routinely consumed by other accounts,
# so their permissions are not ours to rewrite.
if [ -d "$DATADIR/db" ]; then
    find "$DATADIR/db" -type f -exec chmod 0640 {} + -exec chown waarp:waarp {} +
fi

# systemctl is absent from container images and from non-systemd systems, and
# a failed reload must not fail the installation.
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload >/dev/null 2>&1 || :
fi

# The service is deliberately neither enabled nor started: the database has to
# be migrated before the daemon may run against it.
echo ">>> Upgrade the database before restarting the service."
echo ">>> waarp-gatewayd migrate -c /etc/waarp-gateway/gatewayd.ini"
echo ">>> More information on the migration process at \"https://doc.waarp.org/waarp-gateway/latest/fr/migrate.html\""
