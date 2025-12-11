# Sidecar HostPath Support

## Overview

This feature extends KubeVirt sidecars to support mounting hostPath volumes. This is useful when sidecars need access to files on the host filesystem, such as backing images for qemu-img operations.

## Use Case

A common use case is when a sidecar needs to run `qemu-img` to create a copy-on-write (COW) overlay file with a backing image that exists on the host filesystem. Without hostPath support, the sidecar cannot access these backing images.

## Configuration

To use hostPath volumes in a sidecar, add a `hostPath` field to the sidecar configuration in the `hooks.kubevirt.io/hookSidecars` annotation:

```yaml
annotations:
  hooks.kubevirt.io/hookSidecars: '[
    {
      "image": "my-sidecar-image:latest",
      "args": ["--version", "v1alpha2"],
      "hostPath": {
        "path": "/var/lib/kubevirt/images",
        "volumePath": "/host-images",
        "type": "Directory"
      }
    }
  ]'
```

### HostPath Fields

- **path** (required): The path on the host filesystem to mount
- **volumePath** (required): The path inside the sidecar container where the host path will be mounted
- **type** (optional): The type of hostPath volume. Defaults to "Directory". Valid values:
  - `Directory` - A directory must exist at the given path
  - `DirectoryOrCreate` - If nothing exists at the given path, an empty directory will be created
  - `File` - A file must exist at the given path
  - `FileOrCreate` - If nothing exists at the given path, an empty file will be created
  - `Socket` - A UNIX socket must exist at the given path
  - `CharDevice` - A character device must exist at the given path
  - `BlockDevice` - A block device must exist at the given path

## Example: QEMU COW Image Creation

Here's an example of a sidecar that creates a COW overlay image with a backing file from the host:

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  name: vmi-with-cow-overlay
  annotations:
    hooks.kubevirt.io/hookSidecars: '[
      {
        "image": "my-qemu-img-sidecar:latest",
        "imagePullPolicy": "IfNotPresent",
        "args": ["--version", "v1alpha2"],
        "command": ["/usr/bin/create-cow.sh"],
        "hostPath": {
          "path": "/var/lib/kubevirt/images",
          "volumePath": "/host-images",
          "type": "Directory"
        }
      }
    ]'
spec:
  domain:
    devices:
      disks:
      - disk:
          bus: virtio
        name: rootdisk
    resources:
      requests:
        memory: 2048M
  volumes:
  - name: rootdisk
    containerDisk:
      image: quay.io/kubevirt/fedora-cloud-container-disk-demo:latest
```

The sidecar script (`create-cow.sh`) could then access backing images:

```bash
#!/bin/bash
# Inside the sidecar container
qemu-img create -f qcow2 -b /host-images/backing-image.qcow2 -F qcow2 /output/overlay.qcow2
```

## Security Considerations

- HostPath volumes provide access to the host filesystem, which can be a security risk
- Ensure that the paths mounted are necessary and appropriate for your use case
- Consider using read-only mounts when write access is not required
- Be aware that hostPath volumes bypass Kubernetes storage abstractions and can make workloads less portable

## See Also

- [Sidecar Hook Documentation](../cmd/sidecars/README.md)
- [Example VMI with HostPath Sidecar](vmi-with-sidecar-hostpath.yaml)

