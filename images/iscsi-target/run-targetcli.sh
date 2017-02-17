#!/bin/sh
# See https://wiki.alpinelinux.org/wiki/Linux_iSCSI_Target_(TCM)

set -e

die() { echo "ERR: $@" ; exit 2 ; }

SIZE_MB=${SIZE:-1024}
WWN=iqn.2017-01.io.kubevirt:sn.42

if [[ -n "$GENERATE_DEMO_OS_SEED" ]]; then
  echo "Creating demo OS image as requested"
  mkdir -p /volume
  wget http://download.qemu-project.org/linux-0.2.img.bz2 -O - | bunzip2 > /volume/file.img ;
else
  # Otherwise do the usual checks
  echo "Checking volume"
  [[ -d /volume ]] || die "No persistent volume provided"
  [[ -f /volume/file.img ]] || truncate -s ${SIZE} /volume/file.img
fi

echo "Generating /etc/target/saveconfig.json"
sed -i \
  -e "s/__SIZE__/$(stat -c %s /volume/file.img)/" \
  -e "s/__WWN__/$WWN/" \
  /etc/target/saveconfig.json

echo "Restoring target configuration from /etc/target/saveconfig.json"
targetctl restore

echo "Start monitoring"
while true ; do
  date
  COUNT=$(cat /sys/kernel/config/target/iscsi/*/tpgt_1/dynamic_sessions | wc -l)
  echo "  connection count: $COUNT"
  [[ $COUNT -gt 0 ]] && {
    echo "  sessions:"
    sed "/^.\+/ s/^/  - /" /sys/kernel/config/target/iscsi/*/tpgt_1/dynamic_sessions
  }
  sleep 3
done
