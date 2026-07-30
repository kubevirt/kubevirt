#!/usr/bin/env bash

set -ex

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

repofiles=(
    --repofile "rpm/repo.yaml"
    --repofile "rpm/repo-cs10.yaml"
)

# verify that RPMs with given SHASUMs in WORKSPACE files
# are signed with known GPG keys in the configured repo files
bazel run \
    --config=${ARCHITECTURE} \
    //:bazeldnf -- verify \
    "${repofiles[@]}"
