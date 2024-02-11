KubeVirt Deployment
=

# Preface
This document covers the additional [Kubernetes](https://kubernetes.io/) objects that are added to a cluster as the result of deploying [KubeVirt](https://kubevirt.io/).

When deploying KubeVirt onto a Kubernetes cluster, we deploy the following:

- `virt-operator` and its dependencies
- KubeVirt CustomResource

This is done using the [kubectl](https://kubernetes.io/docs/reference/kubectl/) command line tool or programmatically.

```shell
# Point at latest stable release
export RELEASE=$(curl https://storage.googleapis.com/kubevirt-prow/release/kubevirt/kubevirt/stable.txt)
# Deploy the virt-operator
kubectl apply -f https://github.com/kubevirt/kubevirt/releases/download/${RELEASE}/kubevirt-operator.yaml
# Create the KubeVirt CR (instance deployment request) which triggers the actual installation
kubectl apply -f https://github.com/kubevirt/kubevirt/releases/download/${RELEASE}/kubevirt-cr.yaml
# Wait until all KubeVirt components are up
kubectl -n kubevirt wait kubevirt kubevirt --for condition=Available
```

# Virt Operator Deployment

`virt-operator` is the component responsible for deploying the components of the KubeVirt project.

## Namespace
A dedicated [Namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) is created for KubeVirt's components.

```shell
kubectl get namespace kubevirt -o yaml
```

## KubeVirt CustomResourceDefinition
KubeVirt extends Kubernetes using [CustomResourceDefinitions](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/) (CRDs).

The first deployed CRD is `kubevirts.kubevirt.io` which allows us to later create the `KubeVirt` CustomResource.

```shell
kubectl get customresourcedefinition kubevirts.kubevirt.io -o yaml
```

## Priority Class
`kubevirt-cluster-critical` [PriorityClass](https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/#priorityclass) is deployed and will be used by `virt-operator`.

```shell
kubectl get priorityclasses.scheduling.k8s.io kubevirt-cluster-critical -o yaml
```

## ServiceAccount
A [ServiceAccount](https://kubernetes.io/docs/concepts/security/service-accounts/) is deployed to be used by `virt-operator`.

```shell
kubectl get -n kubevirt serviceaccount kubevirt-operator -o yaml
```

## ClusterRoles
```shell
kubectl get clusterrole kubevirt.io:operator -o yaml
kubectl get clusterrole kubevirt-operator -o yaml
```

## ClusterRoleBinding

Binds the `kubevirt-operator` ClusterRole to the `kubevirt:kubevirt-operator` ServiceAccount.

```shell
kubectl get clusterrolebinding kubevirt-operator -o yaml
```

## Role
```shell
kubectl get -n kubevirt role kubevirt-operator -o yaml
```

## RoleBinding
Binds the `kubevirt-operator` Role to the `kubevirt:kubevirt-operator` ServiceAccount.

```shell
kubectl get -n kubevirt rolebinding kubevirt-operator-rolebinding -o yaml
```

## Deployment
A [Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) is created to execute `virt-operator`.

```shell
kubectl get -n kubevirt deployment virt-operator -o yaml
```

## KubeVirt CustomResource
After the `virt-operator` and its dependencies are deployed, we deploy the KubeVirt CustomResource in order to trigger virt-operator to reconcile it and deploy KubeVirt's main components.

```shell
kubectl get -n kubevirt kubevirts.kubevirt.io kubevirt -o yaml
```

# Operands
## CustomResourceDefinitions
`virt-operator` deploys several CustomResourceDefinitions that will be described in the following sections.

### VirtualMachineInstance
```shell
kubectl get customresourcedefinition virtualmachineinstances.kubevirt.io -o yaml
```
For additional information please read the API [documentation](https://kubevirt.io/api-reference/main/definitions.html#_v1_virtualmachineinstance).

### VirtualMachine
```shell
kubectl get customresourcedefinition virtualmachines.kubevirt.io -o yaml
```

For additional information please read the API [documentation](https://kubevirt.io/api-reference/main/definitions.html#_v1_virtualmachine).

### VirtualMachineInstanceReplicaSet
```shell
kubectl get customresourcedefinition virtualmachineinstancereplicasets.kubevirt.io -o yaml
```
For additional information please read the API [documentation](https://kubevirt.io/api-reference/main/definitions.html#_v1_virtualmachineinstancereplicaset).

### VirtualMachinePool
```shell
kubectl get customresourcedefinition virtualmachinepools.pool.kubevirt.io -o yaml
```

For additional information please read the API [documentation](https://kubevirt.io/api-reference/main/definitions.html#_v1alpha1_virtualmachinepool).

### VirtualMachineInstanceMigration
```shell
kubectl get customresourcedefinition virtualmachineinstancemigrations.kubevirt.io -o yaml
```

For additional information please read the API [documentation](https://kubevirt.io/api-reference/main/definitions.html#_v1_virtualmachineinstancemigration).

### MigrationPolicy
```shell
kubectl get customresourcedefinition migrationpolicies.migrations.kubevirt.io -o yaml
```

For additional information please read the API [documentation](https://kubevirt.io/api-reference/main/definitions.html#_v1alpha1_migrationpolicy).

### VirtualMachineSnapshot
```shell
kubectl get customresourcedefinition virtualmachinesnapshots.snapshot.kubevirt.io -o yaml
```

For additional information please read the API [documentation](https://kubevirt.io/api-reference/main/definitions.html#_v1alpha1_virtualmachinesnapshot).

### VirtualMachineSnapshotContent
```shell
kubectl get customresourcedefinition virtualmachinesnapshotcontents.snapshot.kubevirt.io -o yaml
```

For additional information please read the API [documentation](https://kubevirt.io/api-reference/main/definitions.html#_v1alpha1_virtualmachinesnapshotcontent).

### VirtualMachineRestore
```shell
kubectl get customresourcedefinition virtualmachinerestores.snapshot.kubevirt.io -o yaml
```

For additional information please read the API [documentation](https://kubevirt.io/api-reference/main/definitions.html#_v1alpha1_virtualmachinerestore).

### VirtualMachineClone
```shell
kubectl get customresourcedefinition virtualmachineclones.clone.kubevirt.io -o yaml
```

For additional information please read the API [documentation](https://kubevirt.io/api-reference/main/definitions.html#_v1alpha1_virtualmachineclone).

### VirtualMachineExport
```shell
kubectl get customresourcedefinition virtualmachineexports.export.kubevirt.io -o yaml
```

For additional information please read the API [documentation](https://kubevirt.io/api-reference/main/definitions.html#_v1alpha1_virtualmachineexport).

### VirtualMachineClusterInstancetype
```shell
kubectl get customresourcedefinition virtualmachineclusterinstancetypes.instancetype.kubevirt.io -o yaml
```
For additional information please read the API [documentation](https://kubevirt.io/api-reference/main/definitions.html#_v1beta1_virtualmachineclusterinstancetype).

### VirtualMachineInstancetype
```shell
kubectl get customresourcedefinition virtualmachineinstancetypes.instancetype.kubevirt.io -o yaml
```

For additional information please read the API [documentation](https://kubevirt.io/api-reference/main/definitions.html#_v1beta1_virtualmachineinstancetype).

### VirtualMachineClusterPreference
```shell
kubectl get customresourcedefinition virtualmachineclusterpreferences.instancetype.kubevirt.io -o yaml
```

For additional information please read the API [documentation](https://kubevirt.io/api-reference/main/definitions.html#_v1beta1_virtualmachineclusterpreference).

### VirtualMachinePreference
```shell
kubectl get customresourcedefinition virtualmachinepreferences.instancetype.kubevirt.io -o yaml
```

For additional information please read the API [documentation](https://kubevirt.io/api-reference/main/definitions.html#_v1beta1_virtualmachinepreference).

### VirtualMachineInstancePreset
**Deprecated.**

```shell
kubectl get customresourcedefinition virtualmachineinstancepresets.kubevirt.io -o yaml
```
For additional information please read the API [documentation](https://kubevirt.io/api-reference/main/definitions.html#_v1_virtualmachineinstancepreset).

## ClusterRoles
The following ClusterRoles could be bound to users / groups. 
### Default
This ClusterRole is bound to the `system:authenticated` group.

```shell
kubectl get clusterrole kubevirt.io:default -o yaml
kubectl get clusterrolebinding kubevirt.io:default -o yaml
```

### Admin
```shell
kubectl get clusterrole kubevirt.io:admin -o yaml
```

### Edit
```shell
kubectl get clusterrole kubevirt.io:edit -o yaml
```

### View
```shell
kubectl get clusterrole kubevirt.io:view -o yaml
```

### Instance Type View
This ClusterRole is bound to the `system:authenticated` group.

```shell
kubectl get clusterrole instancetype.kubevirt.io:view -o yaml
kubectl get clusterrolebinding instancetype.kubevirt.io:view -o yaml
```

## Virt Operator
Following the initial deployment of `virt-operator` it deploys additional objects.

### Secret
```shell
kubectl get -n kubevirt secret kubevirt-operator-certs -o yaml
```

### Service
```shell
kubectl get -n kubevirt service kubevirt-operator-webhook -o yaml
```

### Validating Webhook Configurations
`virt-operator` serves and registers [validating webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/):
```shell
kubectl get validatingwebhookconfigurations.admissionregistration.k8s.io virt-operator-validator -o yaml
```

## Virt API
### ServiceAccount
```shell
kubectl get -n kubevirt serviceaccount kubevirt-apiserver -o yaml
```

### ClusterRole
```shell
kubectl get clusterrole kubevirt-apiserver -o yaml
```

### ClusterRoleBindings
Binds the `kubevirt-apiserver` ClusterRole to the `kubevirt-apiserver` ServiceAccount.

```shell
kubectl get clusterrolebinding kubevirt-apiserver -o yaml
```

Binds the `system:auth-delegator` ClusterRole to the `kubevirt-apiserver` ServiceAccount.
```shell
kubectl get clusterrolebinding kubevirt-apiserver-auth-delegator -o yaml
```

### Role
```shell
kubectl get -n kubevirt role kubevirt-apiserver -o yaml
```

### RoleBinding
Binds the `kubevirt-apiserver` Role to the `kubevirt-apiserver` ServiceAccount.
```shell
kubectl get -n kubevirt rolebinding kubevirt-apiserver -o yaml
```

### Secret
```shell
kubectl get -n kubevirt secret kubevirt-virt-api-certs -o yaml
```

### Deployment
```shell
kubectl get -n kubevirt deployment virt-api -o yaml
```

### Service
```shell
kubectl get -n kubevirt service virt-api -o yaml
```

### Mutating Webhook Configurations
`virt-api` serves mutating webhooks:

```shell
kubectl get mutatingwebhookconfigurations.admissionregistration.k8s.io virt-api-mutator -o yaml
```

### Validating Webhook Configuration
`virt-api` serves validating webhooks:
```shell
kubectl get validatingwebhookconfigurations.admissionregistration.k8s.io virt-api-validator -o yaml
```

### API Services
`virt-api` serves an [aggregated API](https://kubernetes.io/docs/tasks/extend-kubernetes/configure-aggregation-layer/):

```shell
kubectl get apiservices.apiregistration.k8s.io -l kubevirt.io=virt-api-aggregator
```

## Virt Controller
### ServiceAccount
```shell
kubectl get -n kubevirt serviceaccount kubevirt-controller -o yaml
```

### ClusterRole
```shell
kubectl get clusterrole kubevirt-controller -o yaml
```

### ClusterRoleBinding
Binds the `kubevirt-controller` ClusterRole to the `kubevirt-controller` ServiceAccount.
```shell
kubectl get clusterrolebinding kubevirt-controller -o yaml
```

### Role
```shell
kubectl get -n kubevirt role kubevirt-controller -o yaml
```

### RoleBinding
Binds the `kubevirt-controller` Role to the `kubevirt-controller` ServiceAccount.
```shell
kubectl get -n kubevirt rolebinding kubevirt-controller -o yaml
```

### Secret
```shell
kubectl get -n kubevirt secret kubevirt-controller-certs -o yaml
```

### Deployment
```shell
kubectl get -n kubevirt deployment virt-controller -o yaml
```

## Virt Handler
### ServiceAccount
```shell
kubectl get -n kubevirt serviceaccount kubevirt-handler -o yaml
```

### ClusterRole
```shell
kubectl get clusterrole kubevirt-handler -o yaml
```

### ClusterRoleBinding
Binds the `kubevirt-handler` ClusterRole to the `kubevirt-handler` ServiceAccount.
```shell
kubectl get clusterrolebinding kubevirt-handler -o yaml
```

### Role
```shell
kubectl get -n kubevirt role kubevirt-handler -o yaml
```

### RoleBinding
Binds the `kubevirt-handler` Role to the `kubevirt-handler` ServiceAccount.
```shell
kubectl get -n kubevirt rolebinding kubevirt-handler -o yaml
```

### Secrets
```shell
kubectl get -n kubevirt secret kubevirt-virt-handler-server-certs -o yaml
kubectl get -n kubevirt secret kubevirt-virt-handler-certs -o yaml
```

### DaemonSet
`virt-handler` is deployed as a [DaemonSet](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/).
```shell
kubectl get -n kubevirt daemonset virt-handler -o yaml
```

## Export Proxy
Not deployed by default.

### ServiceAccount
```shell
kubectl get -n kubevirt serviceaccount kubevirt-exportproxy -o yaml
```

### ClusterRole
```shell
kubectl get clusterrole kubevirt-exportproxy -o yaml
```

### ClusterRoleBinding
Binds the `kubevirt-exportproxy` ClusterRole to the `kubevirt-exportproxy` ServiceAccount.
```shell
kubectl get clusterrolebinding kubevirt-exportproxy -o yaml
```

### Role
```shell
kubectl get -n kubevirt role kubevirt-exportproxy -o yaml
```

### RoleBinding
Binds the `kubevirt-exportproxy` Role to the `kubevirt-exportproxy` ServiceAccount.
```shell
kubectl get -n kubevirt rolebinding kubevirt-exportproxy -o yaml
```

### Secret
```shell
kubectl get -n kubevirt secret kubevirt-exportproxy-certs -o yaml
```
### Deployment
```shell
kubectl get -n kubevirt deployment virt-exportproxy -o yaml
```

### Service
```shell
kubectl get -n kubevirt service virt-exportproxy -o yaml
```

## ConfigMaps
```shell
kubectl get -n kubevirt configmap kubevirt-ca -o yaml
kubectl get -n kubevirt configmap kubevirt-export-ca -o yaml
```

## Secrets
```shell
kubectl get -n kubevirt secret kubevirt-ca -o yaml
kubectl get -n kubevirt secret kubevirt-export-ca -o yaml
```

## ClusterInstancetypes
```shell
kubectl get virtualmachineclusterinstancetypes.instancetype.kubevirt.io -o yaml
```

## ClusterPreferences
```shell
kubectl get virtualmachineclusterpreferences.instancetype.kubevirt.io -o yaml
```

## Monitoring
### Service
```shell
kubectl get -n kubevirt service kubevirt-prometheus-metrics -o yaml
```
