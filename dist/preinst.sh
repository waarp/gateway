#!/bin/sh
#
# Runs before the package files are unpacked, on both dpkg and rpm.
#
# /bin/sh and not "/usr/bin/env bash": Debian Policy 10.4 requires a maintainer
# script to name its interpreter directly, and lintian rejects env(1) with
# "unknown-control-interpreter".

set -e

# dpkg passes an action name, rpm passes the number of instances that will be
# installed once the transaction is done. Anything else -- abort-upgrade and
# friends -- must be a no-op.
case "${1:-}" in
    install | upgrade | 1 | 2) ;;
    *) exit 0 ;;
esac

# The account is only ever created, never modified: an existing installation
# may legitimately have had its home or shell adjusted by the administrator.
if ! getent group waarp >/dev/null; then
    groupadd -r waarp
fi

if ! getent passwd waarp >/dev/null; then
    # Home is /var/lib/waarp-gateway, which the package ships, and not the
    # /var/lib/waarp that earlier releases created: the latter matched neither
    # the service's WorkingDirectory nor anything else on the system, and was
    # left behind on purge. --create-home is therefore unnecessary, and would
    # be a no-op anyway since the directory is unpacked after this script.
    #
    # nologin rather than a real shell: this is a service account. Use
    # "runuser -u waarp -- <command>" to run gateway commands as it, which the
    # documentation now recommends for import and export.
    useradd -r -g waarp -d /var/lib/waarp-gateway -s /usr/sbin/nologin \
        -c "Waarp Gateway service account" waarp
fi
