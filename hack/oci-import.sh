#!/bin/bash
#
# Import an exported KubeVirt resource from an OCI artifact back into a cluster.
#
# Supports both VirtualMachine and VirtualMachineTemplate artifacts.
# Accepts either a local OCI TAR file (--tar) or a registry reference
# (--registry). Extracts the config and disk layers, uploads each
# disk as a PVC via virtctl, and creates the resource.
#
# Usage:
#   ./hack/oci-import.sh --tar export.oci.tar [--name my-vm] [--namespace default]
#   ./hack/oci-import.sh --registry registry.example.com/vms/myvm:latest

set -euo pipefail

TAR_PATH=""
REGISTRY_REF=""
RESOURCE_NAME=""
NAMESPACE=""
INSECURE=false

usage() {
    cat <<EOF
Usage: $0 --tar <path> | --registry <ref> [--name <name>] [--namespace <ns>]

Options:
  --tar <path>        Path to local OCI TAR file
  --registry <ref>    OCI registry reference (e.g. registry.example.com/vms/myvm:latest)
  --name <name>       Resource name (default: from config metadata)
  --namespace <ns>    Target namespace (default: current kubectl context)
  --insecure          Skip TLS verification for virtctl image-upload

Supported artifact types:
  application/vnd.kubevirt.virtualmachine.v1
  application/vnd.kubevirt.virtualmachinetemplate.v1

Examples:
  # Export a VM as OCI TAR, then import it into another cluster:
  virtctl vmexport create my-export --vm=my-vm
  virtctl vmexport download my-export --format=oci --output=my-vm.oci.tar
  $0 --tar my-vm.oci.tar --name my-imported-vm --namespace target-ns

  # Push OCI TAR to a registry with skopeo, then import from there:
  podman unshare skopeo copy --multi-arch=all oci-archive:my-vm.oci.tar docker://registry.example.com/vms/my-vm:v1
  $0 --registry registry.example.com/vms/my-vm:v1

  # Export a template as OCI TAR, then import it into another cluster:
  virtctl vmexport create tpl-export --vmtemplate=my-template
  virtctl vmexport download tpl-export --format=oci --output=my-template.oci.tar
  $0 --tar my-template.oci.tar --name my-imported-template --namespace target-ns

  # Push template OCI TAR to a registry with skopeo, then import from there:
  podman unshare skopeo copy --multi-arch=all oci-archive:my-template.oci.tar docker://registry.example.com/templates/my-template:v1
  $0 --registry registry.example.com/templates/my-template:v1

Dependencies: jq, zstd, virtctl, kubectl, skopeo (registry mode only)
EOF
    exit 1
}

while [[ $# -gt 0 ]]; do
    case "$1" in
    --tar)
        TAR_PATH="$2"
        shift 2
        ;;
    --registry)
        REGISTRY_REF="$2"
        shift 2
        ;;
    --name)
        RESOURCE_NAME="$2"
        shift 2
        ;;
    --namespace)
        NAMESPACE="$2"
        shift 2
        ;;
    --insecure)
        INSECURE=true
        shift
        ;;
    -h | --help)
        usage
        ;;
    *)
        echo "Error: unknown option: $1" >&2
        usage
        ;;
    esac
done

if [[ -n "$TAR_PATH" && -n "$REGISTRY_REF" ]]; then
    echo "Error: --tar and --registry are mutually exclusive" >&2
    exit 1
fi
if [[ -z "$TAR_PATH" && -z "$REGISTRY_REF" ]]; then
    echo "Error: one of --tar or --registry is required" >&2
    exit 1
fi

check_dep() {
    if ! command -v "$1" &>/dev/null; then
        echo "Error: $1 is required but not found" >&2
        exit 1
    fi
}

check_dep jq
check_dep zstd
check_dep virtctl
check_dep kubectl
if [[ -n "$REGISTRY_REF" ]]; then
    check_dep skopeo
fi

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

OCI_DIR="$TMPDIR/oci"

if [[ -n "$TAR_PATH" ]]; then
    if [[ ! -f "$TAR_PATH" ]]; then
        echo "Error: $TAR_PATH not found" >&2
        exit 1
    fi
    mkdir -p "$OCI_DIR"
    echo "Extracting OCI TAR..."
    tar xf "$TAR_PATH" -C "$OCI_DIR"
else
    echo "Pulling from registry..."
    skopeo copy "docker://$REGISTRY_REF" "oci:$OCI_DIR"
fi

if [[ ! -f "$OCI_DIR/oci-layout" || ! -f "$OCI_DIR/index.json" ]]; then
    echo "Error: not a valid OCI image layout" >&2
    exit 1
fi

blob() { echo "$OCI_DIR/blobs/${1/://}"; }

MANIFEST_BLOB=$(blob "$(jq -r '.manifests[0].digest' "$OCI_DIR/index.json")")

ARTIFACT_TYPE=$(jq -r '.artifactType // empty' "$MANIFEST_BLOB")
CONFIG_MEDIA_TYPE=$(jq -r '.config.mediaType // empty' "$MANIFEST_BLOB")

IS_VM=false
IS_TEMPLATE=false

