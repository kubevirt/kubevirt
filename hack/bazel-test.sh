set -e

source hack/common.sh
source hack/config.sh

if [ "${CI}" == "true" ]; then
    cat >>ci.bazelrc <<EOF
test --cache_test_results=no --runs_per_test=1
EOF
fi

rm -rf ${ARTIFACTS}/junit ${ARTIFACTS}/testlogs

function collect_results() {
    cd ${KUBEVIRT_DIR}
    mkdir -p ${ARTIFACTS}/junit/
    bazel run //tools/junit-merger:junit-merger -- -o ${ARTIFACTS}/junit/junit.unittests.xml $(find bazel-testlogs/ -name 'test.xml' -printf "%p ")

    for f in $(find bazel-out/ -name 'test.log'); do
        dir=${ARTIFACTS}/testlogs/$(dirname $f)
        mkdir -p ${dir}
        cp -f ${f} ${dir}/test.log
    done
}

trap collect_results EXIT

bazel test \
    --config=${ARCHITECTURE} \
    --features race \
    --test_output=errors -- //staging/src/kubevirt.io/client-go/... //pkg/... //cmd/... //tests/framework/...
