# Provide temporary empty disks

## Motivation

KubeVirt in combination with use-cases like the cluster-autoscaler can be used
to scale and provision vm pools. With the VirtualMachineInstanceReplicaSet and
cloud-init stateless pools can be instantiated and provisioned.
Many of the used cloud images for these use-cases are booted with small disks
with no additional free space, and/or run in read-only mode. A common pattern
to store temporary data which should survive restarts or temporary data which
is too big to fit in memory on temporary disks. The `emptyDir` in k8s provides
a similar functionality.

To provide a similar functionality for our `VirtualMachineInstances` an `emptyDisk`
volume type will be introduced.

## Implementation

A new volume type `emptyDisk` is introduced. The only parameter it supports is
`capacity`. The `emptyDisk` will be placed by `virt-launcher` in an `emptyDir`,
to make sure that the kubelet can take care of clean-up and that
`ephemeral-store` resource definitions apply to the `emptyDisks` too.

The definition looks like this:


```golang
// EmptyDisk represents a temporary disk which shares the vms lifecycle
type EmptyDiskSource struct {
	// Capacity of the sparse disk.
	Capacity resource.Quantity `json:"capacity"`
}
```

A usage example:

```yaml
apiVersion: kubevirt.io/v1alpha2
kind: VirtualMachineInstance
metadata:
  name: testvm
spec:
  terminationGracePeriodSeconds: 0
  domain:
    resources:
      requests:
        memory: 512M
    devices:
      disks:
      - name: disk1
        volumeName: containerDisk
        disk:
          dev: vda
      - name: disk2
        volumeName: emptyDisk
        disk:
          dev: vdb
  volumes:
    - name: containerDisk
      containerDisk:
        image: kubevirt/cirros-container-disk-demo:devel
    - name: emptyDisk
      emptyDisk:
        capacity: 20G
```

This example will boot cirros (which traditionally boots read-only) and attach
a 20 GB temporary disk which can be used to write temporary data.

How and if this will be tied into the pods
`resources.requests.ephemeral-storage` or `resources.limits.ephemeral-storage`
is not yet clear and not part of this implementation.
