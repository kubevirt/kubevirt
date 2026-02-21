#!/usr/bin/env bash
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2026 The Kube-burner Authors.

set -e

_docker_prefix="quay.io/kubevirt/"

export DOCKER_PREFIX=${DOCKER_PREFIX:-${_docker_prefix}}
export DOCKER_TAG=${DOCKER_TAG:-"latest"}
export PROMETHEUS_PORT=${PROMETHEUS_PORT:-30007}
export PROMETHEUS_URL=${PROMETHEUS_URL:-http://127.0.0.1}

echo 'Preparing directory for artifacts'
export AUDIT_CONFIG=${ARTIFACTS}/perfscale-audit-cfg.json
export AUDIT_RESULTS=${ARTIFACTS}/perfscale-audit-results.json
mkdir -p $ARTIFACTS

KUBE_BURNER_VERSION=${KUBE_BURNER_VERSION:-v2.2.2}
OS=$(uname -s)
HARDWARE=$(uname -m)
EXTRA_FLAGS=${EXTRA_FLAGS:-}
KUBE_DIR=${KUBE_DIR:-/tmp}

# in kubevirt CI/CD the node we run the job is on a different cluster, so time is not synced across all nodes
# as a workaround, we collect the timestamps by making a prometheus query, so we can use the exact time that prometheus is using
function get_timestamp() {
    curr_timestamp=$(curl -fs --data-urlencode 'query=container_cpu_usage_seconds_total{pod!="",container="prometheus"}' "http://${PROMETHEUS_URL}:${PROMETHEUS_PORT}/api/v1/query" | jq -r '.data.result[0] | .value[0]')
    date -u +%Y-%m-%dT%TZ -d @$curr_timestamp
}

download_binary(){
    REPO_URL="https://github.com/kube-burner/kube-burner";
    LATEST_TAG=$(curl -s "https://api.github.com/repos/kube-burner/kube-burner/releases/latest" | jq -r '.tag_name');
    TAG_OPTION=$(if [ "$KUBE_BURNER_VERSION" == "latest" ]; then echo "${LATEST_TAG#v}"; else echo "${KUBE_BURNER_VERSION#v}"; fi);
    KUBE_BURNER_URL="https://github.com/kube-burner/kube-burner/releases/download/v${TAG_OPTION}/kube-burner-V${TAG_OPTION}-${OS}-${HARDWARE}.tar.gz"
    curl --fail --retry 8 --retry-all-errors -sS -L "${KUBE_BURNER_URL}" | tar -xzC "${KUBE_DIR}/" kube-burner
}

function perftest() {
    SKIP_INDEXING=${SKIP_INDEXING:-false} JOB_ITERATIONS=${JOB_ITERATIONS:-1} GC=${GC:-false} GC_METRICS=${GC_METRICS:-false} METRICS_FOLDER=${METRICS_FOLDER:-"collected-metrics"} ${KUBE_DIR}/kube-burner init \
        --config ${PERFSCALE_WORKLOAD} \
        --log-level debug \
        ${EXTRA_FLAGS:-}
}

function perfaudit() {
    metrics_folder_name=$(find . -maxdepth 1 -type d -name 'collected-metric*' | head -n 1)
    cp -r "${metrics_folder_name}" "${ARTIFACTS}/"   
}

# run small test to verify the functionality
download_binary
time SKIP_INDEXING=true JOB_ITERATIONS=1 perftest
echo "Sleeping 30 to let system cooldown after the test"
sleep 30
start_timestamp=$(get_timestamp)
# run the test
time GC=true perftest
stop_timestamp=$(get_timestamp)

# run audit tool to transform all kube-burner collected metrics into kubevirt perfscale format
if [[ ${PERFAUDIT} == "true" || ${PERFAUDIT} == "True" ]]; then
    perfaudit
fi

echo "start_timestamp= $start_timestamp"
echo "stop_timestamp= $stop_timestamp"
