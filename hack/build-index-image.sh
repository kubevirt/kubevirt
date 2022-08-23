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
OUT_DIR="${PROJECT_ROOT}/_out"
PACKAGE_NAME="community-kubevirt-hyperconverged"
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
    PREV_VERSION=${PREV_VERSION}-unstable
  fi
  BUNDLE_IMAGE_NAME="${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/${BUNDLE_REGISTRY_IMAGE_NAME}:${CURRENT_VERSION}"
  INDEX_IMAGE_NAME="${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/${INDEX_REGISTRY_IMAGE_NAME}:${CURRENT_VERSION}"

  podman build -t "${BUNDLE_IMAGE_NAME}" -f bundle.Dockerfile --build-arg "VERSION=${CURRENT_VERSION}" .
  podman push "${BUNDLE_IMAGE_NAME}"

  # Referencing the unstable bundle digest in the index image, rather than the floating tag, to avoid
  # unalignment between cached index image and fetched bundle image.
  BUNDLE_IMAGE_NAME=$("${PROJECT_ROOT}/tools/digester/digester" --image "${BUNDLE_IMAGE_NAME}")

  if [[ "${UNSTABLE}" == "UNSTABLE" ]]; then
    # Currently, ci-operator does not support index images with file-based catalogs.
    # To maintain CI functionality, we'll keep using SQLite-based catalogs for the unstable tags
    # until FBC handling will be implemented in openshift ci-operator.
    # shellcheck disable=SC2086
    ${OPM} index add --bundles "${BUNDLE_IMAGE_NAME}" ${INDEX_IMAGE_PARAM} --tag "${INDEX_IMAGE_NAME}" -u podman --mode semver
  else
    mkdir -p "${OUT_DIR}"
    (cd "${OUT_DIR}" && create_file_based_catalog)
  fi

  podman push "${INDEX_IMAGE_NAME}"
}

function create_file_based_catalog() {
  # File-Based Catalog handling
  rm -rf fbc-catalog*
  ${OPM} migrate "${INDEX_IMAGE_NAME}" fbc-catalog || true
  if [ ! -d fbc-catalog ]
  then
    # The index image is already in file-based format. Extracting its catalog file
    oc image extract "${INDEX_IMAGE_NAME}" --file /configs/catalog.json
  else
    # The migration took place
    mv fbc-catalog/community-kubevirt-hyperconverged/catalog.json catalog.json
  fi

  ${OPM} render "${BUNDLE_IMAGE_NAME}" > bundle.json
  CSV_NAME=$(jq -r .name bundle.json)
  VERSION=${CSV_NAME##*.v}
  SKIPRANGE="<"${VERSION}
  CHANNEL=${CURRENT_VERSION%-*}

  # Remove the existing channel schema from the catalog
  jq --arg CHANNEL "$CHANNEL" 'del(. | select(.schema=="olm.channel" and .name==$CHANNEL))' \
   catalog.json > updated_fbc.json

  # Insert the new channel schema for the new bundle instead of the one we removed previously
  jq --arg CSV_NAME "$CSV_NAME" --arg SKIPRANGE "$SKIPRANGE" --arg CHANNEL "$CHANNEL" '. |
    select(.schema=="olm.channel" and .name==$CHANNEL) |
    .entries[0].name=$CSV_NAME |
    .entries[0].skipRange=$SKIPRANGE' \
      catalog.json >> updated_fbc.json

  mv updated_fbc.json updated_fbc.json.tmp
  # Remove the existing bundle schema from the catalog
  jq --arg CHANNEL "$CHANNEL" 'del(. |
    select(.schema=="olm.bundle" and (.name | contains($CHANNEL))))' \
    updated_fbc.json.tmp > updated_fbc.json

  mkdir -p fbc-catalog
  cat bundle.json >> updated_fbc.json
  sed -i '/null/d' updated_fbc.json
  mv updated_fbc.json fbc-catalog/catalog.json
  ${OPM} validate fbc-catalog
  ${OPM} generate dockerfile fbc-catalog
  podman build -t "${INDEX_IMAGE_NAME}" -f fbc-catalog.Dockerfile
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
