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
# Copyright 2022 IBM, Inc.

set -e

_docker_prefix="quay.io/kubevirt/"
_perfscale_workload="tools/perfscale-load-generator/examples/workload/kubevirt-density/kubevirt-burst-100.yaml"
_perfscale_workload_warmup="tools/perfscale-load-generator/examples/workload/kubevirt-density/kubevirt-primer.yaml"

export DOCKER_PREFIX=${DOCKER_PREFIX:-${_docker_prefix}}
export DOCKER_TAG=${DOCKER_TAG:-"latest"}
export PROMETHEUS_PORT=${PROMETHEUS_PORT:-30007}
export PROMETHEUS_URL=${PROMETHEUS_URL:-http://127.0.0.1}
export PERFSCALE_WORKLOAD=${PERFSCALE_WORKLOAD:-${_perfscale_workload}}

echo 'Preparing directory for artifacts'
export ARTIFACTS=${ARTIFACTS}/performance-density
export AUDIT_CONFIG=${ARTIFACTS}/perfscale-audit-cfg.json
export AUDIT_RESULTS=${ARTIFACTS}/perfscale-audit-results.json
rm -rf $ARTIFACTS
mkdir -p $ARTIFACTS

# in kubevirt CI/CD the node we run the job is on a different cluster, so time is not synced across all nodes
# as a workaround, we collect the timestamps by making a prometheus query, so we can use the exact time that prometheus is using
function get_timestamp() {
    curr_timestamp=$(curl -fs --data-urlencode 'query=container_cpu_usage_seconds_total{pod!="",container="prometheus"}' http://localhost:9090/api/v1/query | jq -r '.data.result[0] | .value[0]')
    date -u +%Y-%m-%dT%TZ -d @$curr_timestamp
}

function perftest() {
    _out/cmd/perfscale-load-generator/perfscale-load-generator \
        -v 6 \
        -delete \
        -workload ${PERFSCALE_WORKLOAD}
}

function perfaudit() {
    start_timestamp=$1
    stop_timestamp=$2
    cat <<EOF >${ARTIFACTS}/perfscale-audit-cfg.json
{
	"prometheusURL": "${PROMETHEUS_URL}:${PROMETHEUS_PORT}",
	"startTime": "$start_timestamp",
	"endTime": "$stop_timestamp"
}
EOF
    _out/cmd/perfscale-audit/perfscale-audit \
        --config-file=${AUDIT_CONFIG} \
        --results-file=${AUDIT_RESULTS}
}

# as workaround to collect all pod events, we first need to warmup the kubevirt cluster creating a VMI to prevent Prometheus zero metrics problem
# more info in https://github.com/kubevirt/kubevirt/issues/7083
if [[ ${PERFAUDIT} == "true" || ${PERFAUDIT} == "True" ]]; then
    _perfscale_workload_tmp=${PERFSCALE_WORKLOAD}
    export PERFSCALE_WORKLOAD=$_perfscale_workload_warmup
    # run small test to warm up the system and be able to collect metrics
    perftest ${additional_test_args} ${FUNC_TEST_ARGS}
    export PERFSCALE_WORKLOAD=$_perfscale_workload_tmp
fi

start_timestamp=$(get_timestamp)
# wait 30 before running the test to let prometheus scrape metrics
echo "Sleeping 30 to let prometheus scrape all metrics"
sleep 30
# run the test
time perftest ${additional_test_args} ${FUNC_TEST_ARGS}
stop_timestamp=$(get_timestamp)

# run audit tool to dump metrics
if [[ ${PERFAUDIT} == "true" || ${PERFAUDIT} == "True" ]]; then
    # wait 30s after finished the test. More info in https://github.com/kubevirt/kubevirt/issues/7083
    echo "Sleeping 30s to let prometheus scrape all metrics"
    sleep 30s
    stop_timestamp=$(get_timestamp)
    perfaudit $start_timestamp $stop_timestamp
fi

echo "start_timestamp= $start_timestamp"
echo "stop_timestamp= $stop_timestamp"
