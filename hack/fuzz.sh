set -e

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

bazel test \
    --config=fuzz \
    --@io_bazel_rules_go//go/config:race \
    --test_output=errors -- //pkg/...
