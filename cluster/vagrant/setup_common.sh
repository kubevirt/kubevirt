#!/bin/bash
master_ip=$1
nodes=$2

sed -i -e "s/PasswordAuthentication no/PasswordAuthentication yes/" /etc/ssh/sshd_config
systemctl restart sshd
# FIXME, sometimes eth1 does not come up on Vagrant on latest fc26
sudo ifup eth1
sed -i "/$(hostname)/d" /etc/hosts
grep 'master' /etc/hosts || echo "$master_ip master" >> /etc/hosts
IFS=. read ip1 ip2 ip3 ip4 <<< "$master_ip"
for node in $(seq 0 $(($nodes - 1))); do
  node_hostname="node$node"
  node_ip="$ip1.$ip2.$ip3.$(($ip4 + node + 1))"
  grep $node_hostname /etc/hosts || echo "$node_ip $node_hostname" >> /etc/hosts
done
