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


PROVIDER=${PROVIDER:-vagrant-kubernetes}

source hack/common.sh
source cluster/$PROVIDER/provider.sh
source hack/config.sh

echo "Deploying Gluster Storage..."
disks=$(fdisk -l | grep "/dev/vd.:" | grep "536\.9" | awk '{print $2}' | cut -d":" -f1)
i=1
for each in $disks
do
  sed -i "s#{DISK${i}}#${each}#g" storage/topology.json
  i=$((i+1))
done
sed -i "s#{master_ip}#${master_ip}#g" storage/topology.json
. $(dirname $0)/gk-deploy  --cli="_kubectl" -y --single-node -g storage/topology.json
heketi_service_url=$(_kubectl describe svc/heketi | grep "Endpoints:" | awk '{print $2}')
sed -i "s#{HEKETI_REST_URL}#${heketi_service_url}#g" storage/kube-templates/glusterfs-singlenode-storageclass.yaml
_kubectl create -f storage/kube-templates/glusterfs-singlenode-storageclass.yaml
echo "Done"
