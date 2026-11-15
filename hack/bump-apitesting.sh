#!/usr/bin/env bash
set -euo pipefail

if [ -z "${VERSION:-}" ]; then
    echo "No VERSION specified. Please specify a VERSION to bump, e.g. VERSION=release-1.9"
    exit 1
fi

# The compatibility runner only discovers fixtures under testdata/release-*, so
# the destination name is validated independently of the source ref.
if [[ ! "${VERSION}" =~ ^release-[0-9]+\.[0-9]+$ ]]; then
    echo "Invalid VERSION '${VERSION}'. Expected a fixture directory name like release-1.9"
    exit 1
fi

# Ref the fixtures are taken from. Defaults to the release branch, but a tag
# such as v1.9.0 makes the snapshot reproducible.
SOURCE_REF="${SOURCE_REF:-${VERSION}}"

if ! git rev-parse --verify --quiet "${SOURCE_REF}^{commit}" >/dev/null; then
    echo "Ref ${SOURCE_REF} not found locally. Fetch it first, e.g.:"
    echo "    git fetch <remote> ${SOURCE_REF}"
    exit 1
fi

TESTDATA_DIR="staging/src/kubevirt.io/api/apitesting/testdata"
DEST_DIR="${TESTDATA_DIR}/${VERSION}"

if [ -e "${DEST_DIR}" ]; then
    echo "Destination ${DEST_DIR} already exists. Remove it first to regenerate the snapshot."
    exit 1
fi

OLD_VERSIONS=$(cd "${TESTDATA_DIR}" && ls -d release-* 2>/dev/null || true)
echo "Bumping API testdata: ${OLD_VERSIONS:-<none>} -> ${VERSION} (from ${SOURCE_REF})"

# Extract into a temporary directory so a failure can never touch fixtures
# already in the worktree.
TMP_DIR=$(mktemp -d)
trap 'rm -rf "${TMP_DIR}"' EXIT

git archive "${SOURCE_REF}" -- "${TESTDATA_DIR}/HEAD/" | tar -x -C "${TMP_DIR}"

SNAPSHOT_DIR="${TMP_DIR}/${TESTDATA_DIR}/HEAD"
if [ -z "$(ls -A "${SNAPSHOT_DIR}" 2>/dev/null)" ]; then
    echo "No testdata found in ${TESTDATA_DIR}/HEAD at ${SOURCE_REF}"
    exit 1
fi

for old_version in "${TESTDATA_DIR}"/release-*; do
    [ -d "${old_version}" ] || continue
    echo "Removing old testdata: ${old_version}"
    git rm -rf "${old_version}"
done

mv "${SNAPSHOT_DIR}" "${DEST_DIR}"
git add "${DEST_DIR}"

echo "Done $0"
