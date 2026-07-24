# Network Policies Integration

## Overview

KubeVirt supports network policies to enable secure deployments in environments with strict network isolation
requirements. This integration provides the necessary network policies for KubeVirt components to communicate with each
other and with cluster services (Kubernetes API server and DNS).

Network policies are particularly useful in multi-tenant environments or security-hardened clusters where a default-deny
network policy is enforced.

## Network Policy Categories

KubeVirt network policies are divided into two main categories:

### 1. Cluster Services Access Policies

These policies allow KubeVirt components to communicate with the Kubernetes API server and DNS services. Since API
server and DNS configurations can vary between clusters (custom ports, namespaces, labels), these policies **must be
defined by the cluster administrator** based on their specific cluster configuration.

### 2. Inter-Component Communication Policies

These policies govern communication between KubeVirt components (virt-operator, virt-api, virt-controller, virt-handler,
etc.). These are provided by KubeVirt and can be generated using the `csv-generator` tool.

## Architecture

The following KubeVirt components require network access:

- **virt-operator**: Manages the lifecycle of KubeVirt
- **virt-api**: Provides the KubeVirt API server
- **virt-controller**: Orchestrates virtual machine instances
- **virt-handler**: Manages VMs on each node
- **virt-exportproxy**: Handles VM export operations
- **virt-synchronization-controller**: Synchronizes VM state
- **virt-template-apiserver**: Provides the VM template API server
- **virt-template-controller**: Manages VM template lifecycle
- **virt-launcher**: Runs VM workloads (created per VM)

## Cluster Services Access Configuration

### The `np.kubevirt.io/allow-access-cluster-services` Label

KubeVirt components that require access to the Kubernetes API server and DNS are automatically labeled with:

```yaml
np.kubevirt.io/allow-access-cluster-services: "true"
```

This label is applied to the following components:

- virt-operator
- virt-api
- virt-handler
- virt-controller
- virt-exportproxy
- virt-synchronization-controller
- virt-template-apiserver
- virt-template-controller
- Installer strategy job pods

### Creating Cluster Services Network Policy

As a cluster administrator, you **must** create a network policy that allows pods with the above label to access your
cluster's API server and DNS services.

**Important**: This network policy must be applied **before** installing KubeVirt, otherwise virt-operator will not be
able to communicate with the API server and DNS.

#### Example Network Policy

Here's an example for a standard Kubernetes cluster (customize based on your environment):

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: kv-allow-egress-to-api-server
  namespace: kubevirt
spec:
  podSelector:
    matchExpressions:
      - key: np.kubevirt.io/allow-access-cluster-services
        operator: In
        values:
          - "true"
  policyTypes:
    - Egress
  egress:
    # Allow access to Kubernetes API server
    - ports:
        - protocol: TCP
          port: 6443
    # Allow access to DNS
    - to:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: kube-system
          podSelector:
            matchLabels:
              k8s-app: kube-dns
      ports:
        - protocol: TCP
          port: dns-tcp
        - protocol: UDP
          port: dns
