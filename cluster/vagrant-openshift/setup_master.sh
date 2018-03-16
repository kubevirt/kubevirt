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
nodes=$2

bash /vagrant/cluster/vagrant-openshift/setup_common.sh $master_ip $nodes

# Disable host key checking under ansible.cfg file
sed -i '/host_key_checking/s/^#//g' /etc/ansible/ansible.cfg

# Save nodes to add it under ansible inventory file
inv_nodes=""
IFS=. read ip1 ip2 ip3 ip4 <<<"$master_ip"
for node in $(seq 0 $(($nodes - 1))); do
    node_ip="$ip1.$ip2.$ip3.$(($ip4 + node + 1))"
    node_hostname="node$node openshift_node_labels=\"{'region': 'infra','zone': 'default'}\" openshift_ip=$node_ip"
    inv_nodes="$inv_nodes$node_hostname\n"
done

openshift_ansible_dir="/root/openshift-ansible"
inventory_file="/root/inventory"

mkdir -p /root/openshift-ansible
# Checkout to the specific version as W/A for https://github.com/openshift/openshift-ansible/issues/6756
git clone https://github.com/openshift/openshift-ansible.git $openshift_ansible_dir -b openshift-ansible-3.9.0-0.40.0

# Create ansible inventory file
cat >$inventory_file <<EOF
[OSEv3:children]
masters
nodes

[OSEv3:vars]
ansible_ssh_user=root
ansible_ssh_pass=vagrant
deployment_type=origin
openshift_deployment_type=origin
openshift_clock_enabled=true
openshift_master_identity_providers=[{'name': 'allow_all_auth', 'login': 'true', 'challenge': 'true', 'kind': 'AllowAllPasswordIdentityProvider'}]
openshift_disable_check=memory_availability,disk_availability,docker_storage,package_availability,docker_image_availability
openshift_repos_enable_testing=True
openshift_image_tag=v3.9.0-alpha.4
containerized=true
enable_excluders=false

[masters]
master openshift_ip=$master_ip

[etcd]
master openshift_ip=$master_ip

[nodes]
master openshift_node_labels="{'region': 'infra','zone': 'default'}" openshift_schedulable=true openshift_ip=$master_ip
$inv_nodes
EOF

# Run OpenShift ansible playbook
ansible-playbook -i $inventory_file $openshift_ansible_dir/playbooks/deploy_cluster.yml

# Create OpenShift user
/usr/local/bin/oc create user admin
/usr/local/bin/oc create identity allow_all_auth:admin
/usr/local/bin/oc create useridentitymapping allow_all_auth:admin admin
/usr/local/bin/oc adm policy add-cluster-role-to-user cluster-admin admin

# Set SELinux to permessive mode
setenforce 0
sed -i "s/^SELINUX=.*/SELINUX=permissive/" /etc/selinux/config

echo -e "\033[0;32m Deployment was successful!\033[0m"
