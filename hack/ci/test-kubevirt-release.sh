#!/usr/bin/env bash
set -euo pipefail

release="$(/get-release-tag-for-xy.sh "$1")"
export DOCKER_TAG="$release"
/deploy-release.sh "$release"

/wait-for-kubevirt-ready.sh

/test.sh
