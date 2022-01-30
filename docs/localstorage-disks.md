# Local Storage Placement for VM Disks

This document describes a special handling of `DataVolumes` in the `WaitForFirstConsumer` state. 
`WaitForFirstConsumer` state is available from [CDI v1.21.0](https://github.com/kubevirt/containerized-data-importer/releases/tag/v1.21.0), and the logic to handle this is available from [KubeVirt v0.36.0](https://github.com/kubevirt/kubevirt/releases/tag/v0.36.0)

## Use-case

When the `Virtual Machine` has a `DataVolume` disk (or disks) then bind Local Storage `PVC` to a `PV` on the same `node` where the `VMI` is going to be scheduled.

## The problem

Virtual Machines are able to have a DataVolume disks that are based on Local Storage PVs. Local Storage PVs are bound to a specific node.
Since DataVolumes involve preparing storage with an image before being consumed by the VMI, 
it's possible to result in an Unschedulable VMI in the event that a VMI can not be scheduled to the node the local storage PV was previously pinned to. 

When the VM with a DataVolumeTemplate is defined a DataVolume is created from the template and the `CDI` creates a worker Pod to import/upload/clone data to the PVC (specified in a template).
To run a VMI kubevirt creates a virtlauncher pod with all the VMI requirements. Kubernetes uses the virtlauncher pod requirements to schedule it on a specific node.
Worker Pod might have different constraints than a kubevirt VM. When the VM is scheduled on a different node than the PVC it becomes unusable. 
This is especially problematic when using a VM with DataVolumeTemplate with many disks managed by CDI. 

## The solution

The solution is to leverage Kubernetes pod scheduler to bind the PVC to a PV on a correct node.
By using a StorageClass with `volumeBindingMode` set to `WaitForFirstConsumer` the binding and provisioning of PV is delayed until a Pod using the PersistentVolumeClaim is created. 
KubeVirt can schedule a special ephemeral pod that becomes a first consumer of the PersistentVolumeClaim.
Its only purpose is to be scheduled to a node capable of running VM and by using PVCs to trigger kubernetes to provision and bind PV's on the same node.
After PVC are bound the `CDI` can do its work and KubeVirt can start the actual VM. 
  
## Implementation

### Flow

1. A StorageClass with volumeBindingMode=WaitForFirstConsumer is created
2. User creates the VM with DataVolumeTemplate containing 
3. `KubeVirt` creates DataVolume
4. The `CDI` sees that new DV has unbound PVC with storage class with volumeBindingMode=WaitForFirstConsumer, sets the phase of DV to `WaitForFirstConsumer` and waits for PVC to be bound by some external action. 
5. `KubeVirt` sees the DV in phase `WaitForFirstConsumer`, so it creates an ephemeral pod (basically a virtlauncher pod
without a VM payload and with `kubevirt.io/ephemeral-provisioning` annotation) only used to force PV provisioning 
6. Kubernetes schedules the ephemeral pod, (the node selected meets all the VM requirements), pod requires 
 the same PVC as the VM so kubenertes has to provision and bind the PV to PVC on a correct node before the pod can be started
7. `CDI` sees that PVC is Bound, changes DV status to "ImportScheduled" (or clone/upload), and tries to start worker pods
8. `KubeVirt` sees DV status is `ImportScheduled`, it can terminate the ephemeral provisioning pod
8. `CDI` does the Import, marks DV as `Succeeded`
9. `KubeVirt` creates the virtlauncher pod to start a VM 

This flow differs from standard scenario (import/upload/clone on storage with Immediate binding) by steps 4, 5, 6 and 8. 

Note: 
`WaitForFirstConsumer` state for DataVolumes is available in CDI from v1.21.0 and toggled by a `HonorWaitForFirstConsumer` feature gate. 
When the `HonorWaitForFirstConsumer` feature gate is enabled, the `CDI` is not starting any worker pods when the PVCs StorageClass binding mode is `WaitForFirstConsumer`. In such case the `CDI` puts the DataVolume in a new state `WaitForFirstConsumer`.
More in CDI docs [here](https://github.com/kubevirt/containerized-data-importer/blob/main/doc/waitforfirstconsumer-storage-handling.md).

## Interaction with virtctl

This change solves the problem of node placement for the `VMs`. And everything works automatically when using VM with DV Template, 
but it makes it harder to use virtctl image-upload to upload images to cloud storage.

Right now it is not supported by the virtctl and the user has to do additional steps to upload an image to the storage with `WaitForFirstConsumer` binding.

1. Start an upload with virtctl 
```
virtctl image-upload dv fedora-dv --uploadproxy-url=https://cdi-uploadproxy.mycluster.com --image-path=/images/fedora30.qcow2

DataVolume default/fedora-dv created
Waiting for PVC fedora-dv upload pod to be ready...
cannot upload to DataVolume in state WaitForFirstConsumer
```

Current version of virtctl should immediatetly detect the problem and error. The older version will take all the time limit and finish with timeout.

2. To bind the `PVC` create a consumer `POD`. With a `nodeSelector` to select a specific node ot without for a random node.

Find a pvc:
`k get pvc -l app=containerized-data-importer -o custom-columns=NAME:.metadata.name,OWNER:.metadata.ownerReferences  | grep fedora-dv
`

Create a pod to bind a pvc to any node using the following snippet with correct pvc name and namespace.

```
PVC=<PVC_NAME>
NAMESPACE=<PVC_NAMESPACE>

cat <<EOF | kubectl create -n $NAMESPACE -f -   
apiVersion: v1
kind: Pod
metadata:
  name: consumer-$PVC
spec:
  volumes:
    - name: pod1-storage
      persistentVolumeClaim:
        claimName: $PVC
  containers:
  - name: test-pod-container
    image: busybox
    command: ['sh', '-c', 'echo "Will bind the pvc!" ']
    volumeMounts:
      - mountPath: /disk
        name: pod1-storage
  nodeSelector:
    kubernetes.io/hostname: <NODE_HOSTNAME>

EOF
```

Check if the pvc is bound and kill the temporary pod.
```
kubectl delete pod consumer-$PVC -n $NAMESPACE
```

3. Repeat the virtctl upload command.

```
virtctl image-upload dv fedora-dv --uploadproxy-url=https://cdi-uploadproxy.mycluster.com --image-path=/images/fedora30.qcow2

DataVolume default/fedora-dv created
Waiting for PVC fedora-dv upload pod to be ready...
Pod now ready
Uploading data to https://localhost:9443

 319.13 MiB / 319.13 MiB [================================================================================================================] 100.00% 0s

Uploading data completed successfully, waiting for processing to complete, you can hit ctrl-c without interrupting the progress
Processing completed successfully
Uploading /images/fedora30.qcow2 completed successfully

```
