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
# One-node scaling curve for the virt-handler VM start pool.
#
# Default run measures two builds on the same cluster:
#   baseline  merge-base of virt-handler-start-pool and origin/main
#   pool      virt-handler-start-pool
# It checks out each revision in THIS worktree, cluster-syncs, then creates
# the same cirros 90Mi container-disk VMIs at 1, 10, and 30 with kubectl.
# make perftest is not used: ginkgo AfterSuite dies on TLS handshake timeout
# once the nested node's published ports stall. The primary checkout is never
# switched. On the way out, HEAD is put back where it started.
#
# hack/dockerized rsyncs this worktree into the build container and does not
# follow the worktree .git file, so git describe fails there. Bazel’s
# workspace status command then aborts in hack/print-workspace-status.sh
# because KUBEVIRT_GIT_VERSION is unset. cluster-sync and perftest both build
# that way. For each phase, this script records the host git version and
# points the stamp script at that file, then restores the stamp script.
#
# Rootless podman stops its API service 5s after the last client disconnects.
# cluster-sync then leaves rootlessport orphaned, and the published API port
# accepts TCP and resets or hangs it. `podman info` does not connect to that
# service. This script curls the user podman socket for the whole run, which
# does. A cluster whose rootlessport is already orphaned has to be recreated
# with --fresh; cluster-up on top of it does not replace the port forward.
#
# Prometheus is scraped on the kubevirtci-published node port (podman port
# node01 30007), one short HTTP call at a time. No kubectl port-forward.
#
#   hack/virt-handler-start-perf.sh --fresh             # baseline, then pool
#   hack/virt-handler-start-perf.sh --only pool         # pool build only
#   hack/virt-handler-start-perf.sh --only baseline     # parent build only
#   hack/virt-handler-start-perf.sh --fresh --only pool
#
# Run it from a worktree of this branch (it refuses a primary clone).
#
# The pass bar, decided before looking at the numbers:
#   * On the 30-VMI row, virt-handler-vm queue wait (fraction of items that
#     waited longer than 1s) drops, and vmiCreationToRunning P50 drops by more
#     than the spread of the repeats.
#   * On the 1-VMI row, P50 stays inside that spread.
#   * work_duration P95 drops too, because execute() returns without waiting
#     for SyncVMI.
#   * If the baseline 30-VMI queue wait is already near zero, this change is
#     not the 15s median.
#
# Flags:
#   --fresh              cluster-down, then cluster-up, once, before either phase.
#   --only baseline      measure the parent build and return to this branch.
#   --only pool          measure virt-handler-start-pool only.
#
# Env:
#   KUBEVIRT_MEMORY_SIZE   default 32G (16G fills at ~21 virt-launchers)
#   KUBEVIRT_NUM_NODES     default 1
#   KUBEVIRT_PROVIDER      left as already set
#   PERF_VM_COUNTS         default "1 10 30"
#   PERF_REPEATS           default 3
#   PERF_RESULTS_DIR       default virt-handler-start-perf-results (not under _out;
#                          dockerized rsync --delete of _out after cluster-sync)
#   PERF_BASELINE_REF      default: merge-base with origin/main
#   PERF_POOL_REF          default: virt-handler-start-pool
#   SKIP_CLUSTER_SYNC=1    do not rebuild images (both phases then share one binary)
#   PROMETHEUS_PORT        default 30007

set -euo pipefail

if [[ -z "${PERF_SCRIPT_COPY:-}" ]]; then
    root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
    copy="$(mktemp --suffix=-virt-handler-start-perf.sh)"
    cp "${BASH_SOURCE[0]}" "${copy}"
    chmod +x "${copy}"
    export PERF_SCRIPT_COPY=1 PERF_REPO="${root}"
    exec "${copy}" "$@"
fi

ROOT="${PERF_REPO}"
cd "${ROOT}"

