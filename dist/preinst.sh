#!/usr/bin/env bash

getent group waarp >/dev/null || groupadd -r waarp
getent passwd waarp >/dev/null || \
    useradd -r -g waarp -d /var/lib/waarp -s /bin/bash \
    -c "Waarp user" --create-home waarp

