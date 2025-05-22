# EKS IRSA Support Sidecar

This sidecar container provides support for AWS IAM Roles for Service Accounts (IRSA) in KubeVirt virtual machines running on Amazon EKS.

## Overview

The sidecar implements two hooks that work together to enable IRSA support in VMs:

1. `preCloudInitIso` - Injects AWS credentials into the VM's cloud-init configuration
2. `onDefineDomain` - Sets up a virtiofs device to mount the AWS token

## Important: Required Webhook

To make this sidecar work correctly, you must also deploy the [IRSA Mutation Webhook](https://github.com/kubevirt/irsa-mutation-webhook) in your cluster. The complete IRSA solution consists of two components:

1. **IRSA Mutation Webhook** - Injects the virtio-fs container that shares the AWS IAM token from the host with the VM
2. **EKS IRSA Support Sidecar** - Configures the VM to use the shared token

The mutation webhook must be deployed on your cluster first to enable automatic injection of the virtio-fs container that shares the AWS IAM token with your VMs.

## AWS IAM Token Volume

This sidecar uses the `onDefineDomain` hook to add a virtiofs filesystem to the VM's domain XML that mounts the AWS IAM token. The token is automatically projected by the IRSA mutation webhook, which injects a container with a volume mounted as follows:

```yaml
volumes:
- name: aws-iam-token
  projected:
    defaultMode: 420
    sources:
    - serviceAccountToken:
        audience: sts.amazonaws.com
        expirationSeconds: 86400
        path: token
```

This projected volume contains the AWS IAM token needed for authentication with AWS services. The sidecar ensures this token is accessible within the VM through a virtiofs mount.

The complete flow works as follows:
1. IRSA creates the projected volume with the AWS token in the pod
2. The mutation webhook injects a virtiofs container that makes this token accessible via a socket
3. This sidecar adds the virtiofs filesystem to the VM domain XML
4. The VM can then mount and use this token for AWS authentication

## Features

- Supports both Linux and Windows VMs
- Automatically detects VM OS type
- Injects AWS environment variables into the VM
- Provides secure token access via virtiofs
- Handles both cloud-init and domain XML modifications

## Usage

The sidecar is automatically invoked by KubeVirt when:
- Creating a cloud-init ISO for a VM
- Defining the domain XML for a VM

### Prerequisites

1. Deploy the [IRSA Mutation Webhook](https://github.com/kubevirt/irsa-mutation-webhook) to your cluster:
   ```bash
   git clone https://github.com/kubevirt/irsa-mutation-webhook.git
   cd irsa-mutation-webhook
   make deploy
   ```

### Enabling the Sidecar

To enable the EKS IRSA support sidecar for a VM, add the following annotations to your VirtualMachine or VirtualMachineInstance:

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  annotations:
    hooks.kubevirt.io/hookSidecars: |
      [
        {
          "args": ["--version", "v1alpha3"],
          "image": "quay.io/kubevirt/eks-irsa-sidecar"
        }
      ]
  name: example-vmi
spec:
  domain:
    devices:
      filesystems:
        - name: serviceaccount-fs
          virtiofs: {}
      disks:
        - disk:
            bus: virtio
          name: containerdisk
    machine:
      type: ""
    resources:
      requests:
        memory: 1024M
  volumes:
    - name: containerdisk
      containerDisk:
        image: quay.io/containerdisks/fedora:latest
    - cloudInitNoCloud:
        userData: |-
          #cloud-config
          chpasswd:
            expire: false
          password: fedora
          user: fedora
          bootcmd:
            # mount the ConfigMap
            - "sudo mkdir -p /mnt/serviceaccount"
            - "sudo mkdir -p /mnt/aws-iam-token"
            - "sudo mount -t virtiofs serviceaccount-fs /mnt/serviceaccount"
            - "sudo mount -t virtiofs aws-iam-token /mnt/aws-iam-token"
      name: cloudinitdisk
    - name: serviceaccount-fs
      serviceAccount:
        serviceAccountName: example-sa
```

### Environment Variables

The sidecar automatically collects and injects all AWS-related environment variables (those starting with `AWS_`) from the host into the VM.

### Linux VMs

For Linux VMs, the sidecar:
1. Adds AWS environment variables to `/etc/environment`
2. Sets up a virtiofs mount for the AWS token at `/aws-iam-token`

### Windows VMs

For Windows VMs, the sidecar:
1. Creates PowerShell scripts to set AWS environment variables
2. Sets up a virtiofs mount for the AWS token

## Requirements

- KubeVirt running on Amazon EKS
- IRSA configured for the cluster
- Appropriate IAM roles and policies
- [IRSA Mutation Webhook](https://github.com/kubevirt/irsa-mutation-webhook) installed in the cluster

## Configuration

No additional configuration is required. The sidecar automatically:
- Detects the VM's operating system
- Collects AWS environment variables
- Configures the appropriate cloud-init or domain XML

## Security

- AWS credentials are injected securely via cloud-init
- Token access is provided through a secure virtiofs mount
- Environment variables are set at the system level

## Troubleshooting

Common issues and solutions:

1. **Missing AWS credentials**
   - Verify IRSA is properly configured
   - Check IAM role permissions
   - Ensure environment variables are present

2. **Mount issues**
   - Verify virtiofs is supported by the VM
   - Check for proper permissions
   - Ensure the token socket exists
   - Confirm the IRSA Mutation Webhook is properly installed

3. **Cloud-init failures**
   - Check cloud-init logs in the VM
   - Verify the cloud-init configuration
   - Ensure proper OS detection
