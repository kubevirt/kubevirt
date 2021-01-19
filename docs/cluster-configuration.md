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

### hotplugVolumes Feature Gate
Set the `hotplugVolumes` feature gate in order to allow attaching a data volume to a running VMI.

### withHostModelCPU Feature Gate
Set the `withHostModelCPU`  feature gate in order to enable support migration for VMs with host-model CPU mode

Additional information: [LibvirtXMLCPUModel](https://wiki.openstack.org/wiki/LibvirtXMLCPUModel)

### withHostPassthroughCPU Feature Gate
Set the `withHostPassthroughCPU`  feature gate in order to allow migrating a virtual machine with CPU host-passthrough mode. 

Additional information: [LibvirtXMLCPUModel](https://wiki.openstack.org/wiki/LibvirtXMLCPUModel)

**note**: This should be enabled only when the Cluster is homogeneous from CPU HW perspective doc here

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
    hotplugVolumes: true
    withHostModelCPU: true
    withHostPassthroughCPU: true
```
