#!/usr/bin/env bash
# Build selected KubeVirt images, push only changed content, and update every
# installed workload that references those images.
set -eo pipefail
cd "$(dirname "$0")/.."

core_specs='virt-operator //cmd/virt-operator:virt-operator-image
virt-api //cmd/virt-api:virt-api-image
virt-controller //cmd/virt-controller:virt-controller-image
virt-handler //cmd/virt-handler:virt-handler-image
virt-launcher //cmd/virt-launcher:virt-launcher-image
virt-exportserver //cmd/virt-exportserver:virt-exportserver-image
virt-exportproxy //cmd/virt-exportproxy:virt-exportproxy-image
virt-synchronization-controller //cmd/synchronization-controller:virt-synchronization-controller-image'
template_specs='virt-template-apiserver //images/virt-template:virt-template-apiserver-image
virt-template-controller //images/virt-template:virt-template-controller-image'

load_specs() {
    local specs=$core_specs image target digest_path
    local -a requested=()
    local -A selected=()
    if [[ ${TILT_VIRT_TEMPLATE_ENABLED:-false} == true ]]; then
        specs+=$'\n'"$template_specs"
    fi
    if [[ -n ${PUSH_TARGETS:-} ]]; then
        read -r -a requested <<< "${PUSH_TARGETS//,/ }"
        ((${#requested[@]})) || { echo 'PUSH_TARGETS is empty' >&2; return 2; }
        for image in "${requested[@]}"; do selected[$image]=1; done
    fi
    images=()
    targets=()
    digest_files=()
    while read -r image target; do
        [[ -n $image ]] || continue
        if ((${#requested[@]})) && [[ ! ${selected[$image]:-} ]]; then continue; fi
        images+=("$image")
        targets+=("$target")
        digest_path=${target#//}
        digest_files+=("bazel-bin/${digest_path/:/\/}_digest")
        unset 'selected[$image]'
    done <<< "$specs"
    if ((${#selected[@]})); then
        echo "Unknown or disabled PUSH_TARGETS: ${!selected[*]}" >&2
        return 2
    fi
}

build_images() {
    # Digest paths exist inside KubeVirt's builder container.
    # shellcheck disable=SC1091
    source hack/common.sh
    # shellcheck disable=SC1091
    source hack/bootstrap.sh
    # shellcheck disable=SC1091
    source hack/config.sh

    read -r -a bazel_configs <<< "${BAZEL_CS_CONFIG:-}"
    bazel build --config="$ARCHITECTURE" "${bazel_configs[@]}" \
        --define container_prefix= --define image_prefix= \
        --define container_tag= "${targets[@]}"

    for i in "${!images[@]}"; do
        file=${digest_files[$i]}
        [[ -f $file ]] || { echo "Missing Bazel digest: $file" >&2; return 1; }
        digest=$(sha256sum "$file" | cut -d ' ' -f 1)
        printf 'TILT_IMAGE_DIGEST=%s:%s\n' "${images[$i]}" "$digest"
    done
}

if [[ ${1:-} == build ]]; then
    load_specs
    build_images
    exit
fi

kubectl_cmd=${KUBEVIRT_TILT_KUBECTL:-./kubevirtci/cluster-up/kubectl.sh}
dockerized_cmd=${KUBEVIRT_TILT_DOCKERIZED:-./hack/dockerized}
namespace=${KUBEVIRT_INSTALLED_NAMESPACE:-kubevirt}

# shellcheck disable=SC1091
source hack/common.sh
# shellcheck disable=SC1090
source "kubevirtci/cluster-up/cluster/${KUBEVIRT_PROVIDER}/provider.sh"
# shellcheck disable=SC1091
source hack/config.sh
# shellcheck disable=SC1091
source hack/virt-template/default.sh

operator_image=$("$kubectl_cmd" -n "$namespace" get deployment virt-operator \
    -o=jsonpath='{.spec.template.spec.containers[?(@.name=="virt-operator")].image}')
[[ -n $operator_image ]] || { echo "No installed virt-operator in $namespace" >&2; exit 1; }
installed_tag=${operator_image##*:}
tag=${DOCKER_TAG:-$installed_tag}
if [[ $tag != "$installed_tag" ]]; then
    echo "DOCKER_TAG=$tag differs from the installed operator tag $installed_tag; regenerate and apply the manifests first" >&2
    exit 1
fi
alt_tag=${DOCKER_TAG_ALT:-}
template_tag=${VIRT_TEMPLATE_VERSION:-$virt_template_version}

template_setting=$("$kubectl_cmd" -n "$namespace" get kv kubevirt \
    -o=jsonpath='{.spec.configuration.virtTemplateDeployment.enabled}')
enabled_gates=$("$kubectl_cmd" -n "$namespace" get kv kubevirt \
    -o=jsonpath='{range .spec.configuration.developerConfiguration.featureGates[*]}{.}{"\n"}{end}')
disabled_gates=$("$kubectl_cmd" -n "$namespace" get kv kubevirt \
    -o=jsonpath='{range .spec.configuration.developerConfiguration.disabledFeatureGates[*]}{.}{"\n"}{end}')

# Template is a beta feature gate: enabled by default unless explicitly
# disabled. An explicit featureGates entry takes precedence over disablement.
TILT_VIRT_TEMPLATE_ENABLED=true
if [[ $template_setting == false ]] || {
    ! grep -Fxq Template <<< "$enabled_gates" &&
    grep -Fxq Template <<< "$disabled_gates"
}; then
    TILT_VIRT_TEMPLATE_ENABLED=false
fi

load_specs
if [[ $TILT_VIRT_TEMPLATE_ENABLED == true ]]; then
    for template in "${images[@]}"; do
        [[ $template == virt-template-* ]] || continue
        installed_image=$("$kubectl_cmd" -n "$namespace" get "deployment/$template" \
            -o=jsonpath='{.spec.template.spec.containers[0].image}')
        if [[ ${installed_image##*:} != "$template_tag" ]]; then
            echo "$template uses ${installed_image##*:}; requested VirtTemplate tag is $template_tag. Update the operator's embedded manifest first." >&2
            exit 1
        fi
    done
fi

# A new cluster or registry needs fresh pushes even when source digests match.
cluster_uid=$("$kubectl_cmd" get namespace "$namespace" -o=jsonpath='{.metadata.uid}')
[[ -n $cluster_uid ]] || { echo "Cannot identify namespace $namespace" >&2; exit 1; }
# These values come from the sourced provider and hack/config.sh.
# shellcheck disable=SC2154
state_scope="$cluster_uid|$docker_prefix|$image_prefix|$image_prefix_alt"
state_dir=${KUBEVIRT_TILT_STATE_DIR:-.tilt-kubevirt-state}
mkdir -p "$state_dir"
state_file="$state_dir/kubevirt-images.tsv"

wait_kubevirt_ready() {
    local timeout=${KUBEVIRT_TILT_ROLLOUT_TIMEOUT:-10m}
    for condition in Available=True Progressing=False Degraded=False; do
        "$kubectl_cmd" -n "$namespace" wait kv/kubevirt \
            --for="condition=$condition" --timeout="$timeout"
    done
}

build_output=$("$dockerized_cmd" \
    "KUBEVIRT_PROVIDER='$KUBEVIRT_PROVIDER' BUILD_ARCH='${BUILD_ARCH:-}' PUSH_TARGETS='${PUSH_TARGETS:-}' TILT_VIRT_TEMPLATE_ENABLED='$TILT_VIRT_TEMPLATE_ENABLED' ./hack/tilt-cluster-sync.sh build")
printf '%s\n' "$build_output"

declare -A built_digests changed
while IFS= read -r line; do
    [[ $line == TILT_IMAGE_DIGEST=* ]] || continue
    value=${line#TILT_IMAGE_DIGEST=}
    built_digests[${value%%:*}]=${value#*:}
done <<< "$build_output"

core_push=()
template_push=()
records=()
for image in "${images[@]}"; do
    digest=${built_digests[$image]:-}
    [[ -n $digest ]] || { echo "Build returned no digest for $image" >&2; exit 1; }
    image_tag=$tag
    image_alt_tag=$alt_tag
    if [[ $image == virt-template-* ]]; then
        image_tag=$template_tag
        image_alt_tag=
    fi
    record="$image"$'\t'"$digest"$'\t'"$image_tag"$'\t'"$image_alt_tag"$'\t'"$state_scope"
    records+=("$record")
    if ! grep -Fqx -- "$record" "$state_file" 2>/dev/null; then
        changed[$image]=1
        if [[ $image == virt-template-* ]]; then
            template_push+=("$image")
        else
            core_push+=("$image")
        fi
    fi
done

if ((${#core_push[@]} == 0 && ${#template_push[@]} == 0)); then
    echo 'No KubeVirt image changed'
    exit 0
fi

push_images() {
    local push_tag=$1 push_alt_tag=$2
    shift 2
    local push_targets="$*"
    "$dockerized_cmd" "PUSH_TARGETS='$push_targets' BUILD_ARCH='${BUILD_ARCH:-}' \
DOCKER_PREFIX='${DOCKER_PREFIX:-}' DOCKER_TAG='$push_tag' DOCKER_TAG_ALT='$push_alt_tag' \
IMAGE_PREFIX='${IMAGE_PREFIX:-$image_prefix}' IMAGE_PREFIX_ALT='${IMAGE_PREFIX_ALT:-$image_prefix_alt}' \
KUBEVIRT_PROVIDER='$KUBEVIRT_PROVIDER' ./hack/bazel-push-images.sh"
}

if ((${#core_push[@]})); then push_images "$tag" "$alt_tag" "${core_push[@]}"; fi
if ((${#template_push[@]})); then push_images "$template_tag" '' "${template_push[@]}"; fi

rollout() {
    local workload=$1
    "$kubectl_cmd" -n "$namespace" rollout restart "$workload"
    "$kubectl_cmd" -n "$namespace" rollout status "$workload" \
        --timeout="${KUBEVIRT_TILT_ROLLOUT_TIMEOUT:-10m}"
}

# A changed launcher image appears in both the controller's --launcher-image
# argument and virt-handler's init container. Exportserver appears in the
# controller's --exporter-image argument. Roll each consumer out once.
if [[ ${changed[virt-operator]:-} ]]; then rollout deployment/virt-operator; fi
if [[ ${changed[virt-api]:-} ]]; then rollout deployment/virt-api; fi
if [[ ${changed[virt-controller]:-} || ${changed[virt-launcher]:-} || ${changed[virt-exportserver]:-} ]]; then
    rollout deployment/virt-controller
fi
if [[ ${changed[virt-exportproxy]:-} ]]; then rollout deployment/virt-exportproxy; fi
if [[ ${changed[virt-synchronization-controller]:-} ]]; then rollout deployment/virt-synchronization-controller; fi
if [[ ${changed[virt-handler]:-} || ${changed[virt-launcher]:-} ]]; then
    rollout daemonset/virt-handler
fi
if [[ ${changed[virt-template-apiserver]:-} ]]; then rollout deployment/virt-template-apiserver; fi
if [[ ${changed[virt-template-controller]:-} ]]; then rollout deployment/virt-template-controller; fi

wait_kubevirt_ready

tmp=$(mktemp "$state_file.XXXXXX")
trap 'rm -f "$tmp"' EXIT
declare -A selected_images=()
for image in "${images[@]}"; do selected_images[$image]=1; done
if [[ -f $state_file ]]; then
    while IFS= read -r old_record; do
        old_image=${old_record%%$'\t'*}
        if [[ ! ${selected_images[$old_image]:-} ]]; then
            printf '%s\n' "$old_record" >> "$tmp"
        fi
    done < "$state_file"
fi
printf '%s\n' "${records[@]}" >> "$tmp"
mv "$tmp" "$state_file"
updated=("${core_push[@]}" "${template_push[@]}")
echo "Updated ${updated[*]}"
