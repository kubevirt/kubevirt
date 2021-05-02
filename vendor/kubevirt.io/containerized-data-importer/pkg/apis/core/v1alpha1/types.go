/*
Copyright 2018 The CDI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/api"
)

// DataVolume is an abstraction on top of PersistentVolumeClaims to allow easy population of those PersistentVolumeClaims with relation to VirtualMachines
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=dv;dvs,categories=all
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="The phase the data volume is in"
// +kubebuilder:printcolumn:name="Progress",type="string",JSONPath=".status.progress",description="Transfer progress in percentage if known, N/A otherwise"
// +kubebuilder:printcolumn:name="Restarts",type="integer",JSONPath=".status.restartCount",description="The number of times the transfer has been restarted."
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type DataVolume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataVolumeSpec   `json:"spec"`
	Status DataVolumeStatus `json:"status,omitempty"`
}

// DataVolumeSpec defines the DataVolume type specification
type DataVolumeSpec struct {
	//Source is the src of the data for the requested DataVolume
	Source DataVolumeSource `json:"source"`
	//PVC is the PVC specification
	PVC *corev1.PersistentVolumeClaimSpec `json:"pvc,omitempty"`
	// Storage is the requested storage specification
	Storage *StorageSpec `json:"storage,omitempty"`
	//PriorityClassName for Importer, Cloner and Uploader pod
	PriorityClassName string `json:"priorityClassName,omitempty"`
	//DataVolumeContentType options: "kubevirt", "archive"
	// +kubebuilder:validation:Enum="kubevirt";"archive"
	ContentType DataVolumeContentType `json:"contentType,omitempty"`
	// Checkpoints is a list of DataVolumeCheckpoints, representing stages in a multistage import.
	Checkpoints []DataVolumeCheckpoint `json:"checkpoints,omitempty"`
	// FinalCheckpoint indicates whether the current DataVolumeCheckpoint is the final checkpoint.
	FinalCheckpoint bool `json:"finalCheckpoint,omitempty"`
	// Preallocation controls whether storage for DataVolumes should be allocated in advance.
	Preallocation *bool `json:"preallocation,omitempty"`
}

// StorageSpec defines the Storage type specification
type StorageSpec struct {
	// AccessModes contains the desired access modes the volume should have.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1
	// +optional
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`
	// A label query over volumes to consider for binding.
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
	// Resources represents the minimum resources the volume should have.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// VolumeName is the binding reference to the PersistentVolume backing this claim.
	// +optional
	VolumeName string `json:"volumeName,omitempty"`
	// Name of the StorageClass required by the claim.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
	// volumeMode defines what type of volume is required by the claim.
	// Value of Filesystem is implied when not included in claim spec.
	// +optional
	VolumeMode *corev1.PersistentVolumeMode `json:"volumeMode,omitempty"`
	// This field can be used to specify either:
	// * An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot - Beta)
	// * An existing PVC (PersistentVolumeClaim)
	// * An existing custom resource/object that implements data population (Alpha)
	// In order to use VolumeSnapshot object types, the appropriate feature gate
	// must be enabled (VolumeSnapshotDataSource or AnyVolumeDataSource)
	// If the provisioner or an external controller can support the specified data source,
	// it will create a new volume based on the contents of the specified data source.
	// If the specified data source is not supported, the volume will
	// not be created and the failure will be reported as an event.
	// In the future, we plan to support more data source types and the behavior
	// of the provisioner may change.
	// +optional
	DataSource *corev1.TypedLocalObjectReference `json:"dataSource,omitempty"`
}

// DataVolumeCheckpoint defines a stage in a warm migration.
type DataVolumeCheckpoint struct {
	// Previous is the identifier of the snapshot from the previous checkpoint.
	Previous string `json:"previous"`
	// Current is the identifier of the snapshot created for this checkpoint.
	Current string `json:"current"`
}

// DataVolumeContentType represents the types of the imported data
type DataVolumeContentType string

const (
	// DataVolumeKubeVirt is the content-type of the imported file, defaults to kubevirt
	DataVolumeKubeVirt DataVolumeContentType = "kubevirt"
	// DataVolumeArchive is the content-type to specify if there is a need to extract the imported archive
	DataVolumeArchive DataVolumeContentType = "archive"
)

// DataVolumeSource represents the source for our Data Volume, this can be HTTP, Imageio, S3, Registry or an existing PVC
type DataVolumeSource struct {
	HTTP     *DataVolumeSourceHTTP     `json:"http,omitempty"`
	S3       *DataVolumeSourceS3       `json:"s3,omitempty"`
	Registry *DataVolumeSourceRegistry `json:"registry,omitempty"`
	PVC      *DataVolumeSourcePVC      `json:"pvc,omitempty"`
	Upload   *DataVolumeSourceUpload   `json:"upload,omitempty"`
	Blank    *DataVolumeBlankImage     `json:"blank,omitempty"`
	Imageio  *DataVolumeSourceImageIO  `json:"imageio,omitempty"`
	VDDK     *DataVolumeSourceVDDK     `json:"vddk,omitempty"`
}

// DataVolumeSourcePVC provides the parameters to create a Data Volume from an existing PVC
type DataVolumeSourcePVC struct {
	// The namespace of the source PVC
	Namespace string `json:"namespace"`
	// The name of the source PVC
	Name string `json:"name"`
}

// DataVolumeBlankImage provides the parameters to create a new raw blank image for the PVC
type DataVolumeBlankImage struct{}

// DataVolumeSourceUpload provides the parameters to create a Data Volume by uploading the source
type DataVolumeSourceUpload struct {
}

// DataVolumeSourceS3 provides the parameters to create a Data Volume from an S3 source
type DataVolumeSourceS3 struct {
	//URL is the url of the S3 source
	URL string `json:"url"`
	//SecretRef provides the secret reference needed to access the S3 source
	SecretRef string `json:"secretRef,omitempty"`
	// CertConfigMap is a configmap reference, containing a Certificate Authority(CA) public key, and a base64 encoded pem certificate
	// +optional
	CertConfigMap string `json:"certConfigMap,omitempty"`
}

// DataVolumeSourceRegistry provides the parameters to create a Data Volume from an registry source
type DataVolumeSourceRegistry struct {
	//URL is the url of the Docker registry source
	URL string `json:"url"`
	//SecretRef provides the secret reference needed to access the Registry source
	SecretRef string `json:"secretRef,omitempty"`
	//CertConfigMap provides a reference to the Registry certs
	CertConfigMap string `json:"certConfigMap,omitempty"`
}

// DataVolumeSourceHTTP can be either an http or https endpoint, with an optional basic auth user name and password, and an optional configmap containing additional CAs
type DataVolumeSourceHTTP struct {
	// URL is the URL of the http(s) endpoint
	URL string `json:"url"`
	// SecretRef A Secret reference, the secret should contain accessKeyId (user name) base64 encoded, and secretKey (password) also base64 encoded
	// +optional
	SecretRef string `json:"secretRef,omitempty"`
	// CertConfigMap is a configmap reference, containing a Certificate Authority(CA) public key, and a base64 encoded pem certificate
	// +optional
	CertConfigMap string `json:"certConfigMap,omitempty"`
}

// DataVolumeSourceImageIO provides the parameters to create a Data Volume from an imageio source
type DataVolumeSourceImageIO struct {
	//URL is the URL of the ovirt-engine
	URL string `json:"url"`
	// DiskID provides id of a disk to be imported
	DiskID string `json:"diskId"`
	//SecretRef provides the secret reference needed to access the ovirt-engine
	SecretRef string `json:"secretRef,omitempty"`
	//CertConfigMap provides a reference to the CA cert
	CertConfigMap string `json:"certConfigMap,omitempty"`
}

// DataVolumeSourceVDDK provides the parameters to create a Data Volume from a Vmware source
type DataVolumeSourceVDDK struct {
	// URL is the URL of the vCenter or ESXi host with the VM to migrate
	URL string `json:"url,omitempty"`
	// UUID is the UUID of the virtual machine that the backing file is attached to in vCenter/ESXi
	UUID string `json:"uuid,omitempty"`
	// BackingFile is the path to the virtual hard disk to migrate from vCenter/ESXi
	BackingFile string `json:"backingFile,omitempty"`
	// Thumbprint is the certificate thumbprint of the vCenter or ESXi host
	Thumbprint string `json:"thumbprint,omitempty"`
	// SecretRef provides a reference to a secret containing the username and password needed to access the vCenter or ESXi host
	SecretRef string `json:"secretRef,omitempty"`
}

// DataVolumeStatus contains the current status of the DataVolume
type DataVolumeStatus struct {
	//Phase is the current phase of the data volume
	Phase    DataVolumePhase    `json:"phase,omitempty"`
	Progress DataVolumeProgress `json:"progress,omitempty"`
	// RestartCount is the number of times the pod populating the DataVolume has restarted
	RestartCount int32                 `json:"restartCount,omitempty"`
	Conditions   []DataVolumeCondition `json:"conditions,omitempty" optional:"true"`
}

//DataVolumeList provides the needed parameters to do request a list of Data Volumes from the system
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type DataVolumeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	// Items provides a list of DataVolumes
	Items []DataVolume `json:"items"`
}

// DataVolumeCondition represents the state of a data volume condition.
type DataVolumeCondition struct {
	Type               DataVolumeConditionType `json:"type" description:"type of condition ie. Ready|Bound|Running."`
	Status             corev1.ConditionStatus  `json:"status" description:"status of the condition, one of True, False, Unknown"`
	LastTransitionTime metav1.Time             `json:"lastTransitionTime,omitempty"`
	LastHeartbeatTime  metav1.Time             `json:"lastHeartbeatTime,omitempty"`
	Reason             string                  `json:"reason,omitempty" description:"reason for the condition's last transition"`
	Message            string                  `json:"message,omitempty" description:"human-readable message indicating details about last transition"`
}

// DataVolumePhase is the current phase of the DataVolume
type DataVolumePhase string

// DataVolumeProgress is the current progress of the DataVolume transfer operation. Value between 0 and 100 inclusive, N/A if not available
type DataVolumeProgress string

// DataVolumeConditionType is the string representation of known condition types
type DataVolumeConditionType string

const (
	// PhaseUnset represents a data volume with no current phase
	PhaseUnset DataVolumePhase = ""

	// Pending represents a data volume with a current phase of Pending
	Pending DataVolumePhase = "Pending"
	// PVCBound represents a data volume with a current phase of PVCBound
	PVCBound DataVolumePhase = "PVCBound"

	// ImportScheduled represents a data volume with a current phase of ImportScheduled
	ImportScheduled DataVolumePhase = "ImportScheduled"

	// ImportInProgress represents a data volume with a current phase of ImportInProgress
	ImportInProgress DataVolumePhase = "ImportInProgress"

	// CloneScheduled represents a data volume with a current phase of CloneScheduled
	CloneScheduled DataVolumePhase = "CloneScheduled"

	// CloneInProgress represents a data volume with a current phase of CloneInProgress
	CloneInProgress DataVolumePhase = "CloneInProgress"

	// SnapshotForSmartCloneInProgress represents a data volume with a current phase of SnapshotForSmartCloneInProgress
	SnapshotForSmartCloneInProgress DataVolumePhase = "SnapshotForSmartCloneInProgress"

	// SmartClonePVCInProgress represents a data volume with a current phase of SmartClonePVCInProgress
	SmartClonePVCInProgress DataVolumePhase = "SmartClonePVCInProgress"

	// UploadScheduled represents a data volume with a current phase of UploadScheduled
	UploadScheduled DataVolumePhase = "UploadScheduled"

	// UploadReady represents a data volume with a current phase of UploadReady
	UploadReady DataVolumePhase = "UploadReady"

	// WaitForFirstConsumer represents a data volume with a current phase of WaitForFirstConsumer
	WaitForFirstConsumer DataVolumePhase = "WaitForFirstConsumer"

	// Succeeded represents a DataVolumePhase of Succeeded
	Succeeded DataVolumePhase = "Succeeded"
	// Failed represents a DataVolumePhase of Failed
	Failed DataVolumePhase = "Failed"
	// Unknown represents a DataVolumePhase of Unknown
	Unknown DataVolumePhase = "Unknown"
	// Paused represents a DataVolumePhase of Paused
	Paused DataVolumePhase = "Paused"

	// DataVolumeReady is the condition that indicates if the data volume is ready to be consumed.
	DataVolumeReady DataVolumeConditionType = "Ready"
	// DataVolumeBound is the condition that indicates if the underlying PVC is bound or not.
	DataVolumeBound DataVolumeConditionType = "Bound"
	// DataVolumeRunning is the condition that indicates if the import/upload/clone container is running.
	DataVolumeRunning DataVolumeConditionType = "Running"
)

// DataVolumeCloneSourceSubresource is the subresource checked for permission to clone
const DataVolumeCloneSourceSubresource = "source"

// this has to be here otherwise informer-gen doesn't recognize it
// see https://github.com/kubernetes/code-generator/issues/59
// +genclient:nonNamespaced

// CDI is the CDI Operator CRD
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=cdi;cdis,scope=Cluster
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
type CDI struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CDISpec `json:"spec"`
	// +optional
	Status CDIStatus `json:"status"`
}

// CertConfig contains the tunables for TLS certificates
type CertConfig struct {
	// The requested 'duration' (i.e. lifetime) of the Certificate.
	Duration *metav1.Duration `json:"duration,omitempty"`

	// The amount of time before the currently issued certificate's `notAfter`
	// time that we will begin to attempt to renew the certificate.
	RenewBefore *metav1.Duration `json:"renewBefore,omitempty"`
}

// CDICertConfig has the CertConfigs for CDI
type CDICertConfig struct {
	// CA configuration
	// CA certs are kept in the CA bundle as long as they are valid
	CA *CertConfig `json:"ca,omitempty"`

	// Server configuration
	// Certs are rotated and discarded
	Server *CertConfig `json:"server,omitempty"`
}

// CDISpec defines our specification for the CDI installation
type CDISpec struct {
	// +kubebuilder:validation:Enum=Always;IfNotPresent;Never
	// PullPolicy describes a policy for if/when to pull a container image
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty" valid:"required"`
	// +kubebuilder:validation:Enum=RemoveWorkloads;BlockUninstallIfWorkloadsExist
	// CDIUninstallStrategy defines the state to leave CDI on uninstall
	UninstallStrategy *CDIUninstallStrategy `json:"uninstallStrategy,omitempty"`
	// Rules on which nodes CDI infrastructure pods will be scheduled
	Infra sdkapi.NodePlacement `json:"infra,omitempty"`
	// Restrict on which nodes CDI workload pods will be scheduled
	Workloads sdkapi.NodePlacement `json:"workload,omitempty"`
	// Clone strategy override: should we use a host-assisted copy even if snapshots are available?
	// +kubebuilder:validation:Enum="copy";"snapshot"
	CloneStrategyOverride *CDICloneStrategy `json:"cloneStrategyOverride,omitempty"`
	// CDIConfig at CDI level
	Config *CDIConfigSpec `json:"config,omitempty"`
	// certificate configuration
	CertConfig *CDICertConfig `json:"certConfig,omitempty"`
}

// CDICloneStrategy defines the preferred method for performing a CDI clone (override snapshot?)
type CDICloneStrategy string

const (
	// CloneStrategyHostAssisted specifies slower, host-assisted copy
	CloneStrategyHostAssisted = "copy"

	// CloneStrategySnapshot specifies snapshot-based copying
	CloneStrategySnapshot = "snapshot"
)

// CDIUninstallStrategy defines the state to leave CDI on uninstall
type CDIUninstallStrategy string

const (
	// CDIUninstallStrategyRemoveWorkloads specifies clean uninstall
	CDIUninstallStrategyRemoveWorkloads CDIUninstallStrategy = "RemoveWorkloads"

	// CDIUninstallStrategyBlockUninstallIfWorkloadsExist "leaves stuff around"
	CDIUninstallStrategyBlockUninstallIfWorkloadsExist CDIUninstallStrategy = "BlockUninstallIfWorkloadsExist"
)

// CDIPhase is the current phase of the CDI deployment
type CDIPhase string

// CDIStatus defines the status of the installation
type CDIStatus struct {
	sdkapi.Status `json:",inline"`
}

const (
	// CDIPhaseDeploying signals that the CDI resources are being deployed
	CDIPhaseDeploying CDIPhase = "Deploying"

	// CDIPhaseDeployed signals that the CDI resources are successflly deployed
	CDIPhaseDeployed CDIPhase = "Deployed"

	// CDIPhaseDeleting signals that the CDI resources are being removed
	CDIPhaseDeleting CDIPhase = "Deleting"

	// CDIPhaseDeleted signals that the CDI resources are deleted
	CDIPhaseDeleted CDIPhase = "Deleted"

	// CDIPhaseError signals that the CDI deployment is in an error state
	CDIPhaseError CDIPhase = "Error"

	// CDIPhaseUpgrading signals that the CDI resources are being deployed
	CDIPhaseUpgrading CDIPhase = "Upgrading"

	// CDIPhaseEmpty is an uninitialized phase
	CDIPhaseEmpty CDIPhase = ""
)

//CDIList provides the needed parameters to do request a list of CDIs from the system
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CDIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	// Items provides a list of CDIs
	Items []CDI `json:"items"`
}

// this has to be here otherwise informer-gen doesn't recognize it
// see https://github.com/kubernetes/code-generator/issues/59
// +genclient:nonNamespaced

// CDIConfig provides a user configuration for CDI
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
type CDIConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CDIConfigSpec   `json:"spec"`
	Status CDIConfigStatus `json:"status,omitempty"`
}

//Percent is a string that can only be a value between [0,1)
// (Note: we actually rely on reconcile to reject invalid values)
// +kubebuilder:validation:Pattern=`^(0(?:\.\d{1,3})?|1)$`
type Percent string

//FilesystemOverhead defines the reserved size for PVCs with VolumeMode: Filesystem
type FilesystemOverhead struct {
	// Global is how much space of a Filesystem volume should be reserved for overhead. This value is used unless overridden by a more specific value (per storageClass)
	Global Percent `json:"global,omitempty"`
	// StorageClass specifies how much space of a Filesystem volume should be reserved for safety. The keys are the storageClass and the values are the overhead. This value overrides the global value
	StorageClass map[string]Percent `json:"storageClass,omitempty"`
}

//CDIConfigSpec defines specification for user configuration
type CDIConfigSpec struct {
	// Override the URL used when uploading to a DataVolume
	UploadProxyURLOverride *string `json:"uploadProxyURLOverride,omitempty"`
	// ImportProxy contains importer pod proxy configuration.
	// +optional
	ImportProxy *ImportProxy `json:"importProxy,omitempty"`
	// Override the storage class to used for scratch space during transfer operations. The scratch space storage class is determined in the following order: 1. value of scratchSpaceStorageClass, if that doesn't exist, use the default storage class, if there is no default storage class, use the storage class of the DataVolume, if no storage class specified, use no storage class for scratch space
	ScratchSpaceStorageClass *string `json:"scratchSpaceStorageClass,omitempty"`
	// ResourceRequirements describes the compute resource requirements.
	PodResourceRequirements *corev1.ResourceRequirements `json:"podResourceRequirements,omitempty"`
	// FeatureGates are a list of specific enabled feature gates
	FeatureGates []string `json:"featureGates,omitempty"`
	// FilesystemOverhead describes the space reserved for overhead when using Filesystem volumes. A value is between 0 and 1, if not defined it is 0.055 (5.5% overhead)
	FilesystemOverhead *FilesystemOverhead `json:"filesystemOverhead,omitempty"`
	// Preallocation controls whether storage for DataVolumes should be allocated in advance.
	Preallocation *bool `json:"preallocation,omitempty"`
	// InsecureRegistries is a list of TLS disabled registries
	InsecureRegistries []string `json:"insecureRegistries,omitempty"`
}

//CDIConfigStatus provides the most recently observed status of the CDI Config resource
type CDIConfigStatus struct {
	// The calculated upload proxy URL
	UploadProxyURL *string `json:"uploadProxyURL,omitempty"`
	// ImportProxy contains importer pod proxy configuration.
	// +optional
	ImportProxy *ImportProxy `json:"importProxy,omitempty"`
	// The calculated storage class to be used for scratch space
	ScratchSpaceStorageClass string `json:"scratchSpaceStorageClass,omitempty"`
	// ResourceRequirements describes the compute resource requirements.
	DefaultPodResourceRequirements *corev1.ResourceRequirements `json:"defaultPodResourceRequirements,omitempty"`
	// FilesystemOverhead describes the space reserved for overhead when using Filesystem volumes. A percentage value is between 0 and 1
	FilesystemOverhead *FilesystemOverhead `json:"filesystemOverhead,omitempty"`
	// Preallocation controls whether storage for DataVolumes should be allocated in advance.
	Preallocation bool `json:"preallocation,omitempty"`
}

//CDIConfigList provides the needed parameters to do request a list of CDIConfigs from the system
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CDIConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	// Items provides a list of CDIConfigs
	Items []CDIConfig `json:"items"`
}

//ImportProxy provides the information on how to configure the importer pod proxy.
type ImportProxy struct {
	// HTTPProxy is the URL http://<username>:<pswd>@<ip>:<port> of the import proxy for HTTP requests.  Empty means unset and will not result in the import pod env var.
	// +optional
	HTTPProxy *string `json:"HTTPProxy,omitempty"`
	// HTTPSProxy is the URL https://<username>:<pswd>@<ip>:<port> of the import proxy for HTTPS requests.  Empty means unset and will not result in the import pod env var.
	// +optional
	HTTPSProxy *string `json:"HTTPSProxy,omitempty"`
	// NoProxy is a comma-separated list of hostnames and/or CIDRs for which the proxy should not be used. Empty means unset and will not result in the import pod env var.
	// +optional
	NoProxy *string `json:"noProxy,omitempty"`
	// TrustedCAProxy is the name of a ConfigMap in the cdi namespace that contains a user-provided trusted certificate authority (CA) bundle.
	// The TrustedCAProxy field is consumed by the import controller that is resposible for coping it to a config map named trusted-ca-proxy-bundle-cm in the cdi namespace.
	// Here is an example of the ConfigMap (in yaml):
	//
	// apiVersion: v1
	// kind: ConfigMap
	// metadata:
	//   name: trusted-ca-proxy-bundle-cm
	//   namespace: cdi
	// data:
	//   ca.pem: |
	//     -----BEGIN CERTIFICATE-----
	// 	   ... <base64 encoded cert> ...
	// 	   -----END CERTIFICATE-----
	// +optional
	TrustedCAProxy *string `json:"trustedCAProxy,omitempty"`
}
