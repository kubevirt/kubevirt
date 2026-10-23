#!/usr/bin/env bash
# Incrementally build, push, and roll selected images from a YAML manifest.
set -eo pipefail
if ! command -v yq >/dev/null 2>&1; then
    echo "cluster-sync-fast requires Mike Farah yq v4 on the host (https://github.com/mikefarah/yq). Install it and retry." >&2
    exit 127
fi
if ! yq --version 2>&1 | grep -Eq 'version v?4\.'; then
    echo "cluster-sync-fast requires Mike Farah yq v4; found: $(yq --version 2>&1)" >&2
    exit 2
fi
source_manifest=${CLUSTER_SYNC_FAST_MANIFEST:-${KUBEVIRT_FAST_MANIFEST:-hack/cluster-sync-fast.yaml}}
manifest=$source_manifest
state_file=${CLUSTER_SYNC_FAST_STATE:-${KUBEVIRT_FAST_STATE:-.cluster-sync-fast-state.tsv}}
# The sourced adapter uses this to distinguish an explicit tag from the
# installed operator's tag.
# shellcheck disable=SC2034
requested_docker_tag=${DOCKER_TAG:-}
DOCKER_TAG=${DOCKER_TAG:-devel}
DOCKER_TAG_ALT=${DOCKER_TAG_ALT:-}
export DOCKER_TAG DOCKER_TAG_ALT
namespace=${CLUSTER_SYNC_FAST_NAMESPACE:-default}
kubectl_cmd=${CLUSTER_SYNC_FAST_KUBECTL:-kubectl}
hooks_file=$(yq -r '.hooks.file // "hack/cluster-sync-fast-kubevirt-hooks.sh"' "$manifest")
hooks_overlay=${CLUSTER_SYNC_FAST_HOOKS:-}
if [[ ! -r "$hooks_file" ]]; then
    echo "Cannot read cluster-sync-fast hooks file '$hooks_file'" >&2
    exit 2
fi
# The adapter path comes from the YAML manifest.
# shellcheck disable=SC1090
source "$hooks_file"
if [[ -n "$hooks_overlay" ]]; then
    if [[ ! -r "$hooks_overlay" ]]; then
        echo "Cannot read cluster-sync-fast hooks overlay '$hooks_overlay'" >&2
        exit 2
    fi
    # This optional overlay path comes from the environment.
    # shellcheck disable=SC1090
    source "$hooks_overlay"
fi
if declare -F cluster_sync_fast_init >/dev/null; then
    cluster_sync_fast_init
