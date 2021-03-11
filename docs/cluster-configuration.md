# Cluster Configuration
## Introduction
The HyperConverged Cluster allows modifying the KubeVirt cluster configuration by editing the HyperConverged Cluster CR
(Custom Resource).

The HyperConverged Cluster operator copies the cluster configuration values to the other operand's CRs.

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

## Listing Permitted Devices
Administrators can control which host devices are exposed and permitted to be used in the cluster. Permitted host
devices in the cluster will need to be allowlisted in KubeVirt CR by its `vendor:product` selector for PCI devices or
mediated device names. Use the `permittedHostDevices` field in order to manage the permitted host devices.

The `permittedHostDevices` field is an optional field under the HyperConverged `spec` field.

The `permittedHostDevices` field contains two optional arrays: the `pciHostDevices` and the `mediatedDevices` array.

HCO propagates these arrays as is to the KubeVirt custom resource; i.e. no merge is done, but a replacement.

The `pciHostDevices` array is an array of `PciHostDevice` objects. The fields of this object are:
* `pciVendorSelector` - a combination of a **`vendor_id:product_id`** required to identify a PCI device on a host.

   This identifier 10de:1eb8 can be found using `lspci`; for example:
   ```shell
   lspci -nnv | grep -i nvidia
   ```
  
* `resourceName` - name by which a device is advertised and being requested.
* `externalResourceProvider` - indicates that this resource is being provided by an external device plugin.
  
  KubeVirt in this case will only permit the usage of this device in the cluster but will leave the allocation and 
  monitoring to an external device plugin.

  **default**: `false`

The `mediatedDevices` array is an array of `MediatedDevice` objects. The fields of this object are:
* `mdevNameSelector` - name of a mediated device type required to identify a mediated device on a host.

   For example: mdev type nvidia-226 represents GRID T4-2A.
  
   The selector is matched against the content of `/sys/class/mdev_bus/$mdevUUID/mdev_type/name`.                   
* `resourceName` - name by which a device is advertised and being requested.    
* `externalResourceProvider` - indicates that this resource is being provided by an external device plugin.

  KubeVirt in this case will only permit the usage of this device in the cluster but will leave the allocation and
  monitoring to an external device plugin.
  
  **default**: `false`

### Permitted Host Devises Example
```yaml
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
spec:
  permittedHostDevices:
    pciHostDevices:
      - pciVendorSelector: "10DE:1EB8"
        resourceName: "nvidia.com/TU104GL_Tesla_T4"
        externalResourceProvider: true
      - pciVendorSelector: "8086:6F54"
        resourceName: "intel.com/qat"
    mediatedDevices:
      - mdevNameSelector: "GRID T4-1Q"
        resourceName: "nvidia.com/GRID_T4-1Q"
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
The patch’s path is relative to the `spec` field in each CR.

#### Examples
* The user wants to set the KubeVirt CR’s `spec.configuration.migrations.allowPostCopy` field to `true`. In order to do that, the following annotation should be added to the HyperConverged CR:
```yaml
metadata:
  annotations:
    kubevirt.kubevirt.io/jsonpatch: |-
      [
        {
          "op": "add",
          "path": "/configuration/migrations",
          "value": '{"allowPostCopy": "true"}'
        }
      ]
```
* The user wants to override the default URL used when uploading to a DataVolume, by setting the CDI CR's `spec.config.uploadProxyURLOverride` to `myproxy.example.com`. In order to do that, the following annotation should be added to the HyperConverged CR:
```yaml
metadata:
  annotations:
    containerizeddataimporter.kubevirt.io/jsonpatch: |-
      [
        {
          "op": "add",
          "path": "/config/uploadProxyURLOverride",
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
