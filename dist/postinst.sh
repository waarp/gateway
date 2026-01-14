#!/usr/bin/env bash

# Configuration # waarp:waarp 644
chown -R waarp:waarp /etc/waarp-gateway
chmod 755 /etc/waarp-gateway
chmod 644 /etc/waarp-gateway/gatewayd.ini

# Logs # waarp:waarp 755
chown waarp:waarp /var/log/waarp-gateway/
chmod 755 /var/log/waarp-gateway/

# Data # waarp:waarp 774
chown -R waarp:waarp /var/lib/waarp-gateway/
chmod -R 774 /var/lib/waarp-gateway/

# Share # root:waarp 750
chown -R root:waarp /usr/share/waarp-gateway/
chmod -R 750 /usr/share/waarp-gateway/

systemctl daemon-reload
if systemctl is-active waarp-gatewayd; then
  systemctl restart waarp-gatewayd
fi
