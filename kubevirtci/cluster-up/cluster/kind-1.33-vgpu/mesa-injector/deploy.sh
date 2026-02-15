#!/bin/bash
# Copyright 2024 The KubeVirt Authors.
# Licensed under the Apache License, Version 2.0.

# Deploy mesa-injector webhook to inject OpenGL libraries into virt-launcher pods

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="mesa-injector"
SERVICE="mesa-injector"
SECRET="mesa-injector-certs"

usage() {
    echo "Usage: $0 [deploy|undeploy|status]"
    echo ""
    echo "Commands:"
    echo "  deploy   - Build and deploy the mesa-injector webhook"
    echo "  undeploy - Remove the mesa-injector webhook"
    echo "  status   - Show deployment status"
    exit 1
}

build_image() {
    echo "Building mesa-injector image..."
    
    # Build using docker
    docker build -t mesa-injector:latest "$SCRIPT_DIR"
    
    # Load into kind cluster
    local cluster_name="${CLUSTER_NAME:-vgpu}"
    echo "Loading image into kind cluster '$cluster_name'..."
    kind load docker-image mesa-injector:latest --name "$cluster_name"
}

deploy() {
    echo "Deploying mesa-injector webhook..."
    
    # Build and load image
    build_image
    
    # Create temp directory for certs
    local tmpdir=$(mktemp -d)
    
    echo "Generating TLS certificates..."
    
    # Generate CA
    openssl genrsa -out "$tmpdir/ca.key" 2048 2>/dev/null
    openssl req -x509 -new -nodes -key "$tmpdir/ca.key" -days 365 -out "$tmpdir/ca.crt" \
        -subj "/CN=mesa-injector-ca" 2>/dev/null
    
    # Generate server key and CSR
    openssl genrsa -out "$tmpdir/server.key" 2048 2>/dev/null
    
    cat > "$tmpdir/csr.conf" <<EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names
[alt_names]
DNS.1 = ${SERVICE}
DNS.2 = ${SERVICE}.${NAMESPACE}
DNS.3 = ${SERVICE}.${NAMESPACE}.svc
DNS.4 = ${SERVICE}.${NAMESPACE}.svc.cluster.local
EOF

    openssl req -new -key "$tmpdir/server.key" -out "$tmpdir/server.csr" \
        -subj "/CN=${SERVICE}.${NAMESPACE}.svc" -config "$tmpdir/csr.conf" 2>/dev/null
    
    # Sign server cert
    openssl x509 -req -in "$tmpdir/server.csr" -CA "$tmpdir/ca.crt" -CAkey "$tmpdir/ca.key" \
        -CAcreateserial -out "$tmpdir/server.crt" -days 365 -extensions v3_req -extfile "$tmpdir/csr.conf" 2>/dev/null
    
    # Get CA bundle
    local ca_bundle=$(base64 -w0 < "$tmpdir/ca.crt")
    
    # Create namespace
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    # Create secret with certs
    kubectl -n "$NAMESPACE" create secret tls "$SECRET" \
        --cert="$tmpdir/server.crt" \
        --key="$tmpdir/server.key" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # Create modified manifest with CA bundle
    cp "$SCRIPT_DIR/manifests/deploy.yaml" "$tmpdir/deploy.yaml"
    
    # Use awk to replace caBundle (more robust than sed for this)
    awk -v ca="$ca_bundle" '{gsub(/caBundle: ""/, "caBundle: \"" ca "\""); print}' \
        "$SCRIPT_DIR/manifests/deploy.yaml" > "$tmpdir/deploy.yaml"
    
    # Apply manifests
    kubectl apply -f "$tmpdir/deploy.yaml"
    
    # Cleanup
    rm -rf "$tmpdir"
    
    # Wait for deployment
    echo "Waiting for mesa-injector to be ready..."
    kubectl -n "$NAMESPACE" rollout status deployment/mesa-injector --timeout=60s
    
    echo ""
    echo "Mesa-injector deployed successfully!"
    echo "virt-launcher pods will now have OpenGL libraries injected."
}

undeploy() {
    echo "Removing mesa-injector webhook..."
    
    kubectl delete mutatingwebhookconfiguration mesa-injector --ignore-not-found
    kubectl delete namespace "$NAMESPACE" --ignore-not-found
    
    echo "Mesa-injector removed."
}

status() {
    echo "=== Mesa Injector Status ==="
    echo ""
    
    echo "Namespace:"
    kubectl get namespace "$NAMESPACE" 2>/dev/null || echo "  Not found"
    echo ""
    
    echo "Deployment:"
    kubectl -n "$NAMESPACE" get deployment 2>/dev/null || echo "  Not found"
    echo ""
    
    echo "Pods:"
    kubectl -n "$NAMESPACE" get pods 2>/dev/null || echo "  Not found"
    echo ""
    
    echo "Webhook Configuration:"
    kubectl get mutatingwebhookconfiguration mesa-injector 2>/dev/null || echo "  Not found"
}

case "${1:-}" in
    deploy)
        deploy
        ;;
    undeploy)
        undeploy
        ;;
    status)
        status
        ;;
    *)
        usage
        ;;
esac
