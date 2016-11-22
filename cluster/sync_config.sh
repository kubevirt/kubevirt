#!/bin/bash
set -ex

vagrant ssh-config master 2>&1 | grep "not yet ready for SSH" >/dev/null \
        && { echo "Master node is not up"; exit 1; }

OPTIONS=`vagrant ssh-config master | grep -v '^Host ' | awk -v ORS=' ' 'NF{print "-o " $1 "=" $2}'`

scp $OPTIONS master:/usr/bin/kubectl cluster/.kubectl
chmod u+x cluster/.kubectl

vagrant ssh master -c "sudo cat /etc/kubernetes/kubelet.conf" > cluster/.kubeconfig
