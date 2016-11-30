#!/bin/bash
set -ex

vagrant ssh-config master 2>&1 | grep "not yet ready for SSH" >/dev/null \
        && { echo "Master node is not up"; exit 1; }

OPTIONS=`vagrant ssh-config master | grep -v '^Host ' | awk -v ORS=' ' 'NF{print "-o " $1 "=" $2}'`

scp $OPTIONS master:/usr/bin/kubectl cluster/.kubectl
chmod u+x cluster/.kubectl
vagrant ssh master -c "sudo cat /etc/kubernetes/kubelet.conf" > cluster/.kubeconfig

make all contrib

for VM in `vagrant status | grep running | cut -d " " -f1`; do
  vagrant rsync $VM # if you do not use NFS
  vagrant ssh $VM -c "cd /vagrant && sudo hack/build-docker.sh"
done

# Deploy all manifests files
set +x
for i in `ls contrib/manifest/*.yaml`; do
    cluster/kubectl.sh --core delete -f $i --grace-period 0 2>/dev/null || :
done
sleep 2
for i in `ls contrib/manifest/*.yaml`; do
    cluster/kubectl.sh --core create -f $i
done