FRESH=0
ONLY=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --fresh) FRESH=1 ;;
        --only)
            ONLY="${2:?--only needs baseline or pool}"
            shift
            ;;
        --only=*) ONLY="${1#--only=}" ;;
        *)
            echo "unknown argument: $1" >&2
            echo "usage: $0 [--fresh] [--only baseline|pool]" >&2
            exit 2
            ;;
    esac
    shift
done
case "${ONLY}" in
    "" | baseline | pool) ;;
    *)
        echo "--only must be baseline or pool" >&2
        exit 2
        ;;
esac

if [[ ! -f .git ]]; then
    echo "refusing to switch branches here: ${ROOT} is a primary checkout (.git is a directory)." >&2
    echo "run this from the virt-handler-start-pool worktree." >&2
    exit 1
fi

ORIGINAL_REF="$(git rev-parse --abbrev-ref HEAD)"
if [[ "${ORIGINAL_REF}" == thinmon* ]]; then
    echo "refusing to switch ${ORIGINAL_REF}." >&2
    exit 1
fi
if ! git diff-index --quiet HEAD --; then
    echo "worktree has uncommitted changes. Commit or stash them first; the script checks out the baseline and restores ${ORIGINAL_REF} on the way out." >&2
    exit 1
fi

export KUBEVIRT_NUM_NODES="${KUBEVIRT_NUM_NODES:-1}"
export KUBEVIRT_MEMORY_SIZE="${KUBEVIRT_MEMORY_SIZE:-32G}"
export KUBEVIRT_DEPLOY_PROMETHEUS="${KUBEVIRT_DEPLOY_PROMETHEUS:-true}"

PERF_VM_COUNTS="${PERF_VM_COUNTS:-1 10 30}"
PERF_REPEATS="${PERF_REPEATS:-3}"
PERF_RESULTS_DIR="${PERF_RESULTS_DIR:-${ROOT}/virt-handler-start-perf-results}"
PROMETHEUS_PORT="${PROMETHEUS_PORT:-30007}"
PERF_NS="${PERF_NS:-kubevirt-perf-start}"
POOL_REF="${PERF_POOL_REF:-virt-handler-start-pool}"
MIN_ALLOCATABLE_KI=$((24 * 1024 * 1024))

if [[ -n "${PERF_BASELINE_REF:-}" ]]; then
    BASE_REF="${PERF_BASELINE_REF}"
elif git rev-parse --verify --quiet origin/main >/dev/null; then
    BASE_REF="$(git merge-base "${POOL_REF}" origin/main)"
else
    BASE_REF="${POOL_REF}^"
fi

command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }
command -v python3 >/dev/null || { echo "python3 is required" >&2; exit 1; }

KUBECTL="${ROOT}/kubevirtci/cluster-up/kubectl.sh"
RUN_ID="$(date +%Y%m%d-%H%M%S)"
OUT="${PERF_RESULTS_DIR}/${RUN_ID}"
mkdir -p "${OUT}"
ln -sfn "${RUN_ID}" "${PERF_RESULTS_DIR}/latest"

DENSITY_BACKUP=""
PORT_FORWARD_PID=""
PODMAN_HOLD_PID=""
STAMP_PATCHED=0
STAMP_MARK="# virt-handler-start-perf: host git stamp for the build container"

to_ki() {
    local raw="$1"
    case "${raw}" in
        *Ki) echo "${raw%Ki}" ;;
        *Mi) echo $(( ${raw%Mi} * 1024 )) ;;
        *Gi) echo $(( ${raw%Gi} * 1024 * 1024 )) ;;
        *) echo "${raw}" ;;
    esac
}

restore_count() {
    if [[ -n "${DENSITY_BACKUP}" && -f "${DENSITY_BACKUP}" ]]; then
        mv "${DENSITY_BACKUP}" tests/performance/density.go
        DENSITY_BACKUP=""
    fi
}

# Drop the temporary stamp patch before any checkout. A killed run can leave
# hack/print-workspace-status.sh modified; the dirty-tree check then refuses
# to switch revisions until that file is restored.
restore_stamp() {
    if [[ "${STAMP_PATCHED}" == "1" ]]; then
        git checkout --quiet -- hack/print-workspace-status.sh
        STAMP_PATCHED=0
    fi
    rm -f hack/.git-version-env
}

