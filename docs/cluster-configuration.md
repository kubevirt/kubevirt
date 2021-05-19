# Cluster Configuration
## Introduction
The HyperConverged Cluster allows modifying the KubeVirt cluster configuration by editing the HyperConverged Cluster CR
(Custom Resource).

The HyperConverged Cluster operator copies the cluster configuration values to the other operand's CRs.

The Hyperconverged Cluster Operator configures kubevirt and its supporting operators in an opinionated way and overwrites its operands when there is an unexpected change to them.
Users are expected to not modify the operands directly. The HyperConverged custom resource is the source of truth for the configuration.

To make it more visible and clear for end users, the Hyperconverged Cluster Operator will count the number of these revert actions in a metric named kubevirt_hco_out_of_band_modifications_count.
According to the value of that metric in the last 10 minutes, an alert named KubevirtHyperconvergedClusterOperatorCRModification will be eventually fired:
```
Labels
    alertname=KubevirtHyperconvergedClusterOperatorCRModification
    component_name=kubevirt-kubevirt-hyperconverged
    severity=warning
```
The alert is supposed to resolve after 10 minutes if there isn't a manual intervention to operands in the last 10 minutes.

***Note***: The cluster configurations are supported only in API version `v1beta1` or higher.
## Infra and Workloads Configuration
Some configurations are done separately to Infra and Workloads. The CR's Spec object contains the `infra` and the 
`workloads` objects.

The structures of the `infra` and the `workloads` objects are the same. The HyperConverged Cluster operator will update 
the other operator CRs, according to the specific CR structure. The meaning is if, for example, the other CR does not 
support Infra cluster configurations, but only Workloads configurations, the HyperConverged Cluster operator will only
copy the Workloads configurations to this operator's CR.

Below are the cluster configuration details. Currently, only "Node Placement" configuration is supported.

### Node Placement
Kubernetes lets the cluster admin influence node placement in several ways, see 
https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/ for a general overview.

The HyperConverged Cluster's CR is the single entry point to let the cluster admin influence the placement of all the pods directly and indirectly managed by the HyperConverged Cluster Operator.

The `nodePlacement` object is an optional field in the HyperConverged Cluster's CR, under `spec.infra` and `spec.workloads`
fields.

***Note***: The HyperConverged Cluster operator does not allow modifying of the workloads' node placement configurations if there are already
existing virtual machines or data volumes. 

The `nodePlacement` object contains the following fields:
* `nodeSelector` is the node selector applied to the relevant kind of pods. It specifies a map of key-value pairs: for 
the pod to be eligible to run on a node,	the node must have each of the indicated key-value pairs as labels 	(it can
have additional labels as well). See https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#nodeselector.
* `affinity` enables pod affinity/anti-affinity placement expanding the types of constraints
that can be expressed with nodeSelector.
affinity is going to be applied to the relevant kind of pods in parallel with nodeSelector
See https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity.
* `tolerations` is a list of tolerations applied to the relevant kind of pods.
See https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/ for more info.

