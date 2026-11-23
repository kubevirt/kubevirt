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
# Copyright 2026 Red Hat, Inc.

set -euo pipefail

if [[ "${1:-}" != "stop" ]]; then
    echo "Usage: $0 stop" >&2
    exit 2
fi

pid_file="${TMPDIR:-/tmp}/kubevirt-bazel-image-registry.pid"
if [[ ! -f "${pid_file}" ]]; then
    echo "Cluster-sync Bazel registry is not running"
    exit 0
fi

registry_pid=$(cat "${pid_file}")
if ! ps -p "${registry_pid}" -o args= | grep -Fq "hack/bazel_image_registry.py"; then
    rm -f "${pid_file}"
    echo "Removed stale cluster-sync registry pid file"
    exit 0
fi

kill -TERM "${registry_pid}"
for _ in $(seq 1 30); do
    if ! kill -0 "${registry_pid}" 2>/dev/null; then
        rm -f "${pid_file}"
        echo "Stopped cluster-sync Bazel registry"
        exit 0
    fi
    sleep 0.1
done

echo "Cluster-sync registry process ${registry_pid} did not stop" >&2
exit 1
