#!/usr/bin/env bash

chown -R waarp:waarp /etc/waarp-gateway /var/lib/waarp-gateway /var/log/waarp-gateway

systemctl daemon-reload
if systemctl is-active waarp-gatewayd; then
  systemctl restart waarp-gatewayd
fi
