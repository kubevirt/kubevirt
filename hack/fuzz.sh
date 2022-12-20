set -e

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

bazel test \
    --config=fuzz \
    --features race \
    --test_output=errors -- //pkg/...
