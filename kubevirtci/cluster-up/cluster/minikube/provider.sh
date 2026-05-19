#!/usr/bin/env bash

set -e

: ${MINIKUBE:="minikube --profile=kubevirtci"}
: ${BASE_PATH:=${KUBEVIRTCI_CONFIG_PATH:-$PWD}}
: ${CI_CONFIG:=${BASE_PATH}/${KUBEVIRT_PROVIDER}}
: ${KUBECONFIG:=${CI_CONFIG}/.kubeconfig}
: ${CRI_BIN:="podman"}

function _kubectl() {
  ${CI_CONFIG}/.kubectl "$@"
}

function prepare_config() {
  # Copy kubectl and kubeconfig
  cp $(which kubectl) ${CI_CONFIG}/.kubectl 2>/dev/null || true
  ${MINIKUBE} kubectl -- config view --flatten > "${CI_CONFIG}/.kubeconfig"
  # Create provider config
  cat >"${CI_CONFIG}/config-provider-${KUBEVIRT_PROVIDER}.sh" <<EOF
master_ip="127.0.0.1"
kubeconfig="${CI_CONFIG}/.kubeconfig"
kubectl="${CI_CONFIG}/.kubectl"
docker_prefix=\${DOCKER_PREFIX:-localhost:${HOST_PORT}/kubevirt}
manifest_docker_prefix=\${DOCKER_PREFIX:-registry:5000/kubevirt}
EOF
}

function up() {
  ${MINIKUBE} --driver=${CRI_BIN} start
  prepare_config
  echo "${KUBEVIRT_PROVIDER} cluster is ready"
}

function down() {
  ${MINIKUBE} delete
  echo "${KUBEVIRT_PROVIDER} cluster is down"
}
