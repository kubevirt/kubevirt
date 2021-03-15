
# API Docs

This Document documents the types introduced by the hyperconverged-cluster-operator to be consumed by users.

> Note this document is generated from code comments. When contributing a change to this document please do so by changing the code comments.

## Table of Contents
* [CertRotateConfig](#certrotateconfig)
* [HyperConverged](#hyperconverged)
* [HyperConvergedCertConfig](#hyperconvergedcertconfig)
* [HyperConvergedConfig](#hyperconvergedconfig)
* [HyperConvergedFeatureGates](#hyperconvergedfeaturegates)
* [HyperConvergedList](#hyperconvergedlist)
* [HyperConvergedSpec](#hyperconvergedspec)
* [HyperConvergedStatus](#hyperconvergedstatus)
* [LiveMigrationConfigurations](#livemigrationconfigurations)
* [MediatedHostDevice](#mediatedhostdevice)
* [PciHostDevice](#pcihostdevice)
* [PermittedHostDevices](#permittedhostdevices)
* [Version](#version)

## CertRotateConfig

CertConfig contains the tunables for TLS certificates.

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| duration | The requested 'duration' (i.e. lifetime) of the Certificate. This should comply with golang's ParseDuration format (https://golang.org/pkg/time/#ParseDuration) | metav1.Duration |  | false |
| renewBefore | The amount of time before the currently issued certificate's `notAfter` time that we will begin to attempt to renew the certificate. This should comply with golang's ParseDuration format (https://golang.org/pkg/time/#ParseDuration) | metav1.Duration |  | false |

[Back to TOC](#table-of-contents)

## HyperConverged

HyperConverged is the Schema for the hyperconvergeds API

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| metadata |  | [metav1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#objectmeta-v1-meta) |  | false |
| spec |  | [HyperConvergedSpec](#hyperconvergedspec) |  | false |
| status |  | [HyperConvergedStatus](#hyperconvergedstatus) |  | false |

[Back to TOC](#table-of-contents)

## HyperConvergedCertConfig

HyperConvergedCertConfig holds the CertConfig entries for the HCO operands

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| ca | CA configuration - CA certs are kept in the CA bundle as long as they are valid | *[CertRotateConfig](#certrotateconfig) | {duration: "48h", renewBefore: "24h"} | false |
| server | Server configuration - Certs are rotated and discarded | *[CertRotateConfig](#certrotateconfig) | {duration: "24h", renewBefore: "12h"} | false |

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
| withHostPassthroughCPU | Allow migrating a virtual machine with CPU host-passthrough mode. This should be enabled only when the Cluster is homogeneous from CPU HW perspective doc here | FeatureGate | false | false |

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
| featureGates | featureGates is a map of feature gate flags. Setting a flag to `true` will enable the feature. Setting `false` or removing the feature gate, disables the feature. | [HyperConvergedFeatureGates](#hyperconvergedfeaturegates) |  | false |
| liveMigrationConfig | Live migration limits and timeouts are applied so that migration processes do not overwhelm the cluster. | [LiveMigrationConfigurations](#livemigrationconfigurations) |  | false |
| permittedHostDevices | PermittedHostDevices holds inforamtion about devices allowed for passthrough | *[PermittedHostDevices](#permittedhostdevices) |  | false |
| certConfig | certConfig holds the rotation policy for internal, self-signed certificates | *[HyperConvergedCertConfig](#hyperconvergedcertconfig) |  | false |
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

## LiveMigrationConfigurations

LiveMigrationConfigurations - Live migration limits and timeouts are applied so that migration processes do not overwhelm the cluster.

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| parallelMigrationsPerCluster | Number of migrations running in parallel in the cluster. | *uint32 | 5 | false |
| parallelOutboundMigrationsPerNode | Maximum number of outbound migrations per node. | *uint32 | 2 | false |
| bandwidthPerMigration | Bandwidth limit of each migration, in MiB/s. | *string | "64Mi" | false |
| completionTimeoutPerGiB | The migration will be canceled if it has not completed in this time, in seconds per GiB of memory. For example, a virtual machine instance with 6GiB memory will timeout if it has not completed migration in 4800 seconds. If the Migration Method is BlockMigration, the size of the migrating disks is included in the calculation. | *int64 | 800 | false |
| progressTimeout | The migration will be canceled if memory copy fails to make progress in this time, in seconds. | *int64 | 150 | false |

[Back to TOC](#table-of-contents)

## MediatedHostDevice

MediatedHostDevice represents a host mediated device allowed for passthrough

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| mdevNameSelector |  | string |  | true |
| resourceName |  | string |  | true |
| externalResourceProvider |  | bool |  | false |

[Back to TOC](#table-of-contents)

## PciHostDevice

PciHostDevice represents a host PCI device allowed for passthrough

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| pciVendorSelector |  | string |  | true |
| resourceName |  | string |  | true |
| externalResourceProvider |  | bool |  | false |

[Back to TOC](#table-of-contents)

## PermittedHostDevices

PermittedHostDevices holds inforamtion about devices allowed for passthrough

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| pciHostDevices |  | [][PciHostDevice](#pcihostdevice) |  | false |
| mediatedDevices |  | [][MediatedHostDevice](#mediatedhostdevice) |  | false |

[Back to TOC](#table-of-contents)

## Version



| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| name |  | string |  | false |
| version |  | string |  | false |

[Back to TOC](#table-of-contents)
