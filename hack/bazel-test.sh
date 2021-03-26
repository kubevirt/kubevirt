set -e

source hack/common.sh
source hack/config.sh

if [ "${CI}" == "true" ]; then
    cat >>ci.bazelrc <<EOF
test --cache_test_results=no --runs_per_test=1
EOF
fi

rm -rf ${ARTIFACTS}/junit

function collect_results() {
    cd ${KUBEVIRT_DIR}
    for f in $(find bazel-testlogs/ -name 'test.xml'); do
        dir=${ARTIFACTS}/junit/$(dirname $f)
        mkdir -p ${dir}
        cp -f ${f} ${dir}/junit.xml
    done
}

trap collect_results EXIT

bazel test \
    --config=${ARCHITECTURE} \
    --test_output=errors -- //staging/src/kubevirt.io/client-go/... //pkg/... //cmd/... //tests/framework/...
