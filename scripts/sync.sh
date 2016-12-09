#!/bin/bash
set -ex

contrib/vagrant/sync_config.sh

make all

for VM in `vagrant status | grep running | cut -d " " -f1`; do
  vagrant rsync $VM # if you do not use NFS
  vagrant ssh $VM -c "cd /vagrant && sudo hack/build-docker.sh"
done

# Deploy all manifests files
set +x
for i in `ls cluster/manifest/*.yaml`; do
    contrib/vagrant/kubectl.sh --core delete -f $i --grace-period 0 2>/dev/null || :
done
sleep 2
for i in `ls cluster/manifest/*.yaml`; do
    contrib/vagrant/kubectl.sh --core create -f $i
done
