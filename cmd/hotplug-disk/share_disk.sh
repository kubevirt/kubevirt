#!/bin/bash

set -ex

READ_ONLY=""
if [ "$1" == "-r" ]; then
    READ_ONLY="-r"
    shift
fi

NBD_SOURCE=$1
SOCKET_PATH=$2

qemu-nbd $READ_ONLY -t -f raw -k "$SOCKET_PATH" $NBD_SOURCE &
NBD_PID=$!

# FIXME: this is a race condition. virt-launcher might already
# be trying to attach the disk to the domain
# unfortunately we can't assume NBD_SOURCE is readable unless root
if [ "x$READ_ONLY" != "x-r" ] ; then
    RETRY_LIMIT=100
    RETRIES=0
    DONE=0
    while [ "$DONE" -eq "0" ] ; do
        chown qemu:qemu "$SOCKET_PATH" && DONE=1 || RETRIES=$(($RETRIES + 1))
        [ $RETRIES -gt $RETRY_LIMIT ] && fail "timed out waiting for socket"
        sleep 0.1
    done
fi

tail --pid="$NBD_PID" -f /dev/null
