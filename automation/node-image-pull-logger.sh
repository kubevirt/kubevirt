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
# Copyright The KubeVirt Authors.
#

set -euo pipefail

readonly NODE_IMAGE_PULL_LOG_FILE="/var/log/kubevirt-image-pulls.log"
readonly NODE_IMAGE_PULL_LOGGER_PID_FILE="/var/run/kubevirt-image-pull-logger.pid"
readonly SCRIPT_DIR="$(
    cd "$(dirname "${BASH_SOURCE[0]}")"
    pwd
)"
readonly PROJECT_ROOT="$(
    cd "${SCRIPT_DIR}/.."
    pwd
)"
readonly CLI="${PROJECT_ROOT}/kubevirtci/cluster-up/cli.sh"
readonly KUBECTL="${PROJECT_ROOT}/kubevirtci/cluster-up/kubectl.sh"

usage() {
    cat <<'EOF'
Usage:
  automation/node-image-pull-logger.sh start [--provider <provider>]
  automation/node-image-pull-logger.sh collect --artifacts-path <path> [--provider <provider>]

Examples:
  KUBEVIRT_PROVIDER=k8s-1.35 automation/node-image-pull-logger.sh start
  KUBEVIRT_PROVIDER=k8s-1.35 automation/node-image-pull-logger.sh collect --artifacts-path ./exported-artifacts
EOF
}

get_nodes() {
    "${KUBECTL}" get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'
}

start_logging_on_nodes() {
    local node
    local nodes
    local remote_start_cmd
    local remote_start_cmd_b64

    if ! nodes="$(get_nodes)"; then
        echo "failed to list nodes via kubevirtci kubectl wrapper" >&2
        return 1
    fi

    while IFS= read -r node; do
        [[ -n "${node}" ]] || continue
        remote_start_cmd=$(cat <<EOF
if [ -s ${NODE_IMAGE_PULL_LOGGER_PID_FILE} ]; then
    old_pid=\$(cat ${NODE_IMAGE_PULL_LOGGER_PID_FILE} 2>/dev/null || true)
    if [ -n "\${old_pid}" ]; then
        kill "\${old_pid}" >/dev/null 2>&1 || true
    fi
fi
pkill -f ${NODE_IMAGE_PULL_LOG_FILE} >/dev/null 2>&1 || true
: > ${NODE_IMAGE_PULL_LOG_FILE}
nohup bash -c '
has_source=false
if [ -r /var/log/messages ]; then
    has_source=true
fi
if command -v journalctl >/dev/null 2>&1; then
    has_source=true
fi
if [ -r /var/log/kubelet.log ]; then
    has_source=true
fi
if [ "\${has_source}" != "true" ]; then
    echo "no kubelet/container runtime log source found"
    exit 0
fi
cat \
  <([ -r /var/log/messages ] && tail -n0 -F /var/log/messages || true) \
  <(command -v journalctl >/dev/null 2>&1 && journalctl -u kubelet -u crio -u containerd -f -n0 -o short-iso || true) \
  <([ -r /var/log/kubelet.log ] && tail -n0 -F /var/log/kubelet.log || true) \
| grep --line-buffered -Ei "Pulling image|Successfully pulled image|Pulling image:|PullImage|Pulled image" >> ${NODE_IMAGE_PULL_LOG_FILE}
' >/dev/null 2>&1 < /dev/null &
echo \$! > ${NODE_IMAGE_PULL_LOGGER_PID_FILE}
EOF
)

        remote_start_cmd_b64="$(printf '%s' "${remote_start_cmd}" | base64 | tr -d '\n')"
        if ! "${CLI}" ssh "${node}" -- "echo '${remote_start_cmd_b64}' | base64 -d | sudo bash"; then
            echo "failed to start image pull logging on ${node}" >&2
        fi
    done <<< "${nodes}"
}