# Compute the same semver hack/version.sh would, while git still works on the
# host, and teach the in-container stamp script to read it. The container has
# no worktree git dir, so without this file KUBEVIRT_GIT_VERSION stays unset
# and print-workspace-status.sh aborts under nounset.
prepare_stamp() {
    local saved_dir commit tree version
    unset KUBEVIRT_GIT_COMMIT KUBEVIRT_GIT_TREE_STATE KUBEVIRT_GIT_VERSION
    saved_dir="${KUBEVIRT_DIR-}"
    export KUBEVIRT_DIR="${ROOT}"
    # shellcheck disable=SC1091
    source "${ROOT}/hack/version.sh"
    kubevirt::version::get_version_vars
    if [[ -n "${saved_dir}" ]]; then
        export KUBEVIRT_DIR="${saved_dir}"
    else
        unset KUBEVIRT_DIR
    fi
    commit="${KUBEVIRT_GIT_COMMIT-}"
    tree="${KUBEVIRT_GIT_TREE_STATE-}"
    version="${KUBEVIRT_GIT_VERSION-}"
    unset KUBEVIRT_GIT_COMMIT KUBEVIRT_GIT_TREE_STATE KUBEVIRT_GIT_VERSION
    if [[ -z "${commit}" || -z "${tree}" || -z "${version}" ]]; then
        echo "could not compute KUBEVIRT_GIT_VERSION on the host. cluster-sync cannot stamp a worktree without it." >&2
        exit 1
    fi
    echo "workspace stamp ${version} (${tree})"
    # The container runs get_version_vars again. A dirty tree makes that call
    # append -dirty, so give it the version from before the suffix.
    if [[ "${tree}" == "dirty" && "${version}" == *-dirty ]]; then
        version="${version%-dirty}"
    fi
    cat > hack/.git-version-env <<EOF
KUBEVIRT_GIT_COMMIT='${commit}'
KUBEVIRT_GIT_TREE_STATE='${tree}'
KUBEVIRT_GIT_VERSION='${version}'
EOF
    python3 - "${ROOT}/hack/print-workspace-status.sh" "${STAMP_MARK}" <<'PY'
import pathlib, sys
path, mark = sys.argv[1], sys.argv[2]
file = pathlib.Path(path)
text = file.read_text()
if mark in text:
    sys.exit(0)
needle = "kubevirt::version::get_version_vars\n"
idx = text.find(needle)
if idx < 0:
    sys.exit("hack/print-workspace-status.sh has no kubevirt::version::get_version_vars call")
insert = (
    mark + "\n"
    'if [[ -f "${KUBEVIRT_DIR}/hack/.git-version-env" ]]; then\n'
    '    source "${KUBEVIRT_DIR}/hack/.git-version-env"\n'
    "fi\n"
)
file.write_text(text[:idx] + insert + text[idx:])
PY
    STAMP_PATCHED=1
}

# podman.service exits 5s after the last API client goes away. A detached
# bazel build is long enough for that, and the leftover rootlessport then
# resets or hangs the kube-apiserver port. `podman info` and `podman events`
# do not hold that service open. A curl to the user socket does.
start_podman_hold() {
    local sock="${XDG_RUNTIME_DIR:-}/podman/podman.sock"
    if [[ -n "${PODMAN_HOLD_PID}" ]] && kill -0 "${PODMAN_HOLD_PID}" 2>/dev/null; then
        return 0
    fi
    if [[ ! -S "${sock}" ]]; then
        echo "podman socket ${sock:-<unset>} is not there, so this script cannot keep the API service alive." >&2
        exit 1
    fi
    (
        while true; do
            curl -sf --max-time 3 --unix-socket "${sock}" http://d/v3.0.0/libpod/info >/dev/null 2>&1 || true
            sleep 2
        done
    ) &
    PODMAN_HOLD_PID=$!
}

