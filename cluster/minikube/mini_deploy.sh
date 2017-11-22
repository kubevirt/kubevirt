#!/bin/bash 
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2017 Red Hat, Inc.

# get the config used widely in the kubevirt 
docker_images="cmd/virt-controller" # cmd/virt-launcher cmd/virt-handler cmd/virt-api cmd/virt-manifest" # images/haproxy images/iscsi-demo-target-tgtd images/vm-killer images/libvirt-kubevirt images/spice-proxy cmd/virt-migrator cmd/registry-disk-v1alpha images/cirros-registry-disk-demo cmd/virt-dhcp"
docker_prefix=kubevirt
docker_tag=devel

# common helper function
bold() { echo -e "\e[1m$@\e[0m" ; }
red() { echo -e "\e[31m$@\e[0m" ; }
green() { echo -e "\e[32m$@\e[0m" ; }

die() { red "ERR: $@" >&2 ; exit 2 ; }
silent() { "$@" > /dev/null 2>&1 ; }
has_bin() { silent which $1 ; }
title() { bold "$@" ; }
par() { echo -e "- $@" ; }
parn() { echo -en "- $@ ... " ; }
ok() { green "${@:-OK}" ; }

pushd() { command pushd "$@" >/dev/null ; }
popd() { command popd "$@" >/dev/null ; }

LOCALBIN=$HOME/.local/bin
export PATH=$LOCALBIN:$PATH

TMPD=/var/tmp/kubevirt-demo
PLUGIN_PATH="$HOME/.kube/plugins/virtctl"

check_kubectl() {
  parn "Checking kubectl version"
  local CTLVER=$(kubectl version --short --client)
  egrep -q "1.[78]" <<< $CTLVER || \
    die "kubectl needs to be 1.7 or higher: $CTLVER"
  ok
}

check_for_minikube() {
  parn "Checking for minikube"
  has_bin minikube || \
    die "minikube not found. Please install minikube, see
https://github.com/kubernetes/minikube for details."
  ( minikube status | grep -qsi stopped ) && \
    die "minikube is installed but not started. Please start minikube."
  ok
}

deploy_kubevirt() {
  par "Deploying manifests - this can take several minutes!"
  {
    # set the docker environment to minicube internal one
    eval $(minikube docker-env)

    # build the docker so it can be deployed on the minikube
    _op_docker

    # deploy the manifests to the minikube
    _op_manifests apply

    par "Waiting for the cluster to be ready ..."
    kubectl get pods -w | while read LINE 
    do
      echo -n "  Cluster changed, checking if KubeVirt is ready ... "
      if ! kubectl get pods | grep -qs ContainerCreating; then
        ok "KubeVirt is now ready on you local minicube"
        kill $(pidof -- kubectl get pods -w)
        break
      fi
      echo "Not yet."
    done
  }
}

undeploy_kubevirt() {
  parn "Removing KubeVirt from minikube"
  _op_manifests delete
  eval $(minikube docker-env --unset)
  ok
}

_op_docker() {
  parn "Building the docker images"

  # build docker images used by the kubevirt
  for arg in $docker_images; do
    (cd $arg; docker build -t ${docker_prefix}/$(basename $arg):${docker_tag} -f Dockerfile .)
  done

  ok
}

_op_manifests() {
  local OP=$1

  pushd manifests
    # Fill in templates
    local MASTER_IP=$(minikube ip)
    local DOCKER_PREFIX=kubevirt
    local DOCKER_TAG=${DOCKER_TAG}
    local PRIMARY_NIC=eth0
    for TPL in *.yaml.in; do
       sed -e "s/{{ master_ip }}/$MASTER_IP/g" \
           -e "s/{{ docker_prefix }}/$DOCKER_PREFIX/g" \
           -e "s/{{ docker_tag }}/$DOCKER_TAG/g" \
           -e "s/{{ primary_nic }}/$PRIMARY_NIC/g" \
           -e "s#qemu.*/system#qemu+tcp://minikube/system#"  \
           -e "s#kubernetes.io/hostname:.*#kubernetes.io/hostname: minikube#" \
           $TPL > ${TPL%.in}
    done
  popd

  # Deploying
  for M in manifests/*.yaml; do
    silent kubectl $OP -f $M
  done

  [[ "$OP" != "delete" ]] && kubectl $OP -f cluster/vm.json
}

main() {
  bold "KubeVirt dev minikube env"

  case $1 in
    help) cat <<EOF
Usage: $0 [deploy|undeploy]
  deploy   - (default) Deploy KubeVirt to the local minikube
  undeploy - Remove the previously deployed KubeVirt deployment
EOF
;;
    undeploy)
      check_kubectl; check_for_minikube
      undeploy_kubevirt ;;
    deploy)
      check_kubectl; check_for_minikube
      deploy_kubevirt ;;
    build)
      check_kubectl; check_for_minikube
      eval $(minikube docker-env)
      _op_docker
      ;;
    *)
      check_kubectl; check_for_minikube
      ;;
esac
}

main $@

# vim: et ts=2
