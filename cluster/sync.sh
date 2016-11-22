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
vagrant ssh master -c "kubectl delete -f /vagrant/contrib/manifest/virt-controller.yaml" || :
vagrant ssh master -c "kubectl create -f /vagrant/contrib/manifest/virt-controller.yaml"
vagrant ssh master -c "kubectl delete -f /vagrant/contrib/manifest/virt-controller-service.yaml" || :
vagrant ssh master -c "kubectl create -f /vagrant/contrib/manifest/virt-controller-service.yaml"
vagrant ssh master -c "kubectl delete -f /vagrant/contrib/manifest/virt-handler.yaml" || :
vagrant ssh master -c "kubectl create -f /vagrant/contrib/manifest/virt-handler.yaml"
vagrant ssh master -c "kubectl delete -f /vagrant/contrib/manifest/vm-resource.yaml"
vagrant ssh master -c "kubectl create -f /vagrant/contrib/manifest/vm-resource.yaml"
