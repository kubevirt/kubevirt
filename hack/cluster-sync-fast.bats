#!/usr/bin/env bats

setup() {
    repo_root=$(git rev-parse --show-toplevel)
    cd "$repo_root" || exit
    test_dir=$(mktemp -d)
    export CLUSTER_SYNC_FAST_MANIFEST="$test_dir/manifest.yaml"
    export CLUSTER_SYNC_FAST_STATE="$test_dir/state.tsv"
    export TEST_PUSH_LOG="$test_dir/push.log"
    export TEST_KUBECTL_LOG="$test_dir/kubectl.log"
    export TEST_KUBECTL="$test_dir/kubectl"
    export BUILD_REV=1
    export DOCKER_TAG=test-tag
    export TEST_IMAGE_TAG=template-v1

    cat > "$CLUSTER_SYNC_FAST_MANIFEST" <<EOF_MANIFEST
hooks:
  file: $test_dir/hooks.sh
default_components: [virt-operator, virt-api]
components:
  virt-operator:
    build_target: operator-target
    digest_file: operator.digest
    push_target: virt-operator
    workload: {kind: deployment, name: virt-operator}
    deploy_order: 10
    flow: rollout
  virt-api:
    build_target: api-target
    digest_file: api.digest
    push_target: virt-api
    workload: {kind: deployment, name: virt-api}
    deploy_order: 20
    flow: rollout
  virt-launcher:
    build_target: launcher-target
    digest_file: launcher.digest
    push_target: virt-launcher
    workload: {kind: deployment, name: virt-controller}
    deploy_order: 30
    flow: rollout
  virt-exportserver:
    build_target: exportserver-target
    digest_file: exportserver.digest
    push_target: virt-exportserver
    workload: {kind: deployment, name: virt-controller}
    deploy_order: 30
    flow: rollout
  template:
    build_target: template-target
    digest_file: template.digest
    push_target: template
    push_tag_env: TEST_IMAGE_TAG
    workload: {kind: deployment, name: template}
    deploy_order: 40
    flow: rollout
flows:
  rollout: [restart_workload, wait_for_rollout]
combination_flows:
  - contains: [virt-operator]
    steps: [virt-operator, remaining_changed_components]
  - exactly: [virt-api, virt-operator]
    steps: [virt-api, virt-operator]
EOF_MANIFEST

    cat > "$test_dir/hooks.sh" <<'EOF_HOOKS'
cluster_sync_fast_cluster_ready() {
    namespace=test
    kubectl_cmd=$TEST_KUBECTL
    if [[ -z "$requested_docker_tag" ]]; then
        DOCKER_TAG=installed-tag
        export DOCKER_TAG
    fi
}
cluster_sync_fast_build() {
    local component
    for component in ${1//,/ }; do
        printf 'FAST_SYNC_DIGEST=%s:%s-%s\n' "$component" "$component" "$BUILD_REV"
    done
}
cluster_sync_fast_push() {
    printf '%s|%s|%s\n' "$1" "$2" "$3" >> "$TEST_PUSH_LOG"
}
EOF_HOOKS

    cat > "$TEST_KUBECTL" <<'EOF_KUBECTL'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$TEST_KUBECTL_LOG"
EOF_KUBECTL
    chmod +x "$TEST_KUBECTL"
}

teardown() {
    rm -rf "$test_dir"
}

@test "exact combination rule takes precedence over contains rule" {
    run ./hack/cluster-sync-fast-plan.sh "$CLUSTER_SYNC_FAST_MANIFEST" virt-operator virt-api
    [ "$status" -eq 0 ]
    [ "$output" = $'virt-api\nvirt-operator' ]
}

@test "planner rejects an incomplete or duplicate combination flow" {
    yq -i '(.combination_flows[] | select(has("exactly"))).steps = ["virt-api", "virt-api"]' "$CLUSTER_SYNC_FAST_MANIFEST"
    run ./hack/cluster-sync-fast-plan.sh "$CLUSTER_SYNC_FAST_MANIFEST" virt-operator virt-api
    [ "$status" -ne 0 ]
    [[ "$output" == *"duplicate component"* ]]
}

@test "runner builds, pushes, and rolls only when digest or tag changes" {
    run env CLUSTER_SYNC_FAST_COMPONENTS='virt-operator,virt-api' ./hack/cluster-sync-fast.sh
    [ "$status" -eq 0 ]
    [[ "$output" == *"Changed images: virt-operator virt-api"* ]]
    [ "$(wc -l < "$TEST_PUSH_LOG")" -eq 1 ]
    [ "$(wc -l < "$TEST_KUBECTL_LOG")" -eq 6 ]

    : > "$TEST_PUSH_LOG"
    : > "$TEST_KUBECTL_LOG"
    run env CLUSTER_SYNC_FAST_COMPONENTS='virt-operator,virt-api' ./hack/cluster-sync-fast.sh
    [ "$status" -eq 0 ]
    [[ "$output" == *"No selected component image changed."* ]]
    [ ! -s "$TEST_PUSH_LOG" ]
    [ ! -s "$TEST_KUBECTL_LOG" ]

    run env CLUSTER_SYNC_FAST_COMPONENTS=template ./hack/cluster-sync-fast.sh
    [ "$status" -eq 0 ]
    [ "$(wc -l < "$TEST_PUSH_LOG")" -eq 1 ]
    export TEST_IMAGE_TAG=template-v2
    : > "$TEST_PUSH_LOG"
    run env CLUSTER_SYNC_FAST_COMPONENTS=template ./hack/cluster-sync-fast.sh
    [ "$status" -eq 0 ]
    [[ "$output" == *"Changed images: template"* ]]
    [[ "$(cat "$TEST_PUSH_LOG")" == *"|template-v2|"* ]]
}

@test "shared workload is restarted and awaited once for two changed images" {
    run env CLUSTER_SYNC_FAST_COMPONENTS='virt-launcher virt-exportserver' ./hack/cluster-sync-fast.sh
    [ "$status" -eq 0 ]
    [ "$(grep -c 'rollout restart deployment/virt-controller' "$TEST_KUBECTL_LOG")" -eq 1 ]
    [ "$(grep -c 'rollout status deployment/virt-controller' "$TEST_KUBECTL_LOG")" -eq 1 ]
}

@test "implicit core image tag comes from cluster readiness hook" {
    run env -u DOCKER_TAG CLUSTER_SYNC_FAST_COMPONENTS=virt-api ./hack/cluster-sync-fast.sh
    [ "$status" -eq 0 ]
    [[ "$(cat "$TEST_PUSH_LOG")" == *"|installed-tag|"* ]]
}

@test "dry run loads hook overrides and reports selected image tags" {
    cat > "$test_dir/override.sh" <<'EOF_OVERRIDE'
cluster_sync_fast_init() {
    TEST_IMAGE_TAG=from-hook
    export TEST_IMAGE_TAG
}
EOF_OVERRIDE
    run env CLUSTER_SYNC_FAST_COMPONENTS=template CLUSTER_SYNC_FAST_DRY_RUN=true \
        CLUSTER_SYNC_FAST_HOOKS="$test_dir/override.sh" ./hack/cluster-sync-fast.sh
    [ "$status" -eq 0 ]
    [[ "$output" == *"template: from-hook"* ]]
    [ ! -s "$TEST_PUSH_LOG" ]
    [ ! -s "$TEST_KUBECTL_LOG" ]
}
