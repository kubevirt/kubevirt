#!/bin/bash

set -euo pipefail

rm -f loop0
dd if=/dev/zero of=loop0 bs=50M count=10
if [ -e "/dev/loop0" ]; then
  losetup -d /dev/loop0
fi
rm -rf /dev/loop0
mknod -m 0660 /dev/loop0 b 7 0
losetup /dev/loop0 loop0
rm -f /local-storage/block-device/loop0
ln -s /dev/loop0 /local-storage/block-device

# for some reason without sleep, container sometime fails to create the file
sleep 10

# let the monitoring script know we're done
echo "done" >/ready

while true; do
    sleep 60
done
