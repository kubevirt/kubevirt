
# API Docs

This Document documents the types introduced by the hyperconverged-cluster-operator to be consumed by users.

> Note this document is generated from code comments. When contributing a change to this document please do so by changing the code comments.

## Table of Contents
* [HyperConverged](#hyperconverged)
* [HyperConvergedConfig](#hyperconvergedconfig)
* [HyperConvergedFeatureGates](#hyperconvergedfeaturegates)
* [HyperConvergedList](#hyperconvergedlist)
* [HyperConvergedSpec](#hyperconvergedspec)
* [HyperConvergedStatus](#hyperconvergedstatus)
* [Version](#version)

## HyperConverged

HyperConverged is the Schema for the hyperconvergeds API

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| metadata |  | [metav1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#objectmeta-v1-meta) |  | false |
| spec |  | [HyperConvergedSpec](#hyperconvergedspec) |  | false |
| status |  | [HyperConvergedStatus](#hyperconvergedstatus) |  | false |

[Back to TOC](#table-of-contents)

## HyperConvergedConfig

HyperConvergedConfig defines a set of configurations to pass to components

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| nodePlacement | NodePlacement describes node scheduling configuration. | *[sdkapi.NodePlacement](https://github.com/kubevirt/controller-lifecycle-operator-sdk/blob/bbf16167410b7a781c7b08a3f088fc39551c7a00/pkg/sdk/api/types.go#L49) |  | false |

[Back to TOC](#table-of-contents)

## HyperConvergedFeatureGates

HyperConvergedFeatureGates is a set of optional feature gates to enable or disable new features that are not enabled by default yet.

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| sriovLiveMigration | Allow migrating a virtual machine with SRIOV interfaces. When enabled virt-launcher pods of virtual machines with SRIOV interfaces run with CAP_SYS_RESOURCE capability. This may degrade virt-launcher security. | *bool | false | false |
| hotplugVolumes | Allow attaching a data volume to a running VMI | *bool | false | false |
| gpu | Allow assigning GPU and vGPU devices to virtual machines | *bool | false | false |
| hostDevices | Allow assigning host devices to virtual machines | *bool | false | false |
| withHostPassthroughCPU | Allow migrating a virtual machine with CPU host-passthrough mode. This should be enabled only when the Cluster is homogeneous from CPU HW perspective doc here | *bool | false | false |
| withHostModelCPU | Support migration for VMs with host-model CPU mode | *bool | true | false |
| hypervStrictCheck | Enable HyperV strict host checking for HyperV enlightenments Defaults to true, even when HyperConvergedFeatureGates is empty | *bool | true | false |

[Back to TOC](#table-of-contents)

## HyperConvergedList

HyperConvergedList contains a list of HyperConverged

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| metadata |  | [metav1.ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#listmeta-v1-meta) |  | false |
| items |  | [][HyperConverged](#hyperconverged) |  | true |

[Back to TOC](#table-of-contents)

## HyperConvergedSpec

HyperConvergedSpec defines the desired state of HyperConverged

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| localStorageClassName | LocalStorageClassName the name of the local storage class. | string |  | false |
| infra | infra HyperConvergedConfig influences the pod configuration (currently only placement) for all the infra components needed on the virtualization enabled cluster but not necessarely directly on each node running VMs/VMIs. | [HyperConvergedConfig](#hyperconvergedconfig) |  | false |
| workloads | workloads HyperConvergedConfig influences the pod configuration (currently only placement) of components which need to be running on a node where virtualization workloads should be able to run. Changes to Workloads HyperConvergedConfig can be applied only without existing workload. | [HyperConvergedConfig](#hyperconvergedconfig) |  | false |
| featureGates | featureGates is a map of feature gate flags. Setting a flag to `true` will enable the feature. Setting `false` or removing the feature gate, disables the feature. | *[HyperConvergedFeatureGates](#hyperconvergedfeaturegates) | {sriovLiveMigration: false, hotplugVolumes: false, gpu: false, hostDevices: false, withHostPassthroughCPU: false, withHostModelCPU: true, hypervStrictCheck: true} | false |
| version | operator version | string |  | false |

[Back to TOC](#table-of-contents)

## HyperConvergedStatus

HyperConvergedStatus defines the observed state of HyperConverged

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| conditions | Conditions describes the state of the HyperConverged resource. | []conditionsv1.Condition |  | false |
| relatedObjects | RelatedObjects is a list of objects created and maintained by this operator. Object references will be added to this list after they have been created AND found in the cluster. | []corev1.ObjectReference |  | false |
| versions | Versions is a list of HCO component versions, as name/version pairs. The version with a name of \"operator\" is the HCO version itself, as described here: https://github.com/openshift/cluster-version-operator/blob/master/docs/dev/clusteroperator.md#version | Versions |  | false |

[Back to TOC](#table-of-contents)

## Version



| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| name |  | string |  | false |
| version |  | string |  | false |

[Back to TOC](#table-of-contents)
