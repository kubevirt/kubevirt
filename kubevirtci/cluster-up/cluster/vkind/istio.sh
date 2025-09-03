#!/bin/bash
set -e

ISTIO_VERSION="1.24.4"

export KUBECONFIG=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig

echo "Deploying Istio with CNAO enabled..."

# Create istio-system namespace
echo "Creating istio-system namespace..."
kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: istio-system
EOF

# Install Istio using the operator configuration
echo "Installing Istio with istioctl..."
$KUBEVIRTCI_PATH/cluster/vkind/istioctl install -y -f $KUBEVIRTCI_PATH/cluster/vkind/manifests/istio-operator-with-cnao.cr.yaml

patch_cni_daemonset() {
    echo "Waiting for CNI DaemonSet to be created and patching it to be privileged..."
    
    # Wait up to 3 minutes for the DaemonSet to exist
    local max_attempts=18
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if kubectl get daemonset istio-cni-node -n kube-system >/dev/null 2>&1; then
            echo "Found istio-cni-node DaemonSet, patching to be privileged..."
            
            # Patch the DaemonSet to set privileged: true
            kubectl patch daemonset istio-cni-node -n kube-system -p '{"spec":{"template":{"spec":{"containers":[{"name":"install-cni","securityContext":{"privileged":true}}]}}}}'
            
            if [ $? -eq 0 ]; then
                echo "Successfully patched CNI DaemonSet to be privileged"
                return 0
            else
                echo "Failed to patch CNI DaemonSet, retrying..."
            fi
        else
            echo "CNI DaemonSet not found yet, waiting... (attempt $((attempt+1))/$max_attempts)"
        fi
        
        sleep 10
        ((attempt++))
    done
    
    echo "Warning: Could not find or patch CNI DaemonSet within timeout period"
    return 1
}

# Run the CNI DaemonSet patching in the background (like the Go code does)
patch_cni_daemonset

echo "Istio operator is now ready!"
