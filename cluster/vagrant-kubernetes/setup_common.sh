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

setenforce 0
sed -i "s/^SELINUX=.*/SELINUX=permissive/" /etc/selinux/config

# Disable swap
swapoff -a
sed -i '/ swap / s/^/#/' /etc/fstab

systemctl stop firewalld NetworkManager || :
systemctl disable firewalld NetworkManager || :
# Make sure the firewall is never enabled again
# Enabling the firewall destroys the iptable rules
yum -y remove NetworkManager firewalld

# Install epel
yum -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm
yum -y install jq sshpass

yum -y install bind-utils net-tools iscsi-initiator-utils

# Install storage requirements for iscsi and cluster
yum -y install centos-release-gluster
yum -y install --nogpgcheck -y glusterfs-fuse
yum -y install iscsi-initiator-utils

# if there is a second disk, use it for docker
if ls /dev/*db; then
    # We use the loopback docker dm support, and not a VG for now
    mkdir -p /var/lib/docker/
    restorecon -r /var/lib/docker
    mount LABEL=dockerdata /var/lib/docker/ || {
        mkfs.xfs -L dockerdata -f /dev/?db
    }
    # FAILS because of vdsms multpoath stuff
    #echo -e "\nLABEL=dockerdata /var/lib/docker/ xfs defaults 0 0" >> /etc/fstab
    mkdir -p /etc/systemd/system/docker.service.d/
    cat >/etc/systemd/system/docker.service.d/mount.conf <<EOT
[Service]
ExecStartPre=/usr/bin/sleep 5
ExecStartPre=-/usr/bin/mount LABEL=dockerdata /var/lib/docker
MountFlags=shared
EOT
    mount LABEL=dockerdata /var/lib/docker/
fi

cat <<EOF >/etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=http://yum.kubernetes.io/repos/kubernetes-el7-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
       https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOF
yum install -y docker

# Log to json files instead of journald
sed -i 's/--log-driver=journald //g' /etc/sysconfig/docker

# Omit pgp checks until https://github.com/kubernetes/kubeadm/issues/643 is resolved.
yum install --nogpgcheck -y \
    kubeadm \
    kubelet \
    kubectl \
    kubernetes-cni

# Latest docker on CentOS uses systemd for cgroup management
cat <<EOT >>/etc/systemd/system/kubelet.service.d/09-kubeadm.conf
[Service]
Environment="KUBELET_EXTRA_ARGS=--cgroup-driver=systemd --runtime-cgroups=/systemd/system.slice --kubelet-cgroups=/systemd/system.slice"
EOT
systemctl daemon-reload

systemctl enable docker && systemctl start docker
systemctl enable kubelet && systemctl start kubelet

# Needed for kubernetes service routing and dns
# https://github.com/kubernetes/kubernetes/issues/33798#issuecomment-250962627
modprobe bridge
sysctl -w net.bridge.bridge-nf-call-iptables=1
sysctl -w net.bridge.bridge-nf-call-ip6tables=1
