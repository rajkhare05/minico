#!/usr/bin/env bash

sshpass -p "epa" ssh -T -p 2222 ape@localhost <<'EOF'
TARFILE="alpine-minirootfs-x86_64.tar.gz"
echo "Downloading $TARFILE"
curl -s "https://dl-cdn.alpinelinux.org/alpine/v3.21/releases/x86_64/alpine-minirootfs-3.21.3-x86_64.tar.gz" \
     -o /tmp/$TARFILE

# TODO: handle download error

echo "Setting up /tmp/rootfs"
mkdir -p /tmp/rootfs

# TODO: handle tar extraction error
tar xf /tmp/$TARFILE -C /tmp/rootfs

echo "Done"
EOF

