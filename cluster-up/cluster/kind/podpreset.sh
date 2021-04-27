 #!/usr/bin/env bash

set -e

source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh

function podpreset::enable_admission_plugin() {
    local -r cluster_name=$1

    docker exec "$cluster_name-control-plane" bash -c 'sed -i \
    -e "s/NodeRestriction/NodeRestriction,PodPreset/" \
    -e "/NodeRestriction,PodPreset/ a\    - --runtime-config=settings.k8s.io/v1alpha1=true" \
    /etc/kubernetes/manifests/kube-apiserver.yaml'
}

function podpreset::validate_admission_plugin_is_enabled() {
    local -r cluster_name=$1
    local -r wait_time=$2
    local -r control_plane_container="$cluster_name-control-plane"

    if ! timeout "${wait_time}s" bash <<EOT
function is_admission_plugin_enabled() {
    docker top $control_plane_container |
        grep -Po "kube-apiserver.*--enable-admission-plugins=.*\KPodPreset"
}
until is_admission_plugin_enabled; do
    sleep 1
done
EOT
    then
        echo "FATAL: failed to enable PodPreset admission plugin
        cluster:    $cluster_name
        container:  $control_plane_container" >&2
        return 1
    fi
}

function podpreset::create_virt_launcher_fake_product_uuid_podpreset() {
    local -r namespace=$1

    if ! _kubectl get ns "$namespace" &>2; then
        _kubectl create ns "$namespace"
    fi

    _kubectl apply -f "$KIND_MANIFESTS_DIR/product-uuid-podpreset.yaml" -n "$namespace"
}

function podpreset::expose_unique_product_uuid_per_node() {
    local -r cluster_name=$1
    local -r namespace=$2

    podpreset::enable_admission_plugin "$cluster_name"
    podpreset::validate_admission_plugin_is_enabled "$cluster_name" "10"
    podpreset::create_virt_launcher_fake_product_uuid_podpreset "$namespace"
}
