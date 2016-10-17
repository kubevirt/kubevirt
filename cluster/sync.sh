#!/bin/bash
set -ex

make all contrib
vagrant rsync # if you do not use NFS
vagrant ssh master -c "cd /vagrant && sudo hack/build-docker.sh"
vagrant ssh node -c "cd /vagrant && sudo hack/build-docker.sh"
vagrant ssh master -c "kubectl delete -f /vagrant/contrib/manifest/virt-controller.yaml" || :
vagrant ssh master -c "kubectl create -f /vagrant/contrib/manifest/virt-controller.yaml"
vagrant ssh master -c "kubectl delete -f /vagrant/contrib/manifest/virt-controller-service.yaml" || :
vagrant ssh master -c "kubectl create -f /vagrant/contrib/manifest/virt-controller-service.yaml"
