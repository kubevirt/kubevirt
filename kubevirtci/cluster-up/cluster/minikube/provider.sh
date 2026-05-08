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
  # Copy kubectl binary
  cp $(which kubectl) ${CI_CONFIG}/.kubectl 2>/dev/null || true
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
