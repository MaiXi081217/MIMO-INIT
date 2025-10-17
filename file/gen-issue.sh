#!/bin/bash
HOST=$(hostname)
VERSION="1.0.0"
DATE=$(date "+%Y-%m-%d %H:%M:%S")

cat <<ISSUE > /etc/issue
┌──────────────────────────────────┐
│     Welcome to MIMO Storage      │
└──────────────────────────────────┘
ISSUE

cat <<EOF2 >> /etc/issue
Hostname : $HOST
Version  : $VERSION
Updated  : $DATE

Network Interfaces:
EOF2

ip -o link show | awk -F": " '!/ lo:/ {print $2}' | while read -r IFACE; do
  STATE=$(ip -o link show dev "$IFACE" | awk '/state/ {print $9}')
  [[ -z "$STATE" ]] && STATE=$(cat /sys/class/net/$IFACE/operstate 2>/dev/null || echo none)

  IPADDR=$(ip -o -4 addr show dev "$IFACE" | awk '{print $4; exit}')
  [[ -z "$IPADDR" ]] && IPADDR="(no IP)"

  echo "    $IFACE: $IPADDR ($STATE)" >> /etc/issue
done

echo >> /etc/issue
