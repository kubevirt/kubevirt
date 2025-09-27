source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

set -ex

bazel run \
    --config="${ARCHITECTURE}" \
    -- :buildozer -types go_test 'set race "on"' \
    //staging/src/kubevirt.io/...:* \
    //pkg/...:* \
    //cmd/...:* \
    //tools/util/...:* \
    //tools/cache/...:* \
    //tests/framework/...:*

bazel run \
    --config="${ARCHITECTURE}" \
    -- :buildozer -types go_test 'set race "off"' \
    //pkg/virt-api/webhooks/fuzz/...:*
