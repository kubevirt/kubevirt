#!/bin/bash

set -euo pipefail

if [[ "$#" -eq 0 ]] || [[ -z "$1" ]]; then
    TARGET_COMMIT=$(git log -1 --format=%H --merges)
else
    TARGET_COMMIT="$1"
fi
git diff --name-only "${TARGET_COMMIT}".. -- tests/ \
    | grep -E '_test\.go$' \
    | sed -E 's/tests\/(.*_test)\.go/\1/' \
    | tr '\n' '|' \
    | sed -E 's/\|$//'
