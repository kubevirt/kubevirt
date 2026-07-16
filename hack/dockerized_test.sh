#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

assert_timeout_is_forwarded() {
    local timeout="$1"
    local use_default="$2"
    local temp_dir
    temp_dir="$(mktemp -d)"
    trap 'rm -rf "${temp_dir}"' RETURN

    mkdir -p "${temp_dir}/bin" "${temp_dir}/home"
    cat > "${temp_dir}/bin/podman" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "${PODMAN_CALL_LOG}"
case "$1" in
    volume)
        ;;
    run)
        if [[ " $* " == *" --expose 873 "* ]]; then
            printf '%s\n' rsyncd-container
        fi
        ;;
    port)
        printf '%s\n' 0.0.0.0:1873
        ;;
    ps|stop|rm|exec)
        ;;
    *)
        echo "unexpected podman command: $*" >&2
        exit 1
        ;;
esac
EOF
    cat > "${temp_dir}/bin/rsync" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    chmod +x "${temp_dir}/bin/podman" "${temp_dir}/bin/rsync"

    if [[ "${use_default}" == "true" ]]; then
        env -u PULLER_TIMEOUT \
            PODMAN_CALL_LOG="${temp_dir}/podman.log" \
            PATH="${temp_dir}/bin:${PATH}" \
            HOME="${temp_dir}/home" \
            KUBEVIRT_CRI=podman \
            KUBEVIRT_RUN_UNNESTED=false \
            KUBEVIRT_CENTOS_STREAM_VERSION=9 \
            "${repo_root}/hack/dockerized" true >/dev/null
    else
        PODMAN_CALL_LOG="${temp_dir}/podman.log" \
            PATH="${temp_dir}/bin:${PATH}" \
            HOME="${temp_dir}/home" \
            KUBEVIRT_CRI=podman \
            KUBEVIRT_RUN_UNNESTED=false \
            KUBEVIRT_CENTOS_STREAM_VERSION=9 \
            PULLER_TIMEOUT="${timeout}" \
            "${repo_root}/hack/dockerized" true >/dev/null
    fi

    if ! grep -E -- "^run .*--env PULLER_TIMEOUT=${timeout}([[:space:]]|$).*hack/bazel-server.sh$" "${temp_dir}/podman.log" >/dev/null; then
        echo "missing PULLER_TIMEOUT=${timeout} in bazel-server run call" >&2
        return 1
    fi
    if ! grep -E -- "^exec .*--env PULLER_TIMEOUT=${timeout}([[:space:]]|$).*/entrypoint.sh true$" "${temp_dir}/podman.log" >/dev/null; then
        echo "missing PULLER_TIMEOUT=${timeout} in bazel-server exec call" >&2
        return 1
    fi
}

assert_timeout_is_forwarded 600 true
assert_timeout_is_forwarded 1234 false
