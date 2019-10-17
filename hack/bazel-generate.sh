#!/usr/bin/env bash

source hack/common.sh
source hack/config.sh

# remove libvirt BUILD file to regenerate it each time
rm -f vendor/github.com/libvirt/libvirt-go/BUILD.bazel

# generate BUILD files
bazel run \
    --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64_cgo \
    --workspace_status_command=./hack/print-workspace-status.sh \
    --host_force_python=${bazel_py} \
    //:gazelle

# inject changes to libvirt BUILD file
bazel run \
    --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64_cgo \
    --workspace_status_command=./hack/print-workspace-status.sh \
    --host_force_python=${bazel_py} \
    -- @com_github_bazelbuild_buildtools//buildozer 'add cdeps //:libvirt-libs //:libvirt-headers' //vendor/github.com/libvirt/libvirt-go:go_default_library
bazel run \
    --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64_cgo \
    --workspace_status_command=./hack/print-workspace-status.sh \
    --host_force_python=${bazel_py} \
    -- @com_github_bazelbuild_buildtools//buildozer 'add copts -Ibazel-out/k8-fastbuild/genfiles' //vendor/github.com/libvirt/libvirt-go:go_default_library
# allign BAZEL files to a single format
bazel run \
    --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64_cgo \
    --workspace_status_command=./hack/print-workspace-status.sh \
    --host_force_python=${bazel_py} \
    //:buildifier
