#!/bin/bash

set -e

source $(dirname "$0")/common.sh

cd ${KUBEVIRT_DIR}

test -f .glide.yaml.hash || md5sum glide.yaml >.glide.yaml.hash
if [ "$(md5sum glide.yaml)" != "$(cat .glide.yaml.hash)" ]; then
    glide cc
    glide update --strip-vendor
    md5sum glide.yaml >.glide.yaml.hash
    md5sum glide.lock >.glide.lock.hash
elif [ "$(md5sum glide.lock)" != "$(cat .glide.lock.hash)" ]; then
    glide install --strip-vendor && md5sum glide.lock >.glide.lock.hash
fi
