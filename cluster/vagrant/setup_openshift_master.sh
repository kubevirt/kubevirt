#!/bin/bash
master_ip=$1
nodes=$2

bash /vagrant/cluster/vagrant/setup_openshift_common.sh

sed -i '/host_key_checking/s/^#//g' /etc/ansible/ansible.cfg
IFS=. read ip1 ip2 ip3 ip4 <<< "$master_ip"
nodes=""
for node in $(seq 0 $(($2 - 1))); do
  node_ip="$ip1.$ip2.$ip3.$(($ip4 + node + 1))"
  node_hostname="node$node openshift_node_labels=\"{'region': 'infra','zone': 'default'}\" openshift_ip=$node_ip"
  nodes="$nodes$node_hostname\n"
done
cat > inventory <<EOF
[OSEv3:children]
masters
nodes

[OSEv3:vars]
ansible_ssh_user=root
ansible_ssh_pass=vagrant
openshift_deployment_type=origin
openshift_clock_enabled=true
openshift_master_identity_providers=[{'name': 'allow_all_auth', 'login': 'true', 'challenge': 'true', 'kind': 'AllowAllPasswordIdentityProvider'}]
openshift_disable_check=memory_availability,disk_availability,docker_storage
openshift_repos_enable_testing=True

[masters]
master openshift_ip=$master_ip

[etcd]
master openshift_ip=$master_ip

[nodes]
master openshift_node_labels="{'region': 'infra','zone': 'default'}" openshift_schedulable=true openshift_ip=$master_ip
$nodes

EOF

ansible-playbook -i inventory /usr/share/ansible/openshift-ansible/playbooks/byo/config.yml

# Create OpenShift user
oc create user admin
oc create identity allow_all_auth:admin
oc create useridentitymapping allow_all_auth:admin admin
oadm policy add-cluster-role-to-user cluster-admin admin

echo -e "\033[0;32m Deployment was successful!\033[0m"
