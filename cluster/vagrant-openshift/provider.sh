#!/bin/bash

function _main_ip() {
    echo 192.168.200.2
}

function up() {
    export USING_KUBE_SCRIPTS=true
    # Make sure that the vagrant environment is up and running
    vagrant up --provider=libvirt
    # Synchronize kubectl config
    vagrant ssh-config master 2>&1 | grep "not yet ready for SSH" >/dev/null &&
        {
            echo "Master node is not up"
            exit 1
        }

    OPTIONS=$(vagrant ssh-config master | grep -v '^Host ' | awk -v ORS=' ' 'NF{print "-o " $1 "=" $2}')

    scp $OPTIONS master:/usr/local/bin/oc ${KUBEVIRT_PATH}cluster/vagrant-openshift/.oc
    chmod u+x cluster/vagrant-openshift/.oc

    # Login to OpenShift
    export KUBECONFIG=${KUBEVIRT_PATH}cluster/os-3.9.0-alpha.4/.kubeconfig
    cluster/vagrant-openshift/.oc login $(_main_ip):8443 --insecure-skip-tls-verify=true -u admin -p admin

    # Make sure that local config is correct
    prepare_config
}

function prepare_config() {
    BASE_PATH=${KUBEVIRT_PATH:-$PWD}
    cat >hack/config-provider-vagrant-openshift.sh <<EOF
master_ip=$(_main_ip)
docker_tag=devel
kubeconfig=${BASE_PATH}/cluster/vagrant-openshift/.kubeconfig
EOF
}

function build() {
    make build manifests
    for VM in $(vagrant status | grep -v "^The Libvirt domain is running." | grep running | cut -d " " -f1); do
        vagrant rsync $VM # if you do not use NFS
        vagrant ssh $VM -c "cd /vagrant && export DOCKER_TAG=${docker_tag} && sudo -E hack/build-docker.sh build"
    done
}

function _kubectl() {
    export KUBECONFIG=${KUBEVIRT_PATH}cluster/vagrant-openshift/.kubeconfig
    ${KUBEVIRT_PATH}cluster/vagrant-openshift/.oc "$@"
}

function down() {
    vagrant halt
}
