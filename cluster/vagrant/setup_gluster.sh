#/bin/bash -xe
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

# gluster volume dir, must match glusterfs-data-vol mountpoint
DIR=/vol

# prep
export KUBECONFIG=/etc/kubernetes/admin.conf
cp /etc/kubernetes/admin.conf ~/.kube/config

# install glusterfs mount util
yum install -y glusterfs-client

# label nodes 
kubectl label node --all storagenode=glusterfs 

# create glusterfs cluster
kubectl create -f gluster-daemonset.yaml

# get glusterfs bricks
BRICKS=$(kubectl get pods --selector=glusterfs-node=pod -o template --template="{{range .items}} {{.status.podIP}} {{end}}")
while [[ -z "${BRICKS// }" ]]
do
    sleep 3
    echo "waiting for gluster up"
    BRICKS=$(kubectl get pods --selector=glusterfs-node=pod -o template --template="{{range .items}} {{.status.podIP}} {{end}}")
done

BRICK_PATH=""
for i in ${BRICKS}
do
    if [ ! -z ${i} ]; then
        if [ -z ${BRICK_PATH} ]; then
            BRICK_PATH="${i}:${DIR}"
        else BRICK_PATH=${BRICK_PATH}",${i}:${DIR}"
        fi
    fi
done


# create glusterfs provisioner
kubectl create -f glusterfs-provisioner-deploy.yaml

# create storage class
echo 'kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: glusterfs-simple
provisioner: gluster.org/glusterfs-simple
parameters:
  forceCreate: "true"
  brickrootPaths: "'${BRICK_PATH}'"' | kubectl create -f -

# create a PVC
kubectl create -f https://raw.githubusercontent.com/kubernetes-incubator/external-storage/master/gluster/glusterfs/deploy/pvc.yaml


