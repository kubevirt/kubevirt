#!/usr/bin/env bash

set -ex

source hack/common.sh
source hack/config.sh
source hack/install-bazeldnf.sh

# verify that RPMs with given SHASUMs in WORKSPACE files
# are signed with known GPG keys in repo.yaml
bazeldnf verify \
    --repofile "rpm/repo.yaml"
