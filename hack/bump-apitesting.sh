#!/usr/bin/env bash
set -e

if [ -z "${VERSION}" ]; then
    echo "No VERSION specified. Please specify a VERSION to bump"
    exit 1
fi

TESTDATA_DIR="staging/src/kubevirt.io/api/apitesting/testdata"

OLD_VERSIONS=$(cd "${TESTDATA_DIR}" && echo release-* 2>/dev/null)
echo "Bumping API testdata: ${OLD_VERSIONS} -> ${VERSION}"

mkdir -p "${TESTDATA_DIR}/${VERSION}"
trap 'rm -rf "${TESTDATA_DIR}/${VERSION}"' ERR
git archive "${VERSION}" -- "${TESTDATA_DIR}/HEAD/" | tar -x --strip-components=7 -C "${TESTDATA_DIR}/${VERSION}/"

for old_version in "${TESTDATA_DIR}"/release-*; do
    [ -d "${old_version}" ] || continue
    [ "${old_version}" = "${TESTDATA_DIR}/${VERSION}" ] && continue
    echo "Removing old testdata: ${old_version}"
    git rm -rf "${old_version}"
done

git add "${TESTDATA_DIR}/${VERSION}"

echo "Done $0"
