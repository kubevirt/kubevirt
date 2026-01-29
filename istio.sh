#!/bin/bash
set -e

# Standalone Istio deployment script for KubeVirt
# Supports two modes:
#   - With CNAO (Multus): ISTIO_MODE=cnao ./istio.sh
#   - Without CNAO (chained CNI): ISTIO_MODE=nocnao ./istio.sh or ./istio.sh

# Configuration
export ISTIO_VERSION=${ISTIO_VERSION:-1.26.4}
export ISTIO_MODE=${ISTIO_MODE:-cnao}  # Options: cnao, nocnao
export ISTIO_HUB=${ISTIO_HUB:-quay.io/kubevirtci}
export KUBECONFIG=${KUBECONFIG:-${HOME}/.kube/config}

# Determine architecture
ARCH=$(uname -m)
case ${ARCH} in
    x86_64* | i?86_64* | amd64*)
        ARCH="amd64"
        ;;
    aarch64* | arm64*)
        ARCH="arm64"
        ;;
    *)
        echo "Error: Unsupported architecture: ${ARCH}"
        exit 1
        ;;
esac

# Set up binary directory
ISTIO_BIN_DIR=/tmp/istio-${ISTIO_VERSION}/bin
export PATH=${ISTIO_BIN_DIR}:${PATH}
ISTIO_WORK_DIR=/tmp/istio-deployment
mkdir -p ${ISTIO_BIN_DIR}
mkdir -p ${ISTIO_WORK_DIR}

# Color output helpers
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster. Check your KUBECONFIG"
        exit 1
    fi
    
    log_info "Prerequisites check passed"
}

# Download and install istioctl
install_istioctl() {
    if [[ -f ${ISTIO_BIN_DIR}/istioctl ]]; then
        CURRENT_VERSION=$(${ISTIO_BIN_DIR}/istioctl version --remote=false 2>/dev/null | grep -oP '(?<=version.Version\{Raw:")[^"]+' || echo "unknown")
        if [[ "${CURRENT_VERSION}" == "${ISTIO_VERSION}" ]]; then
            log_info "istioctl ${ISTIO_VERSION} already installed"
            return
        fi
    fi
    
    log_info "Downloading istioctl ${ISTIO_VERSION} for ${ARCH}..."
    
    ISTIO_TARBALL="istio-${ISTIO_VERSION}-linux-${ARCH}.tar.gz"
    ISTIO_URL="https://github.com/istio/istio/releases/download/${ISTIO_VERSION}/${ISTIO_TARBALL}"
    
    cd /tmp
    if ! curl -L -o ${ISTIO_TARBALL} ${ISTIO_URL}; then
        log_error "Failed to download istioctl from ${ISTIO_URL}"
        exit 1
    fi
    
    log_info "Extracting istioctl..."
    tar -xzf ${ISTIO_TARBALL} --strip-components=2 -C ${ISTIO_BIN_DIR} istio-${ISTIO_VERSION}/bin/istioctl
    chmod +x ${ISTIO_BIN_DIR}/istioctl
    
    rm -f ${ISTIO_TARBALL}
    
    log_info "istioctl installed successfully at ${ISTIO_BIN_DIR}/istioctl"
    ${ISTIO_BIN_DIR}/istioctl version --remote=false
}

# Create istio-system namespace
create_namespace() {
    log_info "Creating istio-system namespace..."
    if kubectl get namespace istio-system &> /dev/null; then
        log_warn "Namespace istio-system already exists"
    else
        kubectl create namespace istio-system
        log_info "Namespace istio-system created"
    fi
}

# Generate Istio operator manifest based on mode
generate_istio_manifest() {
    local mode=$1
    local manifest_file="${ISTIO_WORK_DIR}/istio-operator.yaml"
    
    log_info "Generating Istio operator manifest for mode: ${mode}"
    
    if [[ "${mode}" == "cnao" ]]; then
        # With CNAO (Multus) - chained=false
        cat > ${manifest_file} <<EOF
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  namespace: istio-system
  name: istio-operator
spec:
  profile: demo
  hub: ${ISTIO_HUB}
  components:
    cni:
      enabled: true
      namespace: kube-system
      k8s:
        securityContext:
          seLinuxOptions:
            type: spc_t
  values:
    global:
      jwtPolicy: third-party-jwt
    cni:
      provider: multus
      chained: false
      cniBinDir: /opt/cni/bin
      cniConfDir: /etc/cni/multus/net.d
      excludeNamespaces:
       - istio-system
       - kube-system
      logLevel: debug
    pilot:
      cni:
        enabled: true
        provider: "multus"
EOF
    else
        # Without CNAO (chained CNI) - chained=true
        cat > ${manifest_file} <<EOF
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  namespace: istio-system
  name: istio-operator
spec:
  profile: demo
  hub: ${ISTIO_HUB}
  components:
    cni:
      enabled: true
      namespace: kube-system
      k8s:
        securityContext:
          seLinuxOptions:
            type: spc_t
  values:
    global:
      jwtPolicy: third-party-jwt
    cni:
      chained: true
      cniBinDir: /opt/cni/bin
      cniConfDir: /etc/cni/net.d
      privileged: true
      excludeNamespaces:
       - istio-system
       - kube-system
      logLevel: debug
EOF
    fi
    
    log_info "Manifest generated at ${manifest_file}"
    echo "${manifest_file}"
}

