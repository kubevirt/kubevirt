#!/usr/bin/env bats

setup() {
    cd "$BATS_TEST_DIRNAME/.."
}

@test "selects only requested image targets" {
    run env FAST_VIRT_TEMPLATE_ENABLED=true PUSH_TARGETS='virt-controller,virt-launcher' \
        ./hack/cluster-resync.sh list
    [ "$status" -eq 0 ]
    [ "${#lines[@]}" -eq 2 ]
    [ "${lines[0]}" = 'virt-controller //cmd/virt-controller:virt-controller-image' ]
    [ "${lines[1]}" = 'virt-launcher //cmd/virt-launcher:virt-launcher-image' ]
}

@test "rejects unknown and disabled targets" {
    run env FAST_VIRT_TEMPLATE_ENABLED=false PUSH_TARGETS=virt-template-apiserver \
        ./hack/cluster-resync.sh list
    [ "$status" -eq 2 ]
    [[ "$output" == *'Unknown or disabled PUSH_TARGETS: virt-template-apiserver'* ]]
}

@test "launcher and controller share one controller rollout" {
    run ./hack/cluster-resync.sh plan virt-controller virt-launcher
    [ "$status" -eq 0 ]
    [ "${#lines[@]}" -eq 2 ]
    [ "${lines[0]}" = 'deployment/virt-controller' ]
    [ "${lines[1]}" = 'daemonset/virt-handler' ]
}

@test "launcher and exportserver only restart handler" {
    run ./hack/cluster-resync.sh plan virt-launcher virt-exportserver
    [ "$status" -eq 0 ]
    [ "${#lines[@]}" -eq 1 ]
    [ "${lines[0]}" = 'daemonset/virt-handler' ]
}

@test "exportserver alone needs no control-plane rollout" {
    run ./hack/cluster-resync.sh plan virt-exportserver
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "operator is handled before other changed workloads" {
    run ./hack/cluster-resync.sh plan virt-api virt-operator virt-handler
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = 'deployment/virt-operator' ]
    [ "${lines[1]}" = 'deployment/virt-api' ]
    [ "${lines[2]}" = 'daemonset/virt-handler' ]
}
