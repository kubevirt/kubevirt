# Incremental image updates for an already installed KubeVirt cluster.
# Bootstrap with `make cluster-sync` before running Tilt.

# Watch the whole source tree: Bazel, not this file, knows which shared code
# contributes to each image. The helper compares the resulting image digests
# and pushes only images that changed.

local_resource(
    'kubevirt-images',
    cmd=['./hack/tilt-cluster-sync.sh'],
    deps=['.'],
    ignore=['.tilt-kubevirt-state', '_out', 'bazel-*', '.rsynctemp', '.dockerized.lock'],
)
