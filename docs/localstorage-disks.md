# Local Storage Placement for VM Disks


## Introduction

Virtual Machines are able to have a DataVolume disks that are based on Local Storage PVs. Local Storage PVs are bound to a specific node.
It might happen to have an `Unschedulable` `VMI`, when the PVC is bound to a PV on a different node than virt-launcher pod. 

When the VM with a DataVolumeTemplate is defined a DataVolume is created from the template and the `CDI` creates a worker Pod to import/upload/clone data to the PVC (specified in a template).
To run a VMI kubevirt creates a virtlauncher pod with all the VMI requirements. Kubernetes uses the virtlauncher pod requirements to schedule it on a specific node.
Worker Pod might have different constraints than a kubevirt VM. When the VM is scheduled on a different node than the PVC it becomes unusable. 
This is especially problematic when using a VM with DataVolumeTemplate with many disks managed by CDI. 

This document describes a special handling of `DataVolumes` in the `WaitForFirstConsumer` state (available from [CDI v1.21.0](https://github.com/kubevirt/containerized-data-importer/releases/tag/v1.21.0)).


## Use-case

When the `Virtual Machine` has a `DataVolume` disk (or disks) then bind Local Storage `PVC` to a `PV` on the same `node` where the `VMI` is going to be scheduled. 

## Design Overview

The solution is to leverage Kubernetes pod scheduler to bind the PVC to a PV on a correct node.
By using a StorageClass with `volumeBindingMode` set to `WaitForFirstConsumer` the binding and provisioning of PV is delayed until a Pod using the PersistentVolumeClaim is created. 
Kubevirt can schedule a special ephemeral pod that becomes a first consumer of the PersistentVolumeClaim.
Its only purpose is to be scheduled to a node capable of running VM and by using PVCs to trigger kubernetes to provision and bind PV's on the same node.
After PVC are bound the `CDI` can do its work and Kubevirt can start the actual VM. 
 
 
## Implementation

### Flow

1. A StorageClass with volumeBindingMode=WaitForFirstConsumer is created
2. User creates the VM with DataVolumeTemplate containing 
3. `Kubevirt` creates DataVolume
4. The `CDI` sees that new DV has unbound PVC with storage class with volumeBindingMode=WaitForFirstConsumer, sets the phase of DV to `WaitForFirstConsumer` and waits for PVC to be bound by some external action. 
5. `Kubevirt` sees the DV in phase `WaitForFirstConsumer`, so it creates an ephemeral pod (basically a virtlauncher pod
without a VM payload and with `kubevirt.io/ephemeral-provisioning` annotation) only used to force PV provisioning 
6. Kubernetes schedules the ephemeral pod, (the node selected meets all the VM requirements), pod requires 
 the same PVC as the VM so kubenertes has to provision and bind the PV to PVC on a correct node before the pod can be started
7. `CDI` sees that PVC is Bound, changes DV status to "ImportScheduled" (or clone/upload), and tries to start worker pods
8. `Kubevirt` sees DV status is `ImportScheduled`, it can terminate the ephemeral provisioning pod
8. `CDI` does the Import, marks DV as `Succeeded`
9. `Kubevirt` creates the virtlauncher pod to start a VM 

This flow differs from standard scenario (import/upload/clone on storage with Immediate binding) by steps 4, 5, 6 and 8. 

Note: 
`WaitForFirstConsumer` state for DataVolumes is available in CDI from v1.21.0 and toggled by a `HonorWaitForFirstConsumer` feature gate. 
When the `HonorWaitForFirstConsumer` feature gate is enabled, the `CDI` is not starting any worker pods when the PVCs StorageClass binding mode is `WaitForFirstConsumer`. In such case the `CDI` puts the DataVolume in a new state `WaitForFirstConsumer`.
More in CDI docs [here](https://github.com/kubevirt/containerized-data-importer/blob/master/doc/waitforfirstconsumer-storage-handling.md).

