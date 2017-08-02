#!/usr/bin/bash
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

echo "Cleaning up ..."
# Work around https://github.com/kubernetes/kubernetes/issues/33517
cluster/kubectl.sh --core delete -f manifests/virt-handler.yaml --cascade=false --grace-period 0 2>/dev/null || :
cluster/kubectl.sh --core delete pods -l=daemon=virt-handler --force --grace-period 0 2>/dev/null || :

cluster/kubectl.sh --core delete -f manifests/libvirt.yaml --cascade=false --grace-period 0 2>/dev/null || :
cluster/kubectl.sh --core delete pods -l=daemon=libvirt --force --grace-period 0 2>/dev/null || :

# Delete everything else
for i in `ls manifests/*.yaml`; do
    $KUBECTL delete -f $i --grace-period 0 2>/dev/null || :
done

sleep 2

echo "Deploying ..."
for i in `ls manifests/*.yaml`; do
    $KUBECTL create -f $i
done

echo "Done"