fi
component_push_tag() {
    local component=$1 push_tag tag_env
    push_tag=$(COMPONENT=$component yq -r '.components[strenv(COMPONENT)].push_tag // ""' "$manifest")
    if [[ -n "$push_tag" ]]; then
        printf '%s\n' "$push_tag"
        return
    fi
    tag_env=$(COMPONENT=$component yq -r '.components[strenv(COMPONENT)].push_tag_env // ""' "$manifest")
    if [[ -n "$tag_env" ]]; then
        if [[ ! "$tag_env" =~ ^[a-zA-Z_][a-zA-Z0-9_]*$ ]]; then
            echo "Manifest component '$component' has invalid push_tag_env '$tag_env'" >&2
            return 2
        fi
        if [[ -z "${!tag_env:-}" ]]; then
            echo "Manifest component '$component' names unset push tag environment variable '$tag_env'" >&2
            return 2
        fi
        printf '%s\n' "${!tag_env}"
        return
    fi
    printf '%s\n' "$DOCKER_TAG"
}
component_push_alt_tag() {
    local component=$1 configured_value configured_tag_env
    configured_value=$(COMPONENT=$component yq -r '.components[strenv(COMPONENT)].push_alt_tag // ""' "$manifest")
    case "$configured_value" in
        true) printf '%s\n' "$DOCKER_TAG_ALT"; return ;;
        false) printf '\n'; return ;;
        "") ;;
        *) echo "Component '$component' has invalid push_alt_tag '$configured_value' (expected true or false)" >&2; return 2 ;;
    esac
    configured_value=$(COMPONENT=$component yq -r '.components[strenv(COMPONENT)].push_tag // ""' "$manifest")
    configured_tag_env=$(COMPONENT=$component yq -r '.components[strenv(COMPONENT)].push_tag_env // ""' "$manifest")
    if [[ -n "$configured_value" || -n "$configured_tag_env" ]]; then
        printf '\n'
    else
        printf '%s\n' "$DOCKER_TAG_ALT"
    fi
}
cleanup() {
    [[ -z "${state_tmp:-}" ]] || rm -f "$state_tmp"
}
trap cleanup EXIT INT TERM
components=${CLUSTER_SYNC_FAST_COMPONENTS:-${KUBEVIRT_FAST_COMPONENTS:-$(yq -er '.default_components | join(" ")' "$manifest")}}
components=${components//,/ }
read -r -a selected_components <<< "$components"
if ((${#selected_components[@]} == 0)); then
    echo "No components selected; set CLUSTER_SYNC_FAST_COMPONENTS or configure default_components" >&2
    exit 2
fi
declare -A selected_lookup
for component in "${selected_components[@]}"; do
    if [[ -n "${selected_lookup[$component]:-}" ]]; then
        echo "Component '$component' was selected more than once" >&2
        exit 2
    fi
    selected_lookup[$component]=1
done
components=${selected_components[*]}
if [[ ${CLUSTER_SYNC_FAST_DRY_RUN:-${KUBEVIRT_FAST_DRY_RUN:-false}} == true ]]; then
    printf 'Dry run: treating selected components as changed; no build, push, or cluster access.\n'
    printf 'Planned deployment order:\n'
    dry_run_order_output=$(./hack/cluster-sync-fast-plan.sh "$manifest" "${selected_components[@]}")
    mapfile -t dry_run_order <<< "$dry_run_order_output"
    printf '  %s\n' "${dry_run_order[@]}"
    printf 'Image tags:\n'
    for component in "${selected_components[@]}"; do
        tag_value=$(component_push_tag "$component")
        alt_tag_value=$(component_push_alt_tag "$component")
        printf '  %s: %s (alternate: %s)\n' "$component" "$tag_value" "${alt_tag_value:-none}"
    done
    exit 0
fi
for hook in cluster_sync_fast_cluster_ready cluster_sync_fast_build cluster_sync_fast_push; do
    if ! declare -F "$hook" >/dev/null; then
        echo "Configured hooks must define $hook" >&2
        exit 2
    fi
done

build_targets=()
digest_files=()
declare -A push_target_by_component push_tag_by_component push_alt_tag_by_component
for component in "${selected_components[@]}"; do
    COMPONENT=$component yq -e '.components[strenv(COMPONENT)] != null' "$manifest" >/dev/null || {
        echo "Unknown component '$component' in CLUSTER_SYNC_FAST_COMPONENTS" >&2; exit 2;
    }
    build_targets+=("$(COMPONENT=$component yq -er '.components[strenv(COMPONENT)].build_target' "$manifest")")
    digest_files+=("$(COMPONENT=$component yq -er '.components[strenv(COMPONENT)].digest_file' "$manifest")")
    push_target_by_component[$component]=$(COMPONENT=$component yq -er '.components[strenv(COMPONENT)].push_target' "$manifest")
done
cluster_sync_fast_cluster_ready
for component in "${selected_components[@]}"; do
    push_tag_by_component[$component]=$(component_push_tag "$component")
    push_alt_tag_by_component[$component]=$(component_push_alt_tag "$component")
done
mkdir -p "$(dirname "$state_file")"

build_output=$(cluster_sync_fast_build "$components" "${build_targets[*]}" "${digest_files[*]}")
printf '%s\n' "$build_output"
state_tmp=
declare -A candidate_digests
while IFS= read -r line; do
    [[ "$line" == FAST_SYNC_DIGEST=* ]] || continue
    value=${line#FAST_SYNC_DIGEST=}
    component=${value%%:*}
    digest=${value#*:}
    if [[ -z "$component" || "$digest" == "$value" || -z "$digest" ]]; then
        echo "Build hook returned an invalid digest record: '$line'" >&2
        exit 1
    fi
    if [[ -z "${selected_lookup[$component]:-}" || -n "${candidate_digests[$component]:-}" ]]; then
        echo "Build hook returned an unknown or duplicate component '$component'" >&2
        exit 1
    fi
    candidate_digests[$component]=$digest
done <<< "$build_output"
if ((${#candidate_digests[@]} == 0)); then
    echo "Build hook returned no image digests" >&2
    exit 1
fi

for component in "${selected_components[@]}"; do
    if [[ -z "${candidate_digests[$component]:-}" ]]; then
        echo "Build hook did not return a digest for '$component'" >&2
        exit 1
    fi
done

declare -A previous_digests
declare -A previous_tags previous_alt_tags
if [[ -f "$state_file" ]]; then
    while IFS=$'\t' read -r component digest tag alt_tag; do
        if [[ -n "$component" ]]; then
            previous_digests[$component]=$digest
            previous_tags[$component]=$tag
            previous_alt_tags[$component]=$alt_tag
        fi
    done < "$state_file"
fi

changed=()
for component in "${selected_components[@]}"; do
    if [[ "${candidate_digests[$component]:-}" != "${previous_digests[$component]:-}" ||
        "${push_tag_by_component[$component]}" != "${previous_tags[$component]:-}" ||
        "${push_alt_tag_by_component[$component]}" != "${previous_alt_tags[$component]:-}" ]]; then
        changed+=("$component")
    fi
done
if ((${#changed[@]} == 0)); then echo "No selected component image changed."; exit 0; fi
printf 'Changed images: %s\n' "${changed[*]}"

declare -A push_targets_by_group group_tag_by_key group_alt_tag_by_key
push_groups=()
for component in "${changed[@]}"; do
    push_target=${push_target_by_component[$component]}
    push_tag=${push_tag_by_component[$component]}
    push_tag_alt=${push_alt_tag_by_component[$component]}
    group_key="$push_tag|$push_tag_alt"
    if [[ -z "${push_targets_by_group[$group_key]+set}" ]]; then
        push_groups+=("$group_key")
        group_tag_by_key[$group_key]=$push_tag
        group_alt_tag_by_key[$group_key]=$push_tag_alt
    fi
    push_targets_by_group[$group_key]+=" $push_target"
done
for group_key in "${push_groups[@]}"; do
    cluster_sync_fast_push "${push_targets_by_group[$group_key]}" "${group_tag_by_key[$group_key]}" "${group_alt_tag_by_key[$group_key]}"
done

declare -A restarted_workloads waited_workloads deferred_components

run_component_flow() {
    local component=$1 flow action kind name workload timeout
    flow=$(COMPONENT=$component yq -er '.components[strenv(COMPONENT)].flow' "$manifest")
    kind=$(COMPONENT=$component yq -r '.components[strenv(COMPONENT)].workload.kind // ""' "$manifest")
    name=$(COMPONENT=$component yq -r '.components[strenv(COMPONENT)].workload.name // ""' "$manifest")
    timeout=$(COMPONENT=$component yq -r '.components[strenv(COMPONENT)].rollout_timeout // "5m"' "$manifest")
    workload=
    if [[ -n "$kind" && -n "$name" ]]; then
        workload="$kind/$name"
    elif [[ -n "$kind" || -n "$name" ]]; then
        echo "Component '$component' must configure both workload.kind and workload.name" >&2
        return 2
    fi
    if [[ -n "$workload" ]] && ! "$kubectl_cmd" -n "$namespace" get "$workload" >/dev/null 2>&1; then
        if [[ $(COMPONENT=$component yq -r '.components[strenv(COMPONENT)].optional // false' "$manifest") == true ]]; then
            echo "Optional workload '$workload' for '$component' is not installed; leaving its digest unrecorded."
            deferred_components[$component]=1
            return 0
        fi
        echo "Configured workload '$workload' for '$component' is not installed" >&2
        return 1
    fi
    while IFS= read -r action; do
        case "$action" in
            restart_workload)
                if [[ -z "$workload" ]]; then
                    echo "Flow '$flow' uses restart_workload but component '$component' has no workload configured" >&2
                    return 2
                fi
                if [[ -z "${restarted_workloads[$workload]:-}" ]]; then
                    "$kubectl_cmd" -n "$namespace" rollout restart "$workload"
                    restarted_workloads[$workload]=1
                fi
                ;;
            wait_for_rollout)
                if [[ -z "$workload" ]]; then
                    echo "Flow '$flow' uses wait_for_rollout but component '$component' has no workload configured" >&2
                    return 2
                fi
                if [[ -z "${waited_workloads[$workload]:-}" ]]; then
                    "$kubectl_cmd" -n "$namespace" rollout status "$workload" --timeout="${KUBEVIRT_FAST_ROLLOUT_TIMEOUT:-$timeout}"
                    waited_workloads[$workload]=1
                fi
                ;;
            *)
                if declare -F cluster_sync_fast_action >/dev/null; then
                    cluster_sync_fast_action "$action" "$component" "$workload" "$namespace" "$kubectl_cmd" "$timeout"
                else
                    echo "Unknown flow action '$action' for component '$component'; define cluster_sync_fast_action in a hooks file to handle it" >&2
                    return 2
                fi
                ;;
        esac
    done < <(FLOW=$flow yq -er '.flows[strenv(FLOW)][]' "$manifest")
}

planned_output=$(./hack/cluster-sync-fast-plan.sh "$manifest" "${changed[@]}")
mapfile -t planned <<< "$planned_output"
for component in "${planned[@]}"; do run_component_flow "$component"; done

state_tmp=$(mktemp "${state_file}.XXXXXX")
if [[ -f "$state_file" ]]; then cp "$state_file" "$state_tmp"; else : > "$state_tmp"; fi
for component in "${changed[@]}"; do
    [[ -n "${deferred_components[$component]:-}" ]] && continue
    awk -F '\t' -v c="$component" '$1 != c' "$state_tmp" > "$state_tmp.next"
    printf '%s\t%s\t%s\t%s\n' "$component" "${candidate_digests[$component]}" \
        "${push_tag_by_component[$component]}" "${push_alt_tag_by_component[$component]}" >> "$state_tmp.next"
    mv "$state_tmp.next" "$state_tmp"
done
mv "$state_tmp" "$state_file"
