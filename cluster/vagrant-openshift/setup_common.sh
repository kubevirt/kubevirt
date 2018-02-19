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

# Enter master and nodes to /etc/hosts
sed -i "/$(hostname)/d" /etc/hosts
grep 'master' /etc/hosts || echo "$1 master" >>/etc/hosts
IFS=. read ip1 ip2 ip3 ip4 <<<"$1"
for node in $(seq 0 $(($2 - 1))); do
    node_hostname="node$node"
    node_ip="$ip1.$ip2.$ip3.$(($ip4 + node + 1))"
    grep $node_hostname /etc/hosts || echo "$node_ip $node_hostname" >>/etc/hosts
done

# Install storage requirements for iscsi and cluster
yum -y install centos-release-gluster
yum -y install --nogpgcheck -y glusterfs-fuse
yum -y install iscsi-initiator-utils

# Install OpenShift packages
yum install -y centos-release-openshift-origin
yum install -y yum-utils ansible wget git net-tools bind-utils iptables-services bridge-utils bash-completion kexec-tools sos psacct docker
systemctl start docker
systemctl enable docker
