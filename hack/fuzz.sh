set -e

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

function collect_results() {
	cd ${KUBEVIRT_DIR}
	for f in $(find bazel-out/ -name 'test.log'); do
		dir=${ARTIFACTS}/testlogs/$(dirname $f)
		mkdir -p ${dir}
		cp -f ${f} ${dir}/test.log
	done
}
trap collect_results EXIT

bazel test \
	--config=fuzz \
	--features race \
	--test_output=errors -- //pkg/...
