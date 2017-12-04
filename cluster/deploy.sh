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

# Create rbac resources first to avoid unauthorized exceptions
$KUBECTL create -f manifests/dev -R $i
$KUBECTL create -f manifests/testing -R $i

$KUBECTL expose deployment haproxy --port 8184 -l 'kubevirt.io=' -n kube-system --external-ip $master_ip
$KUBECTL expose deployment spice-proxy --port 3128 -l 'kubevirt.io=' -n kube-system --external-ip $master_ip

echo "Done"
