#!/usr/bin/env bash

ARCHITECTURES="amd64 arm64 s390x"

if [[ -z ${IMAGE_NAME} ]]; then
  echo "IMAGE_NAME must be defined"
  exit 1
fi

if [[ -z ${DOCKER_FILE} ]]; then
  echo "DOCKER_FILE must be defined"
  exit 1
fi

SHA=$(git describe --no-match  --always --abbrev=40 --dirty)

IMAGES=
for arch in ${ARCHITECTURES}; do
  . "hack/cri-bin.sh" && ${CRI_BIN} build  --platform=linux/${arch} -f ${DOCKER_FILE} -t "${IMAGE_NAME}-${arch}" --build-arg git_sha=${SHA} .
  IMAGES="${IMAGES} ${IMAGE_NAME}-${arch}"
done

. "hack/cri-bin.sh" && ${CRI_BIN} manifest create "${IMAGE_NAME}" ${IMAGES}
