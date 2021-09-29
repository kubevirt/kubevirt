
# API Docs

This Document documents the types introduced by the hyperconverged-cluster-operator to be consumed by users.

> Note this document is generated from code comments. When contributing a change to this document please do so by changing the code comments.

## Table of Contents
* [CertRotateConfigCA](#certrotateconfigca)
* [CertRotateConfigServer](#certrotateconfigserver)
* [HyperConverged](#hyperconverged)
* [HyperConvergedCertConfig](#hyperconvergedcertconfig)
* [HyperConvergedConfig](#hyperconvergedconfig)
* [HyperConvergedFeatureGates](#hyperconvergedfeaturegates)
* [HyperConvergedList](#hyperconvergedlist)
* [HyperConvergedObsoleteCPUs](#hyperconvergedobsoletecpus)
* [HyperConvergedSpec](#hyperconvergedspec)
* [HyperConvergedStatus](#hyperconvergedstatus)
* [HyperConvergedWorkloadUpdateStrategy](#hyperconvergedworkloadupdatestrategy)
* [LiveMigrationConfigurations](#livemigrationconfigurations)
* [MediatedHostDevice](#mediatedhostdevice)
* [OperandResourceRequirements](#operandresourcerequirements)
* [PciHostDevice](#pcihostdevice)
* [PermittedHostDevices](#permittedhostdevices)
* [StorageImportConfig](#storageimportconfig)
* [Version](#version)

## CertRotateConfigCA

CertRotateConfigCA contains the tunables for TLS certificates.

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| duration | The requested 'duration' (i.e. lifetime) of the Certificate. This should comply with golang's ParseDuration format (https://golang.org/pkg/time/#ParseDuration) | metav1.Duration | "48h0m0s" | false |
| renewBefore | The amount of time before the currently issued certificate's `notAfter` time that we will begin to attempt to renew the certificate. This should comply with golang's ParseDuration format (https://golang.org/pkg/time/#ParseDuration) | metav1.Duration | "24h0m0s" | false |

[Back to TOC](#table-of-contents)

## CertRotateConfigServer

CertRotateConfigServer contains the tunables for TLS certificates.

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| duration | The requested 'duration' (i.e. lifetime) of the Certificate. This should comply with golang's ParseDuration format (https://golang.org/pkg/time/#ParseDuration) | metav1.Duration | "24h0m0s" | false |
| renewBefore | The amount of time before the currently issued certificate's `notAfter` time that we will begin to attempt to renew the certificate. This should comply with golang's ParseDuration format (https://golang.org/pkg/time/#ParseDuration) | metav1.Duration | "12h0m0s" | false |

[Back to TOC](#table-of-contents)

## HyperConverged

HyperConverged is the Schema for the hyperconvergeds API

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| metadata |  | [metav1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#objectmeta-v1-meta) |  | false |
| spec |  | [HyperConvergedSpec](#hyperconvergedspec) | {"certConfig": {"ca": {"duration": "48h0m0s", "renewBefore": "24h0m0s"}, "server": {"duration": "24h0m0s", "renewBefore": "12h0m0s"}}, "featureGates": {"withHostPassthroughCPU": false, "sriovLiveMigration": true, "enableCommonBootImageImport": false}, "liveMigrationConfig": {"bandwidthPerMigration": "64Mi", "completionTimeoutPerGiB": 800, "parallelMigrationsPerCluster": 5, "parallelOutboundMigrationsPerNode": 2, "progressTimeout": 150}} | false |
| status |  | [HyperConvergedStatus](#hyperconvergedstatus) |  | false |

[Back to TOC](#table-of-contents)

## HyperConvergedCertConfig

HyperConvergedCertConfig holds the CertConfig entries for the HCO operands

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| ca | CA configuration - CA certs are kept in the CA bundle as long as they are valid | [CertRotateConfigCA](#certrotateconfigca) | {"duration": "48h0m0s", "renewBefore": "24h0m0s"} | false |
| server | Server configuration - Certs are rotated and discarded | [CertRotateConfigServer](#certrotateconfigserver) | {"duration": "24h0m0s", "renewBefore": "12h0m0s"} | false |

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
| withHostPassthroughCPU | Allow migrating a virtual machine with CPU host-passthrough mode. This should be enabled only when the Cluster is homogeneous from CPU HW perspective doc here | bool | false | true |
| sriovLiveMigration | Allow migrating a virtual machine with SRIOV interfaces. | bool | true | true |
| enableCommonBootImageImport | Opt-in to automatic delivery/updates of the common data import cron templates. There are two sources for the data import cron templates: hard coded list of common templates, and custom templates that can be added to the dataImportCronTemplates field. This feature gates only control the common templates. It is possible to use custom templates by adding them to the dataImportCronTemplates field. | bool | false | true |

[Back to TOC](#table-of-contents)

## HyperConvergedList

HyperConvergedList contains a list of HyperConverged

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| metadata |  | [metav1.ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#listmeta-v1-meta) |  | false |
| items |  | [][HyperConverged](#hyperconverged) |  | true |

[Back to TOC](#table-of-contents)

## HyperConvergedObsoleteCPUs

HyperConvergedObsoleteCPUs allows avoiding scheduling of VMs for obsolete CPU models

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| minCPUModel | MinCPUModel is the Minimum CPU model that is used for basic CPU features; e.g. Penryn or Haswell. The default value for this field is nil, but in KubeVirt, the default value is \"Penryn\", if nothing else is set. Use this field to override KubeVirt default value. | string |  | false |
| cpuModels | CPUModels is a list of obsolete CPU models. When the node-labeller obtains the list of obsolete CPU models, it eliminates those CPU models and creates labels for valid CPU models. The default values for this field is nil, however, HCO uses opinionated values, and adding values to this list will add them to the opinionated values. | []string |  | false |

[Back to TOC](#table-of-contents)

## HyperConvergedSpec

HyperConvergedSpec defines the desired state of HyperConverged

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| localStorageClassName | LocalStorageClassName the name of the local storage class. | string |  | false |
| infra | infra HyperConvergedConfig influences the pod configuration (currently only placement) for all the infra components needed on the virtualization enabled cluster but not necessarely directly on each node running VMs/VMIs. | [HyperConvergedConfig](#hyperconvergedconfig) |  | false |
| workloads | workloads HyperConvergedConfig influences the pod configuration (currently only placement) of components which need to be running on a node where virtualization workloads should be able to run. Changes to Workloads HyperConvergedConfig can be applied only without existing workload. | [HyperConvergedConfig](#hyperconvergedconfig) |  | false |
| featureGates | featureGates is a map of feature gate flags. Setting a flag to `true` will enable the feature. Setting `false` or removing the feature gate, disables the feature. | [HyperConvergedFeatureGates](#hyperconvergedfeaturegates) | {"withHostPassthroughCPU": false, "sriovLiveMigration": true, "enableCommonBootImageImport": false} | false |
| liveMigrationConfig | Live migration limits and timeouts are applied so that migration processes do not overwhelm the cluster. | [LiveMigrationConfigurations](#livemigrationconfigurations) | {"bandwidthPerMigration": "64Mi", "completionTimeoutPerGiB": 800, "parallelMigrationsPerCluster": 5, "parallelOutboundMigrationsPerNode": 2, "progressTimeout": 150} | false |
| permittedHostDevices | PermittedHostDevices holds information about devices allowed for passthrough | *[PermittedHostDevices](#permittedhostdevices) |  | false |
| certConfig | certConfig holds the rotation policy for internal, self-signed certificates | [HyperConvergedCertConfig](#hyperconvergedcertconfig) | {"ca": {"duration": "48h0m0s", "renewBefore": "24h0m0s"}, "server": {"duration": "24h0m0s", "renewBefore": "12h0m0s"}} | false |
| resourceRequirements | ResourceRequirements describes the resource requirements for the operand workloads. | *[OperandResourceRequirements](#operandresourcerequirements) |  | false |
| scratchSpaceStorageClass | Override the storage class used for scratch space during transfer operations. The scratch space storage class is determined in the following order: value of scratchSpaceStorageClass, if that doesn't exist, use the default storage class, if there is no default storage class, use the storage class of the DataVolume, if no storage class specified, use no storage class for scratch space | *string |  | false |
| vddkInitImage | VDDK Init Image eventually used to import VMs from external providers | *string |  | false |
| obsoleteCPUs | ObsoleteCPUs allows avoiding scheduling of VMs for obsolete CPU models | *[HyperConvergedObsoleteCPUs](#hyperconvergedobsoletecpus) |  | false |
| commonTemplatesNamespace | CommonTemplatesNamespace defines namespace in which common templates will be deployed. It overrides the default openshift namespace. | *string |  | false |
| storageImport | StorageImport contains configuration for importing containerized data | *[StorageImportConfig](#storageimportconfig) |  | false |
| workloadUpdateStrategy | WorkloadUpdateStrategy defines at the cluster level how to handle automated workload updates | *[HyperConvergedWorkloadUpdateStrategy](#hyperconvergedworkloadupdatestrategy) | {"workloadUpdateMethods": {"LiveMigrate"}, "batchEvictionSize": 10, "batchEvictionInterval": "1m0s"} | false |
| dataImportCronTemplates | DataImportCronTemplates holds list of data import cron templates (golden images) | []sspv1beta1.DataImportCronTemplate |  | false |

[Back to TOC](#table-of-contents)

## HyperConvergedStatus

HyperConvergedStatus defines the observed state of HyperConverged

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| conditions | Conditions describes the state of the HyperConverged resource. | []metav1.Condition |  | false |
| relatedObjects | RelatedObjects is a list of objects created and maintained by this operator. Object references will be added to this list after they have been created AND found in the cluster. | []corev1.ObjectReference |  | false |
| versions | Versions is a list of HCO component versions, as name/version pairs. The version with a name of \"operator\" is the HCO version itself, as described here: https://github.com/openshift/cluster-version-operator/blob/master/docs/dev/clusteroperator.md#version | Versions |  | false |
| observedGeneration | ObservedGeneration reflects the HyperConverged resource generation. If the ObservedGeneration is less than the resource generation in metadata, the status is out of date | int64 |  | false |
| dataImportSchedule | DataImportSchedule is the cron expression that is used in for the hard-coded data import cron templates. HCO generates the value of this field once and stored in the status field, so will survive restart. | string |  | false |

[Back to TOC](#table-of-contents)

## HyperConvergedWorkloadUpdateStrategy

HyperConvergedWorkloadUpdateStrategy defines options related to updating a KubeVirt install

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| workloadUpdateMethods | WorkloadUpdateMethods defines the methods that can be used to disrupt workloads during automated workload updates. When multiple methods are present, the least disruptive method takes precedence over more disruptive methods. For example if both LiveMigrate and Shutdown methods are listed, only VMs which are not live migratable will be restarted/shutdown. An empty list defaults to no automated workload updating. | []string | {"LiveMigrate"} | false |
| batchEvictionSize | BatchEvictionSize Represents the number of VMIs that can be forced updated per the BatchShutdownInteral interval | *int | 10 | false |
| batchEvictionInterval | BatchEvictionInterval Represents the interval to wait before issuing the next batch of shutdowns | *metav1.Duration | "1m0s" | false |

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
| mdevNameSelector | name of a mediated device type required to identify a mediated device on a host | string |  | true |
| resourceName | name by which a device is advertised and being requested | string |  | true |
| externalResourceProvider | indicates that this resource is being provided by an external device plugin | bool |  | false |
| disabled | HCO enforces the existence of several MediatedHostDevice objects. Set disabled field to true instead of remove these objects. | bool |  | false |

[Back to TOC](#table-of-contents)

## OperandResourceRequirements

OperandResourceRequirements is a list of resource requirements for the operand workloads pods

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| storageWorkloads | StorageWorkloads defines the resources requirements for storage workloads. It will propagate to the CDI custom resource | *corev1.ResourceRequirements |  | false |

[Back to TOC](#table-of-contents)

## PciHostDevice

PciHostDevice represents a host PCI device allowed for passthrough

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| pciDeviceSelector | a combination of a vendor_id:product_id required to identify a PCI device on a host. | string |  | true |
| resourceName | name by which a device is advertised and being requested | string |  | true |
| externalResourceProvider | indicates that this resource is being provided by an external device plugin | bool |  | false |
| disabled | HCO enforces the existence of several PciHostDevice objects. Set disabled field to true instead of remove these objects. | bool |  | false |

[Back to TOC](#table-of-contents)

## PermittedHostDevices

PermittedHostDevices holds information about devices allowed for passthrough

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| pciHostDevices |  | [][PciHostDevice](#pcihostdevice) |  | false |
| mediatedDevices |  | [][MediatedHostDevice](#mediatedhostdevice) |  | false |

[Back to TOC](#table-of-contents)

## StorageImportConfig

StorageImportConfig contains configuration for importing containerized data

| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| insecureRegistries | InsecureRegistries is a list of image registries URLs that are not secured. Setting an insecure registry URL in this list allows pulling images from this registry. | []string |  | false |

[Back to TOC](#table-of-contents)

## Version



| Field | Description | Scheme | Default | Required |
| ----- | ----------- | ------ | -------- |-------- |
| name |  | string |  | false |
| version |  | string |  | false |

[Back to TOC](#table-of-contents)
