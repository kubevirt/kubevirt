#!/bin/bash

function _main_ip {
  echo 192.168.200.2
}

function up () {
  # Make sure that the vagrant environment is up and running
  vagrant up --provider=libvirt
  # Synchronize kubectl config
  vagrant ssh-config master 2>&1 | grep "not yet ready for SSH" >/dev/null \
        && { echo "Master node is not up"; exit 1; }

  OPTIONS=`vagrant ssh-config master | grep -v '^Host ' | awk -v ORS=' ' 'NF{print "-o " $1 "=" $2}'`

  scp $OPTIONS master:/usr/bin/kubectl ${KUBEVIRT_PATH}cluster/vagrant/.kubectl
  chmod u+x cluster/vagrant/.kubectl

  vagrant ssh master -c "sudo cat /etc/kubernetes/admin.conf" > ${KUBEVIRT_PATH}cluster/vagrant/.kubeconfig
}

function prepare_config () {
  cat > hack/config-local.sh <<EOF
master_ip=$(_main_ip)
docker_tag=devel
EOF
}

function build () {
  make build manifests
  for VM in `vagrant status | grep -v "^The Libvirt domain is running." | grep running | cut -d " " -f1`; do
    vagrant rsync $VM # if you do not use NFS
    vagrant ssh $VM -c "cd /vagrant && export DOCKER_TAG=devel && sudo -E hack/build-docker.sh build optional"
  done
}

function _kubectl () {
  export KUBECONFIG=${KUBEVIRT_PATH}cluster/vagrant/.kubeconfig
  ${KUBEVIRT_PATH}cluster/vagrant/.kubectl "$@"
}

function down () {
  vagrant halt
}

# Make sure that local config is correct
prepare_config
