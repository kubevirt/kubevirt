#!/usr/bin/env bash

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

# remove libvirt BUILD file to regenerate it each time
rm -f vendor/libvirt.org/go/libvirt/BUILD.bazel

# generate BUILD files
bazel run \
    --config=${ARCHITECTURE} ${BAZEL_CS_CONFIG} \
    //:gazelle -- --exclude kubevirtci/cluster-up

# inject changes to libvirt BUILD file
bazel run \
    --config=${ARCHITECTURE} ${BAZEL_CS_CONFIG} \
    -- :buildozer 'add cdeps //:libvirt-libs' //vendor/libvirt.org/go/libvirt:go_default_library
# align BAZEL files to a single format
bazel run \
    --config=${ARCHITECTURE} ${BAZEL_CS_CONFIG} \
    //:buildifier