case "$ARTIFACT_TYPE" in
"application/vnd.kubevirt.virtualmachine.v1")
    if [[ "$CONFIG_MEDIA_TYPE" != "application/vnd.kubevirt.virtualmachine.config.v1+json" ]]; then
        echo "Error: unexpected config media type for VM artifact: ${CONFIG_MEDIA_TYPE:-<none>}" >&2
        exit 1
    fi
    IS_VM=true
    ;;
"application/vnd.kubevirt.virtualmachinetemplate.v1")
    if [[ "$CONFIG_MEDIA_TYPE" != "application/vnd.kubevirt.virtualmachinetemplate.config.v1+json" ]]; then
        echo "Error: unexpected config media type for VMTemplate artifact: ${CONFIG_MEDIA_TYPE:-<none>}" >&2
        exit 1
    fi
    IS_TEMPLATE=true
    ;;
*)
    echo "Error: unexpected artifact type: ${ARTIFACT_TYPE:-<none>}" >&2
    exit 1
    ;;
esac

CONFIG_BLOB=$(blob "$(jq -r '.config.digest' "$MANIFEST_BLOB")")

if [[ -z "$RESOURCE_NAME" ]]; then
    RESOURCE_NAME=$(jq -r '.metadata.name' "$CONFIG_BLOB")
fi
if [[ -z "$NAMESPACE" ]]; then
    NAMESPACE=$(kubectl config view --minify -o jsonpath='{..namespace}' 2>/dev/null || true)
    NAMESPACE="${NAMESPACE:-default}"
fi

if [[ "$IS_VM" == true ]]; then
    echo "Importing VM '$RESOURCE_NAME' into namespace '$NAMESPACE'"
else
    echo "Importing VM template '$RESOURCE_NAME' into namespace '$NAMESPACE'"
fi

# Extract all layer metadata in one jq call
LAYERS=$(jq -c '.layers[] | select(.mediaType == "application/vnd.kubevirt.disk.raw+zstd") | {
    digest: .digest,
    name: .annotations["io.kubevirt.disk.name"],
    size: .annotations["io.kubevirt.disk.size"]
}' "$MANIFEST_BLOB")

declare -A PVC_MAP

while IFS= read -r layer; do
    DISK_NAME=$(jq -r '.name' <<<"$layer")
    LAYER_BLOB=$(blob "$(jq -r '.digest' <<<"$layer")")
    RAW_FILE="$TMPDIR/${DISK_NAME}.raw"

    echo "Decompressing disk '$DISK_NAME'..."
    zstd -d "$LAYER_BLOB" -o "$RAW_FILE" --no-progress

    PVC_NAME="${DISK_NAME}-${RESOURCE_NAME}"
    PVC_MAP["$DISK_NAME"]="$PVC_NAME"

    UPLOAD_ARGS=(
        virtctl image-upload dv "$PVC_NAME"
        --size="$(jq -r '.size' <<<"$layer")"
        --image-path="$RAW_FILE"
        --namespace="$NAMESPACE"
    )

    if [[ "$INSECURE" == true ]]; then
        UPLOAD_ARGS+=(--insecure)
    fi

    echo "Uploading disk '$DISK_NAME' as PVC '$PVC_NAME'..."
    "${UPLOAD_ARGS[@]}"

    rm -f "$RAW_FILE"
done <<<"$LAYERS"

# Build jq rewrite expression for PVC claim name mapping
# shellcheck disable=SC2016
jq_rewrite='.metadata.name = $name | .metadata.namespace = $ns'
jq_args=(--arg name "$RESOURCE_NAME" --arg ns "$NAMESPACE")

DVTS_PATH=""
if [[ "$IS_VM" == true ]]; then
    VOLUMES_PATH=".spec.template.spec.volumes[]"
else
    VOLUMES_PATH=".spec.virtualMachine.spec.template.spec.volumes[]"
    DVTS_PATH=".spec.virtualMachine.spec.dataVolumeTemplates[]"
fi

idx=0
for DISK_NAME in "${!PVC_MAP[@]}"; do
    PVC_NAME="${PVC_MAP[$DISK_NAME]}"
    jq_rewrite+=" | (${VOLUMES_PATH} |
        select(.persistentVolumeClaim.claimName != null) |
        select((.persistentVolumeClaim.claimName | gsub(\"\\\\.\"; \"-\")) == \$d${idx}) |
        .persistentVolumeClaim.claimName) = \$p${idx}"
    if [[ "$IS_TEMPLATE" == true ]]; then
        jq_rewrite+=" | (${DVTS_PATH} |
            select(.spec.source.pvc.name != null) |
            select((.spec.source.pvc.name | gsub(\"\\\\.\"; \"-\")) == \$d${idx}) |
            .spec.source.pvc.name) = \$p${idx}"
    fi
    jq_args+=(--arg "d${idx}" "$DISK_NAME" --arg "p${idx}" "$PVC_NAME")
    idx=$((idx + 1))
done

jq "${jq_args[@]}" "$jq_rewrite" "$CONFIG_BLOB" >"$TMPDIR/resource.json"

echo "Creating resource..."
kubectl create -f "$TMPDIR/resource.json"

if [[ "$IS_VM" == true ]]; then
    echo "VM '$RESOURCE_NAME' imported successfully in namespace '$NAMESPACE'"
else
    echo "VirtualMachineTemplate '$RESOURCE_NAME' imported successfully in namespace '$NAMESPACE'"
fi
