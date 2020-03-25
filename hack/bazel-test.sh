set -e

source hack/common.sh
source hack/config.sh

bazel test \
    --config=${ARCHITECTURE} \
    --stamp \
    --test_output=errors -- //pkg/... //cmd/...
