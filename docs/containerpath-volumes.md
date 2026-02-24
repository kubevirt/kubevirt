# ContainerPath Volumes

## Overview

ContainerPath volumes allow VirtualMachines (VMs) to access files and directories from the virt-launcher pod's filesystem via virtiofs. This enables VMs to consume data that is dynamically injected into the pod by Kubernetes or platform-specific mechanisms.

## Motivation

Kubernetes and cloud platforms provide mechanisms for injecting configuration and credentials into pods at runtime:
- Projected service account tokens for workload identity (e.g., AWS IRSA, GCP Workload Identity)
- Dynamic secrets and configuration via admission webhooks (e.g., Vault agent)

These mechanisms inject data directly into pod containers, but VMs running inside those pods cannot natively access this injected data. ContainerPath volumes bridge this gap by exposing specific paths from the virt-launcher container to the guest VM via virtiofs.

## Primary Use Case: AWS IRSA

A common use case is enabling VMs to authenticate to AWS using IAM Roles for Service Accounts (IRSA).

When a pod uses a ServiceAccount with an IRSA annotation, EKS automatically injects a projected token at `/var/run/secrets/eks.amazonaws.com/serviceaccount/`. The VM can access this token via a ContainerPath volume.

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: aws-workload-sa
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/my-pod-role
---
apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  name: aws-workload
spec:
  runStrategy: Always
  template:
    spec:
      domain:
        devices:
          filesystems:
          - name: sa-volume
            virtiofs: {}
          - name: aws-token
            virtiofs: {}
          disks:
          - name: containerdisk
            disk:
              bus: virtio
        resources:
          requests:
            memory: 2Gi
      volumes:
      - name: containerdisk
        containerDisk:
          image: quay.io/containerdisks/fedora:latest
      # Required: causes the pod to use the IRSA-annotated ServiceAccount
      - name: sa-volume
        serviceAccount:
          serviceAccountName: aws-workload-sa
      # Exposes the IRSA-injected token path to the VM
      - name: aws-token
        containerPath:
          path: /var/run/secrets/eks.amazonaws.com/serviceaccount
```

**Why both volumes?**
- `sa-volume`: The serviceAccount volume ensures the virt-launcher pod uses the specified ServiceAccount, which triggers EKS to inject the IRSA token.
- `aws-token`: The containerPath volume exposes the EKS-injected token path to the VM via virtiofs.

Inside the VM:
```bash
# Mount the IRSA token filesystem
mount -t virtiofs aws-token /mnt/aws-creds

# Configure AWS SDK to use web identity token
export AWS_WEB_IDENTITY_TOKEN_FILE=/mnt/aws-creds/token
export AWS_ROLE_ARN=arn:aws:iam::123456789012:role/my-pod-role

# AWS CLI and SDKs will use these credentials
aws s3 ls
```

## Requirements and Limitations

### Feature Gate

ContainerPath volumes require the `ContainerPathVolumes` feature gate:

```yaml
apiVersion: kubevirt.io/v1
kind: KubeVirt
metadata:
  name: kubevirt
  namespace: kubevirt
spec:
  configuration:
    developerConfiguration:
      featureGates:
      - ContainerPathVolumes
```

### Path Requirements

- The specified `path` must exist within the virt-launcher pod's `compute` container
- The path must be populated before the VM starts, or populated by a sidecar that runs continuously
- Paths are read-only from the VM's perspective
- The path must not conflict with KubeVirt-internal mount points

### Supported Volume Types

ContainerPath only supports paths backed by the following Kubernetes volume types:

- **ConfigMap** - Configuration data
- **Secret** - Sensitive data like credentials
- **Projected** - Combinations of ConfigMaps, Secrets, DownwardAPI, and ServiceAccountToken
- **DownwardAPI** - Pod and container metadata
- **EmptyDir** - Ephemeral pod-local storage

Other volume types (PVC, HostPath, etc.) are not supported.

### Live Migration

ContainerPath volumes do not block live migration, but whether the data remains accessible after migration depends on how the path is populated:

**Generally works with migration:**
- **Secrets, ConfigMaps, ServiceAccount tokens** - Kubernetes re-projects these on the target node
- **Sidecar-populated volumes** - If a sidecar container populates the path (e.g., Vault agent), the sidecar runs on the target node and repopulates the data

**Does not work with migration:**
- **EmptyDir volumes without sidecars** - Data is not copied to the target node
- **ReadWriteOnce (RWO) PVCs** - Cannot be attached to multiple nodes simultaneously

When using ContainerPath volumes with live migration, verify that the mechanism populating your container path will function correctly on the target node. Test migration in your environment before relying on it in production.

### Security Considerations

- VMs gain access to any files within the specified container path
- Only expose paths containing data intended for VM consumption
- Use RBAC and admission policies to control which service accounts and roles can be used with VMs
- ContainerPath volumes inherit the security context of the virt-launcher pod
- Only supported volume types are allowed (see [Supported Volume Types](#supported-volume-types))

## Additional Use Cases

### Webhook-Injected Volumes

Admission webhooks can inject volumes into virt-launcher pods. Use ContainerPath to expose these to VMs (partial example showing only the relevant volume):

```yaml
volumes:
- name: injected-config
  containerPath:
    path: /opt/injected-config
```

A corresponding filesystem device is also required in `spec.domain.devices.filesystems`.

### Kubernetes Service Account Tokens

Access the pod's default service account token (partial example):

```yaml
volumes:
- name: kube-api-token
  containerPath:
    path: /var/run/secrets/kubernetes.io/serviceaccount
```

A corresponding filesystem device is also required in `spec.domain.devices.filesystems`.

## Troubleshooting

### VM Stuck with Synchronized=False

If a VM has `Synchronized=False` with reason `MissingVirtiofsContainers`, the specified path does not exist in the virt-launcher pod.

Check the path in the compute container:
```bash
kubectl exec -n <namespace> <virt-launcher-pod> -c compute -- ls -la /path/to/volume
```

Common issues:
- Path typo in VM spec
- Volume not injected by expected mechanism (check pod spec)
- Timing issue: path populated after virtiofs initialization
- Feature gate `ContainerPathVolumes` not enabled

## Implementation Details

ContainerPath volumes use virtiofs to expose container paths to VMs. When a ContainerPath volume is defined:

1. virt-controller validates that the `ContainerPathVolumes` feature gate is enabled
2. virt-controller skips generating a virtiofs sidecar container (the path already exists in the compute container)
3. virt-launcher verifies the path exists before starting the VM
4. The VM mounts the filesystem: `mount -t virtiofs <name> <mount-point>`
