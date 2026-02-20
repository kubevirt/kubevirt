#!/usr/bin/env bash

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

# remove libvirt BUILD file to regenerate it each time
rm -f vendor/libvirt.org/go/libvirt/BUILD.bazel

# generate BUILD files
bazel run \
    --config=${ARCHITECTURE} \
    //:gazelle -- --exclude kubevirtci/cluster-up

# inject changes to libvirt BUILD file
bazel run \
    --config=${ARCHITECTURE} \
    -- :buildozer 'add cdeps //:libvirt-libs' //vendor/libvirt.org/go/libvirt:go_default_library

# inject changes to libnbd BUILD file
bazel run \
    --config=${HOST_ARCHITECTURE} \
    -- :buildozer 'add clinkopts -lnbd' //vendor/libguestfs.org/libnbd/:go_default_library

# libvirt and libnbd cgo bindings share freeCallbackId, override libnbd to avoid conflict
bazel run \
    --config=${HOST_ARCHITECTURE} \
    -- :buildozer 'add copts -D_GNU_SOURCE=1 -DfreeCallbackId=nbd_freeCallbackId' //vendor/libguestfs.org/libnbd/:go_default_library

# align BAZEL files to a single format
bazel run \
    --config=${ARCHITECTURE} \
    //:buildifier
