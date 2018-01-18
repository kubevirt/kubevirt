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
master_ip=$1
network_provider=$3

wait_for_kubernetes() {
    local timeout="$1"
    local start_time=$(date +%s)
    local time_left=$timeout
    local current_time

    while [[ $time_left -gt 0 ]]; do
        kubectl version && return 0
        sleep 60
        current_time=$(date +%s)
        time_left=$((timeout - (current_time - start_time)))
        echo \
            "Waiting for Kubernetes cluster to become functional," \
            "$time_left seconds left..."
    done

    echo "Failed to create Kubernetes cluster in $timeout seconds, aborting"

    return 1
}

export KUBERNETES_MASTER=true
bash /vagrant/cluster/vagrant-kubernetes/setup_common.sh

# Cockpit with kubernetes plugin
yum install -y cockpit cockpit-kubernetes
systemctl enable cockpit.socket && systemctl start cockpit.socket

# W/A for https://github.com/kubernetes/kubernetes/issues/53356
rm -rf /var/lib/kubelet

# Create the master
cat >/etc/kubernetes/kubeadm.conf <<EOF
apiVersion: kubeadm.k8s.io/v1alpha1
kind: MasterConfiguration
token: abcdef.1234567890123456
networking:
  podSubnet: 10.244.0.0/16
EOF

kubeadm init --config /etc/kubernetes/kubeadm.conf

# Tell kubectl which config to use
export KUBECONFIG=/etc/kubernetes/admin.conf

wait_for_kubernetes $((60 * 15))

# Additional network providers are available from
# https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/#pod-network
case "$network_provider" in
"weave")
    kubever=$(kubectl version | base64 | tr -d '\n')
    kubectl apply -f "https://cloud.weave.works/k8s/net?k8s-version=$kubever"
    ;;

"flannel")
    kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/v0.9.1/Documentation/kube-flannel.yml
    ;;
esac

# Allow scheduling pods on master
# Ignore retval because it might not be dedicated already
kubectl taint nodes master node-role.kubernetes.io/master:NoSchedule- || :

mkdir -p /exports/share1

chmod 0755 /exports/share1
chown 36:36 /exports/share1

echo "/exports/share1  *(rw,anonuid=36,anongid=36,all_squash,sync,no_subtree_check)" >/etc/exports

systemctl enable nfs-server && systemctl start nfs-server

echo -e "\033[0;32m Deployment was successful!"
echo -e "Cockpit is accessible at https://$master_ip:9090."
echo -e "Credentials for Cockpit are 'root:vagrant'.\033[0m"
