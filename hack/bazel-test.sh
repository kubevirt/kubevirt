set -e

source hack/common.sh
source hack/config.sh

if [ "${CI}" == "true" ]; then
    cat >>ci.bazelrc <<EOF
test --cache_test_results=no --runs_per_test=4
EOF
fi

bazel test \
    --config=${ARCHITECTURE} \
    --test_output=errors -- //staging/src/kubevirt.io/client-go/... //pkg/... //cmd/... //tests/framework/...
