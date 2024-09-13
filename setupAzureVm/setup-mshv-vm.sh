#!/bin/bash

sudo setenforce 0
sudo modprobe br_netfilter
sudo sysctl -w net.ipv4.ip_forward=1

sudo bash -c 'cat << EOF > /etc/yum.repos.d/cloud-native-2.0-prod.repo
[mariner-official-cloud-native]
name=CBL-Mariner Official Cloud Native \$releasever \$basearch
baseurl=https://packages.microsoft.com/cbl-mariner/\$releasever/prod/cloud-native/\$basearch
gpgkey=file:///etc/pki/rpm-gpg/MICROSOFT-RPM-GPG-KEY file:///etc/pki/rpm-gpg/MICROSOFT-METADATA-GPG-KEY
gpgcheck=1
repo_gpgcheck=1
enabled=1
skip_if_unavailable=False
sslverify=1
EOF'

v=1.28.12
sudo tdnf  -y install helm vim kubeadm==$v kubectl==$v kubelet==$v
cat << EOF > $HOME/kubeadm-config.yaml
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
kubernetesVersion: v1.28.3
networking:
    podSubnet: 10.244.0.0/16
EOF

sudo kubeadm init --config kubeadm-config.yaml 
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config 
sudo chown $(id -u):$(id -g) $HOME/.kube/config
kubectl taint node --all node-role.kubernetes.io/master:NoSchedule-
kubectl taint node --all node-role.kubernetes.io/control-plane:NoSchedule-
kubectl apply --filename https://raw.githubusercontent.com/rancher/local-path-provisioner/v0.0.21/deploy/local-path-storage.yaml
kubectl apply -f https://raw.githubusercontent.com/flannel-io/flannel/v0.19.2/Documentation/kube-flannel.yml
kubectl get pods -A
