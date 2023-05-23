# KubeVirt Updates

## Overview

Starting with KubeVirt release `v0.17.0` zero downtime updates are supported
through the use of the KubeVirt operator.

`Zero Downtime` in the context of KubeVirt means the following will hold true.

- The KubeVirt control plane components will remain available throughout the
update process allowing for uninterrupted VM/VMI creations, deletions, and
modifications.

- Existing VMI workloads will not be interrupted during the update process.

There are some known disruptions that will occur during updates though. This is
limited to anything that involves persistent connections to a control plane
component.

1. In-flight migrations will fail. The VMI will remain available, but the
migration itself will fail due to a severed TLS connection occurring when
virt-handler updates.

2. `virtctl console` and `virtctl vnc` connections will get dropped. This is
due to the virt-api instance that is proxying the connection getting shut down
in favor of a an updated virt-api instance. The new virt-api instance will be
available before the old one is terminated which is why creation, deletions,
and modifications are not impacted.

### Methods to Trigger KubeVirt Update

There are two ways to trigger an update of KubeVirt. 

#### By Patching KubeVirt CR's imageTag value

If an imageTag is specified in the KubeVirt CR, then KubeVirt can be updated
by patching the imageTag value with a new release tag.

**Example:**

The cluster has the following KubeVirt CR deployed.

```
apiVersion: kubevirt.io/v1
kind: KubeVirt
metadata:
  name: kubevirt
  namespace: kubevirt
spec:
  imageTag: v0.17.0
  imagePullPolicy: IfNotPresent
```

To update the KubeVirt release to a new version, we'd patch the KubeVirt object.

```
kubectl patch kv kubevirt -n kubevirt --type=json -p '[{ "op": "add", "path": "/spec/imageTag", "value": "v0.18.0" }]'
```

#### By Updating KubeVirt Operator

If neither the imageTag nor the imageRegistry values are specified, the system
assumes that the KubeVirt CR's version is locked to the KubeVirt operator's
version. This means that updating the operator will automatically trigger
updating KubeVirt as well.

**Example:**

The cluster has the following KubeVirt CR deployed.

```
apiVersion: kubevirt.io/v1
kind: KubeVirt
metadata:
  name: kubevirt
  namespace: kubevirt
spec:
  imagePullPolicy: IfNotPresent
```

Since the CR does not have the imageTag or imageRegistry values set, the
KubeVirt and KubeVirt Operator versions are locked.

Updating the KubeVirt operator to a new version will result in the
underlying KubeVirt installation to also be updated.

```
export RELEASE=v0.17.0
kubectl apply -f https://github.com/kubevirt/kubevirt/releases/download/${RELEASE}/kubevirt-operator.yaml
```

## Implementation Details

### Component Update Ordering

Controllers (virt-controller and virt-handler) are updated before virt-api.

The old virt-api acts as a feature gate ensuring that new behavior can not be
utilized until all the new controllers are available to support the new features.

Once all the controllers are updated, virt-api is updated which will allow usage
of new functionality. 

### RBAC 

Since during the update our control plane will be briefly running both old and
new versions, we have to ensure that RBAC permissions are available that allow
both the old and new versions of components to operate at the same time.

This means that during the update we are using the RBAC rules defined by both
the old and new KubeVirt versions. Once the update completes, only the new
RBAC rules will remain.

### New APIs

New APIs are not available until after the entire update process has completed.
This ensures that a new API object can't be posted to the cluster until every
component within the cluster is updated to learn of the object.

## Notes

### `v1.0.0` Migration To New Storage Versions

With the `v1.0.0` release of KubeVirt the storage version of all core
`kubevirt.io` APIs will be moving to version `v1`. To
accommodate the eventual removal of the `v1alpha3` version with KubeVirt >=
`v1.2.0` it is recommended that operators deploy the
[`kube-storage-version-migrator`](https://github.com/kubernetes-sigs/kube-storage-version-migrator)
tool within their environment. This will ensure any existing `v1alpha3`
stored objects are migrated to `v1` well in advance of the removal of the
underlying `v1alpha3` version.
