#!/bin/bash

set -ex

READ_ONLY=""
if [ "$1" == "-r" ]; then
    READ_ONLY="-r"
    shift
fi

NBD_SOURCE=$1
SOCKET_PATH=$2
TMP_SOCKET_PATH="$(dirname $SOCKET_PATH)/.$(basename $SOCKET_PATH)"

qemu-nbd $READ_ONLY -f raw -k "$TMP_SOCKET_PATH" $NBD_SOURCE &
NBD_PID=$!

if [ "x$READ_ONLY" != "x-r" ] ; then
    RETRY_LIMIT=100
    RETRIES=0
    DONE=0
    while [ "$DONE" -eq "0" ] ; do
        chown qemu:qemu "$TMP_SOCKET_PATH" && DONE=1 || RETRIES=$(($RETRIES + 1))
        [ $RETRIES -gt $RETRY_LIMIT ] && false "timed out waiting for socket"
        sleep 0.1
    done
else
    DONE=1
fi

if [ "$DONE" -eq "1" ] ; then
    mv "$TMP_SOCKET_PATH" "$SOCKET_PATH"
fi

tail --pid="$NBD_PID" -f /dev/null
rm -f "$TMP_SOCKET_PATH" "$SOCKET_PATH"
