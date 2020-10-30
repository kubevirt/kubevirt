#!/usr/bin/env bash
set -euo pipefail

export DOCKER_PREFIX='kubevirtnightlybuilds'
DOCKER_TAG="$(/get-release-tag-for-master.sh)"
export DOCKER_TAG
/deploy-master.sh

/wait-for-kubevirt-ready.sh

/test.sh
