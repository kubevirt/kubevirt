#!/bin/bash -x

  #    this script bumps hco to the next version.
  #
  #    By default, the next version is the next minor version, so if the
  #    current version is `1.12.0` or `1.12.3`, the next version will be
  #    `1.13.0`.
  #
  #    It is possible to change this default behavior by the bump level parameter.
  #    The supported values are `major`, `minor` (default) or `patch`; e.g.
  #
  #      ./hack/bump-hco patch
  #
  #    The script creates a new branch with the name of "bump_hco_to_v<NEXT_VERSION>"

BUMP_LEVEL=minor
if [[ -n $1 ]]; then
  BUMP_LEVEL=$1
fi

git diff --quiet
DIFF=$?
if [[ "0" != "${DIFF}" ]]; then
  echo "there are unstaged changes. this script must start with no git diffs"
  exit 1
fi

source ./hack/config

CURRENT_VERSION=${CSV_VERSION}
NEXT_VERSION=$(./hack/get-next-version.sh "${CURRENT_VERSION}" "${BUMP_LEVEL}")

echo "Bumping hyperconverged-cluster-operator to v${NEXT_VERSION}"

UPSTREAM=$(date "+%Y-%m-%dT%H-%M-upstream")
git remote add "${UPSTREAM}" https://github.com/kubevirt/hyperconverged-cluster-operator.git
git fetch "${UPSTREAM}" main
git checkout -b "bump_hco_to_v${NEXT_VERSION}" "${UPSTREAM}/main"
git remote remove "${UPSTREAM}"

echo "modify files..."

SHORT_NEXT="${NEXT_VERSION%.*}"
#SHORT_CURRENT="${CURRENT_VERSION%.*}"
VERSION_4_SED=$(echo "${NEXT_VERSION}" | sed -E "s|\.|\\\\\\\\\\\.|g")

sed -i -E "s#(quay.io/kubevirt/hyperconverged-cluster-[^:]+:)[0-9\.]+(-unstable)#\1${NEXT_VERSION}\2#g;s|(channel: \"candidate-v)[^\"]+|\1${SHORT_NEXT}|g" README.md
sed -i -E "s|(ARG VERSION=).*|\1${NEXT_VERSION}|g" deploy/index-image/bundle.Dockerfile
sed -i -E "s|(ARG INITIAL_VERSION=).*|\1${NEXT_VERSION}|g;s|(ARG INITIAL_VERSION_SED=).*|\1\"${VERSION_4_SED}\"|g" deploy/index-image/Dockerfile.bundle.ci-index-image-upgrade
sed -i -E "s|(ARG VERSION=).*|\1${NEXT_VERSION}|g" deploy/olm-catalog/bundle.Dockerfile
sed -i -E "s|(ARG INITIAL_VERSION=).*|\1${NEXT_VERSION}|g;s|(ARG INITIAL_VERSION_SED=).*|\1\"${VERSION_4_SED}\"|g" deploy/olm-catalog/Dockerfile.bundle.ci-index-image-upgrade
sed -i -E "s|(quay.io/kubevirt/hyperconverged-cluster-bundle:).*|\1${NEXT_VERSION}|g" deploy/olm-catalog/community-kubevirt-hyperconverged/index-template-release.yaml
sed -i -E "s|(quay.io/kubevirt/hyperconverged-cluster-bundle:).*|\1${NEXT_VERSION}|g" deploy/olm-catalog/community-kubevirt-hyperconverged/index-template-unstable.yaml
sed -i -E "s|(HCO_CHANNEL:-candidate-v)[0-9\.]+|\1${SHORT_NEXT}|g;s|(HCO_INDEX_IMAGE:-quay.io/kubevirt/hyperconverged-cluster-index:)[0-9\.]+(-unstable)|\1${NEXT_VERSION}\2|g" deploy/kustomize/deploy_kustomize.sh
sed -i -E "s|(quay.io/kubevirt/hyperconverged-cluster-functest:)[0-9.]+(-unstable)|\1${NEXT_VERSION}\2|g" docs/functest-container.md
sed -i -E "s|(Version = \")[^\"]+|\1${NEXT_VERSION}|" version/version.go
sed -i -E "s|(MID_VERSION=).+$|\1${NEXT_VERSION}|" hack/consecutive-upgrades-test.sh

mkdir -p "${PACKAGE_DIR}/${NEXT_VERSION}"
make build-manifests

git add ./deploy
git commit -s -a -m "Bump HCO to version ${NEXT_VERSION}"