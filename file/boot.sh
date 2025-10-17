#!/bin/bash
set -euo pipefail
export TERM=xterm
rows=$(tput lines || echo 24)
cols=$(tput cols || echo 80)
clear
logo="
┌──────────────────────────────────┐
│ __  __   _____   __  __    ____  │
│|  \/  | |_   _| |  \/  |  / __ \ │
│| \  / |   | |   | \  / | | |  | |│
│| |\/| |   | |   | |\/| | | |  | |│
│| |  | |  _| |_  | |  | | | |__| |│
│|_|  |_| |_____| |_|  |_|  \____/ │
│                                  │
└──────────────────────────────────┘
"
logo_lines=$(echo "$logo" | wc -l)
pad_lines=$(( (rows - logo_lines - 6) / 2 ))
for ((i=0; i<pad_lines; i++)); do echo; done
while IFS= read -r line; do
    pad_spaces=$(( (cols - ${#line}) / 2 ))
    printf "%*s%s\n" $pad_spaces "" "$line"
done <<< "$logo"
welcome="Welcome to MIMO Storage"
pad_spaces=$(( (cols - ${#welcome}) / 2 ))
printf "%*s%s\n\n" $pad_spaces "" "$welcome"
bar="##################################################"
for i in {0..50}; do
    percent=$((i*2))
    filled=$i
    progress="[${bar:0:filled}$(printf %*s $((50-filled)))] $percent%"
    pad=$(( (cols - ${#progress}) / 2 ))
    printf "\r%*s%s" $pad "" "$progress"
    sleep 0.2
done
echo
msg="✅ System Ready!"
pad=$(( (cols - ${#msg}) / 2 ))
printf "%*s%s\n" $pad "" "$msg"