collect_logs_from_nodes() {
    local artifacts_path="${1:?artifacts path is required}"
    local pulls_dir="${artifacts_path}/node-image-pulls"
    local nodes
    local node
    local log_dest
    local has_logs=false

    mkdir -p "${pulls_dir}"

    if ! nodes="$(get_nodes)"; then
        echo "failed to list nodes while collecting logs" >&2
        nodes=""
    fi

    while IFS= read -r node; do
        [[ -n "${node}" ]] || continue
        log_dest="${pulls_dir}/${node}.log"

        if "${CLI}" ssh "${node}" -- "sudo test -f '${NODE_IMAGE_PULL_LOG_FILE}'" >/dev/null 2>&1; then
            "${CLI}" ssh "${node}" -- "sudo bash -c \"if [ -f '${NODE_IMAGE_PULL_LOGGER_PID_FILE}' ]; then kill \$(cat '${NODE_IMAGE_PULL_LOGGER_PID_FILE}') >/dev/null 2>&1 || true; rm -f '${NODE_IMAGE_PULL_LOGGER_PID_FILE}'; fi; pkill -f '${NODE_IMAGE_PULL_LOG_FILE}' >/dev/null 2>&1 || true\"" || true
            "${CLI}" ssh "${node}" -- "sudo cat '${NODE_IMAGE_PULL_LOG_FILE}'" > "${log_dest}" || true
            has_logs=true
        else
            echo "missing image pull log on ${node}" > "${log_dest}"
        fi
    done <<< "${nodes}"

    if [[ "${has_logs}" == true ]] && ls "${pulls_dir}"/node*.log >/dev/null 2>&1; then
        cat "${pulls_dir}"/node*.log > "${pulls_dir}/all-node-image-pulls.log" || true
        sed -nE \
            -e 's/.*(Pulling image|Successfully pulled image)[[:space:]]+"([^"]+)".*/\2/p' \
            -e 's/.*(Pulling image|Successfully pulled image).*image="?([^" ]+)".*/\2/p' \
            -e 's/.*Pulling image:[[:space:]]*([^" ]+).*/\1/p' \
            -e 's/.*PullImage[[:space:]]+\\"([^"]+)\\".*/\1/p' \
            -e 's/.*PullImage[[:space:]]+"([^"]+)".*/\1/p' \
            -e 's/.*Pulled image[[:space:]]+"([^"]+)".*/\1/p' \
            "${pulls_dir}"/node*.log \
            | awk 'NF' \
            | sort -u > "${pulls_dir}/unique-images.txt" || true
        grep 'registry:5000/' "${pulls_dir}/unique-images.txt" | sort -u > "${pulls_dir}/unique-images-registry5000.txt" || true
    else
        echo "no node image pull logs found" > "${pulls_dir}/all-node-image-pulls.log"
        : > "${pulls_dir}/unique-images.txt"
        : > "${pulls_dir}/unique-images-registry5000.txt"
    fi
}

main() {
    local command="${1:-}"
    local provider_arg=""
    local artifacts_path=""

    [[ -n "${command}" ]] || {
        usage
        exit 1
    }
    shift || true

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --provider)
                provider_arg="${2:-}"
                shift 2
                ;;
            --artifacts-path)
                artifacts_path="${2:-}"
                shift 2
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                echo "unknown argument: $1" >&2
                usage
                exit 1
                ;;
        esac
    done

    if [[ -n "${provider_arg}" ]]; then
        export KUBEVIRT_PROVIDER="${provider_arg}"
    fi

    case "${command}" in
        start)
            start_logging_on_nodes
            ;;
        collect)
            [[ -n "${artifacts_path}" ]] || {
                echo "--artifacts-path is required for collect" >&2
                usage
                exit 1
            }
            collect_logs_from_nodes "${artifacts_path}"
            ;;
        *)
            echo "unknown command: ${command}" >&2
            usage
            exit 1
            ;;
    esac
}

main "$@"
