#!/bin/bash

set -xeuo pipefail

ARCH=$(uname -m | grep -q s390x && echo s390x || echo amd64)

SCRIPT_PATH=$(dirname "$(realpath "$0")")

ARTIFACTS=${ARTIFACTS:-${PWD}/artifacts}
mkdir -p "$ARTIFACTS"

config_file=${1:-}
sonobuoy_version=0.56.9
[[ -f "$config_file" ]] && sonobuoy_version=$(jq -r '.Version' "$config_file" | grep -oE '[0-9\.]+')

conformance_image_config_file="$SCRIPT_PATH/conformance-image-config.yaml"
! [[ -f "$conformance_image_config_file" ]] && echo "FATAL: Conformance image config file does not exists" 1>&2 && exit 1

if [[ -z "$KUBEVIRT_PROVIDER" ]]; then
    echo "KUBEVIRT_PROVIDER is not set" 1>&2
    exit 1
fi

KUBECONFIG=$(cluster-up/kubeconfig.sh)
export KUBECONFIG

teardown() {
    rv=$?

    ./sonobuoy status --json
    ./sonobuoy logs > "${ARTIFACTS}/sonobuoy.log"

    results_tarball=$(./sonobuoy retrieve)
    cp "$results_tarball" "${ARTIFACTS}/"
    tar -ztvf "$results_tarball"

    # Get each plugin junit report rename from 'junit.xml' to 'junit.<plugin-name>.<file number>.xml',
    # and move to artifacts directory
    plugins=$(./sonobuoy status --json | jq -r '[.plugins[]] | unique_by(.plugin) | .[].plugin')
    for plugin in $plugins; do
      tar -xvzf "$results_tarball" "plugins/$plugin/"
      idx=1
      for report in $(find "plugins/$plugin/"* -name "*.xml"); do
        plugin_report=$(basename $report)
        plugin_report="${plugin_report/.xml/.${plugin}.${idx}.xml}"
        cp -f $report "${ARTIFACTS}/$plugin_report"
        idx=$((idx+1))
      done
    done

    passed=$(./sonobuoy status --json | jq  ' .plugins[] | select(."result-status" == "passed")'  | wc -l)
    failed=$(./sonobuoy status --json | jq  ' .plugins[] | select(."result-status" == "failed")'  | wc -l)

   ./sonobuoy delete --wait

    if [ $rv -ne 0 ]; then
        echo "error found, exiting"
        exit $rv
    fi

    if [ "$passed" -eq 0 ] || [ "$failed" -ne 0 ]; then
        echo "sonobuoy failed, running conformance tests with plugins: ($plugins)"
        exit 1
    fi
}

curl -L "https://github.com/vmware-tanzu/sonobuoy/releases/download/v${sonobuoy_version}/sonobuoy_${sonobuoy_version}_linux_${ARCH}.tar.gz" | tar -xz

trap teardown EXIT

run_cmd="./sonobuoy run --wait --e2e-repo-config $conformance_image_config_file"

if [ "$config_file" != "" ]; then
    run_cmd+=" --config $config_file"
fi

SONOBUOY_EXTRA_ARGS=${SONOBUOY_EXTRA_ARGS:-}
if [ -n "$SONOBUOY_EXTRA_ARGS" ]; then
    run_cmd+=" ${SONOBUOY_EXTRA_ARGS}"
fi

$run_cmd
