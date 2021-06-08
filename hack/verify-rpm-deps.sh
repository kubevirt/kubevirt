#!/usr/bin/env bash

set -ex

source hack/common.sh
source hack/config.sh

# verify that RPMs with given SHASUMs in WORKSPACE files
# are signed with known GPG keysin repo.yaml
bazel run \
    --config=${ARCHITECTURE} \
    //:bazeldnf -- verify \
    --repofile rpm/centos-repo.yaml \
    --repofile rpm/fedora-repo.yaml
