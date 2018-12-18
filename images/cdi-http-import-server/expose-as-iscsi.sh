#!/usr/bin/bash

IMAGE_PATH="$1"

if [ ! -f "$IMAGE_PATH" ]; then
    echo "vm image '$IMAGE_PATH' not found"
    exit 1
fi

# USING 'set -e' error detection for everything below this point.
set -e

PORT=${PORT:-3260}
WWN=${WWN:-iqn.2018-01.io.kubevirt:wrapper}
LUNID=1

echo "Starting tgtd at port $PORT"
tgtd -f --iscsi portal="0.0.0.0:${PORT}" &
sleep 5

echo "Adding target and exposing it"
tgtadm --lld iscsi --mode target --op new --tid=1 --targetname $WWN
tgtadm --lld iscsi --mode target --op bind --tid=1 -I ALL

if [ -n "$PASSWORD" ]; then
    echo "Adding authentication for user $USERNAME"
    tgtadm --lld iscsi --op new --mode account --user $USERNAME --password $PASSWORD
    tgtadm --lld iscsi --op bind --mode account --tid=1 --user $USERNAME
fi

echo "Adding volume file as LUN"
tgtadm --lld iscsi --mode logicalunit --op new --tid=1 --lun=$LUNID -b $IMAGE_PATH
tgtadm --lld iscsi --mode logicalunit --op update --tid=1 --lun=$LUNID --params thin_provisioning=1

echo "Start monitoring"
touch previous_state
while true; do
    tgtadm --lld iscsi --mode target --op show >current_state
    diff -q previous_state current_state || (
        date
        cat current_state
    )
    mv -f current_state previous_state
    sleep 5
done
