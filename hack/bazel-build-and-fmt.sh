set -e

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

set +e
hack/bazel-fmt.sh
fmtret=$?
set -e

hack/bazel-build.sh
if [ $fmtret -ne 0 ]; then
    hack/bazel-fmt.sh
fi
