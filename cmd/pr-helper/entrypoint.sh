#!/bin/bash

set -x
MULTIPATH_HOST="${MULTIPATH_HOST:-/run/multipathd.socket}"
MULTIPATH_SOCKET_NAME="${MULTIPATH_SOCKET_NAME:-/run/multipathd.socket}"
ln -s /proc/1/root${MULTIPATH_HOST} ${MULTIPATH_SOCKET_NAME}

set -e

exec /usr/bin/qemu-pr-helper $@
