# KubeVirt Snapshot and Restore API

The `snapshot.kubevirt.io` API Group defines resources for snapshotting and restoring KubeVirt `VirtualMachines`

## Prerequesites

### VolumeSnapshotClass

KubeVirt leverages the `VolumeSnapshot` functionality of Kubernetes [CSI drivers](https://kubernetes-csi.github.io/docs/drivers.html) for capturing persistent `VirtualMachine` state.  So, you should make sure that your `VirtuallMachine` uses `DataVolumes` or `PersistentVolumeClaims` backed by a `StorageClass` that supports `VolumeSnapshots` and a `VolumeSnapshotClass` is properly configured for that `StorageClass`.

To list `VolumeSnapshotClasses`:

```bash
kubectl get volumesnapshotclass
```

Make sure that the `provisioner` property of your `StorageClass` matches the `driver` property of the `VolumeSnapshotClass`

Even if you have no `VolumeSnapshotClasses` in your cluster, `VirtualMachineSnapshots` are not totally useless.  They will still backup your `VirtualMachine` configuration.

### Snapshot Feature Gate

Snapshot/Restore are currently considered alpha features and are disabled by default.

```bash
kubectl patch -n kubevirt kubevirt kubevirt -p '{"spec": {"configuration": { "developerConfiguration": { "featureGates": [ "Snapshot" ] }}}}' -o json --type merge
```

## Snapshot a VirtualMachine

Snapshot a virtualMachine is supported for online and offline vms.

When snapshoting a running vm the controller will check for qemu guest agent in the vm, if it exists it will freeze the vm filesystems before taking the snapshot (and unfreeze after the snapshot). It is recommended to take online snapshot with the guest agent for a better snapshot, if not present a best effort snapshot will be taken.\
\*To check if you're vm has qemu-guest-agent check for 'AgentConnected' in the vm spec.

There will be an indication in the vmSnapshot status if the snapshot was taken online and with or without guest agent participation.

\*Currently online vm snapshot is not supported with hotplug disks, in such case the vm has to be turned off in order to take the snapshot.


To snapshot a `VirtualMachine` named `larry`, apply the following yaml.

```yaml
apiVersion: snapshot.kubevirt.io/v1alpha1
kind: VirtualMachineSnapshot
metadata:
  name: snap-larry
spec:
  source:
    apiGroup: kubevirt.io
    kind: VirtualMachine
    name: larry
```

To wait for a snapshot to complete, execute:

```bash
kubectl wait vmsnapshot snap-larry --for condition=Ready
```

You can check the vmSnapshot phase in the vmSnapshot status, it can be either InProgress/Succeeded/Failed.\
The vmSnapshot has a default deadline of 5min. If the vmSnapshot hasn't completed succeessfully until then it will be marked as Failed and cleanup will be made (unfreeze vm and delete created content snapshot as necessary).
The vmSnapshot will remain in Failed state until deleted by the user.\
To change the default deadline add to the snapshot Spec: 'FailureDeadline' with the new value. In order to cancel this deadline you can mark it as 0 (not recommended).


## Restoring a VirtualMachine

To restore the `VirtualMachine` `larry` from `VirtualMachineSnapshot` `snap-larry`, apply the following yaml.

```yaml
apiVersion: snapshot.kubevirt.io/v1alpha1
kind: VirtualMachineRestore
metadata:
  name: restore-larry
spec:
  target:
    apiGroup: kubevirt.io
    kind: VirtualMachine
    name: larry
  virtualMachineSnapshotName: snap-larry
```

To wait for a restore to complete, execute:

```bash
kubectl wait vmrestore restore-larry --for condition=Ready
```

## Cleanup

Keep `VirtualMachineSnapshots` (and their corresponding `VirtualMachineSnapshotContents`) around as long as you may want to restore from them again.

Feel free to delete `larry-restore` as it is not needed once the restore is complete.
