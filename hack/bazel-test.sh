set -e

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

default_test_query='tests(//staging/src/kubevirt.io/... + //pkg/... + //cmd/... + //tools/... + //tests/framework/...)'

if [[ -n "${WHAT}" ]]; then
    read -r -a bazel_test_targets <<<"${WHAT}"
else
    readarray -t bazel_test_targets < <(bazel query "${default_test_query}")
fi

rm -rf ${ARTIFACTS}/junit ${ARTIFACTS}/testlogs

if [ "${CI}" == "true" ]; then
    cat >>ci.bazelrc <<EOF
build --jobs=4
test --cache_test_results=no --runs_per_test=1
EOF

    function collect_results() {
        cd ${KUBEVIRT_DIR}
        mkdir -p ${ARTIFACTS}/junit/
        bazel run --config=${ARCHITECTURE} ${BAZEL_CS_CONFIG} //tools/junit-merger:junit-merger -- -o ${ARTIFACTS}/junit/junit.unittests.xml $(find bazel-testlogs/ -name 'test.xml' -printf "%p ")

        for f in $(find bazel-out/ -name 'test.log'); do
            dir=${ARTIFACTS}/testlogs/$(dirname $f)
            mkdir -p ${dir}
            cp -f ${f} ${dir}/test.log
        done
    }
    trap collect_results EXIT
fi

${KUBEVIRT_DIR}/hack/bazel-race.sh

bazel test \
    --config=${ARCHITECTURE} ${BAZEL_CS_CONFIG} \
    --test_output=errors -- "${bazel_test_targets[@]}"
