#!/usr/bin/env bash

set -ex

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

# verify that RPMs with given digests in rpms.json
# are signed with known GPG keys in repo-cs9.yaml and repo-cs10.yaml
bazel run \
    --config=${ARCHITECTURE} \
    //:bazeldnf -- verify \
    --lockfile "rpms.json" \
    --repofile "rpm/repo-cs9.yaml" \
    --repofile "rpm/repo-cs10.yaml"
