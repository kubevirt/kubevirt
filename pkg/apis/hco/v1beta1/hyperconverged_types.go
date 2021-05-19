package v1beta1

import (
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/api"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

// HyperConvergedName is the name of the HyperConverged resource that will be reconciled
const HyperConvergedName = "kubevirt-hyperconverged"

// HyperConvergedSpec defines the desired state of HyperConverged
// +k8s:openapi-gen=true
type HyperConvergedSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html

	// LocalStorageClassName the name of the local storage class.
	LocalStorageClassName string `json:"localStorageClassName,omitempty"`

	// infra HyperConvergedConfig influences the pod configuration (currently only placement)
	// for all the infra components needed on the virtualization enabled cluster
	// but not necessarely directly on each node running VMs/VMIs.
	// +optional
	Infra HyperConvergedConfig `json:"infra,omitempty"`

	// workloads HyperConvergedConfig influences the pod configuration (currently only placement) of components
	// which need to be running on a node where virtualization workloads should be able to run.
	// Changes to Workloads HyperConvergedConfig can be applied only without existing workload.
	// +optional
	Workloads HyperConvergedConfig `json:"workloads,omitempty"`

	// featureGates is a map of feature gate flags. Setting a flag to `true` will enable
	// the feature. Setting `false` or removing the feature gate, disables the feature.
	// +kubebuilder:default={"withHostPassthroughCPU": false, "sriovLiveMigration": false}
	// +optional
	FeatureGates HyperConvergedFeatureGates `json:"featureGates,omitempty"`

	// Live migration limits and timeouts are applied so that migration processes do not
	// overwhelm the cluster.
	// +kubebuilder:default={"bandwidthPerMigration": "64Mi", "completionTimeoutPerGiB": 800, "parallelMigrationsPerCluster": 5, "parallelOutboundMigrationsPerNode": 2, "progressTimeout": 150}
	// +optional
	LiveMigrationConfig LiveMigrationConfigurations `json:"liveMigrationConfig,omitempty"`

	// PermittedHostDevices holds information about devices allowed for passthrough
	// +optional
	PermittedHostDevices *PermittedHostDevices `json:"permittedHostDevices,omitempty"`

	// certConfig holds the rotation policy for internal, self-signed certificates
	// +kubebuilder:default={"ca": {"duration": "48h0m0s", "renewBefore": "24h0m0s"}, "server": {"duration": "24h0m0s", "renewBefore": "12h0m0s"}}
	// +optional
	CertConfig HyperConvergedCertConfig `json:"certConfig,omitempty"`

	// ResourceRequirements describes the resource requirements for the operand workloads.
	// +optional
	ResourceRequirements *OperandResourceRequirements `json:"resourceRequirements,omitempty"`

	// Override the storage class used for scratch space during transfer operations. The scratch space storage class
	// is determined in the following order:
	// value of scratchSpaceStorageClass, if that doesn't exist, use the default storage class, if there is no default
	// storage class, use the storage class of the DataVolume, if no storage class specified, use no storage class for
	// scratch space
	// +optional
	ScratchSpaceStorageClass *string `json:"scratchSpaceStorageClass,omitempty"`

	// VDDK Init Image eventually used to import VMs from external providers
	// +optional
	VddkInitImage *string `json:"vddkInitImage,omitempty"`

	// ObsoleteCPUs allows avoiding scheduling of VMs for obsolete CPU models
	// +optional
	ObsoleteCPUs *HyperConvergedObsoleteCPUs `json:"obsoleteCPUs,omitempty"`

	// StorageImport contains configuration for importing containerized data
	// +optional
	StorageImport *StorageImportConfig `json:"storageImport,omitempty"`

	// WorkloadUpdateStrategy defines at the cluster level how to handle automated workload updates
	// +kubebuilder:default={"workloadUpdateMethods": {"LiveMigrate", "Evict"}, "batchEvictionSize": 10, "batchEvictionInterval": "1m"}
	// +optional
	WorkloadUpdateStrategy *HyperConvergedWorkloadUpdateStrategy `json:"workloadUpdateStrategy,omitempty"`

	// operator version
	// +optional
	Version string `json:"version,omitempty"`
}

