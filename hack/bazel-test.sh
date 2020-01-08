set -e

source hack/common.sh
source hack/config.sh

bazel test \
    --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64_cgo \
    --stamp \
    --workspace_status_command=./hack/print-workspace-status.sh \
    --host_force_python=${bazel_py} \
    --test_output=errors -- //pkg/...
