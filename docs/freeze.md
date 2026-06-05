# Guest file system freeze

Kubevirt provides a way to freeze/thaw the guest filesystem. That can be used to ensure the filesystem consistency
when making a backup or a VM snapshot. The freeze requires a running qemu-guest-agent in the guest VM.

Note: To check if your vm has a qemu-guest-agent check for 'AgentConnected' in the vm status.

## freeze/unfreeze subresource API

VirtualMachineInstance API implements a subresource for a freeze and unfreeze commands.
When called it sends respective freeze/unfreeze commands to the virt-launcher.

The freeze/unfreeze can be accessed using HTTP PUT command on a vmi REST API or by a Kubevirt client-go.
API subresource is located under a `virtualmachineinstances` resource
(example uri for VM `example-vm` in namespace `demo` is `apis/subresources.kubevirt.io/v1alpha3/namespaces/demo/virtualmachineinstances/example-vm/freeze`).

It can also be access through kubevirt/client-go. It is available on VirtualMachineInstanceInterface defined in `kubevirt.io/client-go/kubecli/kubevirt.go`.

```
virtClient.VirtualMachineInstance(namespace).Freeze(vmiName, unfreezeTimeout)
virtClient.VirtualMachineInstance(namespace).Unfreeze(vmiName)
```

Freeze command has additional parameter - the unfreezeTimeout - the timeout when an automatic unfreeze command is
executed in the virt launcher to prevent the vmi from being stuck on freeze. It can be set to 0 to disable 
automatic unfreeze, but that is not recommended.

The freeze/unfreeze subresources are internally used by the VM VirtualMachineSnapshot API
(https://kubevirt.io/user-guide/operations/snapshot_restore_api/).


## virt-freezer

To integrate with kubevirt agnostic tools that are not using kubevirt REST APIs a special virt-freezer application is
available in the `compute` container. Virt-freezer binary is located in `/usr/bin/virt-freezer`.

Freeze the `example-vm` in the namespace `demo`:
```bash

kubectl exec virt-launcher-example-vm-v8vxz -n demo -- /usr/bin/virt-freezer --freeze --namespace demo --name example-vm 
```
The freeze command has an unfreeze timeout set to 5 minutes, that can be changed with the use of the `unfreezeTimeoutSeconds` flag. 
Example with unfreeze timeout set to one minute:
```bash
kubectl exec virt-launcher-example-vm-v8vxz -n demo -- /usr/bin/virt-freezer --freeze --namespace demo --name example-vm --unfreezeTimeoutSeconds 60
```
Check if the freeze is effective. This can be done by checking the value of the Fs Freeze Status.
```bash
kubectl describe vmi example-vm -n demo | grep Freeze
  Fs Freeze Status:        frozen
```

Unfreeze the `example-vm` in the namespace `demo` and verify that the filesystem is no longer frozen (the freeze status is empty or missing).
```bash
kubectl exec virt-launcher-example-vm-v8vxz -n demo -- /usr/bin/virt-freezer --unfreeze --namespace demo --name example-vm
kubectl describe vmi example-vm -n demo | grep Freeze
```
