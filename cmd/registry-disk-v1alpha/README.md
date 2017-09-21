# KubeVirt Registry Disk Base Container

The KubeVirt Registry Disk Base Container allows users to store VM disks in
a container registry and attach those disk to VMs automatically using the
KubeVirt runtime.

This Base Container is compatible with disk type RegistryDisk:v1alpha

# RegistryDisk:v1alpha
## Storing Disks in Container Registry

VM disks can be stored in either qcow2 format or raw format by copying the vm
disk into a container image and uploading that container image to a container
registry.

Example: Place a bootable VM disk into a container image in the /disk directory
and upload to the container registry.
```
cat << END > Dockerfile 
FROM kubevirt.io:registry-disk-v1alpha
ADD fedora25.qcow2 /disk
END

docker build -t vmdisks/fedora25:latest .
docker push vmdisks/fedora25:latest
```

## Assigning Ephemeral Disks to VMs

Assign an ephemeral disk backed by an image in the container registry by
adding a RegistryDisk:v1alpha disk to the VM definition and supplying
the container image as the disk's source name.

Example: Create a KubeVirt VM definition with container backed ephemeral disk.

```
cat << END > vm.yaml
metadata:
  name: testvm-ephemeral
apiVersion: kubevirt.io/v1alpha1
kind: VirtualMachine
spec:
  domain:
    devices:
      disks:
      - type: RegistryDisk:v1alpha
        source:
          name: vmdisks/fedora25:devel
        target:
          dev: vda
      interfaces:
      - source:
          network: default
        type: network
    memory:
      unit: MB
      value: 64
    os:
      type:
        os: hvm
    type: qemu
END
```

After creating the VM definition, starting the VM is as simple starting a pod.
 
```
kubectl create -f vm.yaml
```

