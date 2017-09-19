# VirtualMachineReplicaSet

## Overview

In order to allow scaling up or down similar VMs, add a way to specify
`VirtualMachine` templates and the amount of replicas required, to let the
runtime create these `VirtualMachine`s.

## Use-cases

### Ephemeral VMs

Scaling ephemeral VMs which only have read-only mounts, or work with a backing
store, to keep temporary data, which can be deleted after the VM gets
undefined.

## Implementation

A new object `VirtualMachineReplicaSet`, backed with a controller will be
implemented:

```yaml
apiVersion: kubevirt.io/v1alpha1
kind: VirtualMachineReplicaSet
metadata:
  name: myreplicaset
spec:
  replicas: 3
  selector:
    matchLabels:
      mylabel: mylabel
  template:
    metadata:
      name: test
      labels:
        mylabel: mylabel
    spec:
      domain:
        devices:
      [...]  
```

`spec.template` is similar to a `VirtualMachineSpec`. `spec.replicas` specifies
how many instances should be created out of `spec.temlate`. `spec.selector`
contains selectors, which need to match `spec.template.metadata.labels`.

The status looks like this:

```yaml
status:
  conditions: null
  replicas: 3
```

It shows the number of `VirtualMachine`s which are in a non-final state nad
which match `spec.selector`. In case of a scaling error, a `ReplicaFailure`
condition is added to the status.

In case of a delete failure:

```yaml
status:
  conditions:
  - type: "ReplicaFailure"
    status: True
    reason: "FailureDelete"
    message: "no permission to delete VMs"
    lastTransmissionTime: "..."
  replicas: 4
```

In case of a create failure:

```yaml
status:
  conditions:
  - type: "ReplicaFailure"
    status: True
    reason: "FailureCreate"
    message: "no permission to create VMs"
    lastTransmissionTime: "..."
  replicas: 2
```

### Guarantees

Once Graceful Deletes are implemented, the VirtualMachineReplicaSet guarantees
that it will never create more than the requested numbers of VMs in a non-final
state.

As a consequence, in case of node failures, or in case of connection loss to
virt-handlers which run VMs which are part of that set, no new VMs will be
spawned until the VMs in unknown state are explicitly removed, or the node
reconnects. This behaviour allows plugging in fencing controllers in a well
defined way.

It might make sense to weaken these guarantees in the future, and instead add a
`StatefulSet` equivalent to KubeVirt in the future.

### Milestones

 * Basic functionality
 * Support graceful delete [1]
 * Support controller references [2]
 * Support label changes
 * Support adopting orphaned Pods [2]
 * Define a well known scale-down order

The basic functionality includes scaling up, down and reporting errors if
scaling does not work. In this stage it is the full responsibility of the user
to attach labels to the VMs in a way, so that selectors of multiple
VirtualMachineReplicaSets don't overlap.

### References

1. https://kubernetes.io/docs/tasks/run-application/force-delete-stateful-set-pod/#delete-pods
2. https://github.com/kubernetes/community/blob/58b1c30d95719749068497ba35dfe4c64b21aa72/contributors/design-proposals/api-machinery/controller-ref.md
