#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

ARCH=$(uname -m | grep -q s390x && echo s390x || echo amd64)

source "${SCRIPT_DIR}/detect_cri.sh"

export KUBEVIRTCI_TAG=${KUBEVIRTCI_TAG:-$(date +"%y%m%d%H%M")-$(git rev-parse --short HEAD)}
export CRI_BIN=${CRI_BIN:-$(detect_cri)}

if [ -z "${CRI_BIN}" ]; then
    echo "ERROR: Neither podman nor docker is available." >&2
    exit 1
fi

TARGET_REPO="quay.io/kubevirtci"
TARGET_KUBEVIRT_REPO="quay.io/kubevirt"

function build_alpine_container_disk() {
  echo "INFO: build alpine container disk"
  (cd cluster-provision/images/vm-image-builder && ./create-containerdisk.sh alpine-cloud-init)
  if [[ "$ARCH" == "amd64" ]]; then
    ${CRI_BIN} tag alpine-cloud-init:devel "${TARGET_REPO}/alpine-with-test-tooling-container-disk:${KUBEVIRTCI_TAG}"
    ${CRI_BIN} tag alpine-cloud-init:devel "${TARGET_KUBEVIRT_REPO}/alpine-with-test-tooling-container-disk:devel"
  else
    ${CRI_BIN} tag alpine-cloud-init:devel "${TARGET_REPO}/alpine-with-test-tooling-container-disk:${KUBEVIRTCI_TAG}-${ARCH}"
    ${CRI_BIN} tag alpine-cloud-init:devel "${TARGET_KUBEVIRT_REPO}/alpine-with-test-tooling-container-disk:devel-${ARCH}"
  fi
}

function push_alpine_container_disk() {
  echo "INFO: push alpine container disk"
  if [[ "$ARCH" == "amd64" ]]; then
    TARGET_IMAGE="${TARGET_REPO}/alpine-with-test-tooling-container-disk:${KUBEVIRTCI_TAG}"
    TARGET_KUBEVIRT_IMAGE="${TARGET_KUBEVIRT_REPO}/alpine-with-test-tooling-container-disk:devel"
  else
    TARGET_IMAGE="${TARGET_REPO}/alpine-with-test-tooling-container-disk:${KUBEVIRTCI_TAG}-${ARCH}"
    TARGET_KUBEVIRT_IMAGE="${TARGET_KUBEVIRT_REPO}/alpine-with-test-tooling-container-disk:devel-${ARCH}"
  fi
  ${CRI_BIN} push "$TARGET_IMAGE"
  ${CRI_BIN} push "$TARGET_KUBEVIRT_IMAGE"
}

function publish_alpine_container_disk() {
  build_alpine_container_disk
  push_alpine_container_disk
}

publish_alpine_container_disk