stop_podman_hold() {
    if [[ -n "${PODMAN_HOLD_PID}" ]]; then
        kill "${PODMAN_HOLD_PID}" >/dev/null 2>&1 || true
        wait "${PODMAN_HOLD_PID}" 2>/dev/null || true
        PODMAN_HOLD_PID=""
    fi
}

cleanup() {
    set +e
    restore_count
    restore_stamp
    stop_podman_hold
    if [[ -n "${ORIGINAL_REF}" ]]; then
        local now
        now="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)"
        if [[ "${now}" != "${ORIGINAL_REF}" ]]; then
            echo "restoring checkout ${ORIGINAL_REF}"
            git checkout --quiet "${ORIGINAL_REF}"
        fi
    fi
    stop_prometheus_proxy
}
trap cleanup EXIT

checkout_ref() {
    local ref="$1"
    local want have
    restore_count
    restore_stamp
    want="$(git rev-parse "${ref}")"
    have="$(git rev-parse HEAD)"
    if [[ "${want}" == "${have}" ]]; then
        return 0
    fi
    if ! git diff-index --quiet HEAD --; then
        echo "worktree became dirty; refusing to check out ${ref}" >&2
        exit 1
    fi
    if git show-ref --verify --quiet "refs/heads/${ref}"; then
        git checkout --quiet "${ref}"
    else
        git checkout --quiet --detach "${ref}"
    fi
}

cluster_is_up() {
    [[ -x "${KUBECTL}" ]] && timeout 20 "${KUBECTL}" get nodes --request-timeout=8s >/dev/null 2>&1
}

node_container_running() {
    local name
    if [[ -n "${KUBEVIRT_PROVIDER:-}" ]]; then
        name="${KUBEVIRT_PROVIDER}-node01"
        podman ps --format '{{.Names}}' | grep -qx "${name}"
        return
    fi
    podman ps --format '{{.Names}}' | grep -q -- '-node01$'
}

require_one_node_shape() {
    local nodes allocatable raw
    nodes="$("${KUBECTL}" get nodes --no-headers | wc -l | tr -d '[:space:]')"
    if [[ "${nodes}" -ne 1 ]]; then
        echo "cluster has ${nodes} nodes; this curve needs 1 so every VMI lands on one virt-handler. Re-run with --fresh after exporting KUBEVIRT_NUM_NODES=1." >&2
        exit 1
    fi
    raw="$("${KUBECTL}" get nodes -o jsonpath='{.items[0].status.allocatable.memory}')"
    allocatable="$(to_ki "${raw}")"
    if [[ "${allocatable}" -lt "${MIN_ALLOCATABLE_KI}" ]]; then
        echo "node allocatable memory is ${raw}, under 24Gi. 30 virt-launchers need ~14Gi plus the rest of the node. Re-run with --fresh and KUBEVIRT_MEMORY_SIZE=32G." >&2
        exit 1
    fi
    echo "node allocatable memory: ${raw}"
}

prometheus_node_host_port() {
    local name spec
    if [[ -n "${KUBEVIRT_PROVIDER:-}" ]]; then
        name="${KUBEVIRT_PROVIDER}-node01"
    else
        name="$(podman ps --format '{{.Names}}' | grep -- '-node01$' | head -1)"
    fi
    spec="$(podman port "${name}" 30007 2>/dev/null | head -1)"
    echo "${spec##*:}"
}

stop_prometheus_proxy() {
    if [[ -n "${PORT_FORWARD_PID}" ]]; then
        kill "${PORT_FORWARD_PID}" >/dev/null 2>&1 || true
        wait "${PORT_FORWARD_PID}" 2>/dev/null || true
        PORT_FORWARD_PID=""
    fi
}

