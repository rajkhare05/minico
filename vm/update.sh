#!/usr/bin/env bash

# TODO: This script can be triggered automatically if combined with watchman.
# Watchman checks the file changes saved.

current_dir=$(pwd)
minico_path=''

# Check if current folder is vm or not
if [[ $(basename $current_dir) == 'vm' ]]; then
    minico_path=$(echo "${current_dir%/*}")

# Check if current folder is minico or not
elif [[ $(basename $current_dir) == 'minico' ]]; then
    minico_path=$current_dir

else
    echo "cd into the root of project minico first !"
    exit 1
fi

# Assuming the VM port forwarding is enabled.
# Default: host's port 2222 is forwared to VM's port 22

# Create and save the ssh config in the ~/.ssh/config file like:
# Host 0.vm.localhost
#     User ape
#     Hostname localhost
#     Port 2222
#     IdentityFile ~/.ssh/ape

# Remote host saved in the ~/.ssh/config file
VM='0.vm.localhost'

# Remove the previous code in VM
ssh -T $VM <<'EOF'
rm -r ~/projects/minico/*
exit
EOF

# Copy the latest code in VM
scp -r $minico_path/* $VM:~/projects/minico/

# Build the latest code in VM
ssh -T $VM <<'EOF'
cd ~/projects/minico
go build
# ./minico run sh -c whoami
exit
EOF

# Run the newly built minico binary in VM.
# NOTE: When SSHed into the VM, prompt symbol ($ or #) is not shown.
#       Run a command like id or whoami to verify that shell is working.
ssh $VM 'cd ~/projects/minico && ./minico run /bin/sh'

