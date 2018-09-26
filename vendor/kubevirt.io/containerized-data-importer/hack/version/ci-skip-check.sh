#!/usr/bin/env bash

# ci-skip-check.sh
# Travis does not have a way to skip tag builds

if [[ $(git name-rev --name-only --tags HEAD) != *"undefined"* ]] &&
    [[ $(git tag -ln --format '%(subject)' $(git describe --exact-match --tags HEAD)) == *'[ci skip]'* ]]; then
    exit 1
fi