start_prometheus_proxy() {
    local dest
    dest="$(prometheus_node_host_port)"
    if [[ ! "${dest}" =~ ^[0-9]+$ ]]; then
        echo "could not find the host port for node port 30007 (KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER:-unset})" >&2
        exit 1
    fi
    stop_prometheus_proxy
    mkdir -p "${OUT}"
    echo "proxying 127.0.0.1:${PROMETHEUS_PORT} -> 127.0.0.1:${dest} (kubevirtci node port 30007)"
    echo "----- $(date -Is) proxy ${PROMETHEUS_PORT}->${dest} -----" >>"${OUT}/prometheus-port-forward.log"
    cat >"${OUT}/prometheus-proxy.py" <<'PY'
import socket, sys, threading

listen, dest = int(sys.argv[1]), int(sys.argv[2])

def pipe(a, b):
    try:
        while True:
            data = a.recv(65536)
            if not data:
                break
            b.sendall(data)
    except Exception:
        pass
    finally:
        for s in (a, b):
            try:
                s.close()
            except Exception:
                pass

srv = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
srv.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
srv.bind(("127.0.0.1", listen))
srv.listen(64)
while True:
    client, _ = srv.accept()
    try:
        upstream = socket.create_connection(("127.0.0.1", dest), timeout=10)
        upstream.settimeout(None)
    except Exception:
        client.close()
        continue
    threading.Thread(target=pipe, args=(client, upstream), daemon=True).start()
    threading.Thread(target=pipe, args=(upstream, client), daemon=True).start()
PY
    python3 "${OUT}/prometheus-proxy.py" "${PROMETHEUS_PORT}" "${dest}" >>"${OUT}/prometheus-port-forward.log" 2>&1 &
    PORT_FORWARD_PID=$!
}

ensure_prometheus() {
    if curl -sfG --max-time 3 "http://127.0.0.1:${PROMETHEUS_PORT}/api/v1/query" --data-urlencode 'query=up' >/dev/null; then
        return 0
    fi
    start_prometheus_proxy
    local i
    for i in $(seq 1 30); do
        if curl -sfG --max-time 3 "http://127.0.0.1:${PROMETHEUS_PORT}/api/v1/query" --data-urlencode 'query=up' >/dev/null; then
            return 0
        fi
        sleep 1
    done
    echo "prometheus did not become reachable on port ${PROMETHEUS_PORT}" >&2
    exit 1
}

prom_url() {
    local spec port
    spec="$(podman port "${KUBEVIRT_PROVIDER}-node01" 30007 2>/dev/null | head -1)"
    port="${spec##*:}"
    if [[ ! "${port}" =~ ^[0-9]+$ ]]; then
        return 1
    fi
    echo "http://127.0.0.1:${port}"
}

prom_query() {
    local query="$1" url out
    url="$(prom_url)" || { echo na; return 0; }
    out="$(curl -sfG --max-time 8 "${url}/api/v1/query" --data-urlencode "query=${query}" 2>/dev/null)" || { echo na; return 0; }
    echo "${out}" | jq -r 'if (.data.result | length) == 0 then "na" else .data.result[0].value[1] end'
}

kc() {
    "${KUBECTL}" --request-timeout=15s "$@"
}

require_api() {
    if ! cluster_is_up; then
        echo "API is not reachable. Stopping. Partial results: ${TSV}" >&2
        exit 1
    fi
}

cirros_image() {
    local handler
    handler="$(kc -n kubevirt get pod -l kubevirt.io=virt-handler -o jsonpath='{.items[0].spec.containers[0].image}')"
    if [[ -z "${handler}" ]]; then
        echo "could not read the virt-handler image, so the cirros disk image is unknown" >&2
        exit 1
    fi
    echo "${handler/virt-handler/cirros-container-disk-demo}"
}

ensure_perf_ns() {
    kc get ns "${PERF_NS}" >/dev/null 2>&1 || kc create ns "${PERF_NS}" >/dev/null
}

delete_vmi_batch() {
    kc -n "${PERF_NS}" delete vmi --all --timeout=120s >/dev/null 2>&1 || true
}

