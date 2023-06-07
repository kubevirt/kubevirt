#!/bin/bash

set -ex

source "$(dirname "$0")"/../common.sh
source "$(dirname "$0")"/../config.sh

# FIXME(lyarwood): This is pretty dumb, replace if/when other tooling becomes available to determine this
KUBEVIRT_BUILDER_IMAGE_TAG=${KUBEVIRT_BUILDER_IMAGE_TAG:-$(${KUBEVIRT_CRI} search --list-tags --limit 500 quay.io/kubevirt/builder | awk '{print $2}' | sort -rn | head -n3 | grep -Ev "arm64|amd64")}
sed -i "s/KUBEVIRT_BUILDER_IMAGE\=.*/KUBEVIRT_BUILDER_IMAGE=\${KUBEVIRT_BUILDER_IMAGE\:\-\"quay.io\/kubevirt\/builder:${KUBEVIRT_BUILDER_IMAGE_TAG}\"}/g" hack/dockerized
