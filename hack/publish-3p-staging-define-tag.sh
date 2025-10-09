#!/usr/bin/env bash

set -exo pipefail

function define_tag() {
    tag_suffix="v12n"
    # Check if the branch matches the pattern
    if [[ "$BRANCH_NAME" =~ ^(v[0-9]+\.[0-9]+\.[0-9])+-virtualization$ ]]; then
    # Extract last index from existing similar tags
        version="${BASH_REMATCH[1]}"
        latest_tag=$(git tag --list "${version}-${tag_suffix}.*" --sort=-v:refname | head --lines 1)

        if [[ "$latest_tag" =~ ^.+-${tag_suffix}\.([0-9]+)$ ]]; then
            last_index=${BASH_REMATCH[1]}
            next_index=$((last_index + 1))
        else
            next_index=1
        fi

        new_tag="${version}-${tag_suffix}.${next_index}"
    else
    # Use commit hash for other branches
        new_tag="v0.0.0-${tag_suffix}-3p-kubevirt-${COMMIT_HASH}"
    fi

    echo "TARGET_TAG=${new_tag}" >> $GITHUB_ENV
}

define_tag
