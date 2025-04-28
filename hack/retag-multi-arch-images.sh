#!/usr/bin/env bash

source ./hack/architecture.sh

if [[ -z ${IMAGE_REPO} ]]; then
  echo "IMAGE_REPO must be defined"
  exit 1
fi

NEW_IMAGE_REPO=${NEW_IMAGE_REPO:-${IMAGE_REPO}}

if [[ -z ${CURRENT_TAG} ]]; then
  echo "CURRENT_TAG must be defined"
  exit 1
fi

if [[ -z ${NEW_TAG} ]]; then
  echo "NEW_TAG must be defined"
  exit 1
fi

. ./hack/cri-bin.sh && export CRI_BIN=${CRI_BIN}

if [[ "${MULTIARCH}" == "true" ]]; then
  for arch in ${ARCHITECTURES}; do
    NEW_IMAGE="${NEW_IMAGE_REPO}:${NEW_TAG}-${arch}"
    ${CRI_BIN} tag "${IMAGE_REPO}:${CURRENT_TAG}-${arch}" "${NEW_IMAGE}"
    ./hack/retry.sh 3 10 "${CRI_BIN} push ${NEW_IMAGE}"
  done
fi

# retag the manifest
NEW_IMAGE="${NEW_IMAGE_REPO}:${NEW_TAG}"
${CRI_BIN} tag "${IMAGE_REPO}:${CURRENT_TAG}" "${NEW_IMAGE}"
./hack/retry.sh 3 10 "${CRI_BIN} push ${NEW_IMAGE}"