// CertRotateConfigCA contains the tunables for TLS certificates.
// +k8s:openapi-gen=true
type CertRotateConfigCA struct {
	// The requested 'duration' (i.e. lifetime) of the Certificate.
	// This should comply with golang's ParseDuration format (https://golang.org/pkg/time/#ParseDuration)
	// +kubebuilder:default="48h0m0s"
	// +optional
	Duration metav1.Duration `json:"duration,omitempty"`

	// The amount of time before the currently issued certificate's `notAfter`
	// time that we will begin to attempt to renew the certificate.
	// This should comply with golang's ParseDuration format (https://golang.org/pkg/time/#ParseDuration)
	// +kubebuilder:default="24h0m0s"
	// +optional
	RenewBefore metav1.Duration `json:"renewBefore,omitempty"`
}

// CertRotateConfigServer contains the tunables for TLS certificates.
// +k8s:openapi-gen=true
type CertRotateConfigServer struct {
	// The requested 'duration' (i.e. lifetime) of the Certificate.
	// This should comply with golang's ParseDuration format (https://golang.org/pkg/time/#ParseDuration)
	// +kubebuilder:default="24h0m0s"
	// +optional
	Duration metav1.Duration `json:"duration,omitempty"`

	// The amount of time before the currently issued certificate's `notAfter`
	// time that we will begin to attempt to renew the certificate.
	// This should comply with golang's ParseDuration format (https://golang.org/pkg/time/#ParseDuration)
	// +kubebuilder:default="12h0m0s"
	// +optional
	RenewBefore metav1.Duration `json:"renewBefore,omitempty"`
}

// HyperConvergedCertConfig holds the CertConfig entries for the HCO operands
// +k8s:openapi-gen=true
type HyperConvergedCertConfig struct {
	// CA configuration -
	// CA certs are kept in the CA bundle as long as they are valid
	// +kubebuilder:default={"duration": "48h0m0s", "renewBefore": "24h0m0s"}
	// +optional
	CA CertRotateConfigCA `json:"ca,omitempty"`

	// Server configuration -
	// Certs are rotated and discarded
	// +kubebuilder:default={"duration": "24h0m0s", "renewBefore": "12h0m0s"}
	// +optional
	Server CertRotateConfigServer `json:"server,omitempty"`
}

// HyperConvergedConfig defines a set of configurations to pass to components
type HyperConvergedConfig struct {
	// NodePlacement describes node scheduling configuration.
	// +optional
	NodePlacement *sdkapi.NodePlacement `json:"nodePlacement,omitempty"`
}

// LiveMigrationConfigurations - Live migration limits and timeouts are applied so that migration processes do not
// overwhelm the cluster.
// +k8s:openapi-gen=true
type LiveMigrationConfigurations struct {
	// Number of migrations running in parallel in the cluster.
	// +optional
	// +kubebuilder:default=5
	ParallelMigrationsPerCluster *uint32 `json:"parallelMigrationsPerCluster,omitempty"`

	// Maximum number of outbound migrations per node.
	// +optional
	// +kubebuilder:default=2
	ParallelOutboundMigrationsPerNode *uint32 `json:"parallelOutboundMigrationsPerNode,omitempty"`

	// Bandwidth limit of each migration, in MiB/s.
	// +optional
	// +kubebuilder:default="64Mi"
	// +kubebuilder:validation:Pattern=^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
	BandwidthPerMigration *string `json:"bandwidthPerMigration,omitempty"`

	// The migration will be canceled if it has not completed in this time, in seconds per GiB
	// of memory. For example, a virtual machine instance with 6GiB memory will timeout if it has not completed
	// migration in 4800 seconds. If the Migration Method is BlockMigration, the size of the migrating disks is included
	// in the calculation.
	// +kubebuilder:default=800
	// +optional
	CompletionTimeoutPerGiB *int64 `json:"completionTimeoutPerGiB,omitempty"`

	// The migration will be canceled if memory copy fails to make progress in this time, in seconds.
	// +kubebuilder:default=150
	// +optional
	ProgressTimeout *int64 `json:"progressTimeout,omitempty"`
}

