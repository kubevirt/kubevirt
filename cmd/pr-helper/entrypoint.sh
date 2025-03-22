#!/bin/bash

set -x
MULTIPATH_HOST="${MULTIPATH_HOST:-/run/multipathd.socket}"
ln -s /var/run/kubevirt/daemons/pr/multipathd.socket ${MULTIPATH_SOCKET_NAME}

set -e

exec /usr/bin/qemu-pr-helper $@