```

#### Customization Guidelines

You may need to customize this policy based on your cluster configuration:

1. **API Server Port**: The default is TCP/6443, but your cluster may use a different port
2. **DNS Namespace**: The default is `kube-system`, but some clusters use different namespaces
3. **DNS Pod Labels**: The default label is `k8s-app: kube-dns`, but this may vary (e.g., CoreDNS might use different
   labels)
4. **DNS Ports**: Adjust if your cluster uses non-standard DNS ports

## Inter-Component Network Policies

### Generating Network Policies

KubeVirt provides a set of network policies for inter-component communication. These can be generated using the
`csv-generator` tool with the `--dump-network-policies` flag:

```bash
./tools/csv-generator/csv-generator --dump-network-policies
```

The generated policies are available in:

```
manifests/generated/kubevirt-network-policies.yaml.in
```

For releases, the policies are also available at:

```
manifests/release/kubevirt-network-policies.yaml.in
```

### Network Policy Details

The following network policies are generated:

#### 1. kubevirt-allow-ingress-to-metrics

Allows ingress to metrics endpoints (port 8443) for monitoring purposes.

**Direction**: Inbound  
**Applies to**: virt-operator, virt-handler, virt-controller, virt-api, virt-exportproxy,
virt-synchronization-controller

#### 2. kubevirt-allow-ingress-to-virt-api-webhook-server

Allows ingress to the virt-api webhook server for admission control.

**Direction**: Inbound  
**Applies to**: virt-api  
**Port**: TCP/8443

#### 3. kubevirt-allow-virt-api-to-components

Allows virt-api to communicate with other KubeVirt components.

**Direction**: Outbound  
**From**: virt-api  
**To**: virt-operator, virt-handler, virt-controller, virt-api  
**Port**: TCP/8443

#### 4. kubevirt-allow-virt-api-to-launchers

Allows virt-api to communicate with virt-launcher pods across all namespaces.

**Direction**: Outbound  
**From**: virt-api  
**To**: virt-launcher (all namespaces)

#### 5. kubevirt-allow-virt-api-to-virt-handler

Allows virt-api to communicate with virt-handler.

**Direction**: Outbound  
**From**: virt-api  
**To**: virt-handler  
**Port**: TCP

#### 6. kubevirt-allow-ingress-to-virt-handler

Allows virt-api to send requests to virt-handler.

**Direction**: Inbound  
**From**: virt-api  
**To**: virt-handler  
**Port**: TCP

#### 7. kubevirt-allow-ingress-to-virt-operator-webhook-server

Allows ingress to the virt-operator webhook server.

**Direction**: Inbound  
**Applies to**: virt-operator  
**Port**: TCP/8444

#### 8. kubevirt-allow-virt-exportproxy-communications

Allows virt-exportproxy to communicate with export service pods.

**Direction**: Both  
**From**: virt-exportproxy  
**To**: Pods with label `kubevirt.io.virt-export-service`  
**Port**: TCP/8443 (both ingress and egress)

#### 9. kubevirt-allow-handler-to-handler

Allows virt-handler pods to communicate with each other for VM migration and other operations.

**Direction**: Both  
**From**: virt-handler  
**To**: virt-handler

#### 10. kubevirt-allow-handler-to-prometheus

Allows virt-handler to push metrics to Prometheus.

**Direction**: Outbound  
**From**: virt-handler  
**Port**: TCP/8443

### Virt-Template Network Policies

The following network policies are generated for the virt-template subsystem. They are included in the same generated
manifest alongside the core KubeVirt policies.

#### 11. virt-template-allow-apiserver-ingress

Allows ingress to the virt-template API server.

**Direction**: Inbound  
**Applies to**: virt-template-apiserver (`control-plane: apiserver`)  
**Port**: TCP/9443

#### 12. virt-template-allow-metrics-traffic

Allows ingress to the virt-template controller for metrics scraping.

**Direction**: Inbound  
**Applies to**: virt-template-controller (`control-plane: controller-manager`)  
**Port**: TCP/8443

#### 13. virt-template-allow-webhook-traffic

Allows ingress to the virt-template controller for webhook traffic.

**Direction**: Inbound  
**Applies to**: virt-template-controller (`control-plane: controller-manager`)  
**Port**: TCP/9443

## Deployment Guide

### Step 1: Apply Default Deny Policy (Optional)

If you want strict network isolation, apply a default-deny policy:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny
  namespace: kubevirt
spec:
  podSelector: { }
  policyTypes:
    - Ingress
    - Egress
```

### Step 2: Apply Cluster Services Network Policy

Before installing KubeVirt, create and apply the cluster services network policy (see example above):

```bash
kubectl apply -f cluster-services-np.yaml
```

### Step 3: Apply Inter-Component Network Policies

Apply the KubeVirt-generated network policies:

```bash
kubectl apply -f manifests/release/kubevirt-network-policies.yaml
```

### Step 4: Install KubeVirt

Proceed with the standard KubeVirt installation:

```bash
kubectl apply -f https://github.com/kubevirt/kubevirt/releases/download/v<version>/kubevirt-operator.yaml
kubectl apply -f https://github.com/kubevirt/kubevirt/releases/download/v<version>/kubevirt-cr.yaml
```

## Troubleshooting

### virt-operator fails to start

**Symptom**: virt-operator pod is running but cannot communicate with the API server.

**Solution**: Ensure the cluster services network policy is applied before installing KubeVirt, and verify that it
matches your cluster's API server and DNS configuration.

### Components cannot communicate

**Symptom**: KubeVirt components fail to communicate with each other.

**Solution**: Verify that the inter-component network policies are applied correctly:

```bash
kubectl get networkpolicies -n kubevirt
```

### DNS resolution fails

**Symptom**: Components cannot resolve DNS names.

**Solution**: Check that your cluster services network policy correctly identifies your DNS pods. Verify the namespace,
labels, and ports used by your DNS service.

## Development and Testing

### Local Testing

For local testing with a default-deny policy, you can use the example in `hack/cluster-services-np.yaml`:

```bash
kubectl apply -f hack/cluster-services-np.yaml
```

### Generating Updated Policies

If you modify the network policy generation code in `pkg/virt-operator/resource/generate/components/networkpolicy.go`,
regenerate the manifests:

```bash
make generate
```

This will update `manifests/generated/kubevirt-network-policies.yaml.in`.

## Reference

- Source code: `pkg/virt-operator/resource/generate/components/networkpolicy.go`
- Generated manifests: `manifests/generated/kubevirt-network-policies.yaml.in`
- Example cluster services policy: `hack/cluster-services-np.yaml`
- Label constant: `staging/src/kubevirt.io/api/core/v1/types.go:1470`
- PR: [#15195](https://github.com/kubevirt/kubevirt/pull/15195)

## Security Considerations

1. **Defense in Depth**: Network policies provide an additional security layer but should be used alongside other
   security measures (RBAC, Pod Security Standards, etc.)

2. **Namespace Isolation**: The inter-component policies are scoped to the KubeVirt namespace, but virt-launcher
   policies allow cross-namespace communication (required for VM workloads)

3. **Monitoring**: Ensure your monitoring solution can reach the metrics endpoints (port 8443) on KubeVirt components

4. **Updates**: When upgrading KubeVirt, review the network policies to ensure they match the new version's
   communication requirements
