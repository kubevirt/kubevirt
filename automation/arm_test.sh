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

# This test is used to verify if kubevirt cluster can be successfully deployed
# and if vmi-fedora can be successfully boot.
set -ex

kubectl() { cluster-up/kubectl.sh "$@"; }

make cluster-sync
make generate
# check if kubevirt can boot vmi-fedora successfully
kubectl apply -f examples/vmi-fedora.yaml
timeout=300
current_time=0
while [ -n "$(kubectl get pods --no-headers | grep virt-launcher-vmi-fedora| grep -v Running)" ]; do
	echo "Waiting for vmi-fedora to enter the Running state ..."
	sleep 10

	current_time=$((current_time + 10))
	if [ $current_time -gt $timeout ]; then
		echo "start vmi-fedora failed"
		exit 1
	fi
done
kubectl delete -f examples/vmi-fedora.yaml
