#!/usr/bin/env bash
set -ex

##################################################################
# build-index-image build index-image(s) for HCO
# there are three options:
# * run with ALL parameter will generate index images for all the
#   versions
# * run with LATEST parameter will generate index image for the
#   latest version
# * run with a specific version name; e.g. 1.1.0, will generate
#   index image for this version
##################################################################

PROJECT_ROOT="$(readlink -e "$(dirname "${BASH_SOURCE[0]}")"/../)"
DEPLOY_DIR="${PROJECT_ROOT}/deploy/olm-catalog"
PACKAGE_NAME="community-kubevirt-hyperconverged"
INDEX_IMAGE_PARAM=
IMAGE_REGISTRY=${IMAGE_REGISTRY:-quay.io}
REGISTRY_NAMESPACE=${REGISTRY_NAMESPACE:-kubevirt}
BUNDLE_REGISTRY_IMAGE_NAME=${BUNDLE_REGISTRY_IMAGE_NAME:-hyperconverged-cluster-bundle}
INDEX_REGISTRY_IMAGE_NAME=${INDEX_REGISTRY_IMAGE_NAME:-hyperconverged-cluster-index}
OPM=${OPM:-opm}
UNSTABLE=$2


function create_index_image() {
  CURRENT_VERSION=$1
  PREV_VERSION=$2
  if [[ "${UNSTABLE}" == "UNSTABLE" ]]; then
    mv ${PACKAGE_NAME}/${CURRENT_VERSION} ${PACKAGE_NAME}/${CURRENT_VERSION}-unstable
    CURRENT_VERSION=${CURRENT_VERSION}-unstable
  fi
  BUNDLE_IMAGE_NAME="${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/${BUNDLE_REGISTRY_IMAGE_NAME}:${CURRENT_VERSION}"
  INDEX_IMAGE_NAME="${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/${INDEX_REGISTRY_IMAGE_NAME}:${CURRENT_VERSION}"

  INDEX_IMAGE_PARAM=
  if [[ -n ${2} ]]; then
    PREV_INDEX_IMAGE="${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/${INDEX_REGISTRY_IMAGE_NAME}:${PREV_VERSION}"
    INDEX_IMAGE_PARAM=--from-index="${PREV_INDEX_IMAGE}"
  fi

  docker build -t "${BUNDLE_IMAGE_NAME}" -f bundle.Dockerfile --build-arg "VERSION=${CURRENT_VERSION}" .
  docker push "${BUNDLE_IMAGE_NAME}"

  # Extract the digest of the bundle image, to be added to the index image
  BUNDLE_IMAGE_NAME=$("${PROJECT_ROOT}/tools/digester/digester" --image "${BUNDLE_IMAGE_NAME}")

  # shellcheck disable=SC2086
  ${OPM} index add --bundles "${BUNDLE_IMAGE_NAME}" ${INDEX_IMAGE_PARAM} --tag "${INDEX_IMAGE_NAME}" -u docker
  docker push "${INDEX_IMAGE_NAME}"
}

function create_all_versions() {
  PREV_VERSION=
  for version in $(ls -d ${PACKAGE_NAME}/*/ | sort -V | cut -d '/' -f 2); do
    create_index_image "${version}" "${PREV_VERSION}"
    PREV_VERSION=${version}
  done
}

function create_latest_version() {
  CURRENT_VERSION=
  PREV_VERSION=
  for version in $(ls -d ${PACKAGE_NAME}/*/ | sort -V | cut -d '/' -f 2); do
    PREV_VERSION=${CURRENT_VERSION}
    CURRENT_VERSION=${version}
  done
  create_index_image "${CURRENT_VERSION}" "${PREV_VERSION}"
}

function build_specific_version() {
  CURRENT_VERSION=
  PREV_VERSION=
  for version in $(ls -d ${PACKAGE_NAME}/*/ | sort -V | cut -d '/' -f 2); do
    PREV_VERSION=${CURRENT_VERSION}
    CURRENT_VERSION=${version}
    if [[ "$1" == "${CURRENT_VERSION}" ]]; then
      break
    fi
  done
  create_index_image "${CURRENT_VERSION}" "${PREV_VERSION}"
}

function help() {
  echo "usage $0 {ALL|LATEST|<version>} [UNSTABLE]"
}

if [[ $# -gt 2 ]] || [[ $# -lt 1 ]]; then
  help
  exit 1
fi

cd "${DEPLOY_DIR}"

case "$1" in
  [Aa][Ll][Ll])
    create_all_versions
    ;;

  [Ll][Aa][Tt][Ee][Ss][Tt])
    create_latest_version
    ;;

  [Hh][Ee][Ll][Pp])
    help
    ;;

  *)
    build_specific_version $1
    ;;

esac
