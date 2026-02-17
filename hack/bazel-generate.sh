#!/usr/bin/env bash

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

# remove libvirt and libnbd BUILD files to regenerate them each time
rm -f vendor/libvirt.org/go/libvirt/BUILD.bazel
rm -f vendor/libguestfs.org/libnbd/BUILD.bazel

# generate BUILD files
bazel run \
    --config=${ARCHITECTURE} ${BAZEL_CS_CONFIG} \
    //:gazelle -- --exclude kubevirtci/cluster-up

# inject changes to libvirt BUILD file
bazel run \
    --config=${ARCHITECTURE} ${BAZEL_CS_CONFIG} \
    -- :buildozer 'add cdeps //:libvirt-libs' //vendor/libvirt.org/go/libvirt:go_default_library

# inject changes to libnbd BUILD file
bazel run \
    --config=${ARCHITECTURE} ${BAZEL_CS_CONFIG} \
    -- :buildozer 'add cdeps //:libnbd-libs' //vendor/libguestfs.org/libnbd/:go_default_library

# align BAZEL files to a single format
bazel run \
    --config=${ARCHITECTURE} ${BAZEL_CS_CONFIG} \
    //:buildifier