// HyperConvergedFeatureGates is a set of optional feature gates to enable or disable new features that are not enabled
// by default yet.
// +k8s:openapi-gen=true
type HyperConvergedFeatureGates struct {
	// Allow migrating a virtual machine with CPU host-passthrough mode. This should be
	// enabled only when the Cluster is homogeneous from CPU HW perspective doc here
	// +optional
	// +kubebuilder:default=false
	WithHostPassthroughCPU bool `json:"withHostPassthroughCPU"`

	// Allow migrating a virtual machine with SRIOV interfaces.
	// When enabled virt-launcher pods of virtual machines with SRIOV
	// interfaces run with CAP_SYS_RESOURCE capability.
	// This may degrade virt-launcher security.
	// +optional
	// +kubebuilder:default=false
	SRIOVLiveMigration bool `json:"sriovLiveMigration"`
}

// PermittedHostDevices holds information about devices allowed for passthrough
// +k8s:openapi-gen=true
type PermittedHostDevices struct {
	// +listType=map
	// +listMapKey=pciDeviceSelector
	PciHostDevices []PciHostDevice `json:"pciHostDevices,omitempty"`
	// +listType=map
	// +listMapKey=mdevNameSelector
	MediatedDevices []MediatedHostDevice `json:"mediatedDevices,omitempty"`
}

// PciHostDevice represents a host PCI device allowed for passthrough
// +k8s:openapi-gen=true
type PciHostDevice struct {
	// a combination of a vendor_id:product_id required to identify a PCI device on a host.
	PCIDeviceSelector string `json:"pciDeviceSelector"`
	// name by which a device is advertised and being requested
	ResourceName string `json:"resourceName"`
	// indicates that this resource is being provided by an external device plugin
	// +optional
	ExternalResourceProvider bool `json:"externalResourceProvider,omitempty"`
	// HCO enforces the existence of several PciHostDevice objects. Set disabled field to true instead of remove
	// these objects.
	// +optional
	Disabled bool `json:"disabled,omitempty"`
}

// MediatedHostDevice represents a host mediated device allowed for passthrough
// +k8s:openapi-gen=true
type MediatedHostDevice struct {
	// name of a mediated device type required to identify a mediated device on a host
	MDEVNameSelector string `json:"mdevNameSelector"`
	// name by which a device is advertised and being requested
	ResourceName string `json:"resourceName"`
	// indicates that this resource is being provided by an external device plugin
	// +optional
	ExternalResourceProvider bool `json:"externalResourceProvider,omitempty"`
	// HCO enforces the existence of several MediatedHostDevice objects. Set disabled field to true instead of remove
	// these objects.
	// +optional
	Disabled bool `json:"disabled,omitempty"`
}

// OperandResourceRequirements is a list of resource requirements for the operand workloads pods
// +k8s:openapi-gen=true
type OperandResourceRequirements struct {
	// StorageWorkloads defines the resources requirements for storage workloads. It will propagate to the CDI custom
	// resource
	// +optional
	StorageWorkloads *corev1.ResourceRequirements `json:"storageWorkloads,omitempty"`
}

// HyperConvergedObsoleteCPUs allows avoiding scheduling of VMs for obsolete CPU models
// +k8s:openapi-gen=true
type HyperConvergedObsoleteCPUs struct {
	// MinCPUModel is the Minimum CPU model that is used for basic CPU features; e.g. Penryn or Haswell.
	// The default value for this field is nil, but in KubeVirt, the default value is "Penryn", if nothing else is set.
	// Use this field to override KubeVirt default value.
	// +optional
	MinCPUModel string `json:"minCPUModel,omitempty"`
	// CPUModels is a list of obsolete CPU models. When the node-labeller obtains the list of obsolete CPU models, it
	// eliminates those CPU models and creates labels for valid CPU models.
	// The default values for this field is nil, however, HCO uses opinionated values, and adding values to this list
	// will add them to the opinionated values.
	// +optional
	CPUModels []string `json:"cpuModels,omitempty"`
}

// StorageImportConfig contains configuration for importing containerized data
// +k8s:openapi-gen=true
type StorageImportConfig struct {
	// InsecureRegistries is a list of image registries URLs that are not secured. Setting an insecure registry URL
	// in this list allows pulling images from this registry.
	// +optional
	InsecureRegistries []string `json:"insecureRegistries,omitempty"`
}

