Handling Eviction Requests
=

# Preface
The [Kubernetes](https://kubernetes.io/) API supports [API-initiated Eviction](https://kubernetes.io/docs/concepts/scheduling-eviction/api-eviction/) which allows to programmatically evict pods.
The API is used for example by:
- [kubectl drain](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/)
- [Descheduler](https://github.com/kubernetes-sigs/descheduler)

Since KubeVirt virtual machines are running inside `virt-launcher` pods - they are affected by Kubernetes' eviction mechanism.
This requires special handling on KubeVirt's side, since virtual machines eviction is a bit more complex than the average pod.
`Evacuation` is the term used by KubeVirt to describe the migration of a VMI as the result of `virt-launcher` pod eviction.

This document will describe how KubeVirt currently handles eviction requests.

# Eviction Strategies
A VirtualMachineInstance can have one of four Eviction Strategies. The eviction strategy is defined in the VMI spec, with a fallback to a cluster-wide definition in the KubeVirt CustomResource.

The eviction strategy affects the way the VirtualMachineInstance will be evacuated:

| Eviction Strategy     | Meaning                                                                                                                                                                                                                                                                                                                                          |
|-----------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| None                  | No action will be taken, according to the specified 'RunStrategy' the VirtualMachine will be restarted or shutdown                                                                                                                                                                                                                               |
| LiveMigrate           | The VirtualMachine will be migrated instead of being shutdown                                                                                                                                                                                                                                                                                    |
| LiveMigrateIfPossible | Same as `LiveMigrate` but only if the VirtualMachine is Live-Migratable, otherwise it will behave as `None`                                                                                                                                                                                                                                      |
| External              | The VirtualMachine will be protected by a PDB and vmi.Status.EvacuationNodeName will be set on eviction. This is mainly useful for cluster-api-provider-kubevirt (capk) which needs a way for VMI’s to be blocked from eviction, yet signal capk that eviction has been called on the VMI so the capk controller can handle tearing the VMI down |

# Pod Eviction Webhook
`virt-api` serves a [validating webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/) which intercepts **all** eviction requests in the cluster:
```shell
kubectl get validatingwebhookconfigurations.admissionregistration.k8s.io virt-api-validator -o yaml
```

The webhook serves two purposes:
1. To trigger VMI evacuation in cases where it is required.
2. To prevent evacuation of associated hotplug pods.

The way the webhook triggers the evacuation is by setting the VMI's `Status.EvacuationNodeName` field to the node name it is currently running on, so the [evacuation controller](#evacuation-controller) will know it needs to migrate it to another node.

The webhook has the ability to:
1. Approve the request - so it could be further processed
2. Deny the request - the request will be declined without additional processing

The webhook admits eviction requests **before** `kube-api` checks them against [Pod Distribution Budget](https://kubernetes.io/docs/concepts/workloads/pods/disruptions/#pod-disruption-budgets) objects.

In case the pod is not a `virt-launcher` or a `hp-volume-` pod - the eviction request is approved. Otherwise, depending on the VMI's eviction strategy and whether it is migratable - the webhook will potentially mark the VMI for evacuation and approve or deny the eviction request: 

| Eviction Strategy     | Is VMI migratable | Is VMI marked for evacuation | Does Webhook approve eviction | Webhook Response                                 |
|-----------------------|-------------------|------------------------------|-------------------------------|--------------------------------------------------|
| None                  | True/False        | False                        | True                          | 200 - Eviction granted                           |
| LiveMigrate           | True              | True                         | False                         | 429 - Eviction denied (evacuation was triggered) |
| LiveMigrate           | False             | False                        | False                         | 429 - Eviction denied                            |
| LiveMigrateIfPossible | True              | True                         | False                         | 429 - Eviction denied (evacuation was triggered) |
| LiveMigrateIfPossible | False             | False                        | True                          | 200 - Eviction granted                           |
| External              | True/False        | True                         | False                         | 429 - Eviction denied (evacuation was triggered) |

The webhook will approve additional eviction requests on a virt-launcher pod owned by a VMI which had previously been marked for evacuation:

| Eviction Strategy     | Is VMI migratable | Does Webhook approve eviction | Webhook Response       |
|-----------------------|-------------------|-------------------------------|------------------------|
| LiveMigrate           | True              | True                          | 200 - Eviction granted |
| LiveMigrateIfPossible | True              | True                          | 200 - Eviction granted |
| External              | True/False        | True                          | 200 - Eviction granted |

In these cases, a PDB will protect the virt-launcher pod (see explanation bellow).

> **Note**  
> Since the webhook intercepts all eviction requests in the cluster, it is configured to be [ignored](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#failure-policy) in case kube-api fails to get a response from it.
> Ignored in this context means that the eviction request is considered to be approved by the webhook, and will be further checked against the PodDistributionBudget.
> Some virt-launcher pods should be protected from eviction even if the webhook fails, this is the reason PodDistributionBudget objects are required (described in the next section).
> In case the webhook is down, the virt-launcher pod will be protected from eviction by the PDB (if required), but the evacuation will **not** be triggered.

# Pod Distribution Budget
In case the `Pod Eviction Webhook` approved the eviction, kube-api checks whether a PDB protects the `virt-launcher` pod.
If there is a PDB protecting the `virt-launcher` pod - the eviction request is denied, otherwise it is approved and the pod is evicted.

In order for the evacuation of VMIs to happen in a controlled manner, KubeVirt protects part of the `virt-launcher` pods with a PDB which blocks eviction requests.

`virt-controller` has a `Disruption Budget Controller` which decides whether a `virt-launcher` pod should be protected based on the eviction strategy of its controlling VMI:

| Eviction Strategy     | Is a PDB required             | 
|-----------------------|-------------------------------|
| None                  | False                         |
| LiveMigrate           | True                          |
| LiveMigrateIfPossible | Only if the VMI is migratable |
| External              | True                          |

> **Note**  
> During a migration, the PDB that protects the source virt-launcher pod is expended by the migration controller to also protect the target pod.

# Eviction Approval Summary

The eviction request's initiator will observe one of the following responses:

| Eviction Strategy     | Is VMI migratable | Is VMI marked for evacuation | Does Webhook approve eviction | Does PDB allow eviction | Final Response                                   |
|-----------------------|-------------------|------------------------------|-------------------------------|-------------------------|--------------------------------------------------|
| None                  | True/False        | False                        | True                          | True                    | 200 - Eviction granted                           |
| LiveMigrate           | True              | True                         | False                         | False                   | 429 - Eviction denied (evacuation was triggered) |
| LiveMigrate           | False             | False                        | False                         | False                   | 429 - Eviction denied by webhook                 |
| LiveMigrateIfPossible | True              | True                         | False                         | False                   | 429 - Eviction denied (evacuation was triggered) |
| LiveMigrateIfPossible | False             | False                        | True                          | True                    | 200 - Eviction granted                           |
| External              | True/False        | True                         | False                         | False                   | 429 - Eviction denied (evacuation was triggered) |

For additional requests on virt-launcher pods owned by a VMI which had previously been marked for evacuation:

| Eviction Strategy     | Is VMI migratable | Does Webhook approve eviction | Does PDB allow eviction | Final Response                |
|-----------------------|-------------------|-------------------------------|-------------------------|-------------------------------|
| LiveMigrate           | True              | True                          | False                   | 429 - Eviction blocked by PDB |
| LiveMigrateIfPossible | True              | True                          | False                   | 429 - Eviction blocked by PDB |
| External              | True/False        | True                          | False                   | 429 - Eviction blocked by PDB |

To summarize:
1. The eviction request is granted only if both the webhook and the PDB allow them.
2. When the eviction request's initiator gets a 429 response, they can check the (first) response message whether the VMI will be evacuated.

## Example kubectl drain Output
The following output depicts the eviction of a `virt-launcher` pod owned by a migratable VMI (with the `LiveMigrate` eviction strategy):
```shell
$ kubectl drain node01 --ignore-daemonsets --delete-emptydir-data
...
evicting pod default/virt-launcher-vm-cirros-wn5v4
error when evicting pods/"virt-launcher-vm-cirros-wn5v4" -n "default" (will retry after 5s): admission webhook "virt-launcher-eviction-interceptor.kubevirt.io" denied the request: Eviction triggered evacuation of VMI "default/vm-cirros"
...
evicting pod default/virt-launcher-vm-cirros-wn5v4
error when evicting pods/"virt-launcher-vm-cirros-wn5v4" -n "default" (will retry after 5s): Cannot evict pod as it would violate the pod's disruption budget.
...
evicting pod default/virt-launcher-vm-cirros-wn5v4
pod/virt-launcher-vm-cirros-wn5v4 evicted
node/node01 drained
```


# Evacuation Controller
`virt-controller` has an evacuation controller which looks for potential VMIs to evict and tries to migrate them to another node.