create_vmi_batch() {
    local count="$1" image="$2" i
    for i in $(seq 1 "${count}"); do
        kc -n "${PERF_NS}" create -f - >/dev/null <<EOF
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  name: perf-${i}
spec:
  domain:
    resources:
      requests:
        memory: 90Mi
        cpu: 100m
      limits:
        memory: 90Mi
        cpu: 100m
    devices:
      disks:
      - name: disk0
        disk:
          bus: virtio
      - name: cloudinitdisk
        disk:
          bus: virtio
      interfaces:
      - name: default
        masquerade: {}
  networks:
  - name: default
    pod: {}
  volumes:
  - name: disk0
    containerDisk:
      image: ${image}
      imagePullPolicy: IfNotPresent
  - name: cloudinitdisk
    cloudInitNoCloud:
      userData: "#!/bin/sh\necho ci\n"
EOF
        sleep 0.1
    done
}

wait_running_batch() {
    local count="$1"
    if ! kc -n "${PERF_NS}" wait vmi --all --for=jsonpath='{.status.phase}'=Running --timeout=300s >/dev/null; then
        echo "timed out waiting for ${count} VMIs to become Running:" >&2
        kc -n "${PERF_NS}" get vmi -o custom-columns=NAME:.metadata.name,PHASE:.status.phase >&2 || true
        return 1
    fi
}

run_primer() {
    local image="$1"
    echo "primer VMI so the container disk is already pulled"
    delete_vmi_batch
    create_vmi_batch 1 "${image}"
    wait_running_batch 1
    delete_vmi_batch
}

TSV="${OUT}/results.tsv"
echo -e "phase\tsha\tcount\trepeat\tp50\tp95\tqueue_wait_gt_1s\twork_p95\thandler_cpu_max" > "${TSV}"

