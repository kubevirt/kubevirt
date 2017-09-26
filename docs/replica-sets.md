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

`spec.template` is equal to a `VirtualMachineSpec`. `spec.replicas` specifies
how many instances should be created out of `spec.template`. `spec.selector`
contains selectors, which need to match `spec.template.metadata.labels`.

The status looks like this:

```yaml
status:
  conditions: null
  replicas: 3
  readyReplicas : 2
```
In case of a scaling error, a `ReplicaFailure` condition is added to the
`status.conditions`. Further it shows the number of `VirtualMachine`s which
are in a non-final state and which match `spec.selector` in the
`status.replicas` field.  `status.readyReplicas` indicates how many of these
replicas meet the ready condition.

*Note* that at the moment when writing this proposal, there exist no
readiness checks for VirtualMachines in Kubevirt. Therefore a `VirtualMachine` is
considered to be ready, when reported by virt-handler as running or migrating.

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
  readyReplicas: 3
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
  readyReplicas: 3
```

### Guarantees

The VirtualMachineReplicaSet  does **not** guarantee that there will never be
more than the wanted replicas active in the cluster. Based on readiness checks,
unknown VirtualMachine states and graceful deletes, it might decide to already
create new replicas in advance, to make sure that the amount of ready replicas
stays close to the expected replica count.

### Milestones

 * Basic functionality
 * Support label changes
 * Define a well known scale-down order
 * Support graceful delete [1]
 * Support controller references [2]
 * Support adopting orphaned Pods [2]

The basic functionality includes scaling up, down and reporting errors if
scaling does not work. In this stage it is the full responsibility of the user
to attach labels to the VMs in a way, so that selectors of multiple
VirtualMachineReplicaSets don't overlap.

### References

1. https://kubernetes.io/docs/tasks/run-application/force-delete-stateful-set-pod/#delete-pods
2. https://github.com/kubernetes/community/blob/58b1c30d95719749068497ba35dfe4c64b21aa72/contributors/design-proposals/api-machinery/controller-ref.md
