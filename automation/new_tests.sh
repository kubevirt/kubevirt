#!/bin/bash

set -euo pipefail

function usage {
    cat <<EOF
usage: $0 [commit_id]

    generate a regex string containing changed tests for using it with
    [ginkgo.focus](https://onsi.github.io/ginkgo/#focused-specs) parameter.
    Return a string in the form

        'test1|test2|...|testn'

EOF
}

if [[ "$#" -gt 0 ]]; then
    if [[ "$1" =~ '-h' ]]; then
        usage
        exit 0
    fi
    TARGET_COMMIT="$1"
else
    # if there's no commit provided default to the latest merge commit
    TARGET_COMMIT=$(git log -1 --format=%H --merges)
fi

# 1. fetch the changed file names from within the tests/ directory
# 2. grep all files ending with '_test.go'
# 3. remove `tests/` and `_test.go` to only have the test name
# 4. replace newline with `|`
# 5. remove last `|`
git diff --name-only "${TARGET_COMMIT}".. -- tests/ \
    | grep -E '_test\.go$' \
    | sed -E 's/tests\/(.*_test)\.go/\1/' \
    | tr '\n' '|' \
    | sed -E 's/\|$//'
