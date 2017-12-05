#!/bin/bash
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2017 Red Hat, Inc.
#

set -ex

KUBECTL=${KUBECTL:-kubectl}

source hack/config.sh

spiceProxy() {
cat <<EOF
spec:
  template:
    spec:
      containers:
        - name: virt-api
          env:
            - name: SPICE_PROXY
              value: "http://${master_ip}:3128"
---
EOF
}

echo "Cleaning up ..."
# Work around https://github.com/kubernetes/kubernetes/issues/33517
$KUBECTL delete ds -l "kubevirt.io" -n kube-system --cascade=false --grace-period 0 2>/dev/null || :
$KUBECTL delete pods -n kube-system -l="kubevirt.io=libvirt" --force --grace-period 0 2>/dev/null || :
$KUBECTL delete pods -n kube-system -l="kubevirt.io=virt-handler" --force --grace-period 0 2>/dev/null || :

# Delete everything, no matter if release, devel or infra
$KUBECTL delete -f manifests -R --grace-period 1 2>/dev/null || :

# Delete exposures
$KUBECTL delete services -l "kubevirt.io" -n kube-system

sleep 2

echo "Deploying ..."

# Deploy the right manifests for the right target
if [ -z "$TARGET" ] || [ "$TARGET" = "vagrant-dev"  ]; then
    $KUBECTL create -f manifests/dev -R $i
elif [ "$TARGET" = "vagrant-release"  ]; then
    $KUBECTL create -f manifests/release -R $i
   ## Tell virt-api where to look for the spice proxy
   spiceProxy | $KUBECTL patch deployment virt-api -n kube-system --patch "$(cat -)"
fi

## Expose common services
$KUBECTL expose deployment haproxy --port 8184 -l 'kubevirt.io=' -n kube-system --external-ip $master_ip
$KUBECTL expose deployment spice-proxy --port 3128 -l 'kubevirt.io=' -n kube-system --external-ip $master_ip

# Deploy additional infra for testing
$KUBECTL create -f manifests/testing -R $i

echo "Done"
