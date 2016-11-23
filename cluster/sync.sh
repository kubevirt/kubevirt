#!/bin/bash
set -ex

OPTIONS=`vagrant ssh-config master | grep -v '^Host ' | awk -v ORS=' ' 'NF{print "-o " $1 "=" $2}'`

scp $OPTIONS master:/usr/bin/kubectl cluster/.kubectl
chmod u+x cluster/.kubectl
vagrant ssh master -c "sudo cat /etc/kubernetes/kubelet.conf" > cluster/.kubeconfig

make all contrib
vagrant rsync # if you do not use NFS
vagrant ssh master -c "cd /vagrant && sudo hack/build-docker.sh"
vagrant ssh node -c "cd /vagrant && sudo hack/build-docker.sh"

# Deploy all manifests files
set +x
for i in `ls contrib/manifest/*.yaml`; do
    cluster/kubectl.sh --core delete -f $i --grace-period 0 2>/dev/null || :
done
sleep 2
for i in `ls contrib/manifest/*.yaml`; do
    cluster/kubectl.sh --core create -f $i
done
