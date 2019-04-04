#!/usr/bin/env bash

# remove libvirt BUILD file to regenerate it each time
rm -f vendor/github.com/libvirt/libvirt-go/BUILD.bazel

# generate BUILD files
bazel run \
    --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 \
    --workspace_status_command=./hack/print-workspace-status.sh \
    //:gazelle

# inject changes to libvirt BUILD file
bazel run \
    --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 \
    --workspace_status_command=./hack/print-workspace-status.sh \
    -- @com_github_bazelbuild_buildtools//buildozer 'add cdeps //:libvirt-libs //:libvirt-headers' //vendor/github.com/libvirt/libvirt-go:go_default_library
bazel run \
    --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 \
    --workspace_status_command=./hack/print-workspace-status.sh \
    -- @com_github_bazelbuild_buildtools//buildozer 'add copts -Ibazel-out/k8-fastbuild/genfiles' //vendor/github.com/libvirt/libvirt-go:go_default_library
# allign BAZEL files to a single format
bazel run \
    --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 \
    --workspace_status_command=./hack/print-workspace-status.sh \
    //:buildifier
