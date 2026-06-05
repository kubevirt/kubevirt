# KubeVirt Registry Disk Base Container

The KubeVirt Registry Disk Base Container allows users to store VMI disks in
a container registry and attach those disk to VMIs automatically using the
KubeVirt runtime.

This Base Container is compatible with disk type ContainerDisk:v1alpha

# Storing Disks in Container Registry

VMI disks can be stored in either qcow2 format or raw format by copying the vm
disk into a container image and uploading that container image to a container
registry.

Example: Place a bootable VMI disk into a container image in the /disk directory
and upload to the container registry.
```
cat << END > Dockerfile
FROM scratch
ADD fedora25.qcow2 /disk/
END

docker build -t vmdisks/fedora25:latest .
docker push vmdisks/fedora25:latest
```

# Assigning Ephemeral Disks to VMIs

Assign an ephemeral disk backed by an image in the container registry by
adding a ContainerDisk:v2alpha disk to the VMI definition and supplying
the container image as the disk's source name.

Example: Create a KubeVirt VMI definition with container backed ephemeral disk.

```
cat << END > vm.yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  creationTimestamp: null
  name: vm-ephemeral
spec:
  domain:
    devices:
      disks:
      - disk:
          bus: virtio
        name: containerdisk
        volumeName: registryvolume
    machine:
      type: ""
    resources:
      requests:
        memory: 64M
  terminationGracePeriodSeconds: 0
  volumes:
  - name: registryvolume
    containerDisk:
      image: kubevirt/cirros-container-disk-demo:devel
status: {}
END
```

After creating the VMI definition, starting the VMI is as simple starting a pod.

```
kubectl create -f vm.yaml
```
