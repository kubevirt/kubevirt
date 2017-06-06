#!/bin/bash
set -ex

make all DOCKER_TAG=devel

for VM in `vagrant status | grep -v "^The Libvirt domain is running." | grep running | cut -d " " -f1`; do
  vagrant rsync $VM # if you do not use NFS
  vagrant ssh $VM -c "cd /vagrant && export DOCKER_TAG=devel && sudo -E hack/build-docker.sh"
done

