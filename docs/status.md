Status
======

The purpose of this document is to detail how the HyperConverged Cluster
Operator (HCO) handles the status on instances of the `HyperConverged`
Custom Resource (presumably only one instance is allowed per cluster). This
document is a follow-up to the [conditions proposal](conditions.md)
including the implementation details to explain in long form how we are
consolidating the status of component operators on the `HyperConverged`
resource.

In this first implementation the goals are:

1. Avoid ambiguity. If we can't complete reconcile our conditions should reflect
   that. Essentially, the combination of our log messages and the status on our
   `HyperConverged` resource should be enough to tell an admin what's wrong or
   where they need to look next.
1. Keep it simple. The logic for how the conditions on the `HyperConverged`'s
   resource are being set should be easy to follow.
1. Avoid timeouts. It's tempting to say if `KubeVirt` has been progressing for
   an hour that we should allow OLM to upgrade us (by setting the readiness
   probe to succeed). However, OLM should provide a mechanism for overriding
   operators and we want this to be simple.

**NOTE**

Conditions consist of a number of fields including Type, Status, etc. For the
remainder of this document we will be merging the Type and Status, so the
Available condition type with status set to false will simply be written as
!Available.

# HyperConvergedStatus

See [hyperconverged_types.go](../api/v1beta1/hyperconverged_types.go).

The `HyperConvergedStatus` type belongs on the `Status` field of the
`HyperConverged` struct. The `HyperConverged` has two fields:

* Conditions (`conditions`) describes the current state of the `HyperConverged`
    Custom Resource.
* RelatedObjects (`relatedObjects`) is a list of objects created and maintained
    by the HCO. This list includes objects like the `KubeVirt` and `CDI` Custom
    Resources.

## Conditions

See the [conditions proposal](conditions.md). In the first
implementation the intention is to make it as simple as possible. The algorithm
for consolidating status in the `Reconcile()` functions is as follows:

1. Add an initial set of Conditions to the instance
1. Make the in-memory representation of the Conditions `nil` (this is
   intentionally separate from the `Conditions` on the `HyperConvergedStatus`).
   Only negative conditions will be saved on the in-memory representation.
1. For each component to reconcile (in this first pass this is only `KubeVirt`,
   `CDI`, and `NetworkAddonsConfig` components).
   1. If we find the resource and `found.Status.Conditions == nil`, then set the
      following on the in-memory representation of the `Conditions`: !Available,
      Progressing, !Upgradeable. The reason string will always be
      `"${component}Conditions"` with message string `"${component} resource has no
      conditions"` to make it clear who is not reporting conditions as we
      expect.
   1. If we find the resource and `found.Status.Conditions != nil`, then we
      iterate over the conditions.
      1. If !Available then set the in-memory representation !Avaible with
         reason `"${component}NotAvailable"` and add the components condition
         message to ours, `"${component} is not available: "`.
      1. If Progressing then set the in-memory representation Progressing with
         reason `"${component}Progressing"` and add the components condition
         message to ours, `"${component} is progressing: "`. __Also__ set the
         in-memory represenation !Upgradeable with the same reason and message.
      1. If Degraded then set the in-memory representation Degraded with
         reason `"${component}Degraded"` and add the components condition
         message to ours, `"${component} is degraded: "`.
1. Evaluate the in-memory representation of the `Conditions`. If `nil`, then we
   know no component operator has reported negatively and we can mark our
   instance as Available, !Progressing, !Degraded, and Upgradeable (also set
   readiness probe to succeed). If `!nil`, then we simply set those conditions
   on the instance (note: lastTransitionTime and lastHeartbeatTime should be
   handled for us). If any component reports as Progressing then we also set the
   readiness probe to fail.
1. Write the status back to the cluster.

**NOTE**

The distinction between the in-memory slice of Conditions (a field on the
`ReconcileHyperConverged` struct) and the server side or cluster side Conditions
(field on the `HyperConvergedStatus`) is important.

## Related Objects

Maintaining a list of the objects being controlled by the `HyperConverged`
Custom Resource is most valuable for auditing purposes. It may be useful for an
admin to be able to dig deeper to find, for example, the
`KubeVirtCommonTemplatesBundle` that is created in the `openshift` namespace.
During reconcile, we iterate over the objects to ensure that they exist as we
expect them too, if we find the object then we simply add it to the list of
`relatedObjects`. Doing this with the found objects allows us to add the uid and
resourceVersion.
