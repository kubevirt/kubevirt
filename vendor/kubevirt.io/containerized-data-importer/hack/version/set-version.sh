#!/usr/bin/env bash

set -eou pipefail

# containerized-data-importer/
REPO_ROOT="$(readlink -f $(dirname $0)/../../)"

source $REPO_ROOT/hack/version/lib.sh

NEW_VERSION="${1:-}"
CUR_VERSION="$(getCurrentVersion)"

if [ -z "$CUR_VERSION" ]; then
    printf "Could not get current version from git tags\n"
    exit 1
fi
verifyVersionFormat "$NEW_VERSION"
verifyNoDiff
TARGET_FILES=$(getVersionedFiles "$CUR_VERSION" "$REPO_ROOT")
if [ -z "$TARGET_FILES" ]; then
    printf "Zero files found containing current version %s, aborting\n" "$CUR_VERSION"
    exit 1
fi
acceptChanges "$CUR_VERSION" "$NEW_VERSION" "$TARGET_FILES"
for f in $TARGET_FILES; do
    setNewVersion $f "$CUR_VERSION" "$NEW_VERSION"
done
commitAndTag "$NEW_VERSION" "$TARGET_FILES"
