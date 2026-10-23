#!/usr/bin/env bash

set -euo pipefail

kubectl get kubevirt kubevirt -n kubevirt >/dev/null 2>&1 || {
    echo "KubeVirt is not installed in the active cluster. Install KubeVirt first." >&2
    exit 1
}

echo "Configuring KubeVirt to manage fake NVIDIA mediated devices..."
kubectl patch kubevirt kubevirt -n kubevirt --type=merge -p '
{
  "spec": {
    "configuration": {
      "mediatedDevicesConfiguration": {
        "mediatedDeviceTypes": ["nvidia-222"],
        "nodeMediatedDeviceTypes": null,
        "enabled": true
      },
      "permittedHostDevices": {
        "mediatedDevices": [
          {"mdevNameSelector": "GRID T4-1B", "resourceName": "nvidia.com/GRID_T4-1B"},
          {"mdevNameSelector": "GRID T4-2B", "resourceName": "nvidia.com/GRID_T4-2B"}
        ]
      }
    }
  }
}'

echo "KubeVirt CR configured. Waiting for KubeVirt to become Available..."
kubectl wait kubevirt/kubevirt -n kubevirt --for=condition=Available --timeout=10m

generation=$(kubectl get kubevirt kubevirt -n kubevirt -o jsonpath='{.metadata.generation}')
if [[ ! "$generation" =~ ^[0-9]+$ ]]; then
    echo "KubeVirt did not report a valid generation: $generation" >&2
    exit 1
fi
expected_instances=16
resource_name='nvidia.com/GRID_T4-1B'
nodes=$(kubectl get nodes -l vgpu_capable=true -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}')
if [[ -z "$nodes" ]]; then
    echo "No vgpu_capable=true nodes found. Run setup_vgpu_mdev.sh first." >&2
    exit 1
fi

deadline=$((SECONDS + 600))
observed_generation=''
while (( SECONDS < deadline )); do
    observed_generation=$(kubectl get kubevirt kubevirt -n kubevirt -o jsonpath='{.status.observedGeneration}')
    if [[ "$observed_generation" =~ ^[0-9]+$ ]] && (( observed_generation >= generation )); then
        break
    fi
    sleep 5
done
if [[ ! "$observed_generation" =~ ^[0-9]+$ ]] || (( observed_generation < generation )); then
    echo "KubeVirt did not observe generation $generation within 10 minutes." >&2
    exit 1
fi

for node in $nodes; do
    echo "Waiting for $expected_instances nvidia-222 mdevs and $resource_name on $node..."
    mdev_count=''
    allocatable=''
    while (( SECONDS < deadline )); do
        handlers=$(kubectl get pods -n kubevirt -l kubevirt.io=virt-handler \
            --field-selector "spec.nodeName=$node,status.phase=Running" \
            -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}')
        handler=${handlers%%$'\n'*}
        mdev_count=''
        if [[ -n "$handler" ]]; then
            mdev_count=$(kubectl exec -n kubevirt "$handler" -- bash -c '
                count=0
                for dev in /sys/bus/mdev/devices/*; do
                    [[ -e "$dev" ]] || continue
                    if [[ $(basename "$(readlink -f "$dev/mdev_type")") == nvidia-222 ]]; then
                        count=$((count + 1))
                    fi
                done
                echo "$count"
            ' 2>/dev/null) || mdev_count=''
        fi
        allocatable=$(kubectl get node "$node" \
            -o go-template='{{with index .status.allocatable "nvidia.com/GRID_T4-1B"}}{{.}}{{end}}')
        if [[ "$mdev_count" == "$expected_instances" && "$allocatable" == "$expected_instances" ]]; then
            echo "$node is ready: $mdev_count mdevs, $allocatable allocatable $resource_name devices."
            break
        fi
        sleep 5
    done
    if [[ "$mdev_count" != "$expected_instances" || "$allocatable" != "$expected_instances" ]]; then
        echo "$node did not become ready within 10 minutes (mdevs: ${mdev_count:-unknown}, allocatable: ${allocatable:-unknown})." >&2
        exit 1
    fi
done

echo "KubeVirt mediated devices are ready on all vGPU workers."