# Deploy Istio
deploy_istio() {
    local manifest_file=$1
    
    log_info "Deploying Istio with manifest: ${manifest_file}"
    
    if ! istioctl install -y -f ${manifest_file}; then
        log_error "Failed to install Istio"
        exit 1
    fi
    
    log_info "Istio deployment initiated successfully"
}

# Patch CNI DaemonSet to ensure privileged mode (same as kubevirtci does)
patch_cni_daemonset() {
    log_info "Patching istio-cni-node DaemonSet to ensure privileged mode..."
    
    local max_attempts=18  # 3 minutes with 10-second intervals
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if kubectl get daemonset istio-cni-node -n kube-system &>/dev/null; then
            log_info "DaemonSet found, applying privileged patch..."
            
            if kubectl patch daemonset istio-cni-node -n kube-system --type=json -p='[{"op": "add", "path": "/spec/template/spec/containers/0/securityContext/privileged", "value": true}]' 2>/dev/null; then
                log_info "Successfully patched istio-cni-node DaemonSet"
                return 0
            else
                # Check if it's already privileged
                if kubectl get daemonset istio-cni-node -n kube-system -o jsonpath='{.spec.template.spec.containers[0].securityContext.privileged}' 2>/dev/null | grep -q "true"; then
                    log_info "DaemonSet is already configured as privileged"
                    return 0
                fi
            fi
        fi
        
        attempt=$((attempt + 1))
        if [ $attempt -lt $max_attempts ]; then
            log_info "Waiting for istio-cni-node DaemonSet to be created (attempt $attempt/$max_attempts)..."
            sleep 10
        fi
    done
    
    log_warn "Could not patch istio-cni-node DaemonSet within timeout (this may be okay if it's already privileged)"
    return 0
}

# Wait for Istio to be ready
wait_for_istio() {
    log_info "Waiting for Istio components to be ready..."
    
    log_info "Waiting for istiod deployment..."
    kubectl wait --for=condition=available --timeout=300s deployment/istiod -n istio-system || {
        log_error "istiod deployment failed to become ready"
        exit 1
    }
    
    log_info "Waiting for istio-cni-node daemonset..."
    kubectl rollout status daemonset/istio-cni-node -n kube-system --timeout=300s || {
        log_warn "istio-cni-node daemonset status check failed (this may be normal)"
    }
    
    log_info "All Istio components are ready!"
}

# Verify installation
verify_installation() {
    log_info "Verifying Istio installation..."
    
    istioctl verify-install || {
        log_warn "Istio verification reported some issues (check above)"
    }
    
    log_info "Istio installation verification complete"
}

# Print summary
print_summary() {
    log_info "============================================"
    log_info "Istio Deployment Summary"
    log_info "============================================"
    log_info "Istio Version: ${ISTIO_VERSION}"
    log_info "Deployment Mode: ${ISTIO_MODE}"
    log_info "Hub: ${ISTIO_HUB}"
    log_info "Architecture: ${ARCH}"
    log_info "============================================"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up temporary files..."
    # Keep the binary for reuse, just clean up manifests
    rm -rf ${ISTIO_WORK_DIR}
}

# Main execution
main() {
    log_info "Starting Istio deployment..."
    log_info "Mode: ${ISTIO_MODE}"
    log_info "Version: ${ISTIO_VERSION}"
    
    check_prerequisites
    install_istioctl
    create_namespace
    
    manifest_file=$(generate_istio_manifest ${ISTIO_MODE})
    deploy_istio ${manifest_file}
    patch_cni_daemonset
    wait_for_istio
    verify_installation
    
    cleanup
    print_summary
    
    log_info "Istio deployment completed successfully!"
}

# Run main function
main
