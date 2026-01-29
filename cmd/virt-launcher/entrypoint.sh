#!/bin/bash
# Entrypoint script for virt-launcher
# Ensures /var/run is symlinked to /run for libvirt socket compatibility

# In CentOS Stream 10+, libvirt creates sockets in /run/libvirt/ but
# the filesystem package creates /var/run as a directory instead of a symlink.
# This fixes the symlink at runtime.
#
# IMPORTANT: Must use relative symlink (../run) not absolute (/run) because
# virt-handler accesses the container's filesystem via /proc/<pid>/root/var/run.
# With an absolute symlink, the kernel would resolve /run relative to
# virt-handler's root, not the container's root.
if [ -d "/var/run" ] && [ ! -L "/var/run" ]; then
    # Preserve any existing content
    if [ "$(ls -A /var/run 2>/dev/null)" ]; then
        cp -a /var/run/* /run/ 2>/dev/null || true
    fi
    rm -rf /var/run
    ln -sf ../run /var/run
fi

exec /usr/bin/virt-launcher-bin "$@"
