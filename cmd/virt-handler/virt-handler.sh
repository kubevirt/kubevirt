#!/usr/bin/bash

# Configure libvirt username and password
mkdir -p $HOME/.config/libvirt
cat << EOF > $HOME/.config/libvirt/auth.conf
[credentials-defgrp]
authname=$(cat /etc/sasl-secret/username)
password=$(cat /etc/sasl-secret/password)

[auth-libvirt-default]
credentials=defgrp
EOF

/virt-handler "$@"
