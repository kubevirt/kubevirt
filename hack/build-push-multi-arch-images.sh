#!/usr/bin/env bash

source ./hack/architecture.sh

if [[ -z ${IMAGE_NAME} ]]; then
  echo "IMAGE_NAME must be defined"
  exit 1
fi

if [[ -z ${DOCKER_FILE} ]]; then
  echo "DOCKER_FILE must be defined"
  exit 1
fi

SHA=$(git describe --no-match  --always --abbrev=40 --dirty)

. ./hack/cri-bin.sh && export CRI_BIN=${CRI_BIN}

${CRI_BIN} manifest create -a "${IMAGE_NAME}"
for arch in ${ARCHITECTURES}; do
  ${CRI_BIN} build  --platform=linux/${arch} -f ${DOCKER_FILE} -t "${IMAGE_NAME}-${arch}" --build-arg git_sha=${SHA} .
  ./hack/retry.sh 3 10 "${CRI_BIN} push ${IMAGE_NAME}-${arch}"
  ${CRI_BIN} manifest add "${IMAGE_NAME}" "${IMAGE_NAME}-${arch}"
done

./hack/retry.sh 3 10 "${CRI_BIN} manifest push ${IMAGE_NAME}"
