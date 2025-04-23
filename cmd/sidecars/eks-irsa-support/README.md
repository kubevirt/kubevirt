# EKS IRSA Support Sidecar

This sidecar container provides support for AWS IAM Roles for Service Accounts (IRSA) in KubeVirt virtual machines running on Amazon EKS.

## Overview

The sidecar implements two hooks that work together to enable IRSA support in VMs:

1. `preCloudInitIso` - Injects AWS credentials into the VM's cloud-init configuration
2. `onDefineDomain` - Sets up a virtiofs device to mount the AWS token

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

### Enabling the Sidecar

To enable the EKS IRSA support sidecar for a VM, add the following annotations to your VirtualMachine or VirtualMachineInstance:

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  name: my-vm
  annotations:
    hooks.kubevirt.io/hookSidecars: '[{"image": "kubevirt/eks-irsa-support:latest"}]'
spec:
  template:
    spec:
      domain:
        devices:
          filesystems:
            - name: irsa-token
              virtiofs: {}
        volumes:
          - name: irsa-token
            filesystem:
              name: irsa-token
```

### Complete Example

Here's a complete example of a VM with IRSA support:

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  name: eks-irsa-vm
  annotations:
    hooks.kubevirt.io/hookSidecars: '[{"image": "kubevirt/eks-irsa-support:latest"}]'
spec:
  running: true
  template:
    metadata:
      labels:
        kubevirt.io/domain: eks-irsa-vm
    spec:
      domain:
        devices:
          filesystems:
            - name: irsa-token
              virtiofs: {}
        volumes:
          - name: irsa-token
            filesystem:
              name: irsa-token
        resources:
          requests:
            memory: 1Gi
      volumes:
        - name: containerdisk
          containerDisk:
            image: kubevirt/fedora-cloud-container-disk-demo:latest
        - name: cloudinitdisk
          cloudInitNoCloud:
            userData: |
              #cloud-config
              password: fedora
              chpasswd: { expire: False }
```

### Environment Variables

The sidecar automatically collects and injects all AWS-related environment variables (those starting with `AWS_`) from the host into the VM.

### Linux VMs

For Linux VMs, the sidecar:
1. Adds AWS environment variables to `/etc/environment`
2. Sets up a virtiofs mount for the AWS token at `/irsa-token`

### Windows VMs

For Windows VMs, the sidecar:
1. Creates PowerShell scripts to set AWS environment variables
2. Sets up a virtiofs mount for the AWS token

## Requirements

- KubeVirt running on Amazon EKS
- IRSA configured for the cluster
- Appropriate IAM roles and policies

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

3. **Cloud-init failures**
   - Check cloud-init logs in the VM
   - Verify the cloud-init configuration
   - Ensure proper OS detection

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Submit a pull request

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details. 