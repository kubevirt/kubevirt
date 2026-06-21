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
# Copyright 2026 The KubeVirt Authors.
#
# This script should be executed through the makefile via `make kube-burner-perftest`.

set -e

source hack/common.sh
source hack/config.sh

KUBE_BURNER_VERSION=${KUBE_BURNER_VERSION:-v2.7.0}
PROMETHEUS_PORT=${PROMETHEUS_PORT:-30007}
PROMETHEUS_ENDPOINT=${PROMETHEUS_ENDPOINT:-http://127.0.0.1}
PERFSCALE_WORKLOAD=${PERFSCALE_WORKLOAD:-tests/performance/manifests/kube-burner/kubevirt-density/kubevirt-density.yml}
KB_LOG_LEVEL=${KB_LOG_LEVEL:-debug}
export QPS=${QPS:-10}
export BURST=${BURST:-10}
export JOB_ITERATIONS=${JOB_ITERATIONS:-1}

echo 'Preparing directory for artifacts'
mkdir -p "${ARTIFACTS}"

echo "Building kube-burner ${KUBE_BURNER_VERSION} from source"
KUBE_BURNER_BIN="${KUBE_DIR:-/tmp}/kube-burner"
mkdir -p "$(dirname "${KUBE_BURNER_BIN}")"
KUBE_BURNER_SRC=$(mktemp -d)
if [[ "${KUBE_BURNER_VERSION}" == "latest" ]]; then
    git clone --depth 1 "https://github.com/kube-burner/kube-burner.git" "${KUBE_BURNER_SRC}"
else
    git clone --depth 1 --branch "${KUBE_BURNER_VERSION}" "https://github.com/kube-burner/kube-burner.git" "${KUBE_BURNER_SRC}"
fi
(cd "${KUBE_BURNER_SRC}" && GOFLAGS="" go build -o "${KUBE_BURNER_BIN}" ./cmd/kube-burner/)
rm -rf "${KUBE_BURNER_SRC}"

# In kubevirt CI/CD the node running the job is on a different cluster,
# so time is not synced across all nodes. As a workaround we collect
# timestamps via a prometheus query to use its exact clock.
function get_timestamp() {
    curr_timestamp=$(curl -fs --data-urlencode 'query=container_cpu_usage_seconds_total{pod!="",container="prometheus"}' "${PROMETHEUS_ENDPOINT}:${PROMETHEUS_PORT}/api/v1/query" | jq -r '.data.result[0] | .value[0]')
    date -u +%Y-%m-%dT%TZ -d @"${curr_timestamp}"
}

function perftest() {
    SKIP_INDEXING=${SKIP_INDEXING:-false} \
    JOB_ITERATIONS=${JOB_ITERATIONS} \
    GC=${GC:-false} \
    GC_METRICS=${GC_METRICS:-false} \
    METRICS_FOLDER=${METRICS_FOLDER:-collected-metrics} \
    QPS=${QPS} \
    BURST=${BURST} \
    PROMETHEUS_ENDPOINT=${PROMETHEUS_ENDPOINT} \
    PROMETHEUS_PORT=${PROMETHEUS_PORT} \
    PROMETHEUS_TOKEN=${PROMETHEUS_TOKEN:-} \
    "${KUBE_BURNER_BIN}" init \
        --config "${PERFSCALE_WORKLOAD}" \
        --log-level "${KB_LOG_LEVEL}" \
        ${EXTRA_FLAGS:-}
}

function perfaudit() {
    metrics_folder_name=$(find . -maxdepth 1 -type d -name 'collected-metric*' | head -n 1)
    cp -r "${metrics_folder_name}" "${ARTIFACTS}/"
}

# run small test to verify functionality
time SKIP_INDEXING=true JOB_ITERATIONS=1 perftest
echo "Sleeping 30s to let system cool down after the test"
sleep 30
start_timestamp=$(get_timestamp)
# run the actual test
time GC=true perftest
stop_timestamp=$(get_timestamp)

# copy kube-burner collected metrics into artifacts
if [[ "${PERFAUDIT:-true}" == "true" || "${PERFAUDIT:-true}" == "True" ]]; then
    perfaudit
fi

echo "start_timestamp= ${start_timestamp}"
echo "stop_timestamp= ${stop_timestamp}"
