#!/usr/bin/env bash

function check_uptime() {
  retries=$1
  timeout=$2

  for _ in seq 1 $retries; do
    BOOTTIME=$(~/virtctl ssh -i ./hack/test_ssh -n ${VMS_NAMESPACE} testvm --username=cirros --local-ssh=true --local-ssh-opts "-o UserKnownHostsFile=/dev/null" --local-ssh-opts "-o StrictHostKeyChecking=no" -c "echo BOOTTIME=\$((\$(date +%s) - \$(awk '{print int(\$1)}' /proc/uptime)))" 2>>/dev/null | grep "^BOOTTIME" | cut -d= -f2 | tr -dc '[:digit:]')
    if [ -n "${BOOTTIME}" ]
      then
        echo "${BOOTTIME}"
        return 0;
      else
        sleep $timeout;
    fi
  done;
  echo "VM boot time could not be retrieved"
  return 1;
}

