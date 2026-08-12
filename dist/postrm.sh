#!/bin/sh
#
# Runs after the package files are removed, on both dpkg and rpm.

set -e

# dpkg also calls this with "upgrade" between the two versions of a plain
# version bump. Acting there would, for instance, disable a service that the
# administrator had enabled -- so everything below is confined to a genuine
# removal.
case "${1:-}" in
    remove | purge | 0) ;;
    *) exit 0 ;;
esac

if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload >/dev/null 2>&1 || :
fi

#
# Nothing is deleted from /etc/waarp-gateway, not even on purge, and that is a
# decision rather than an omission.
#
# /etc/waarp-gateway/passphrase.aes is not a conffile, so it survives a purge
# today and has done so in every release. It is the master key for every stored
# credential and for the PGP and AES private keys in the crypto_keys table.
# Meanwhile /var/lib/waarp-gateway/db is non-empty on any real installation, so
# dpkg keeps it. Deleting the key while keeping the data it protects gives a
# gateway that reinstalls cleanly, starts with no error at all, shows its rules
# and users intact -- and fails on the first transfer with an authentication
# error, unrecoverably. Restoring the key from a backup does not repair it
# either, since anything written under the replacement key then becomes
# unreadable in turn.
#
# The waarp account is kept for the same reason: it is what makes a purge
# followed by a reinstall non-destructive, by keeping the surviving files owned
# by a user that still exists. Do not "userdel waarp" while /var/lib/waarp-gateway
# is still there -- its uid is the first one useradd -r hands out, and the next
# package to create a system account would inherit ownership of the database.
#