#### Operators placement
The HyperConverged Cluster Operator and the operators for its component are supposed to be deployed by the Operator Lifecycle Manager (OLM).
Thus, the HyperConverged Cluster Operator is not going to directly influence its own placement but that should be influenced by the OLM.
The cluster admin indeed is allowed to influence the placement of the Pods directly created by the OLM configuring a [nodeSelector](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/subscription-config.md#nodeselector) or [tolerations](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/subscription-config.md#tolerations) directly on the OLM subscription object.

#### Node Placement Examples
* Place the infra resources on nodes labeled with "nodeType = infra", and workloads in nodes labeled with "nodeType = nested-virtualization", using node selector:
  ```yaml
  ...
  spec:
    infra:
      nodePlacement:
        nodeSelector:
          nodeType: infra
    workloads:
      nodePlacement:
        nodeSelector:
          nodeType: nested-virtualization
  ```
* Place the infra resources on nodes labeled with "nodeType = infra", and workloads in nodes labeled with 
"nodeType = nested-virtualization", preferring nodes with more than 8 CPUs, using affinity:
  ```yaml
  ...
  spec:
    infra:
      nodePlacement:
        affinity:
          nodeAffinity:
            requiredDuringSchedulingIgnoredDuringExecution:
              nodeSelectorTerms:
              - matchExpressions:
                - key: nodeType
                  operator: In
                  values:
                  - infra
    workloads:
      nodePlacement:
        affinity:
          nodeAffinity:
            requiredDuringSchedulingIgnoredDuringExecution:
              nodeSelectorTerms:
              - matchExpressions:
                - key: nodeType
                  operator: In
                  values:
                  - nested-virtualization
            preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 1
              preference:
                matchExpressions:
                - key: my-cloud.io/num-cpus
                  operator: gt
                  values:
                  - 8
  ```
* In this example, there are several nodes that are saved for KubeVirt resources (e.g. VMs), already set with the 
`key=kubevirt:NoSchedule` taint. This taint will prevent any scheduling to these nodes, except for pods with the matching 
tolerations.
  ```yaml
  ...
  spec:
    workloads:
      nodePlacement:
        tolerations:
        - key: "key"
          operator: "Equal"
          value: "kubevirt"
          effect: "NoSchedule"
  ```

## FeatureGates
The `featureGates` field is an optional set of optional boolean feature enabler. The features in this list are advanced 
or new features that are not enabled by default.

To enable a feature, add its name to the `featureGates` list and set it to `true`. Missing or `false` feature gates 
disables the feature.

### withHostPassthroughCPU Feature Gate
Set the `withHostPassthroughCPU` feature gate in order to allow migrating a virtual machine with CPU host-passthrough
mode. This can provide slightly better CPU performance, but should be enabled only when the Cluster is homogeneous from
CPU HW perspective.

**Default**: `false`

Additional information: [LibvirtXMLCPUModel](https://wiki.openstack.org/wiki/LibvirtXMLCPUModel)

### sriovLiveMigration Feature Gate
Set the `sriovLiveMigration` feature gate in order to allow migrating a virtual machine with SRIOV interfaces.
When enabled virt-launcher pods of virtual machines with SRIOV interfaces run with CAP_SYS_RESOURCE capability.
This may degrade virt-launcher security.

**Default**: `false`

### Feature Gates Example
```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  infra: {}
  workloads: {}
  featureGates:
    withHostPassthroughCPU: true
    sriovLiveMigration: true
```

## Live Migration Configurations

Set the live migration configurations by modifying the fields in the `liveMigrationConfig` under the `spec` field

### bandwidthPerMigration

Bandwidth limit of each migration, in MiB/s. The format is a number and with the `Mi` suffix, e.g. `64Mi`.

**default**: 64Mi

### completionTimeoutPerGiB

The migration will be canceled if it has not completed in this time, in seconds per GiB of memory. For example, a
virtual machine instance with 6GiB memory will timeout if it has not completed migration in 4800 seconds. If the
Migration Method is BlockMigration, the size of the migrating disks is included in the calculation. The format is a
number.

**default**: 800

### parallelMigrationsPerCluster

Number of migrations running in parallel in the cluster. The format is a number.

**default**: 5

### parallelOutboundMigrationsPerNode

Maximum number of outbound migrations per node. The format is a number.

**default**: 2

### progressTimeout:

The migration will be canceled if memory copy fails to make progress in this time, in seconds. The format is a number.

**default**: 150

### Example

```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  liveMigrationConfig:
    bandwidthPerMigration: 64Mi
    completionTimeoutPerGiB: 800
    parallelMigrationsPerCluster: 5
    parallelOutboundMigrationsPerNode: 2
    progressTimeout: 150
```

## Listing Permitted Host Devices
Administrators can control which host devices are exposed and permitted to be used in the cluster. Permitted host
devices in the cluster will need to be allowlisted in KubeVirt CR by its `vendor:product` selector for PCI devices or
mediated device names. Use the `permittedHostDevices` field in order to manage the permitted host devices.

The `permittedHostDevices` field is an optional field under the HyperConverged `spec` field.

The `permittedHostDevices` field contains two optional arrays: the `pciHostDevices` and the `mediatedDevices` array.

HCO propagates these arrays as is to the KubeVirt custom resource; i.e. no merge is done, but a replacement.

The `pciHostDevices` array is an array of `PciHostDevice` objects. The fields of this object are:
* `pciDeviceSelector` - a combination of a **`vendor_id:product_id`** required to identify a PCI device on a host.

   This identifier 10de:1eb8 can be found using `lspci`; for example:
   ```shell
   lspci -nnv | grep -i nvidia
   ```
  
* `resourceName` - name by which a device is advertised and being requested.
* `externalResourceProvider` - indicates that this resource is being provided by an external device plugin.

  KubeVirt in this case will only permit the usage of this device in the cluster but will leave the allocation and
  monitoring to an external device plugin.

  **default**: `false`
* `disabled` - set to `true` to disable default host devices, because these device cannot be removed

The `mediatedDevices` array is an array of `MediatedDevice` objects. The fields of this object are:
* `mdevNameSelector` - name of a mediated device type required to identify a mediated device on a host.

   For example: mdev type nvidia-226 represents GRID T4-2A.

  The selector is matched against the content of `/sys/class/mdev_bus/$mdevUUID/mdev_type/name`.
* `resourceName` - name by which a device is advertised and being requested.
* `externalResourceProvider` - indicates that this resource is being provided by an external device plugin.

  KubeVirt in this case will only permit the usage of this device in the cluster but will leave the allocation and
  monitoring to an external device plugin.

  **default**: `false`

### Permitted Host Devices Default Values

HCO enforces the existence of two PCI Host Devices in the list:

```yaml
pciHostDevices:
- pciDeviceSelector: "10DE:1DB6"
  resourceName: "nvidia.com/GV100GL_Tesla_V100",
- pciDeviceSelector: "10DE:1EB8",
  resourceName: "nvidia.com/TU104GL_Tesla_T4",
```

It is possible to add more devices, but these two devices must be in the list. If you need to remove them, use the
`disabled` filed instead of deleting them.

### Permitted Host Devices Example

In this example, we're adding a new device in addition to two default ones, and disabling the default "
nvidia.com/TU104GL_Tesla_T4" device:

```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  permittedHostDevices:
    pciHostDevices:
    - pciDeviceSelector: "10DE:1DB6"
      resourceName: "nvidia.com/GV100GL_Tesla_V100",
    - pciDeviceSelector: "10DE:1EB8"
      resourceName: "nvidia.com/TU104GL_Tesla_T4"
      disabled: true
    - pciDeviceSelector: "8086:6F54"
      resourceName: "intel.com/qat"
      externalResourceProvider: true
    mediatedDevices:
    - mdevNameSelector: "GRID T4-1Q"
      resourceName: "nvidia.com/GRID_T4-1Q"
```

## Storage Class for Scratch Space

Administrators can Override the storage class used for scratch space during transfer operations by setting the
`scratchSpaceStorageClass` field under the HyperConverged `spec` field.

The scratch space storage class is determined in the following order:

value of scratchSpaceStorageClass, if that doesn't exist, use the default storage class, if there is no default storage
class, use the storage class of the DataVolume, if no storage class specified, use no storage class for scratch space

### Storage Class for Scratch Space Example

```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  scratchSpaceStorageClass: aStorageClassName
```

## Storage Resource Configurations

The administrator can limit storage workloads resources and to require minimal resources. Use the `resourceRequirements`
field under the HyperConverged `spec` filed. Add the `storageWorkloads` field under the `resourceRequirements`. The
content of the `storageWorkloads` field is
the [standard kubernetes resource configuration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#resourcerequirements-v1-core)
.

### Storage Resource Configurations Example

```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  resourceRequirements:
    storageWorkloads:
      limits:
        cpu: "500m"
        memory: "2Gi"
      requests:
        cpu: "250m"
        memory: "1Gi"
```

## CPU Plugin Configurations
You can schedule a virtual machine (VM) on a node where the CPU model and policy attribute of the VM are compatible with
the CPU models and policy attributes that the node supports. By specifying a list of obsolete CPU models in a the 
HyperConverged custom resource, you can exclude them from the list of labels created for CPU models.

Through the process of iteration, the list of base CPU features in the minimum CPU model are eliminated from the list of
labels generated for the node. For example, an environment might have two supported CPU models: `Penryn` and `Haswell`. 

Use the `spec.obsoleteCPUs` to configure the CPU plugin. Add the obsolete CPU list under `spec.obsoleteCPUs.cpuModels`,
and the minCPUModel as the value of `spec.obsoleteCPUs.minCPUModel`.

The default value for the `spec.obsoleteCPUs.minCPUModel` field in KubeVirt is `"Penryn"`, but it won't be visible if 
missing in the CR. The default value for the `spec.obsoleteCPUs.cpuModels` field is hardcoded predefined list and is not
visible. You can add new CPU models to the list, but can't remove CPU models from the predefined list. The predefined list
is not visible in the HyperConverged CR.

The hard-coded predefined list of obsolete CPU modes is:
* `486`
* `pentium`
* `pentium2`
* `pentium3`
* `pentiumpro`
* `coreduo`
* `n270`
* `core2duo`
* `Conroe`
* `athlon`
* `phenom`
* `qemu64`
* `qemu32`
* `kvm64`
* `kvm32`

You don't need to add a CPU model to the `spec.obsoleteCPUs.cpuModels` field if it is in this list. 

### CPU Plugin Configurations Example
```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  obsoleteCPUs:
    cpuModels:
      - "486"
      - "pentium"
      - "pentium2"
      - "pentium3"
      - "pentiumpro"
    minCPUModel: "Penryn"
```

## Enable eventual launcher updates by default
us the HyperConverged `spec.workloadUpdateStrategy` object to define how to handle automated workload updates at the cluster
level.

The `workloadUpdateStrategy` fields are:
* `batchEvictionInterval` - BatchEvictionInterval Represents the interval to wait before issuing the next batch of
  shutdowns. 
  
  The Default value is `1m`
  
* `batchEvictionSize` - Represents the number of VMIs that can be forced updated per the BatchShutdownInteral interval
  
  The default value is `10`

* `workloadUpdateMethods` - defines the methods that can be used to disrupt workloads
  during automated workload updates.
  
  When multiple methods are present, the least disruptive method takes
  precedence over more disruptive methods. For example if both LiveMigrate and Shutdown
  methods are listed, only VMs which are not live migratable will be restarted/shutdown.
  
  An empty list defaults to no automated workload updating.

  The default values are `LiveMigrate` and `Evict`.

### workloadUpdateStrategy example
```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  workloadUpdateStrategy:
    workloadUpdateMethods:
    - LiveMigrate
    - Evict
    batchEvictionSize: 10
    batchEvictionInterval: "1m"
```

## Insecure Registries for Imported Data containerized Images
If there is a need to import data images from an insecure registry, these registries should be added to the
`insecureRegistries` field under the `storageImport` in the `HyperConverged`'s `spec` field.

### Insecure Registry Example
```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  storageImport:
    insecureRegistries:
      - "private-registry-example-1:5000"
      - "private-registry-example-2:5000"
      ...
```


## Configurations via Annotations

In addition to `featureGates` field in HyperConverged CR's spec, the user can set annotations in the HyperConverged CR
to unfold more configuration options.  
**Warning:** Annotations are less formal means of cluster configuration and may be dropped without the same deprecation
process of a regular API, such as in the `spec` section.

### OvS Opt-In Annotation

Starting from HCO version 1.3.0, OvS CNI support is disabled by default on new installations.  
In order to enable the deployment of OvS CNI DaemonSet on all _workload_ nodes, an annotation of `deployOVS: true` must
be set on HyperConverged CR.  
It can be set while creating the HyperConverged custom resource during the initial deployment, or during run time.

* To enable OvS CNI on the cluster, the HyperConverged CR should be similar to:

```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  annotations:
    deployOVS: "true"
...
```
* OvS CNI can also be enabled during run time of HCO, by annotating its CR:
```
kubectl annotate HyperConverged kubevirt-hyperconverged -n kubevirt-hyperconverged deployOVS=true --overwrite
```

If HCO was upgraded to 1.3.0 from a previous version, the annotation will be added as `true` and OvS will be deployed.  
Subsequent upgrades to newer versions will preserve the state from previous version, i.e. OvS will be deployed in the upgraded version if and only if it was deployed in the previous one.

### jsonpatch Annotations
HCO enables users to modify the operand CRs directly using jsonpatch annotations in HyperConverged CR.  
Modifications done to CRs using jsonpatch annotations won't be reconciled back by HCO to the opinionated defaults.  
The following annotations are supported in the HyperConverged CR:
* `kubevirt.kubevirt.io/jsonpatch` - for KubeVirt configurations
* `containerizeddataimporter.kubevirt.io/jsonpatch` - for CDI configurations
* `networkaddonsconfig.kubevirt.io/jsonpatch` - for CNAO configurations

The content of the annotation will be a json array of patch objects, as defined in [RFC6902](https://tools.ietf.org/html/rfc6902).

#### Examples
* The user wants to set the KubeVirt CR’s `spec.configuration.migrations.allowPostCopy` field to `true`. In order to do that, the following annotation should be added to the HyperConverged CR:
```yaml
metadata:
  annotations:
    kubevirt.kubevirt.io/jsonpatch: |-
      [
        {
          "op": "add",
          "path": "/spec/configuration/migrations",
          "value": {"allowPostCopy": true}
        }
      ]
```

From CLI it will be:
```bash
$ kubectl annotate --overwrite -n kubevirt-hyperconverged hco kubevirt-hyperconverged \
  kubevirt.kubevirt.io/jsonpatch='[{"op": "add", \
    "path": "/spec/configuration/migrations", \
    "value": {"allowPostCopy": true} }]'
hyperconverged.hco.kubevirt.io/kubevirt-hyperconverged annotated
$ kubectl get kubevirt -n kubevirt-hyperconverged kubevirt-kubevirt-hyperconverged -o json \
  | jq '.spec.configuration.migrations.allowPostCopy'
true
$ kubectl annotate --overwrite -n kubevirt-hyperconverged hco kubevirt-hyperconverged \
  kubevirt.kubevirt.io/jsonpatch='[{"op": "add", \
    "path": "/spec/configuration/migrations", \
    "value": {"allowPostCopy": false} }]'
hyperconverged.hco.kubevirt.io/kubevirt-hyperconverged annotated
$ kubectl get kubevirt -n kubevirt-hyperconverged kubevirt-kubevirt-hyperconverged -o json \
  | jq '.spec.configuration.migrations.allowPostCopy'
false
$ kubectl get hco -n kubevirt-hyperconverged  kubevirt-hyperconverged -o json \
  | jq '.status.conditions[] | select(.type == "TaintedConfiguration")'
{
  "lastHeartbeatTime": "2021-03-24T17:25:49Z",
  "lastTransitionTime": "2021-03-24T11:33:11Z",
  "message": "Unsupported feature was activated via an HCO annotation",
  "reason": "UnsupportedFeatureAnnotation",
  "status": "True",
  "type": "TaintedConfiguration"
}
```

* The user wants to override the default URL used when uploading to a DataVolume, by setting the CDI CR's `spec.config.uploadProxyURLOverride` to `myproxy.example.com`. In order to do that, the following annotation should be added to the HyperConverged CR:
```yaml
metadata:
  annotations:
    containerizeddataimporter.kubevirt.io/jsonpatch: |-
      [
        {
          "op": "add",
          "path": "/spec/config/uploadProxyURLOverride",
          "value": "myproxy.example.com"
        }
      ]
```

**_Note:_** The full configurations options for Kubevirt, CDI and CNAO which are available on the cluster, can be explored by using `kubectl explain <resource name>.spec`. For example:  
```yaml
$ kubectl explain kv.spec
KIND:     KubeVirt
VERSION:  kubevirt.io/v1

RESOURCE: spec <Object>

DESCRIPTION:
     <empty>

FIELDS:
   certificateRotateStrategy	<Object>

   configuration	<Object>
     holds kubevirt configurations. same as the virt-configMap

   customizeComponents	<Object>

   imagePullPolicy	<string>
     The ImagePullPolicy to use.

   imageRegistry	<string>
     The image registry to pull the container images from Defaults to the same
     registry the operator's container image is pulled from.
  
  <truncated>
```

To inspect lower-level objects onder `spec`, they can be specified in `kubectl explain`, recursively. e.g.  
```yaml
$ kubectl explain kv.spec.configuration.network
KIND:     KubeVirt
VERSION:  kubevirt.io/v1

RESOURCE: network <Object>

DESCRIPTION:
     NetworkConfiguration holds network options

FIELDS:
   defaultNetworkInterface	<string>

   permitBridgeInterfaceOnPodNetwork	<boolean>

   permitSlirpInterface	<boolean>
```

* To explore kubevirt configuration options, use `kubectl explain kv.spec`
* To explore CDI configuration options, use `kubectl explain cdi.spec`
* To explore CNAO configuration options, use `kubectl explain networkaddonsconfig.spec`

### WARNING
Using the jsonpatch annotation feature incorrectly might lead to unexpected results and could potentially render the Kubevirt-Hyperconverged system unstable.  
The jsonpatch annotation feature is particularly dangerous when upgrading Kubevirt-Hyperconverged, as the structure or the semantics of the underlying components' CR might be changed. Please remove any jsonpatch annotation usage prior the upgrade, to avoid any potential issues.
**USE WITH CAUTION!**
