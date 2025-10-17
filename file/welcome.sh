#!/bin/bash
clear
GREEN="\e[32m"
RESET="\e[0m"
RED="\e[31m"
echo "Welcome to MIMO Server!"
echo "* Hostname      : $(hostname)"
echo "* Version       : 1.0.0($(uname -r))"
echo
echo "Network:"
echo
PRIMARY_IP=$(hostname -I | awk "{print \$1}")
if [ -n "$PRIMARY_IP" ]; then
    echo -e "  WebGUI Address: ${RED}http://$PRIMARY_IP${RESET}"
else
    echo "  WebGUI Address: (No IP assigned)"
fi
echo

ip -o link show | awk -F": " '!/ lo:/ {print $2}' | while read -r IFACE; do
    STATE=$(ip -o link show dev "$IFACE" | awk '/state/ {print $9}')
    [[ -z "$STATE" ]] && STATE=$(cat /sys/class/net/$IFACE/operstate 2>/dev/null || echo none)

    IPADDR=$(ip -o -4 addr show dev "$IFACE" | awk '{print $4; exit}')
    [[ -z "$IPADDR" ]] && IPADDR=""

    echo "    $IFACE: $IPADDR ($STATE)"
done
echo
