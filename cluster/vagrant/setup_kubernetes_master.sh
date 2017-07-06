#/bin/bash -xe
#
# This file is part of the kubevirt project
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

export KUBERNETES_MASTER=true
bash ./setup_kubernetes_common.sh

# Cockpit with kubernetes plugin
yum install -y cockpit cockpit-kubernetes
systemctl enable cockpit.socket && systemctl start cockpit.socket

# Create the master
kubeadm init --pod-network-cidr=10.244.0.0/16 --token abcdef.1234567890123456 --apiserver-advertise-address=$ADVERTISED_MASTER_IP

# Tell kubectl which config to use
export KUBECONFIG=/etc/kubernetes/admin.conf

set +e

kubectl version
while [ $? -ne 0 ]; do
  sleep 60
  echo 'Waiting for Kubernetes cluster to become functional...'
  kubectl version
done

set -e

# Work around https://github.com/kubernetes/kubernetes/issues/34101
# Weave otherwise the network provider does not work
kubectl -n kube-system get ds -l 'k8s-app=kube-proxy' -o json \
        | jq '.items[0].spec.template.spec.containers[0].command |= .+ ["--proxy-mode=userspace"]' \
        |   kubectl apply -f - && kubectl -n kube-system delete pods -l 'k8s-app=kube-proxy'

if [ "$NETWORK_PROVIDER" == "weave" ]; then 
  kubectl apply -f https://github.com/weaveworks/weave/releases/download/v1.9.4/weave-daemonset-k8s-1.6.yaml
else
  kubectl create -f kube-$NETWORK_PROVIDER.yaml
fi

# Work around https://github.com/kubernetes/kubeadm/issues/335 until Kubernetes 1.7.1 is released
kubectl apply -f kubernetes-1.7-workaround.yml

# Allow scheduling pods on master
# Ignore retval because it might not be dedicated already
kubectl taint nodes master node-role.kubernetes.io/master:NoSchedule- || :

# TODO better scope the permissions, for now allow the default account everything
kubectl create clusterrolebinding add-on-cluster-admin --clusterrole=cluster-admin --serviceaccount=kube-system:default
kubectl create clusterrolebinding add-on-default-admin --clusterrole=cluster-admin --serviceaccount=default:default

mkdir -p /exports/share1

chmod 0755 /exports/share1
chown 36:36 /exports/share1

echo "/exports/share1  *(rw,anonuid=36,anongid=36,all_squash,sync,no_subtree_check)" > /etc/exports

systemctl enable nfs-server && systemctl start nfs-server