run_curve() {
    local phase="$1"
    local sha="$2"
    local count repeat image t0 elapsed
    local queue_gt1 work_p95 p50 p95 cpu

    echo "recording ${phase} ${sha}; counts: ${PERF_VM_COUNTS}; repeats: ${PERF_REPEATS}"
    require_api
    ensure_perf_ns
    image="$(cirros_image)"
    echo "container disk ${image}"
    run_primer "${image}"

    for count in ${PERF_VM_COUNTS}; do
        for repeat in $(seq 1 "${PERF_REPEATS}"); do
            echo "=== ${phase} ${sha} count=${count} repeat=${repeat} ==="
            require_api
            delete_vmi_batch
            t0="$(date +%s)"
            create_vmi_batch "${count}" "${image}"
            if ! wait_running_batch "${count}"; then
                echo -e "${phase}\t${sha}\t${count}\t${repeat}\tna\tna\tna\tna\tna" | tee -a "${TSV}"
                delete_vmi_batch
                continue
            fi
            sleep 35
            elapsed=$(( $(date +%s) - t0 ))
            if [[ "${elapsed}" -lt 30 ]]; then
                elapsed=30
            fi

            p50="$(prom_query "histogram_quantile(0.50, sum by (le) (increase(kubevirt_vmi_phase_transition_time_from_creation_seconds_bucket{phase=\"Running\"}[${elapsed}s])))")"
            p95="$(prom_query "histogram_quantile(0.95, sum by (le) (increase(kubevirt_vmi_phase_transition_time_from_creation_seconds_bucket{phase=\"Running\"}[${elapsed}s])))")"
            queue_gt1="$(prom_query "1 - sum(increase(kubevirt_workqueue_queue_duration_seconds_bucket{name=\"virt-handler-vm\",le=\"1\"}[${elapsed}s])) / sum(increase(kubevirt_workqueue_queue_duration_seconds_count{name=\"virt-handler-vm\"}[${elapsed}s]))")"
            work_p95="$(prom_query "histogram_quantile(0.95, sum by (le) (increase(kubevirt_workqueue_work_duration_seconds_bucket{name=\"virt-handler-vm\"}[${elapsed}s])))")"
            cpu="$(prom_query "max(rate(container_cpu_usage_seconds_total{pod=~\"virt-handler.*\",container=\"virt-handler\"}[${elapsed}s]))")"

            echo -e "${phase}\t${sha}\t${count}\t${repeat}\t${p50}\t${p95}\t${queue_gt1}\t${work_p95}\t${cpu}" | tee -a "${TSV}"
            delete_vmi_batch
        done
    done
}

run_phase() {
    local phase="$1"
    local ref="$2"
    local sha
    checkout_ref "${ref}"
    sha="$(git rev-parse --short HEAD)"
    echo "=== phase ${phase} checked out ${sha} (${ref}) ==="
    prepare_stamp
    if [[ "${SKIP_CLUSTER_SYNC:-}" != "1" ]]; then
        echo "cluster-sync so virt-handler matches ${sha}"
        make cluster-sync
    else
        echo "warning: SKIP_CLUSTER_SYNC=1, so phase ${phase} is measured with whatever virt-handler is already deployed" >&2
    fi
    run_curve "${phase}" "${sha}"
    restore_stamp
}

start_podman_hold

if [[ "${FRESH}" -eq 1 ]]; then
    echo "cluster-down, then cluster-up with KUBEVIRT_NUM_NODES=${KUBEVIRT_NUM_NODES} KUBEVIRT_MEMORY_SIZE=${KUBEVIRT_MEMORY_SIZE}"
    make cluster-down
fi

if ! cluster_is_up; then
    if node_container_running; then
        echo "The kubevirtci node container is still running, but the API port is resetting connections." >&2
        echo "podman.service idled out during the bazel build and left rootlessport orphaned. cluster-up does not replace it." >&2
        podman ps --format '{{.Names}}' | grep -- '-node01$' >&2 || true
        echo "Recreate it with KUBEVIRT_PROVIDER set, then: hack/virt-handler-start-perf.sh --fresh" >&2
        exit 1
    fi
    echo "cluster-up with KUBEVIRT_NUM_NODES=${KUBEVIRT_NUM_NODES} KUBEVIRT_MEMORY_SIZE=${KUBEVIRT_MEMORY_SIZE} KUBEVIRT_DEPLOY_PROMETHEUS=${KUBEVIRT_DEPLOY_PROMETHEUS}"
    make cluster-up
fi
if ! cluster_is_up; then
    echo "The API is not reachable after cluster-up. Not starting cluster-sync." >&2
    echo "podman.service has likely already exited and orphaned rootlessport." >&2
    exit 1
fi
require_one_node_shape

echo "baseline ${BASE_REF}"
echo "pool     ${POOL_REF}"
echo "results  ${OUT}"

if [[ -z "${ONLY}" || "${ONLY}" == "baseline" ]]; then
    run_phase baseline "${BASE_REF}"
fi
if [[ -z "${ONLY}" || "${ONLY}" == "pool" ]]; then
    run_phase pool "${POOL_REF}"
fi

python3 - "${TSV}" "${OUT}/summary.txt" <<'PY'
import statistics, sys
from collections import defaultdict

src, dest = sys.argv[1], sys.argv[2]
groups = defaultdict(lambda: defaultdict(list))
with open(src) as fh:
    header = fh.readline().rstrip("\n").split("\t")
    for line in fh:
        row = dict(zip(header, line.rstrip("\n").split("\t")))
        key = (row["phase"], row["count"])
        for col in ("p50", "p95", "queue_wait_gt_1s", "work_p95"):
            try:
                groups[key][col].append(float(row[col]))
            except ValueError:
                pass

def med(values):
    return statistics.median(values) if values else float("nan")

lines = ["phase\tcount\tp50_median\tp95_median\tqueue_wait_gt_1s_median\twork_p95_median"]
for (phase, count) in sorted(groups, key=lambda k: (k[0], int(k[1]))):
    cols = groups[(phase, count)]
    lines.append(
        f"{phase}\t{count}\t{med(cols['p50']):.3f}\t{med(cols['p95']):.3f}\t{med(cols['queue_wait_gt_1s']):.3f}\t{med(cols['work_p95']):.3f}"
    )
text = "\n".join(lines) + "\n"
open(dest, "w").write(text)
print(text, end="")
PY

echo "wrote ${TSV}"
echo "wrote ${OUT}/summary.txt"
