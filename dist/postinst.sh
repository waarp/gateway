#!/usr/bin/env bash

# Configuration # root:root 644
chmod 755 /etc/waarp-gateway 
chmod 644 /etc/waarp-gateway/waarp-gatewayd.ini

# Logs # root:waarp 664
chown root:waarp /usr/share/waarp-gateway/* 
chmod 664 /var/log/waarp-gateway/*

# Data # waarp:waarp 774
chown waarp: /var/lib/waarp-gateway/
chmod 774 /var/lib/waarp-gateway/

# Share # root:waarp 750
chown root:waarp /usr/share/waarp-gateway/* 
chmod 750 /usr/share/waarp-gateway/*

systemctl daemon-reload
if systemctl is-active waarp-gatewayd; then
  systemctl restart waarp-gatewayd
fi
