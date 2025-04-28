#!/usr/bin/env bash

source ./hack/architecture.sh

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
ORIG_DEPLOY_DIR="${PROJECT_ROOT}/deploy/olm-catalog"
OUT_DIR="${PROJECT_ROOT}/_out"
DEPLOY_DIR="${OUT_DIR}/deploy/olm-catalog"
PACKAGE_NAME="community-kubevirt-hyperconverged"
IMAGE_REGISTRY=${IMAGE_REGISTRY:-quay.io}
REGISTRY_NAMESPACE=${REGISTRY_NAMESPACE:-kubevirt}
BUNDLE_REGISTRY_IMAGE_NAME=${BUNDLE_REGISTRY_IMAGE_NAME:-hyperconverged-cluster-bundle}
INDEX_REGISTRY_IMAGE_NAME=${INDEX_REGISTRY_IMAGE_NAME:-hyperconverged-cluster-index}
OPM=${OPM:-opm}
UNSTABLE=$2

function create_index_image() {
  CURRENT_VERSION=$1
  INITIAL_VERSION=${CURRENT_VERSION}
  if [[ "${UNSTABLE}" == "UNSTABLE" ]]; then
    mv ${PACKAGE_NAME}/${CURRENT_VERSION} ${PACKAGE_NAME}/${CURRENT_VERSION}-unstable
    CURRENT_VERSION=${CURRENT_VERSION}-unstable
  fi

  if [[ -z ${IMAGE_TAG} ]]; then
    IMAGE_TAG=${CURRENT_VERSION}
  fi

  BUNDLE_IMAGE_NAME="${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/${BUNDLE_REGISTRY_IMAGE_NAME}:${IMAGE_TAG}"
  INDEX_IMAGE_NAME="${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/${INDEX_REGISTRY_IMAGE_NAME}:${IMAGE_TAG}"

  podman build -t "${BUNDLE_IMAGE_NAME}" -f bundle.Dockerfile --build-arg "VERSION=${CURRENT_VERSION}" .
  podman push "${BUNDLE_IMAGE_NAME}"

  # Referencing the unstable bundle digest in the index image, rather than the floating tag, to avoid
  # unalignment between cached index image and fetched bundle image.
  BUNDLE_IMAGE_NAME=$("${PROJECT_ROOT}/tools/digester/digester" --image "${BUNDLE_IMAGE_NAME}")

  create_file_based_catalog "${INITIAL_VERSION}" "${BUNDLE_IMAGE_NAME}" "${UNSTABLE}"
}

function create_file_based_catalog() {
  INITIAL_VERSION=$1
  BUNDLE_IMAGE_NAME=$2
  UNSTABLE=$3

  # File-Based Catalog handling
  cd "${OUT_DIR}"
  template_file=index-template-release.yaml
  if [[ "${UNSTABLE}" == "UNSTABLE" ]]; then
    template_file=index-template-unstable.yaml
  fi

  cp "deploy/olm-catalog/community-kubevirt-hyperconverged/${template_file}" ./index-template.yaml

  if [[ "quay.io/kubevirt/hyperconverged-cluster-bundle:${INITIAL_VERSION}" != "${BUNDLE_IMAGE_NAME}" ]]; then
    sed -i -E "s|quay.io/kubevirt/hyperconverged-cluster-bundle:${INITIAL_VERSION}|${BUNDLE_IMAGE_NAME}|g" index-template.yaml
  fi

  while IFS= read -r line; do
    image=$(echo "${line}" | sed -E 's|.*(quay.io/kubevirt/hyperconverged-cluster-bundle:.*)|\1|g')
    digest=$("${PROJECT_ROOT}/tools/digester/digester" --image "${image}")
    sed -i "s|${image}|${digest}|g" index-template.yaml
  done < <(grep 'quay.io/kubevirt/hyperconverged-cluster-bundle:' index-template.yaml)

  rm -rf fbc-catalog
  mkdir fbc-catalog
  ${OPM} alpha render-template semver --migrate-level=bundle-object-to-csv-metadata index-template.yaml > fbc-catalog/catalog.json
  ${OPM} validate fbc-catalog
  rm -f fbc-catalog.Dockerfile
  ${OPM} generate dockerfile fbc-catalog

  IMAGES=
  for arch in ${ARCHITECTURES}; do
    current_image="${INDEX_IMAGE_NAME}-${arch}"
    podman build --platform="linux/${arch}" -t "${current_image}" -f fbc-catalog.Dockerfile
    podman push "${current_image}"
    IMAGES="${IMAGES} ${current_image}"
  done

  podman manifest create "${INDEX_IMAGE_NAME}" ${IMAGES}
  podman manifest push "${INDEX_IMAGE_NAME}"
}

function create_all_versions() {
  PREV_VERSION=
  for version in $(ls -d ${PACKAGE_NAME}/*/ | sort -V | cut -d '/' -f 2); do
    create_index_image "${version}"
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
  create_index_image "${CURRENT_VERSION}"
}

function build_specific_version() {
  CURRENT_VERSION=
  for version in $(ls -d ${PACKAGE_NAME}/*/ | sort -V | cut -d '/' -f 2); do
    if [[ "$1" == "${version}" ]]; then
      CURRENT_VERSION="${version}"
      break
    fi
  done
  if [[ -z ${CURRENT_VERSION} ]]; then
    echo "can't find version $1"
    exit 1
  fi
  create_index_image "${CURRENT_VERSION}"
}

function help() {
  echo "usage $0 {ALL|LATEST|<version>} [UNSTABLE]"
}

if [[ $# -gt 2 ]] || [[ $# -lt 1 ]]; then
  help
  exit 1
fi

rm -rf "${DEPLOY_DIR}"
mkdir -p "${DEPLOY_DIR}"
cp -r ${ORIG_DEPLOY_DIR}/* "${DEPLOY_DIR}/"
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