//
// HyperConvergedWorkloadUpdateStrategy defines options related to updating a KubeVirt install
//
// +k8s:openapi-gen=true
type HyperConvergedWorkloadUpdateStrategy struct {
	// WorkloadUpdateMethods defines the methods that can be used to disrupt workloads
	// during automated workload updates.
	// When multiple methods are present, the least disruptive method takes
	// precedence over more disruptive methods. For example if both LiveMigrate and Shutdown
	// methods are listed, only VMs which are not live migratable will be restarted/shutdown.
	// An empty list defaults to no automated workload updating.
	//
	// +listType=atomic
	// +kubebuilder:default={"LiveMigrate", "Evict"}
	// +optional
	WorkloadUpdateMethods []string `json:"workloadUpdateMethods,omitempty"`

	// BatchEvictionSize Represents the number of VMIs that can be forced updated per
	// the BatchShutdownInteral interval
	//
	// +kubebuilder:default=10
	// +optional
	BatchEvictionSize *int `json:"batchEvictionSize,omitempty"`

	// BatchEvictionInterval Represents the interval to wait before issuing the next
	// batch of shutdowns
	//
	// +kubebuilder:default="1m"
	// +optional
	BatchEvictionInterval *metav1.Duration `json:"batchEvictionInterval,omitempty"`
}

// HyperConvergedStatus defines the observed state of HyperConverged
// +k8s:openapi-gen=true
type HyperConvergedStatus struct {
	// Conditions describes the state of the HyperConverged resource.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +optional
	Conditions []conditionsv1.Condition `json:"conditions,omitempty"  patchStrategy:"merge" patchMergeKey:"type"`

	// RelatedObjects is a list of objects created and maintained by this
	// operator. Object references will be added to this list after they have
	// been created AND found in the cluster.
	// +optional
	RelatedObjects []corev1.ObjectReference `json:"relatedObjects,omitempty"`

	// Versions is a list of HCO component versions, as name/version pairs. The version with a name of "operator"
	// is the HCO version itself, as described here:
	// https://github.com/openshift/cluster-version-operator/blob/master/docs/dev/clusteroperator.md#version
	// +optional
	Versions Versions `json:"versions,omitempty"`
}

func (hcs *HyperConvergedStatus) UpdateVersion(name, version string) {
	if hcs.Versions == nil {
		hcs.Versions = Versions{}
	}
	hcs.Versions.updateVersion(name, version)
}

func (hcs *HyperConvergedStatus) GetVersion(name string) (string, bool) {
	return hcs.Versions.getVersion(name)
}

type Version struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

func newVersion(name, version string) Version {
	return Version{Name: name, Version: version}
}

type Versions []Version

func (vs *Versions) updateVersion(name, version string) {
	for i, v := range *vs {
		if v.Name == name {
			(*vs)[i].Version = version
			return
		}
	}
	*vs = append(*vs, newVersion(name, version))
}

func (vs *Versions) getVersion(name string) (string, bool) {
	for _, v := range *vs {
		if v.Name == name {
			return v.Version, true
		}
	}
	return "", false
}

const (

	// ConditionReconcileComplete communicates the status of the HyperConverged resource's
	// reconcile functionality. Basically, is the Reconcile function running to completion.
	ConditionReconcileComplete conditionsv1.ConditionType = "ReconcileComplete"

	// ConditionTaintedConfiguration indicates that a hidden/debug configuration
	// has been applied to the HyperConverged resource via a specialized annotation.
	// This condition is exposed only when its value is True, and is otherwise hidden.
	ConditionTaintedConfiguration conditionsv1.ConditionType = "TaintedConfiguration"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HyperConverged is the Schema for the hyperconvergeds API
// +k8s:openapi-gen=true
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:resource:scope=Namespaced,categories={all},shortName={hco,hcos}
// +kubebuilder:subresource:status
type HyperConverged struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:default={"certConfig": {"ca": {"duration": "48h0m0s", "renewBefore": "24h0m0s"}, "server": {"duration": "24h0m0s", "renewBefore": "12h0m0s"}}, "featureGates": {"withHostPassthroughCPU": false, "sriovLiveMigration": false}, "liveMigrationConfig": {"bandwidthPerMigration": "64Mi", "completionTimeoutPerGiB": 800, "parallelMigrationsPerCluster": 5, "parallelOutboundMigrationsPerNode": 2, "progressTimeout": 150}}
	// +optional
	Spec   HyperConvergedSpec   `json:"spec,omitempty"`
	Status HyperConvergedStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HyperConvergedList contains a list of HyperConverged
type HyperConvergedList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HyperConverged `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HyperConverged{}, &HyperConvergedList{})
}
