/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package v1

/*
 ATTENTION: Rerun code generators when comments on structs or fields are modified.
*/

import (
	"encoding/json"
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

const DefaultGracePeriodSeconds int64 = 30

// VirtualMachineInstance is *the* VirtualMachineInstance Definition. It represents a virtual machine in the runtime environment of kubernetes.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
type VirtualMachineInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// VirtualMachineInstance Spec contains the VirtualMachineInstance specification.
	Spec VirtualMachineInstanceSpec `json:"spec" valid:"required"`
	// Status is the high level overview of how the VirtualMachineInstance is doing. It contains information available to controllers and users.
	Status VirtualMachineInstanceStatus `json:"status,omitempty"`
}

func (v *VirtualMachineInstance) MarshalBinary() (data []byte, err error) {
	return json.Marshal(*v)
}

func (v *VirtualMachineInstance) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, v)
}

// VirtualMachineInstanceList is a list of VirtualMachines
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineInstance `json:"items"`
}

type EvictionStrategy string

type StartStrategy string

const (
	StartStrategyPaused StartStrategy = "Paused"
)

// VirtualMachineInstanceSpec is a description of a VirtualMachineInstance.
type VirtualMachineInstanceSpec struct {

	// If specified, indicates the pod's priority.
	// If not specified, the pod priority will be default or zero if there is no
	// default.
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// Specification of the desired behavior of the VirtualMachineInstance on the host.
	Domain DomainSpec `json:"domain"`
	// NodeSelector is a selector which must be true for the vmi to fit on a node.
	// Selector which must match a node's labels for the vmi to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// If affinity is specifies, obey all the affinity rules
	Affinity *k8sv1.Affinity `json:"affinity,omitempty"`
	// If specified, the VMI will be dispatched by specified scheduler.
	// If not specified, the VMI will be dispatched by default scheduler.
	// +optional
	SchedulerName string `json:"schedulerName,omitempty"`
	// If toleration is specified, obey all the toleration rules.
	Tolerations []k8sv1.Toleration `json:"tolerations,omitempty"`

	// EvictionStrategy can be set to "LiveMigrate" if the VirtualMachineInstance should be
	// migrated instead of shut-off in case of a node drain.
	//
	// +optional
	EvictionStrategy *EvictionStrategy `json:"evictionStrategy,omitempty"`
	// StartStrategy can be set to "Paused" if Virtual Machine should be started in paused state.
	//
	// +optional
	StartStrategy *StartStrategy `json:"startStrategy,omitempty"`
	// Grace period observed after signalling a VirtualMachineInstance to stop after which the VirtualMachineInstance is force terminated.
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`
	// List of volumes that can be mounted by disks belonging to the vmi.
	Volumes []Volume `json:"volumes,omitempty"`
	// Periodic probe of VirtualMachineInstance liveness.
	// VirtualmachineInstances will be stopped if the probe fails.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	LivenessProbe *Probe `json:"livenessProbe,omitempty"`
	// Periodic probe of VirtualMachineInstance service readiness.
	// VirtualmachineInstances will be removed from service endpoints if the probe fails.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	ReadinessProbe *Probe `json:"readinessProbe,omitempty"`
	// Specifies the hostname of the vmi
	// If not specified, the hostname will be set to the name of the vmi, if dhcp or cloud-init is configured properly.
	// +optional
	Hostname string `json:"hostname,omitempty"`
	// If specified, the fully qualified vmi hostname will be "<hostname>.<subdomain>.<pod namespace>.svc.<cluster domain>".
	// If not specified, the vmi will not have a domainname at all. The DNS entry will resolve to the vmi,
	// no matter if the vmi itself can pick up a hostname.
	// +optional
	Subdomain string `json:"subdomain,omitempty"`
	// List of networks that can be attached to a vm's virtual interface.
	Networks []Network `json:"networks,omitempty"`
	// Set DNS policy for the pod.
	// Defaults to "ClusterFirst".
	// Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'.
	// DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy.
	// To have DNS options set along with hostNetwork, you have to specify DNS policy
	// explicitly to 'ClusterFirstWithHostNet'.
	// +optional
	DNSPolicy k8sv1.DNSPolicy `json:"dnsPolicy,omitempty" protobuf:"bytes,6,opt,name=dnsPolicy,casttype=DNSPolicy"`
	// Specifies the DNS parameters of a pod.
	// Parameters specified here will be merged to the generated DNS
	// configuration based on DNSPolicy.
	// +optional
	DNSConfig *k8sv1.PodDNSConfig `json:"dnsConfig,omitempty" protobuf:"bytes,26,opt,name=dnsConfig"`
	// Specifies a set of public keys to inject into the vm guest
	// +listType=atomic
	// +optional
	AccessCredentials []AccessCredential `json:"accessCredentials,omitempty"`
}

func (vmiSpec *VirtualMachineInstanceSpec) UnmarshalJSON(data []byte) error {
	type VMISpecAlias VirtualMachineInstanceSpec
	var vmiSpecAlias VMISpecAlias

	if err := json.Unmarshal(data, &vmiSpecAlias); err != nil {
		return err
	}

	if vmiSpecAlias.DNSConfig != nil {
		for i, ns := range vmiSpecAlias.DNSConfig.Nameservers {
			if sanitizedIP, err := sanitizeIP(ns); err == nil {
				vmiSpecAlias.DNSConfig.Nameservers[i] = sanitizedIP
			}
		}
	}

	*vmiSpec = VirtualMachineInstanceSpec(vmiSpecAlias)
	return nil
}

// VirtualMachineInstancePhaseTransitionTimestamp gives a timestamp in relation to when a phase is set on a vmi
type VirtualMachineInstancePhaseTransitionTimestamp struct {
	// Phase is the status of the VirtualMachineInstance in kubernetes world. It is not the VirtualMachineInstance status, but partially correlates to it.
	Phase VirtualMachineInstancePhase `json:"phase,omitempty"`
	// PhaseTransitionTimestamp is the timestamp of when the phase change occurred
	PhaseTransitionTimestamp metav1.Time `json:"phaseTransitionTimestamp,omitempty"`
}

type TopologyHints struct {
	TSCFrequency *int64 `json:"tscFrequency,omitempty"`
}

// VirtualMachineInstanceStatus represents information about the status of a VirtualMachineInstance. Status may trail the actual
// state of a system.
type VirtualMachineInstanceStatus struct {
	// NodeName is the name where the VirtualMachineInstance is currently running.
	NodeName string `json:"nodeName,omitempty"`
	// A brief CamelCase message indicating details about why the VMI is in this state. e.g. 'NodeUnresponsive'
	// +optional
	Reason string `json:"reason,omitempty"`
	// Conditions are specific points in VirtualMachineInstance's pod runtime.
	Conditions []VirtualMachineInstanceCondition `json:"conditions,omitempty"`
	// Phase is the status of the VirtualMachineInstance in kubernetes world. It is not the VirtualMachineInstance status, but partially correlates to it.
	Phase VirtualMachineInstancePhase `json:"phase,omitempty"`
	// PhaseTransitionTimestamp is the timestamp of when the last phase change occurred
	// +listType=atomic
	// +optional
	PhaseTransitionTimestamps []VirtualMachineInstancePhaseTransitionTimestamp `json:"phaseTransitionTimestamps,omitempty"`
	// Interfaces represent the details of available network interfaces.
	Interfaces []VirtualMachineInstanceNetworkInterface `json:"interfaces,omitempty"`
	// Guest OS Information
	GuestOSInfo VirtualMachineInstanceGuestOSInfo `json:"guestOSInfo,omitempty"`
	// Represents the status of a live migration
	MigrationState *VirtualMachineInstanceMigrationState `json:"migrationState,omitempty"`
	// Represents the method using which the vmi can be migrated: live migration or block migration
	MigrationMethod VirtualMachineInstanceMigrationMethod `json:"migrationMethod,omitempty"`
	// This represents the migration transport
	MigrationTransport VirtualMachineInstanceMigrationTransport `json:"migrationTransport,omitempty"`
	// The Quality of Service (QOS) classification assigned to the virtual machine instance based on resource requirements
	// See PodQOSClass type for available QOS classes
	// More info: https://git.k8s.io/community/contributors/design-proposals/node/resource-qos.md
	// +optional
	QOSClass *k8sv1.PodQOSClass `json:"qosClass,omitempty"`

	// LauncherContainerImageVersion indicates what container image is currently active for the vmi.
	LauncherContainerImageVersion string `json:"launcherContainerImageVersion,omitempty"`

	// EvacuationNodeName is used to track the eviction process of a VMI. It stores the name of the node that we want
	// to evacuate. It is meant to be used by KubeVirt core components only and can't be set or modified by users.
	// +optional
	EvacuationNodeName string `json:"evacuationNodeName,omitempty"`

	// ActivePods is a mapping of pod UID to node name.
	// It is possible for multiple pods to be running for a single VMI during migration.
	ActivePods map[types.UID]string `json:"activePods,omitempty"`

	// VolumeStatus contains the statuses of all the volumes
	// +optional
	// +listType=atomic
	VolumeStatus []VolumeStatus `json:"volumeStatus,omitempty"`

	// FSFreezeStatus is the state of the fs of the guest
	// it can be either frozen or thawed
	// +optional
	FSFreezeStatus string `json:"fsFreezeStatus,omitempty"`

	// +optional
	TopologyHints *TopologyHints `json:"topologyHints,omitempty"`

	//VirtualMachineRevisionName is used to get the vm revision of the vmi when doing
	// an online vm snapshot
	// +optional
	VirtualMachineRevisionName string `json:"virtualMachineRevisionName,omitempty"`

	// RuntimeUser is used to determine what user will be used in launcher
	// +optional
	RuntimeUser uint64 `json:"runtimeUser"`
}

// PersistentVolumeClaimInfo contains the relavant information virt-handler needs cached about a PVC
type PersistentVolumeClaimInfo struct {
	// AccessModes contains the desired access modes the volume should have.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1
	// +listType=atomic
	// +optional
	AccessModes []k8sv1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`

	// VolumeMode defines what type of volume is required by the claim.
	// Value of Filesystem is implied when not included in claim spec.
	// +optional
	VolumeMode *k8sv1.PersistentVolumeMode `json:"volumeMode,omitempty"`

	// Capacity represents the capacity set on the corresponding PVC status
	// +optional
	Capacity k8sv1.ResourceList `json:"capacity,omitempty"`

	// Requests represents the resources requested by the corresponding PVC spec
	// +optional
	Requests k8sv1.ResourceList `json:"requests,omitempty"`

	// Preallocated indicates if the PVC's storage is preallocated or not
	// +optional
	Preallocated bool `json:"preallocated,omitempty"`

	// Percentage of filesystem's size to be reserved when resizing the PVC
	// +optional
	FilesystemOverhead *cdiv1.Percent `json:"filesystemOverhead,omitempty"`
}

// VolumeStatus represents information about the status of volumes attached to the VirtualMachineInstance.
type VolumeStatus struct {
	// Name is the name of the volume
	Name string `json:"name"`
	// Target is the target name used when adding the volume to the VM, eg: vda
	Target string `json:"target"`
	// Phase is the phase
	Phase VolumePhase `json:"phase,omitempty"`
	// Reason is a brief description of why we are in the current hotplug volume phase
	Reason string `json:"reason,omitempty"`
	// Message is a detailed message about the current hotplug volume phase
	Message string `json:"message,omitempty"`
	// PersistentVolumeClaimInfo is information about the PVC that handler requires during start flow
	PersistentVolumeClaimInfo *PersistentVolumeClaimInfo `json:"persistentVolumeClaimInfo,omitempty"`
	// If the volume is hotplug, this will contain the hotplug status.
	HotplugVolume *HotplugVolumeStatus `json:"hotplugVolume,omitempty"`
	// Represents the size of the volume
	Size int64 `json:"size,omitempty"`
	// If the volume is memorydump volume, this will contain the memorydump info.
	MemoryDumpVolume *DomainMemoryDumpInfo `json:"memoryDumpVolume,omitempty"`
}

// DomainMemoryDumpInfo represents the memory dump information
type DomainMemoryDumpInfo struct {
	// StartTimestamp is the time when the memory dump started
	StartTimestamp *metav1.Time `json:"startTimestamp,omitempty"`
	// EndTimestamp is the time when the memory dump completed
	EndTimestamp *metav1.Time `json:"endTimestamp,omitempty"`
	// ClaimName is the name of the pvc the memory was dumped to
	ClaimName string `json:"claimName,omitempty"`
	// TargetFileName is the name of the memory dump output
	TargetFileName string `json:"targetFileName,omitempty"`
}

// HotplugVolumeStatus represents the hotplug status of the volume
type HotplugVolumeStatus struct {
	// AttachPodName is the name of the pod used to attach the volume to the node.
	AttachPodName string `json:"attachPodName,omitempty"`
	// AttachPodUID is the UID of the pod used to attach the volume to the node.
	AttachPodUID types.UID `json:"attachPodUID,omitempty"`
}

// VolumePhase indicates the current phase of the hotplug process.
type VolumePhase string

const (
	// VolumePending means the Volume is pending and cannot be attached to the node yet.
	VolumePending VolumePhase = "Pending"
	// VolumeBound means the Volume is bound and can be attach to the node.
	VolumeBound VolumePhase = "Bound"
	// HotplugVolumeAttachedToNode means the volume has been attached to the node.
	HotplugVolumeAttachedToNode VolumePhase = "AttachedToNode"
	// HotplugVolumeMounted means the volume has been attached to the node and is mounted to the virt-launcher pod.
	HotplugVolumeMounted VolumePhase = "MountedToPod"
	// VolumeReady means the volume is ready to be used by the VirtualMachineInstance.
	VolumeReady VolumePhase = "Ready"
	// HotplugVolumeDetaching means the volume is being detached from the node, and the attachment pod is being removed.
	HotplugVolumeDetaching VolumePhase = "Detaching"
	// HotplugVolumeUnMounted means the volume has been unmounted from the virt-launcer pod.
	HotplugVolumeUnMounted VolumePhase = "UnMountedFromPod"
	// MemoryDumpVolumeCompleted means that the requested memory dump was completed and the dump is ready in the volume
	MemoryDumpVolumeCompleted VolumePhase = "MemoryDumpCompleted"
	// MemoryDumpVolumeInProgress means that the volume for the memory dump was attached, and now the command is being triggered
	MemoryDumpVolumeInProgress VolumePhase = "MemoryDumpInProgress"
	// MemoryDumpVolumeInProgress means that the volume for the memory dump was attached, and now the command is being triggered
	MemoryDumpVolumeFailed VolumePhase = "MemoryDumpFailed"
)

func (v *VirtualMachineInstance) IsScheduling() bool {
	return v.Status.Phase == Scheduling
}

func (v *VirtualMachineInstance) IsScheduled() bool {
	return v.Status.Phase == Scheduled
}

func (v *VirtualMachineInstance) IsRunning() bool {
	return v.Status.Phase == Running
}

func (v *VirtualMachineInstance) IsMarkedForEviction() bool {
	return v.Status.EvacuationNodeName != ""
}

func (v *VirtualMachineInstance) IsMigratable() bool {
	for _, cond := range v.Status.Conditions {
		if cond.Type == VirtualMachineInstanceIsMigratable && cond.Status == k8sv1.ConditionTrue {
			return true
		}
	}
	return false
}

func (v *VirtualMachineInstance) IsFinal() bool {
	return v.Status.Phase == Failed || v.Status.Phase == Succeeded
}

func (v *VirtualMachineInstance) IsMarkedForDeletion() bool {
	return v.ObjectMeta.DeletionTimestamp != nil
}

func (v *VirtualMachineInstance) IsUnknown() bool {
	return v.Status.Phase == Unknown
}

func (v *VirtualMachineInstance) IsUnprocessed() bool {
	return v.Status.Phase == Pending || v.Status.Phase == VmPhaseUnset
}

// Checks if CPU pinning has been requested
func (v *VirtualMachineInstance) IsCPUDedicated() bool {
	return v.Spec.Domain.CPU != nil && v.Spec.Domain.CPU.DedicatedCPUPlacement
}

func (v *VirtualMachineInstance) IsBootloaderEFI() bool {
	return v.Spec.Domain.Firmware != nil && v.Spec.Domain.Firmware.Bootloader != nil &&
		v.Spec.Domain.Firmware.Bootloader.EFI != nil
}

// WantsToHaveQOSGuaranteed checks if cpu and memoyr limits and requests are identical on the VMI.
// This is the indicator that people want a VMI with QOS of guaranteed
func (v *VirtualMachineInstance) WantsToHaveQOSGuaranteed() bool {
	resources := v.Spec.Domain.Resources
	return !resources.Requests.Memory().IsZero() && resources.Requests.Memory().Cmp(*resources.Limits.Memory()) == 0 &&
		!resources.Requests.Cpu().IsZero() && resources.Requests.Cpu().Cmp(*resources.Limits.Cpu()) == 0
}

// ShouldStartPaused returns true if VMI should be started in paused state
func (v *VirtualMachineInstance) ShouldStartPaused() bool {
	return v.Spec.StartStrategy != nil && *v.Spec.StartStrategy == StartStrategyPaused
}

func (v *VirtualMachineInstance) IsRealtimeEnabled() bool {
	return v.Spec.Domain.CPU != nil && v.Spec.Domain.CPU.Realtime != nil
}

type VirtualMachineInstanceConditionType string

// These are valid conditions of VMIs.
const (
	// Provisioning means, a VMI depends on DataVolumes which are in Pending/WaitForFirstConsumer status,
	// and some actions are taken to provision the PVCs for the DataVolumes
	VirtualMachineInstanceProvisioning VirtualMachineInstanceConditionType = "Provisioning"

	// Ready means the VMI is able to service requests and should be added to the
	// load balancing pools of all matching services.
	VirtualMachineInstanceReady VirtualMachineInstanceConditionType = "Ready"

	// If there happens any error while trying to synchronize the VirtualMachineInstance with the Domain,
	// this is reported as false.
	VirtualMachineInstanceSynchronized VirtualMachineInstanceConditionType = "Synchronized"

	// If the VMI was paused by the user, this is reported as true.
	VirtualMachineInstancePaused VirtualMachineInstanceConditionType = "Paused"

	// Reflects whether the QEMU guest agent is connected through the channel
	VirtualMachineInstanceAgentConnected VirtualMachineInstanceConditionType = "AgentConnected"

	// Reflects whether the QEMU guest agent updated access credentials successfully
	VirtualMachineInstanceAccessCredentialsSynchronized VirtualMachineInstanceConditionType = "AccessCredentialsSynchronized"

	// Reflects whether the QEMU guest agent is connected through the channel
	VirtualMachineInstanceUnsupportedAgent VirtualMachineInstanceConditionType = "AgentVersionNotSupported"

	// Indicates whether the VMI is live migratable
	VirtualMachineInstanceIsMigratable VirtualMachineInstanceConditionType = "LiveMigratable"
	// Reason means that VMI is not live migratioable because of it's disks collection
	VirtualMachineInstanceReasonDisksNotMigratable = "DisksNotLiveMigratable"
	// Reason means that VMI is not live migratioable because of it's network interfaces collection
	VirtualMachineInstanceReasonInterfaceNotMigratable = "InterfaceNotLiveMigratable"
	// Reason means that VMI is not live migratioable because it uses hotplug
	VirtualMachineInstanceReasonHotplugNotMigratable = "HotplugNotLiveMigratable"
	// Reason means that VMI is not live migratioable because of it's CPU mode
	VirtualMachineInstanceReasonCPUModeNotMigratable = "CPUModeLiveMigratable"
	// Reason means that VMI is not live migratable because it uses virtiofs
	VirtualMachineInstanceReasonVirtIOFSNotMigratable = "VirtIOFSNotLiveMigratable"
	// Reason means that VMI is not live migratable because it uses PCI host devices
	VirtualMachineInstanceReasonHostDeviceNotMigratable = "HostDeviceNotLiveMigratable"
	// Reason means that VMI is not live migratable because it uses Secure Encrypted Virtualization (SEV)
	VirtualMachineInstanceReasonSEVNotMigratable = "SEVNotLiveMigratable"
)

const (
	// PodTerminatingReason indicates on the Ready condition on the VMI if the underlying pod is terminating
	PodTerminatingReason = "PodTerminating"

	// PodNotExistsReason indicates on the Ready condition on the VMI if the underlying pod does not exist
	PodNotExistsReason = "PodNotExists"

	// PodConditionMissingReason indicates on the Ready condition on the VMI if the underlying pod does not report a Ready condition
	PodConditionMissingReason = "PodConditionMissing"

	// GuestNotRunningReason indicates on the Ready condition on the VMI if the underlying guest VM is not running
	GuestNotRunningReason = "GuestNotRunning"
)

type VirtualMachineInstanceMigrationConditionType string

// These are valid conditions of VMIs.
const (
	// VirtualMachineInstanceMigrationAbortRequested indicates that live migration abort has been requested
	VirtualMachineInstanceMigrationAbortRequested VirtualMachineInstanceMigrationConditionType = "migrationAbortRequested"
)

type VirtualMachineInstanceCondition struct {
	Type   VirtualMachineInstanceConditionType `json:"type"`
	Status k8sv1.ConditionStatus               `json:"status"`
	// +nullable
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Message            string      `json:"message,omitempty"`
}

type VirtualMachineInstanceMigrationCondition struct {
	Type   VirtualMachineInstanceMigrationConditionType `json:"type"`
	Status k8sv1.ConditionStatus                        `json:"status"`
	// +nullable
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Message            string      `json:"message,omitempty"`
}

// The migration phase indicates that the job has completed
func (m *VirtualMachineInstanceMigration) IsFinal() bool {
	return m.Status.Phase == MigrationFailed || m.Status.Phase == MigrationSucceeded
}

func (m *VirtualMachineInstanceMigration) IsRunning() bool {
	switch m.Status.Phase {
	case MigrationFailed, MigrationPending, MigrationPhaseUnset, MigrationSucceeded:
		return false
	}
	return true
}

// The migration phase indicates that the target pod should have already been created
func (m *VirtualMachineInstanceMigration) TargetIsCreated() bool {
	return m.Status.Phase != MigrationPhaseUnset &&
		m.Status.Phase != MigrationPending
}

// The migration phase indicates that job has been handed off to the VMI controllers to complete.
func (m *VirtualMachineInstanceMigration) TargetIsHandedOff() bool {
	return m.Status.Phase != MigrationPhaseUnset &&
		m.Status.Phase != MigrationPending &&
		m.Status.Phase != MigrationScheduling &&
		m.Status.Phase != MigrationScheduled
}

type VirtualMachineInstanceNetworkInterface struct {
	// IP address of a Virtual Machine interface. It is always the first item of
	// IPs
	IP string `json:"ipAddress,omitempty"`
	// Hardware address of a Virtual Machine interface
	MAC string `json:"mac,omitempty"`
	// Name of the interface, corresponds to name of the network assigned to the interface
	Name string `json:"name,omitempty"`
	// List of all IP addresses of a Virtual Machine interface
	IPs []string `json:"ipAddresses,omitempty"`
	// The interface name inside the Virtual Machine
	InterfaceName string `json:"interfaceName,omitempty"`
	// Specifies the origin of the interface data collected. values: domain, guest-agent, or both
	InfoSource string `json:"infoSource,omitempty"`
}

type VirtualMachineInstanceGuestOSInfo struct {
	// Name of the Guest OS
	Name string `json:"name,omitempty"`
	// Guest OS Kernel Release
	KernelRelease string `json:"kernelRelease,omitempty"`
	// Guest OS Version
	Version string `json:"version,omitempty"`
	// Guest OS Pretty Name
	PrettyName string `json:"prettyName,omitempty"`
	// Version ID of the Guest OS
	VersionID string `json:"versionId,omitempty"`
	// Kernel version of the Guest OS
	KernelVersion string `json:"kernelVersion,omitempty"`
	// Machine type of the Guest OS
	Machine string `json:"machine,omitempty"`
	// Guest OS Id
	ID string `json:"id,omitempty"`
}

// MigrationConfigSource indicates the source of migration configuration.
//
// +k8s:openapi-gen=true
type MigrationConfigSource string

// +k8s:openapi-gen=true
type VirtualMachineInstanceMigrationState struct {
	// The time the migration action began
	// +nullable
	StartTimestamp *metav1.Time `json:"startTimestamp,omitempty"`
	// The time the migration action ended
	// +nullable
	EndTimestamp *metav1.Time `json:"endTimestamp,omitempty"`
	// The Target Node has seen the Domain Start Event
	TargetNodeDomainDetected bool `json:"targetNodeDomainDetected,omitempty"`
	// The address of the target node to use for the migration
	TargetNodeAddress string `json:"targetNodeAddress,omitempty"`
	// The list of ports opened for live migration on the destination node
	TargetDirectMigrationNodePorts map[string]int `json:"targetDirectMigrationNodePorts,omitempty"`
	// The target node that the VMI is moving to
	TargetNode string `json:"targetNode,omitempty"`
	// The target pod that the VMI is moving to
	TargetPod string `json:"targetPod,omitempty"`
	// The UID of the target attachment pod for hotplug volumes
	TargetAttachmentPodUID types.UID `json:"targetAttachmentPodUID,omitempty"`
	// The source node that the VMI originated on
	SourceNode string `json:"sourceNode,omitempty"`
	// Indicates the migration completed
	Completed bool `json:"completed,omitempty"`
	// Indicates that the migration failed
	Failed bool `json:"failed,omitempty"`
	// Indicates that the migration has been requested to abort
	AbortRequested bool `json:"abortRequested,omitempty"`
	// Indicates the final status of the live migration abortion
	AbortStatus MigrationAbortStatus `json:"abortStatus,omitempty"`
	// The VirtualMachineInstanceMigration object associated with this migration
	MigrationUID types.UID `json:"migrationUid,omitempty"`
	// Lets us know if the vmi is currently running pre or post copy migration
	Mode MigrationMode `json:"mode,omitempty"`
	// Name of the migration policy. If string is empty, no policy is matched
	MigrationPolicyName *string `json:"migrationPolicyName,omitempty"`
	// Migration configurations to apply
	MigrationConfiguration *MigrationConfiguration `json:"migrationConfiguration,omitempty"`
	// If the VMI requires dedicated CPUs, this field will
	// hold the dedicated CPU set on the target node
	// +listType=atomic
	TargetCPUSet []int `json:"targetCPUSet,omitempty"`
	// If the VMI requires dedicated CPUs, this field will
	// hold the numa topology on the target node
	TargetNodeTopology string `json:"targetNodeTopology,omitempty"`
}

type MigrationAbortStatus string

const (
	// MigrationAbortSucceeded means that the VirtualMachineInstance live migration has been aborted
	MigrationAbortSucceeded MigrationAbortStatus = "Succeeded"
	// MigrationAbortFailed means that the vmi live migration has failed to be abort
	MigrationAbortFailed MigrationAbortStatus = "Failed"
	// MigrationAbortInProgress mean that the vmi live migration is aborting
	MigrationAbortInProgress MigrationAbortStatus = "Aborting"
)

type MigrationMode string

const (
	// MigrationPreCopy means the VMI migrations that is currently running is in pre copy mode
	MigrationPreCopy MigrationMode = "PreCopy"
	// MigrationPostCopy means the VMI migrations that is currently running is in post copy mode
	MigrationPostCopy MigrationMode = "PostCopy"
)

type VirtualMachineInstanceMigrationTransport string

const (
	// MigrationTransportUnix means that the VMI will be migrated using the unix URI
	MigrationTransportUnix VirtualMachineInstanceMigrationTransport = "Unix"
)

type VirtualMachineInstanceMigrationMethod string

const (
	// BlockMigration means that all VirtualMachineInstance disks should be copied over to the destination host
	BlockMigration VirtualMachineInstanceMigrationMethod = "BlockMigration"
	// LiveMigration means that VirtualMachineInstance disks will not be copied over to the destination host
	LiveMigration VirtualMachineInstanceMigrationMethod = "LiveMigration"
)

// VirtualMachineInstancePhase is a label for the condition of a VirtualMachineInstance at the current time.
type VirtualMachineInstancePhase string

// These are the valid statuses of pods.
const (
	//When a VirtualMachineInstance Object is first initialized and no phase, or Pending is present.
	VmPhaseUnset VirtualMachineInstancePhase = ""
	// Pending means the VirtualMachineInstance has been accepted by the system.
	Pending VirtualMachineInstancePhase = "Pending"
	// A target Pod exists but is not yet scheduled and in running state.
	Scheduling VirtualMachineInstancePhase = "Scheduling"
	// A target pod was scheduled and the system saw that Pod in runnig state.
	// Here is where the responsibility of virt-controller ends and virt-handler takes over.
	Scheduled VirtualMachineInstancePhase = "Scheduled"
	// Running means the pod has been bound to a node and the VirtualMachineInstance is started.
	Running VirtualMachineInstancePhase = "Running"
	// Succeeded means that the VirtualMachineInstance stopped voluntarily, e.g. reacted to SIGTERM or shutdown was invoked from
	// inside the VirtualMachineInstance.
	Succeeded VirtualMachineInstancePhase = "Succeeded"
	// Failed means that the vmi crashed, disappeared unexpectedly or got deleted from the cluster before it was ever started.
	Failed VirtualMachineInstancePhase = "Failed"
	// Unknown means that for some reason the state of the VirtualMachineInstance could not be obtained, typically due
	// to an error in communicating with the host of the VirtualMachineInstance.
	Unknown VirtualMachineInstancePhase = "Unknown"
)

const (
	// AppLabel and AppName labels marks resources that belong to KubeVirt. An optional value
	// may indicate which specific KubeVirt component a resource belongs to.
	AppLabel string = "kubevirt.io"
	AppName  string = "name"
	// This annotation is used to match virtual machine instances represented as
	// libvirt XML domains with their pods. Among other things, the annotation is
	// used to detect virtual machines with dead pods. Used on Pod.
	DomainAnnotation string = "kubevirt.io/domain"
	// Represents the name of the migration job this target pod is associated with
	MigrationJobNameAnnotation                    string = "kubevirt.io/migrationJobName"
	ControllerAPILatestVersionObservedAnnotation  string = "kubevirt.io/latest-observed-api-version"
	ControllerAPIStorageVersionObservedAnnotation string = "kubevirt.io/storage-observed-api-version"
	// Used by functional tests to force a VMI to fail the migration internally within launcher
	FuncTestForceLauncherMigrationFailureAnnotation string = "kubevirt.io/func-test-force-launcher-migration-failure"
	// Used by functional tests to prevent virt launcher from finishing the target pod preparation.
	FuncTestBlockLauncherPrepareMigrationTargetAnnotation string = "kubevirt.io/func-test-block-migration-target-preparation"

	// Used by functional tests set custom image on migration target pod
	FuncTestMigrationTargetImageOverrideAnnotation string = "kubevirt.io/func-test-migration-target-image-override"

	// Used by functional tests to simulate virt-launcher crash looping
	FuncTestLauncherFailFastAnnotation string = "kubevirt.io/func-test-virt-launcher-fail-fast"
	// This label is used to match virtual machine instance IDs with pods.
	// Similar to kubevirt.io/domain. Used on Pod.
	// Internal use only.
	CreatedByLabel string = "kubevirt.io/created-by"
	// This label is used to indicate that this pod is the target of a migration job.
	MigrationJobLabel string = "kubevirt.io/migrationJobUID"
	// This label indicates the migration name that a PDB is protecting.
	MigrationNameLabel string = "kubevirt.io/migrationName"
	// This label describes which cluster node runs the virtual machine
	// instance. Needed because with CRDs we can't use field selectors. Used on
	// VirtualMachineInstance.
	NodeNameLabel string = "kubevirt.io/nodeName"
	// This label describes which cluster node runs the target Pod for a Virtual
	// Machine Instance migration job. Needed because with CRDs we can't use field
	// selectors. Used on VirtualMachineInstance.
	MigrationTargetNodeNameLabel string = "kubevirt.io/migrationTargetNodeName"
	// This annotation indicates that a migration is the result of an
	// automated evacuation
	EvacuationMigrationAnnotation string = "kubevirt.io/evacuationMigration"
	// This annotation indicates that a migration is the result of an
	// automated workload update
	WorkloadUpdateMigrationAnnotation string = "kubevirt.io/workloadUpdateMigration"
	// This label declares whether a particular node is available for
	// scheduling virtual machine instances on it. Used on Node.
	NodeSchedulable string = "kubevirt.io/schedulable"
	// This annotation is regularly updated by virt-handler to help determine
	// if a particular node is alive and hence should be available for new
	// virtual machine instance scheduling. Used on Node.
	VirtHandlerHeartbeat string = "kubevirt.io/heartbeat"
	// This label indicates what launcher image a VMI is currently running with.
	OutdatedLauncherImageLabel string = "kubevirt.io/outdatedLauncherImage"
	// Namespace recommended by Kubernetes for commonly recognized labels
	AppLabelPrefix = "app.kubernetes.io"
	// This label is commonly used by 3rd party management tools to identify
	// an application's name.
	AppNameLabel = AppLabelPrefix + "/name"
	// This label is commonly used by 3rd party management tools to identify
	// an application's version.
	AppVersionLabel = AppLabelPrefix + "/version"
	// This label is commonly used by 3rd party management tools to identify
	// a higher level application.
	AppPartOfLabel = AppLabelPrefix + "/part-of"
	// This label is commonly used by 3rd party management tools to identify
	// the component this application is a part of.
	AppComponentLabel = AppLabelPrefix + "/component"
	// This label identifies each resource as part of KubeVirt
	AppComponent = "kubevirt"
	// This label will be set on all resources created by the operator
	ManagedByLabel                 = AppLabelPrefix + "/managed-by"
	ManagedByLabelOperatorValue    = "virt-operator"
	ManagedByLabelOperatorOldValue = "kubevirt-operator"
	// This annotation represents the kubevirt version for an install strategy configmap.
	InstallStrategyVersionAnnotation = "kubevirt.io/install-strategy-version"
	// This annotation represents the kubevirt registry used for an install strategy configmap.
	InstallStrategyRegistryAnnotation = "kubevirt.io/install-strategy-registry"
	// This annotation represents the kubevirt deployment identifier used for an install strategy configmap.
	InstallStrategyIdentifierAnnotation = "kubevirt.io/install-strategy-identifier"
	// This annotation shows the enconding used for the manifests in the Install Strategy ConfigMap.
	InstallStrategyConfigMapEncoding = "kubevirt.io/install-strategy-cm-encoding"
	// This annotation is a hash of all customizations that live under spec.CustomizeComponents
	KubeVirtCustomizeComponentAnnotationHash = "kubevirt.io/customizer-identifier"
	// This annotation represents the kubevirt generation that was used to create a resource
	KubeVirtGenerationAnnotation = "kubevirt.io/generation"
	// This annotation represents that this object is for temporary use during updates
	EphemeralBackupObject = "kubevirt.io/ephemeral-backup-object"
	// This annotation represents that the annotated object is for temporary use during pod/volume provisioning
	EphemeralProvisioningObject string = "kubevirt.io/ephemeral-provisioning"

	// This label indicates the object is a part of the install strategy retrieval process.
	InstallStrategyLabel = "kubevirt.io/install-strategy"

	// Set by virt-operator to coordinate component deletion
	VirtOperatorComponentFinalizer string = "kubevirt.io/virtOperatorFinalizer"

	// Set by VMI controller to ensure VMIs are processed during deletion
	VirtualMachineInstanceFinalizer string = "foregroundDeleteVirtualMachine"
	// Set By VM controller on VMIs to ensure VMIs are processed by VM controller during deletion
	VirtualMachineControllerFinalizer        string = "kubevirt.io/virtualMachineControllerFinalize"
	VirtualMachineInstanceMigrationFinalizer string = "kubevirt.io/migrationJobFinalize"
	CPUManager                               string = "cpumanager"
	// This annotation is used to inject ignition data
	// Used on VirtualMachineInstance.
	IgnitionAnnotation           string = "kubevirt.io/ignitiondata"
	PlacePCIDevicesOnRootComplex string = "kubevirt.io/placePCIDevicesOnRootComplex"

	// This label represents supported cpu features on the node
	CPUFeatureLabel = "cpu-feature.node.kubevirt.io/"
	// This label represents supported cpu models on the node
	CPUModelLabel                  = "cpu-model.node.kubevirt.io/"
	SupportedHostModelMigrationCPU = "cpu-model-migration.node.kubevirt.io/"
	CPUTimerLabel                  = "cpu-timer.node.kubevirt.io/"
	// This label represents supported HyperV features on the node
	HypervLabel = "hyperv.node.kubevirt.io/"
	// This label represents vendor of cpu model on the node
	CPUModelVendorLabel = "cpu-vendor.node.kubevirt.io/"

	// This label represents the host model CPU name
	HostModelCPULabel = "host-model-cpu.node.kubevirt.io/"
	// This label represents the host model required features
	HostModelRequiredFeaturesLabel = "host-model-required-features.node.kubevirt.io/"

	LabellerSkipNodeAnnotation        = "node-labeller.kubevirt.io/skip-node"
	VirtualMachineLabel               = AppLabel + "/vm"
	MemfdMemoryBackend         string = "kubevirt.io/memfd"

	MigrationSelectorLabel = "kubevirt.io/vmi-name"

	// This annotation represents vmi running nonroot implementation
	DeprecatedNonRootVMIAnnotation = "kubevirt.io/nonroot"

	// This annotation is to keep virt launcher container alive when an VMI encounters a failure for debugging purpose
	KeepLauncherAfterFailureAnnotation string = "kubevirt.io/keep-launcher-alive-after-failure"

	// MigrationTransportUnixAnnotation means that the VMI will be migrated using the unix URI
	MigrationTransportUnixAnnotation string = "kubevirt.io/migrationTransportUnix"

	// MigrationUnschedulablePodTimeoutSecondsAnnotation represents a custom timeout period used for unschedulable target pods
	// This exists for functional testing
	MigrationUnschedulablePodTimeoutSecondsAnnotation string = "kubevirt.io/migrationUnschedulablePodTimeoutSeconds"

	// MigrationPendingPodTimeoutSecondsAnnotation represents a custom timeout period used for target pods stuck in pending for any reason
	// This exists for functional testing
	MigrationPendingPodTimeoutSecondsAnnotation string = "kubevirt.io/migrationPendingPodTimeoutSeconds"

	// CustomLibvirtLogFiltersAnnotation can be used to customized libvirt log filters. Example value could be
	// "3:remote 4:event 3:util.json 3:util.object 3:util.dbus 3:util.netlink 3:node_device 3:rpc 3:access 1:*".
	// For more info: https://libvirt.org/kbase/debuglogs.html
	CustomLibvirtLogFiltersAnnotation string = "kubevirt.io/libvirt-log-filters"

	// RealtimeLabel marks the node as capable of running realtime workloads
	RealtimeLabel string = "kubevirt.io/realtime"

	// VirtualMachineUnpaused is a custom pod condition set for the virt-launcher pod.
	// It's used as a readiness gate to prevent paused VMs from being marked as ready.
	VirtualMachineUnpaused k8sv1.PodConditionType = "kubevirt.io/virtual-machine-unpaused"

	// SEVLabel marks the node as capable of running workloads with SEV
	SEVLabel string = "kubevirt.io/sev"

	// FlavorAnnotation is the name of a VirtualMachineFlavor
	FlavorAnnotation string = "kubevirt.io/flavor-name"

	// ClusterFlavorAnnotation is the name of a VirtualMachineClusterFlavor
	ClusterFlavorAnnotation string = "kubevirt.io/cluster-flavor-name"

	// FlavorAnnotation is the name of a VirtualMachinePreference
	PreferenceAnnotation string = "kubevirt.io/preference-name"

	// ClusterFlavorAnnotation is the name of a VirtualMachinePreferenceFlavor
	ClusterPreferenceAnnotation string = "kubevirt.io/cluster-preference-name"

	// VirtualMachinePoolRevisionName is used to store the vmpool revision's name this object
	// originated from.
	VirtualMachinePoolRevisionName string = "kubevirt.io/vm-pool-revision-name"

	// VirtualMachineNameLabel is the name of the Virtual Machine
	VirtualMachineNameLabel string = "vm.kubevirt.io/name"
)

func NewVMI(name string, uid types.UID) *VirtualMachineInstance {
	return &VirtualMachineInstance{
		Spec: VirtualMachineInstanceSpec{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			UID:       uid,
			Namespace: k8sv1.NamespaceDefault,
		},
		Status: VirtualMachineInstanceStatus{},
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       VirtualMachineInstanceGroupVersionKind.Kind,
		},
	}
}

type SyncEvent string

const (
	Created                      SyncEvent = "Created"
	Deleted                      SyncEvent = "Deleted"
	PresetFailed                 SyncEvent = "PresetFailed"
	Override                     SyncEvent = "Override"
	Started                      SyncEvent = "Started"
	ShuttingDown                 SyncEvent = "ShuttingDown"
	Stopped                      SyncEvent = "Stopped"
	PreparingTarget              SyncEvent = "PreparingTarget"
	Migrating                    SyncEvent = "Migrating"
	Migrated                     SyncEvent = "Migrated"
	SyncFailed                   SyncEvent = "SyncFailed"
	Resumed                      SyncEvent = "Resumed"
	AccessCredentialsSyncFailed  SyncEvent = "AccessCredentialsSyncFailed"
	AccessCredentialsSyncSuccess SyncEvent = "AccessCredentialsSyncSuccess"
)

func (s SyncEvent) String() string {
	return string(s)
}

// TODO Namespace could be different, also store it somewhere in the domain, so that we can report deletes on handler startup properly
func NewVMIReferenceFromName(name string) *VirtualMachineInstance {
	return NewVMIReferenceFromNameWithNS(k8sv1.NamespaceDefault, name)
}

func NewVMIReferenceFromNameWithNS(namespace string, name string) *VirtualMachineInstance {
	vmi := &VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			SelfLink:  fmt.Sprintf("/apis/%s/namespaces/%s/virtualmachineinstances/%s", GroupVersion.String(), namespace, name),
		},
	}
	vmi.SetGroupVersionKind(schema.GroupVersionKind{Group: GroupVersion.Group, Kind: "VirtualMachineInstance", Version: GroupVersion.Version})
	return vmi
}

func NewVMIReferenceWithUUID(namespace string, name string, uuid types.UID) *VirtualMachineInstance {
	vmi := NewVMIReferenceFromNameWithNS(namespace, name)
	vmi.UID = uuid
	return vmi
}

type VMISelector struct {
	// Name of the VirtualMachineInstance to migrate
	Name string `json:"name" valid:"required"`
}

func NewVMReferenceFromNameWithNS(namespace string, name string) *VirtualMachine {
	vm := &VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			SelfLink:  fmt.Sprintf("/apis/%s/namespaces/%s/virtualmachines/%s", GroupVersion.String(), namespace, name),
		},
	}
	vm.SetGroupVersionKind(schema.GroupVersionKind{Group: GroupVersion.Group, Kind: "VirtualMachine", Version: GroupVersion.Version})
	return vm
}

// Given a VirtualMachineInstance, update all NodeSelectorTerms with anti-affinity for that VirtualMachineInstance's node.
// This is useful for the case when a migration away from a node must occur.
// This method returns the full Affinity structure updated the anti affinity terms
func UpdateAntiAffinityFromVMINode(pod *k8sv1.Pod, vmi *VirtualMachineInstance) *k8sv1.Affinity {
	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &k8sv1.Affinity{}
	}

	if pod.Spec.Affinity.NodeAffinity == nil {
		pod.Spec.Affinity.NodeAffinity = &k8sv1.NodeAffinity{}
	}

	if pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &k8sv1.NodeSelector{}
	}

	selector := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	terms := selector.NodeSelectorTerms

	if len(terms) == 0 {
		selector.NodeSelectorTerms = append(terms, k8sv1.NodeSelectorTerm{})
		terms = selector.NodeSelectorTerms
	}

	for idx, term := range terms {
		if term.MatchExpressions == nil {
			term.MatchExpressions = []k8sv1.NodeSelectorRequirement{}
		}

		term.MatchExpressions = append(term.MatchExpressions, PrepareVMINodeAntiAffinitySelectorRequirement(vmi))
		selector.NodeSelectorTerms[idx] = term
	}

	return pod.Spec.Affinity
}

// Given a VirtualMachineInstance, create a NodeSelectorTerm with anti-affinity for that VirtualMachineInstance's node.
// This is useful for the case when a migration away from a node must occur.
func PrepareVMINodeAntiAffinitySelectorRequirement(vmi *VirtualMachineInstance) k8sv1.NodeSelectorRequirement {
	return k8sv1.NodeSelectorRequirement{
		Key:      "kubernetes.io/hostname",
		Operator: k8sv1.NodeSelectorOpNotIn,
		Values:   []string{vmi.Status.NodeName},
	}
}

// VirtualMachineInstance is *the* VirtualMachineInstance Definition. It represents a virtual machine in the runtime environment of kubernetes.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
type VirtualMachineInstanceReplicaSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// VirtualMachineInstance Spec contains the VirtualMachineInstance specification.
	Spec VirtualMachineInstanceReplicaSetSpec `json:"spec" valid:"required"`
	// Status is the high level overview of how the VirtualMachineInstance is doing. It contains information available to controllers and users.
	// +nullable
	Status VirtualMachineInstanceReplicaSetStatus `json:"status,omitempty"`
}

// VMIList is a list of VMIs
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineInstanceReplicaSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineInstanceReplicaSet `json:"items"`
}

type VirtualMachineInstanceReplicaSetSpec struct {
	// Number of desired pods. This is a pointer to distinguish between explicit
	// zero and not specified. Defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Label selector for pods. Existing ReplicaSets whose pods are
	// selected by this will be the ones affected by this deployment.
	Selector *metav1.LabelSelector `json:"selector" valid:"required"`

	// Template describes the pods that will be created.
	Template *VirtualMachineInstanceTemplateSpec `json:"template" valid:"required"`

	// Indicates that the replica set is paused.
	// +optional
	Paused bool `json:"paused,omitempty" protobuf:"varint,7,opt,name=paused"`
}

type VirtualMachineInstanceReplicaSetStatus struct {
	// Total number of non-terminated pods targeted by this deployment (their labels match the selector).
	// +optional
	Replicas int32 `json:"replicas,omitempty" protobuf:"varint,2,opt,name=replicas"`

	// The number of ready replicas for this replica set.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty" protobuf:"varint,4,opt,name=readyReplicas"`

	Conditions []VirtualMachineInstanceReplicaSetCondition `json:"conditions,omitempty" optional:"true"`

	// Canonical form of the label selector for HPA which consumes it through the scale subresource.
	LabelSelector string `json:"labelSelector,omitempty"`
}

type VirtualMachineInstanceReplicaSetCondition struct {
	Type   VirtualMachineInstanceReplicaSetConditionType `json:"type"`
	Status k8sv1.ConditionStatus                         `json:"status"`
	// +nullable
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Message            string      `json:"message,omitempty"`
}

type VirtualMachineInstanceReplicaSetConditionType string

const (
	// VirtualMachineInstanceReplicaSetReplicaFailure is added in a replica set when one of its vmis
	// fails to be created due to insufficient quota, limit ranges, pod security policy, node selectors,
	// etc. or deleted due to kubelet being down or finalizers are failing.
	VirtualMachineInstanceReplicaSetReplicaFailure VirtualMachineInstanceReplicaSetConditionType = "ReplicaFailure"

	// VirtualMachineInstanceReplicaSetReplicaPaused is added in a replica set when the replica set got paused by the controller.
	// After this condition was added, it is safe to remove or add vmis by hand and adjust the replica count by hand.
	VirtualMachineInstanceReplicaSetReplicaPaused VirtualMachineInstanceReplicaSetConditionType = "ReplicaPaused"
)

type DataVolumeTemplateDummyStatus struct{}

type DataVolumeTemplateSpec struct {
	// TypeMeta only exists on DataVolumeTemplate for API backwards compatibility
	// this field is not used by our controllers and is a no-op.
	// +nullable
	metav1.TypeMeta `json:",inline"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +nullable
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// DataVolumeSpec contains the DataVolume specification.
	Spec cdiv1.DataVolumeSpec `json:"spec"`

	// DataVolumeTemplateDummyStatus is here simply for backwards compatibility with
	// a previous API.
	// +nullable
	// +optional
	Status *DataVolumeTemplateDummyStatus `json:"status,omitempty"`
}

type VirtualMachineInstanceTemplateSpec struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	// +nullable
	ObjectMeta metav1.ObjectMeta `json:"metadata,omitempty"`
	// VirtualMachineInstance Spec contains the VirtualMachineInstance specification.
	Spec VirtualMachineInstanceSpec `json:"spec,omitempty" valid:"required"`
}

// VirtualMachineInstanceMigration represents the object tracking a VMI's migration
// to another host in the cluster
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
type VirtualMachineInstanceMigration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VirtualMachineInstanceMigrationSpec   `json:"spec" valid:"required"`
	Status            VirtualMachineInstanceMigrationStatus `json:"status,omitempty"`
}

// VirtualMachineInstanceMigrationList is a list of VirtualMachineMigrations
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineInstanceMigrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineInstanceMigration `json:"items"`
}

type VirtualMachineInstanceMigrationSpec struct {
	// The name of the VMI to perform the migration on. VMI must exist in the migration objects namespace
	VMIName string `json:"vmiName,omitempty" valid:"required"`
}

// VirtualMachineInstanceMigration reprents information pertaining to a VMI's migration.
type VirtualMachineInstanceMigrationStatus struct {
	Phase      VirtualMachineInstanceMigrationPhase       `json:"phase,omitempty"`
	Conditions []VirtualMachineInstanceMigrationCondition `json:"conditions,omitempty"`
}

// VirtualMachineInstanceMigrationPhase is a label for the condition of a VirtualMachineInstanceMigration at the current time.
type VirtualMachineInstanceMigrationPhase string

// These are the valid migration phases
const (
	MigrationPhaseUnset VirtualMachineInstanceMigrationPhase = ""
	// The migration is accepted by the system
	MigrationPending VirtualMachineInstanceMigrationPhase = "Pending"
	// The migration's target pod is being scheduled
	MigrationScheduling VirtualMachineInstanceMigrationPhase = "Scheduling"
	// The migration's target pod is running
	MigrationScheduled VirtualMachineInstanceMigrationPhase = "Scheduled"
	// The migration's target pod is being prepared for migration
	MigrationPreparingTarget VirtualMachineInstanceMigrationPhase = "PreparingTarget"
	// The migration's target pod is prepared and ready for migration
	MigrationTargetReady VirtualMachineInstanceMigrationPhase = "TargetReady"
	// The migration is in progress
	MigrationRunning VirtualMachineInstanceMigrationPhase = "Running"
	// The migration passed
	MigrationSucceeded VirtualMachineInstanceMigrationPhase = "Succeeded"
	// The migration failed
	MigrationFailed VirtualMachineInstanceMigrationPhase = "Failed"
)

// VirtualMachineInstancePreset defines a VMI spec.domain to be applied to all VMIs that match the provided label selector
// More info: https://kubevirt.io/user-guide/virtual_machines/presets/#overrides
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
type VirtualMachineInstancePreset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// VirtualMachineInstance Spec contains the VirtualMachineInstance specification.
	Spec VirtualMachineInstancePresetSpec `json:"spec,omitempty" valid:"required"`
}

// VirtualMachineInstancePresetList is a list of VirtualMachinePresets
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineInstancePresetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineInstancePreset `json:"items"`
}

type VirtualMachineInstancePresetSpec struct {
	// Selector is a label query over a set of VMIs.
	// Required.
	Selector metav1.LabelSelector `json:"selector"`
	// Domain is the same object type as contained in VirtualMachineInstanceSpec
	Domain *DomainSpec `json:"domain,omitempty"`
}

func NewVirtualMachinePreset(name string, selector metav1.LabelSelector) *VirtualMachineInstancePreset {
	return &VirtualMachineInstancePreset{
		Spec: VirtualMachineInstancePresetSpec{
			Selector: selector,
			Domain:   &DomainSpec{},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: k8sv1.NamespaceDefault,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       VirtualMachineInstancePresetGroupVersionKind.Kind,
		},
	}
}

// VirtualMachine handles the VirtualMachines that are not running
// or are in a stopped state
// The VirtualMachine contains the template to create the
// VirtualMachineInstance. It also mirrors the running state of the created
// VirtualMachineInstance in its status.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
type VirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec contains the specification of VirtualMachineInstance created
	Spec VirtualMachineSpec `json:"spec" valid:"required"`
	// Status holds the current state of the controller and brief information
	// about its associated VirtualMachineInstance
	Status VirtualMachineStatus `json:"status,omitempty"`
}

// Return the current runStrategy for the VirtualMachine
// if vm.spec.running is set, that will be mapped to runStrategy:
//   false: RunStrategyHalted
//   true: RunStrategyAlways
func (vm *VirtualMachine) RunStrategy() (VirtualMachineRunStrategy, error) {
	if vm.Spec.Running != nil && vm.Spec.RunStrategy != nil {
		return RunStrategyUnknown, fmt.Errorf("running and runstrategy are mutually exclusive")
	}
	RunStrategy := RunStrategyHalted
	if vm.Spec.Running != nil {
		if (*vm.Spec.Running) == true {
			RunStrategy = RunStrategyAlways
		}
	} else if vm.Spec.RunStrategy != nil {
		RunStrategy = *vm.Spec.RunStrategy
	}
	return RunStrategy, nil
}

// VirtualMachineList is a list of virtualmachines
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachine `json:"items"`
}

// VirtualMachineRunStrategy is a label for the requested VirtualMachineInstance Running State at the current time.
type VirtualMachineRunStrategy string

// These are the valid VMI run strategies
const (
	// Placeholder. Not a valid RunStrategy.
	RunStrategyUnknown VirtualMachineRunStrategy = ""
	// VMI should always be running.
	RunStrategyAlways VirtualMachineRunStrategy = "Always"
	// VMI should never be running.
	RunStrategyHalted VirtualMachineRunStrategy = "Halted"
	// VMI can be started/stopped using API endpoints.
	RunStrategyManual VirtualMachineRunStrategy = "Manual"
	// VMI will initially be running--and restarted if a failure occurs.
	// It will not be restarted upon successful completion.
	RunStrategyRerunOnFailure VirtualMachineRunStrategy = "RerunOnFailure"
	// VMI will run once and not be restarted upon completion regardless
	// if the completion is of phase Failure or Success
	RunStrategyOnce VirtualMachineRunStrategy = "Once"
)

// VirtualMachineSpec describes how the proper VirtualMachine
// should look like
type VirtualMachineSpec struct {
	// Running controls whether the associatied VirtualMachineInstance is created or not
	// Mutually exclusive with RunStrategy
	Running *bool `json:"running,omitempty" optional:"true"`

	// Running state indicates the requested running state of the VirtualMachineInstance
	// mutually exclusive with Running
	RunStrategy *VirtualMachineRunStrategy `json:"runStrategy,omitempty" optional:"true"`

	// FlavorMatcher references a flavor that is used to fill fields in Template
	Flavor *FlavorMatcher `json:"flavor,omitempty" optional:"true"`

	// PreferenceMatcher references a set of preference that is used to fill fields in Template
	Preference *PreferenceMatcher `json:"preference,omitempty" optional:"true"`

	// Template is the direct specification of VirtualMachineInstance
	Template *VirtualMachineInstanceTemplateSpec `json:"template"`

	// dataVolumeTemplates is a list of dataVolumes that the VirtualMachineInstance template can reference.
	// DataVolumes in this list are dynamically created for the VirtualMachine and are tied to the VirtualMachine's life-cycle.
	DataVolumeTemplates []DataVolumeTemplateSpec `json:"dataVolumeTemplates,omitempty"`
}

// StateChangeRequestType represents the existing state change requests that are possible
type StateChangeRequestAction string

// These are the currently defined state change requests
const (
	StartRequest StateChangeRequestAction = "Start"
	StopRequest  StateChangeRequestAction = "Stop"
)

// VirtualMachinePrintableStatus is a human readable, high-level representation of the status of the virtual machine.
type VirtualMachinePrintableStatus string

// A list of statuses defined for virtual machines
const (
	// VirtualMachineStatusStopped indicates that the virtual machine is currently stopped and isn't expected to start.
	VirtualMachineStatusStopped VirtualMachinePrintableStatus = "Stopped"
	// VirtualMachineStatusProvisioning indicates that cluster resources associated with the virtual machine
	// (e.g., DataVolumes) are being provisioned and prepared.
	VirtualMachineStatusProvisioning VirtualMachinePrintableStatus = "Provisioning"
	// VirtualMachineStatusStarting indicates that the virtual machine is being prepared for running.
	VirtualMachineStatusStarting VirtualMachinePrintableStatus = "Starting"
	// VirtualMachineStatusRunning indicates that the virtual machine is running.
	VirtualMachineStatusRunning VirtualMachinePrintableStatus = "Running"
	// VirtualMachineStatusPaused indicates that the virtual machine is paused.
	VirtualMachineStatusPaused VirtualMachinePrintableStatus = "Paused"
	// VirtualMachineStatusStopping indicates that the virtual machine is in the process of being stopped.
	VirtualMachineStatusStopping VirtualMachinePrintableStatus = "Stopping"
	// VirtualMachineStatusTerminating indicates that the virtual machine is in the process of deletion,
	// as well as its associated resources (VirtualMachineInstance, DataVolumes, …).
	VirtualMachineStatusTerminating VirtualMachinePrintableStatus = "Terminating"
	// VirtualMachineStatusCrashLoopBackOff indicates that the virtual machine is currently in a crash loop waiting to be retried.
	VirtualMachineStatusCrashLoopBackOff VirtualMachinePrintableStatus = "CrashLoopBackOff"
	// VirtualMachineStatusMigrating indicates that the virtual machine is in the process of being migrated
	// to another host.
	VirtualMachineStatusMigrating VirtualMachinePrintableStatus = "Migrating"
	// VirtualMachineStatusUnknown indicates that the state of the virtual machine could not be obtained,
	// typically due to an error in communicating with the host on which it's running.
	VirtualMachineStatusUnknown VirtualMachinePrintableStatus = "Unknown"
	// VirtualMachineStatusUnschedulable indicates that an error has occurred while scheduling the virtual machine,
	// e.g. due to unsatisfiable resource requests or unsatisfiable scheduling constraints.
	VirtualMachineStatusUnschedulable VirtualMachinePrintableStatus = "ErrorUnschedulable"
	// VirtualMachineStatusErrImagePull indicates that an error has occured while pulling an image for
	// a containerDisk VM volume.
	VirtualMachineStatusErrImagePull VirtualMachinePrintableStatus = "ErrImagePull"
	// VirtualMachineStatusImagePullBackOff indicates that an error has occured while pulling an image for
	// a containerDisk VM volume, and that kubelet is backing off before retrying.
	VirtualMachineStatusImagePullBackOff VirtualMachinePrintableStatus = "ImagePullBackOff"
	// VirtualMachineStatusPvcNotFound indicates that the virtual machine references a PVC volume which doesn't exist.
	VirtualMachineStatusPvcNotFound VirtualMachinePrintableStatus = "ErrorPvcNotFound"
	// VirtualMachineStatusDataVolumeNotFound indicates that the virtual machine references a DataVolume volume which doesn't exist.
	VirtualMachineStatusDataVolumeNotFound VirtualMachinePrintableStatus = "ErrorDataVolumeNotFound"
	// VirtualMachineStatusDataVolumeError indicates that an error has been reported by one of the DataVolumes
	// referenced by the virtual machines.
	VirtualMachineStatusDataVolumeError VirtualMachinePrintableStatus = "DataVolumeError"
	// VirtualMachineStatusWaitingForVolumeBinding indicates that some PersistentVolumeClaims backing
	// the virtual machine volume are still not bound.
	VirtualMachineStatusWaitingForVolumeBinding VirtualMachinePrintableStatus = "WaitingForVolumeBinding"
)

// VirtualMachineStartFailure tracks VMIs which failed to transition successfully
// to running using the VM status
type VirtualMachineStartFailure struct {
	ConsecutiveFailCount int          `json:"consecutiveFailCount,omitempty"`
	LastFailedVMIUID     types.UID    `json:"lastFailedVMIUID,omitempty"`
	RetryAfterTimestamp  *metav1.Time `json:"retryAfterTimestamp,omitempty"`
}

// VirtualMachineStatus represents the status returned by the
// controller to describe how the VirtualMachine is doing
type VirtualMachineStatus struct {
	// SnapshotInProgress is the name of the VirtualMachineSnapshot currently executing
	SnapshotInProgress *string `json:"snapshotInProgress,omitempty"`
	// RestoreInProgress is the name of the VirtualMachineRestore currently executing
	RestoreInProgress *string `json:"restoreInProgress,omitempty"`
	// Created indicates if the virtual machine is created in the cluster
	Created bool `json:"created,omitempty"`
	// Ready indicates if the virtual machine is running and ready
	Ready bool `json:"ready,omitempty"`
	// PrintableStatus is a human readable, high-level representation of the status of the virtual machine
	PrintableStatus VirtualMachinePrintableStatus `json:"printableStatus,omitempty"`
	// Hold the state information of the VirtualMachine and its VirtualMachineInstance
	Conditions []VirtualMachineCondition `json:"conditions,omitempty" optional:"true"`
	// StateChangeRequests indicates a list of actions that should be taken on a VMI
	// e.g. stop a specific VMI then start a new one.
	StateChangeRequests []VirtualMachineStateChangeRequest `json:"stateChangeRequests,omitempty" optional:"true"`
	// VolumeRequests indicates a list of volumes add or remove from the VMI template and
	// hotplug on an active running VMI.
	// +listType=atomic
	VolumeRequests []VirtualMachineVolumeRequest `json:"volumeRequests,omitempty" optional:"true"`

	// VolumeSnapshotStatuses indicates a list of statuses whether snapshotting is
	// supported by each volume.
	VolumeSnapshotStatuses []VolumeSnapshotStatus `json:"volumeSnapshotStatuses,omitempty" optional:"true"`

	// StartFailure tracks consecutive VMI startup failures for the purposes of
	// crash loop backoffs
	// +nullable
	// +optional
	StartFailure *VirtualMachineStartFailure `json:"startFailure,omitempty" optional:"true"`

	// MemoryDumpRequest tracks memory dump request phase and info of getting a memory
	// dump to the given pvc
	// +nullable
	// +optional
	MemoryDumpRequest *VirtualMachineMemoryDumpRequest `json:"memoryDumpRequest,omitempty" optional:"true"`
}

type VolumeSnapshotStatus struct {
	// Volume name
	Name string `json:"name"`
	// True if the volume supports snapshotting
	Enabled bool `json:"enabled"`
	// Empty if snapshotting is enabled, contains reason otherwise
	Reason string `json:"reason,omitempty" optional:"true"`
}

type VirtualMachineVolumeRequest struct {
	// AddVolumeOptions when set indicates a volume should be added. The details
	// within this field specify how to add the volume
	AddVolumeOptions *AddVolumeOptions `json:"addVolumeOptions,omitempty" optional:"true"`
	// RemoveVolumeOptions when set indicates a volume should be removed. The details
	// within this field specify how to add the volume
	RemoveVolumeOptions *RemoveVolumeOptions `json:"removeVolumeOptions,omitempty" optional:"true"`
}

type VirtualMachineStateChangeRequest struct {
	// Indicates the type of action that is requested. e.g. Start or Stop
	Action StateChangeRequestAction `json:"action"`
	// Provides additional data in order to perform the Action
	Data map[string]string `json:"data,omitempty" optional:"true"`
	// Indicates the UUID of an existing Virtual Machine Instance that this change request applies to -- if applicable
	UID *types.UID `json:"uid,omitempty" optional:"true" protobuf:"bytes,5,opt,name=uid,casttype=k8s.io/kubernetes/pkg/types.UID"`
}

// VirtualMachineCondition represents the state of VirtualMachine
type VirtualMachineCondition struct {
	Type   VirtualMachineConditionType `json:"type"`
	Status k8sv1.ConditionStatus       `json:"status"`
	// +nullable
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Message            string      `json:"message,omitempty"`
}

type VirtualMachineConditionType string

const (
	// VirtualMachineFailure is added in a virtual machine when its vmi
	// fails to be created due to insufficient quota, limit ranges, pod security policy, node selectors,
	// etc. or deleted due to kubelet being down or finalizers are failing.
	VirtualMachineFailure VirtualMachineConditionType = "Failure"

	// VirtualMachineReady is copied to the virtual machine from its vmi
	VirtualMachineReady VirtualMachineConditionType = "Ready"

	// VirtualMachinePaused is added in a virtual machine when its vmi
	// signals with its own condition that it is paused.
	VirtualMachinePaused VirtualMachineConditionType = "Paused"
)

type HostDiskType string

const (
	// if disk does not exist at the given path,
	// a disk image will be created there
	HostDiskExistsOrCreate HostDiskType = "DiskOrCreate"
	// a disk image must exist at given disk path
	HostDiskExists HostDiskType = "Disk"
)

type NetworkInterfaceType string

const (
	// Virtual machine instance bride interface
	BridgeInterface NetworkInterfaceType = "bridge"
	// Virtual machine instance slirp interface
	SlirpInterface NetworkInterfaceType = "slirp"
	// Virtual machine instance masquerade interface
	MasqueradeInterface NetworkInterfaceType = "masquerade"
)

type DriverCache string

type DriverIO string

const (
	// CacheNone - I/O from the guest is not cached on the host, but may be kept in a writeback disk cache.
	CacheNone DriverCache = "none"
	// CacheWriteThrough - I/O from the guest is cached on the host but written through to the physical medium.
	CacheWriteThrough DriverCache = "writethrough"
	// CacheWriteBack - I/O from the guest is cached on the host.
	CacheWriteBack DriverCache = "writeback"

	// IOThreads - User mode based threads with a shared lock that perform I/O tasks. Can impact performance but offers
	// more predictable behaviour. This method is also takes fewer CPU cycles to submit I/O requests.
	IOThreads DriverIO = "threads"
	// IONative - Kernel native I/O tasks (AIO) offer a better performance but can block the VM if the file is not fully
	// allocated so this method recommended only when the backing file/disk/etc is fully preallocated.
	IONative DriverIO = "native"
	// IODefault - Fallback to the default value from the kernel. With recent Kernel versions (for example RHEL-7) the
	// default is AIO.
	IODefault DriverIO = "default"
)

// Handler defines a specific action that should be taken
// TODO: pass structured data to these actions, and document that data here.
type Handler struct {
	// One and only one of the following should be specified.
	// Exec specifies the action to take, it will be executed on the guest through the qemu-guest-agent.
	// If the guest agent is not available, this probe will fail.
	// +optional
	Exec *k8sv1.ExecAction `json:"exec,omitempty" protobuf:"bytes,1,opt,name=exec"`
	// GuestAgentPing contacts the qemu-guest-agent for availability checks.
	// +optional
	GuestAgentPing *GuestAgentPing `json:"guestAgentPing,omitempty"`
	// HTTPGet specifies the http request to perform.
	// +optional
	HTTPGet *k8sv1.HTTPGetAction `json:"httpGet,omitempty"`
	// TCPSocket specifies an action involving a TCP port.
	// TCP hooks not yet supported
	// TODO: implement a realistic TCP lifecycle hook
	// +optional
	TCPSocket *k8sv1.TCPSocketAction `json:"tcpSocket,omitempty"`
}

// Probe describes a health check to be performed against a VirtualMachineInstance to determine whether it is
// alive or ready to receive traffic.
type Probe struct {
	// The action taken to determine the health of a VirtualMachineInstance
	Handler `json:",inline"`
	// Number of seconds after the VirtualMachineInstance has started before liveness probes are initiated.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	InitialDelaySeconds int32 `json:"initialDelaySeconds,omitempty"`
	// Number of seconds after which the probe times out.
	// For exec probes the timeout fails the probe but does not terminate the command running on the guest.
	// This means a blocking command can result in an increasing load on the guest.
	// A small buffer will be added to the resulting workload exec probe to compensate for delays
	// caused by the qemu guest exec mechanism.
	// Defaults to 1 second. Minimum value is 1.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty"`
	// How often (in seconds) to perform the probe.
	// Default to 10 seconds. Minimum value is 1.
	// +optional
	PeriodSeconds int32 `json:"periodSeconds,omitempty"`
	// Minimum consecutive successes for the probe to be considered successful after having failed.
	// Defaults to 1. Must be 1 for liveness. Minimum value is 1.
	// +optional
	SuccessThreshold int32 `json:"successThreshold,omitempty"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded.
	// Defaults to 3. Minimum value is 1.
	// +optional
	FailureThreshold int32 `json:"failureThreshold,omitempty"`
}

// KubeVirt represents the object deploying all KubeVirt resources
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
type KubeVirt struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KubeVirtSpec   `json:"spec" valid:"required"`
	Status            KubeVirtStatus `json:"status,omitempty"`
}

// KubeVirtList is a list of KubeVirts
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KubeVirtList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubeVirt `json:"items"`
}

type KubeVirtSelfSignConfiguration struct {
	// Deprecated. Use CA.Duration instead
	CARotateInterval *metav1.Duration `json:"caRotateInterval,omitempty"`
	// Deprecated. Use Server.Duration instead
	CertRotateInterval *metav1.Duration `json:"certRotateInterval,omitempty"`
	// Deprecated. Use CA.Duration and CA.RenewBefore instead
	CAOverlapInterval *metav1.Duration `json:"caOverlapInterval,omitempty"`

	// CA configuration
	// CA certs are kept in the CA bundle as long as they are valid
	CA *CertConfig `json:"ca,omitempty"`

	// Server configuration
	// Certs are rotated and discarded
	Server *CertConfig `json:"server,omitempty"`
}

// CertConfig contains the tunables for TLS certificates
type CertConfig struct {
	// The requested 'duration' (i.e. lifetime) of the Certificate.
	Duration *metav1.Duration `json:"duration,omitempty"`

	// The amount of time before the currently issued certificate's "notAfter"
	// time that we will begin to attempt to renew the certificate.
	RenewBefore *metav1.Duration `json:"renewBefore,omitempty"`
}

type KubeVirtCertificateRotateStrategy struct {
	SelfSigned *KubeVirtSelfSignConfiguration `json:"selfSigned,omitempty"`
}

type WorkloadUpdateMethod string

const (
	// WorkloadUpdateMethodLiveMigrate allows VMIs which are capable of being
	// migrated to automatically migrate during automated workload updates.
	WorkloadUpdateMethodLiveMigrate WorkloadUpdateMethod = "LiveMigrate"
	// WorkloadUpdateMethodEvict results in a VMI's pod being evicted. Unless the
	// pod has a pod disruption budget allocated, the eviction will usually result in
	// the VMI being shutdown.
	// Depending on whether a VMI is backed by a VM or not, this will either result
	// in a restart of the VM by rescheduling a new VMI, or the shutdown via eviction
	// of a standalone VMI object.
	WorkloadUpdateMethodEvict WorkloadUpdateMethod = "Evict"
)

//
// KubeVirtWorkloadUpdateStrategy defines options related to updating a KubeVirt install
type KubeVirtWorkloadUpdateStrategy struct {
	// WorkloadUpdateMethods defines the methods that can be used to disrupt workloads
	// during automated workload updates.
	// When multiple methods are present, the least disruptive method takes
	// precedence over more disruptive methods. For example if both LiveMigrate and Shutdown
	// methods are listed, only VMs which are not live migratable will be restarted/shutdown
	//
	// An empty list defaults to no automated workload updating
	//
	// +listType=atomic
	// +optional
	WorkloadUpdateMethods []WorkloadUpdateMethod `json:"workloadUpdateMethods,omitempty"`

	// BatchEvictionSize Represents the number of VMIs that can be forced updated per
	// the BatchShutdownInteral interval
	//
	// Defaults to 10
	//
	// +optional
	BatchEvictionSize *int `json:"batchEvictionSize,omitempty"`

	// BatchEvictionInterval Represents the interval to wait before issuing the next
	// batch of shutdowns
	//
	// Defaults to 1 minute
	//
	// +optional
	BatchEvictionInterval *metav1.Duration `json:"batchEvictionInterval,omitempty"`
}

type KubeVirtSpec struct {
	// The image tag to use for the continer images installed.
	// Defaults to the same tag as the operator's container image.
	ImageTag string `json:"imageTag,omitempty"`
	// The image registry to pull the container images from
	// Defaults to the same registry the operator's container image is pulled from.
	ImageRegistry string `json:"imageRegistry,omitempty"`

	// The ImagePullPolicy to use.
	ImagePullPolicy k8sv1.PullPolicy `json:"imagePullPolicy,omitempty" valid:"required"`

	// The namespace Prometheus is deployed in
	// Defaults to openshift-monitor
	MonitorNamespace string `json:"monitorNamespace,omitempty"`

	// The namespace the service monitor will be deployed
	//  When ServiceMonitorNamespace is set, then we'll install the service monitor object in that namespace
	// otherwise we will use the monitoring namespace.
	ServiceMonitorNamespace string `json:"serviceMonitorNamespace,omitempty"`

	// The name of the Prometheus service account that needs read-access to KubeVirt endpoints
	// Defaults to prometheus-k8s
	MonitorAccount string `json:"monitorAccount,omitempty"`

	// WorkloadUpdateStrategy defines at the cluster level how to handle
	// automated workload updates
	WorkloadUpdateStrategy KubeVirtWorkloadUpdateStrategy `json:"workloadUpdateStrategy,omitempty"`

	// Specifies if kubevirt can be deleted if workloads are still present.
	// This is mainly a precaution to avoid accidental data loss
	UninstallStrategy KubeVirtUninstallStrategy `json:"uninstallStrategy,omitempty"`

	CertificateRotationStrategy KubeVirtCertificateRotateStrategy `json:"certificateRotateStrategy,omitempty"`

	// Designate the apps.kubevirt.io/version label for KubeVirt components.
	// Useful if KubeVirt is included as part of a product.
	// If ProductVersion is not specified, KubeVirt's version will be used.
	ProductVersion string `json:"productVersion,omitempty"`

	// Designate the apps.kubevirt.io/part-of label for KubeVirt components.
	// Useful if KubeVirt is included as part of a product.
	// If ProductName is not specified, the part-of label will be omitted.
	ProductName string `json:"productName,omitempty"`

	// Designate the apps.kubevirt.io/component label for KubeVirt components.
	// Useful if KubeVirt is included as part of a product.
	// If ProductComponent is not specified, the component label default value is kubevirt.
	ProductComponent string `json:"productComponent,omitempty"`

	// holds kubevirt configurations.
	// same as the virt-configMap
	Configuration KubeVirtConfiguration `json:"configuration,omitempty"`

	// selectors and tolerations that should apply to KubeVirt infrastructure components
	// +optional
	Infra *ComponentConfig `json:"infra,omitempty"`

	// selectors and tolerations that should apply to KubeVirt workloads
	// +optional
	Workloads *ComponentConfig `json:"workloads,omitempty"`

	CustomizeComponents CustomizeComponents `json:"customizeComponents,omitempty"`
}

type CustomizeComponents struct {
	// +listType=atomic
	Patches []CustomizeComponentsPatch `json:"patches,omitempty"`

	// Configure the value used for deployment and daemonset resources
	Flags *Flags `json:"flags,omitempty"`
}

// Flags will create a patch that will replace all flags for the container's
// command field. The only flags that will be used are those define. There are no
// guarantees around forward/backward compatibility.  If set incorrectly this will
// cause the resource when rolled out to error until flags are updated.
type Flags struct {
	API        map[string]string `json:"api,omitempty"`
	Controller map[string]string `json:"controller,omitempty"`
	Handler    map[string]string `json:"handler,omitempty"`
}

type CustomizeComponentsPatch struct {
	// +kubebuilder:validation:MinLength=1
	ResourceName string `json:"resourceName"`
	// +kubebuilder:validation:MinLength=1
	ResourceType string    `json:"resourceType"`
	Patch        string    `json:"patch"`
	Type         PatchType `json:"type"`
}

type PatchType string

const (
	JSONPatchType           PatchType = "json"
	MergePatchType          PatchType = "merge"
	StrategicMergePatchType PatchType = "strategic"
)

type KubeVirtUninstallStrategy string

const (
	KubeVirtUninstallStrategyRemoveWorkloads                KubeVirtUninstallStrategy = "RemoveWorkloads"
	KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist KubeVirtUninstallStrategy = "BlockUninstallIfWorkloadsExist"
)

// GenerationStatus keeps track of the generation for a given resource so that decisions about forced updates can be made.
type GenerationStatus struct {
	// group is the group of the thing you're tracking
	Group string `json:"group"`
	// resource is the resource type of the thing you're tracking
	Resource string `json:"resource"`
	// namespace is where the thing you're tracking is
	// +optional
	Namespace string `json:"namespace,omitempty" optional:"true"`
	// name is the name of the thing you're tracking
	Name string `json:"name"`
	// lastGeneration is the last generation of the workload controller involved
	LastGeneration int64 `json:"lastGeneration"`
	// hash is an optional field set for resources without generation that are content sensitive like secrets and configmaps
	// +optional
	Hash string `json:"hash,omitempty" optional:"true"`
}

// KubeVirtStatus represents information pertaining to a KubeVirt deployment.
type KubeVirtStatus struct {
	Phase                                   KubeVirtPhase       `json:"phase,omitempty"`
	Conditions                              []KubeVirtCondition `json:"conditions,omitempty" optional:"true"`
	OperatorVersion                         string              `json:"operatorVersion,omitempty" optional:"true"`
	TargetKubeVirtRegistry                  string              `json:"targetKubeVirtRegistry,omitempty" optional:"true"`
	TargetKubeVirtVersion                   string              `json:"targetKubeVirtVersion,omitempty" optional:"true"`
	TargetDeploymentConfig                  string              `json:"targetDeploymentConfig,omitempty" optional:"true"`
	TargetDeploymentID                      string              `json:"targetDeploymentID,omitempty" optional:"true"`
	ObservedKubeVirtRegistry                string              `json:"observedKubeVirtRegistry,omitempty" optional:"true"`
	ObservedKubeVirtVersion                 string              `json:"observedKubeVirtVersion,omitempty" optional:"true"`
	ObservedDeploymentConfig                string              `json:"observedDeploymentConfig,omitempty" optional:"true"`
	ObservedDeploymentID                    string              `json:"observedDeploymentID,omitempty" optional:"true"`
	OutdatedVirtualMachineInstanceWorkloads *int                `json:"outdatedVirtualMachineInstanceWorkloads,omitempty" optional:"true"`
	ObservedGeneration                      *int64              `json:"observedGeneration,omitempty"`
	// +listType=atomic
	Generations []GenerationStatus `json:"generations,omitempty" optional:"true"`
}

// KubeVirtPhase is a label for the phase of a KubeVirt deployment at the current time.
type KubeVirtPhase string

// These are the valid KubeVirt deployment phases
const (
	// The deployment is processing
	KubeVirtPhaseDeploying KubeVirtPhase = "Deploying"
	// The deployment succeeded
	KubeVirtPhaseDeployed KubeVirtPhase = "Deployed"
	// The deletion is processing
	KubeVirtPhaseDeleting KubeVirtPhase = "Deleting"
	// The deletion succeeeded
	KubeVirtPhaseDeleted KubeVirtPhase = "Deleted"
)

// KubeVirtCondition represents a condition of a KubeVirt deployment
type KubeVirtCondition struct {
	Type   KubeVirtConditionType `json:"type"`
	Status k8sv1.ConditionStatus `json:"status"`
	// +optional
	// +nullable
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// +optional
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Message            string      `json:"message,omitempty"`
}

type KubeVirtConditionType string

// These are the valid KubeVirt condition types
const (
	// Whether the deployment or deletion was successful (only used if false)
	KubeVirtConditionSynchronized KubeVirtConditionType = "Synchronized"
	// Whether all resources were created and up-to-date
	KubeVirtConditionCreated KubeVirtConditionType = "Created"

	// Conditions for HCO, see https://github.com/kubevirt/hyperconverged-cluster-operator/blob/master/docs/conditions.md
	// Whether KubeVirt is functional and available in the cluster.
	KubeVirtConditionAvailable KubeVirtConditionType = "Available"
	// Whether the operator is actively making changes to KubeVirt
	KubeVirtConditionProgressing KubeVirtConditionType = "Progressing"
	// Whether KubeVirt is not functioning completely
	KubeVirtConditionDegraded KubeVirtConditionType = "Degraded"
)

const (
	EvictionStrategyNone        EvictionStrategy = "None"
	EvictionStrategyLiveMigrate EvictionStrategy = "LiveMigrate"
	EvictionStrategyExternal    EvictionStrategy = "External"
)

// RestartOptions may be provided when deleting an API object.
type RestartOptions struct {
	metav1.TypeMeta `json:",inline"`

	// The duration in seconds before the object should be force-restarted. Value must be non-negative integer.
	// The value zero indicates, restart immediately. If this value is nil, the default grace period for deletion of the corresponding VMI for the
	// specified type will be used to determine on how much time to give the VMI to restart.
	// Defaults to a per object value if not specified. zero means restart immediately.
	// Allowed Values: nil and 0
	// +optional
	GracePeriodSeconds *int64 `json:"gracePeriodSeconds,omitempty" protobuf:"varint,1,opt,name=gracePeriodSeconds"`

	// When present, indicates that modifications should not be
	// persisted. An invalid or unrecognized dryRun directive will
	// result in an error response and no further processing of the
	// request. Valid values are:
	// - All: all dry run stages will be processed
	// +optional
	// +listType=atomic
	DryRun []string `json:"dryRun,omitempty" protobuf:"bytes,2,rep,name=dryRun"`
}

// StartOptions may be provided on start request.
type StartOptions struct {
	metav1.TypeMeta `json:",inline"`

	// Indicates that VM will be started in paused state.
	// +optional
	Paused bool `json:"paused,omitempty" protobuf:"varint,7,opt,name=paused"`
	// When present, indicates that modifications should not be
	// persisted. An invalid or unrecognized dryRun directive will
	// result in an error response and no further processing of the
	// request. Valid values are:
	// - All: all dry run stages will be processed
	// +optional
	// +listType=atomic
	DryRun []string `json:"dryRun,omitempty" protobuf:"bytes,5,rep,name=dryRun"`
}

// PauseOptions may be provided on pause request.
type PauseOptions struct {
	metav1.TypeMeta `json:",inline"`

	// When present, indicates that modifications should not be
	// persisted. An invalid or unrecognized dryRun directive will
	// result in an error response and no further processing of the
	// request. Valid values are:
	// - All: all dry run stages will be processed
	// +optional
	// +listType=atomic
	DryRun []string `json:"dryRun,omitempty" protobuf:"bytes,1,rep,name=dryRun"`
}

// UnpauseOptions may be provided on unpause request.
type UnpauseOptions struct {
	metav1.TypeMeta `json:",inline"`

	// When present, indicates that modifications should not be
	// persisted. An invalid or unrecognized dryRun directive will
	// result in an error response and no further processing of the
	// request. Valid values are:
	// - All: all dry run stages will be processed
	// +optional
	// +listType=atomic
	DryRun []string `json:"dryRun,omitempty" protobuf:"bytes,1,rep,name=dryRun"`
}

const (
	StartRequestDataPausedKey  string = "paused"
	StartRequestDataPausedTrue string = "true"
)

// StopOptions may be provided when deleting an API object.
type StopOptions struct {
	metav1.TypeMeta `json:",inline"`

	// this updates the VMIs terminationGracePeriodSeconds during shutdown
	// +optional
	GracePeriod *int64 `json:"gracePeriod,omitempty" protobuf:"varint,1,opt,name=gracePeriod"`
	// When present, indicates that modifications should not be
	// persisted. An invalid or unrecognized dryRun directive will
	// result in an error response and no further processing of the
	// request. Valid values are:
	// - All: all dry run stages will be processed
	// +optional
	// +listType=atomic
	DryRun []string `json:"dryRun,omitempty" protobuf:"bytes,2,rep,name=dryRun"`
}

// MigrateOptions may be provided on migrate request.
type MigrateOptions struct {
	metav1.TypeMeta `json:",inline"`
	// When present, indicates that modifications should not be
	// persisted. An invalid or unrecognized dryRun directive will
	// result in an error response and no further processing of the
	// request. Valid values are:
	// - All: all dry run stages will be processed
	// +optional
	// +listType=atomic
	DryRun []string `json:"dryRun,omitempty" protobuf:"bytes,1,rep,name=dryRun"`
}

// VirtualMachineInstanceGuestAgentInfo represents information from the installed guest agent
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineInstanceGuestAgentInfo struct {
	metav1.TypeMeta `json:",inline"`
	// GAVersion is a version of currently installed guest agent
	GAVersion string `json:"guestAgentVersion,omitempty"`
	// Return command list the guest agent supports
	// +listType=atomic
	SupportedCommands []GuestAgentCommandInfo `json:"supportedCommands,omitempty"`
	// Hostname represents FQDN of a guest
	Hostname string `json:"hostname,omitempty"`
	// OS contains the guest operating system information
	OS VirtualMachineInstanceGuestOSInfo `json:"os,omitempty"`
	// Timezone is guest os current timezone
	Timezone string `json:"timezone,omitempty"`
	// UserList is a list of active guest OS users
	UserList []VirtualMachineInstanceGuestOSUser `json:"userList,omitempty"`
	// FSInfo is a guest os filesystem information containing the disk mapping and disk mounts with usage
	FSInfo VirtualMachineInstanceFileSystemInfo `json:"fsInfo,omitempty"`
	// FSFreezeStatus is the state of the fs of the guest
	// it can be either frozen or thawed
	FSFreezeStatus string `json:"fsFreezeStatus,omitempty"`
}

// List of commands that QEMU guest agent supports
type GuestAgentCommandInfo struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled,omitempty"`
}

// VirtualMachineInstanceGuestOSUserList comprises the list of all active users on guest machine
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineInstanceGuestOSUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineInstanceGuestOSUser `json:"items"`
}

// VirtualMachineGuestOSUser is the single user of the guest os
type VirtualMachineInstanceGuestOSUser struct {
	UserName  string  `json:"userName"`
	Domain    string  `json:"domain,omitempty"`
	LoginTime float64 `json:"loginTime,omitempty"`
}

// VirtualMachineInstanceFileSystemInfo represents information regarding single guest os filesystem
type VirtualMachineInstanceFileSystemInfo struct {
	Filesystems []VirtualMachineInstanceFileSystem `json:"disks"`
}

// VirtualMachineInstanceFileSystemList comprises the list of all filesystems on guest machine
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineInstanceFileSystemList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineInstanceFileSystem `json:"items"`
}

// VirtualMachineInstanceFileSystem represents guest os disk
type VirtualMachineInstanceFileSystem struct {
	DiskName       string `json:"diskName"`
	MountPoint     string `json:"mountPoint"`
	FileSystemType string `json:"fileSystemType"`
	UsedBytes      int    `json:"usedBytes"`
	TotalBytes     int    `json:"totalBytes"`
}

// FreezeUnfreezeTimeout represent the time unfreeze will be triggered if guest was not unfrozen by unfreeze command
type FreezeUnfreezeTimeout struct {
	UnfreezeTimeout *metav1.Duration `json:"unfreezeTimeout"`
}

// VirtualMachineMemoryDumpRequest represent the memory dump request phase and info
type VirtualMachineMemoryDumpRequest struct {
	// ClaimName is the name of the pvc that will contain the memory dump
	ClaimName string `json:"claimName"`
	// Phase represents the memory dump phase
	Phase MemoryDumpPhase `json:"phase"`
	// StartTimestamp represents the time the memory dump started
	StartTimestamp *metav1.Time `json:"startTimestamp,omitempty"`
	// EndTimestamp represents the time the memory dump was completed
	EndTimestamp *metav1.Time `json:"endTimestamp,omitempty"`
	// FileName represents the name of the output file
	FileName *string `json:"fileName,omitempty"`
	// Message is a detailed message about failure of the memory dump
	Message string `json:"message,omitempty"`
}

type MemoryDumpPhase string

const (
	// The memorydump is during pvc Associating
	MemoryDumpAssociating MemoryDumpPhase = "Associating"
	// The memorydump is in progress
	MemoryDumpInProgress MemoryDumpPhase = "InProgress"
	// The memorydump is being unmounted
	MemoryDumpUnmounting MemoryDumpPhase = "Unmounting"
	// The memorydump is completed
	MemoryDumpCompleted MemoryDumpPhase = "Completed"
	// The memorydump is being unbound
	MemoryDumpDissociating MemoryDumpPhase = "Dissociating"
	// The memorydump failed
	MemoryDumpFailed MemoryDumpPhase = "Failed"
)

// AddVolumeOptions is provided when dynamically hot plugging a volume and disk
type AddVolumeOptions struct {
	// Name represents the name that will be used to map the
	// disk to the corresponding volume. This overrides any name
	// set inside the Disk struct itself.
	Name string `json:"name"`
	// Disk represents the hotplug disk that will be plugged into the running VMI
	Disk *Disk `json:"disk"`
	// VolumeSource represents the source of the volume to map to the disk.
	VolumeSource *HotplugVolumeSource `json:"volumeSource"`
	// When present, indicates that modifications should not be
	// persisted. An invalid or unrecognized dryRun directive will
	// result in an error response and no further processing of the
	// request. Valid values are:
	// - All: all dry run stages will be processed
	// +optional
	// +listType=atomic
	DryRun []string `json:"dryRun,omitempty"`
}

// RemoveVolumeOptions is provided when dynamically hot unplugging volume and disk
type RemoveVolumeOptions struct {
	// Name represents the name that maps to both the disk and volume that
	// should be removed
	Name string `json:"name"`
	// When present, indicates that modifications should not be
	// persisted. An invalid or unrecognized dryRun directive will
	// result in an error response and no further processing of the
	// request. Valid values are:
	// - All: all dry run stages will be processed
	// +optional
	// +listType=atomic
	DryRun []string `json:"dryRun,omitempty"`
}

type TokenBucketRateLimiter struct {
	// QPS indicates the maximum QPS to the apiserver from this client.
	// If it's zero, the component default will be used
	QPS float32 `json:"qps"`

	// Maximum burst for throttle.
	// If it's zero, the component default will be used
	Burst int `json:"burst"`
}

type RateLimiter struct {
	TokenBucketRateLimiter *TokenBucketRateLimiter `json:"tokenBucketRateLimiter,omitempty"`
}

// RESTClientConfiguration allows configuring certain aspects of the k8s rest client.
type RESTClientConfiguration struct {
	//RateLimiter allows selecting and configuring different rate limiters for the k8s client.
	RateLimiter *RateLimiter `json:"rateLimiter,omitempty"`
}

// ReloadableComponentConfiguration holds all generic k8s configuration options which can
// be reloaded by components without requiring a restart.
type ReloadableComponentConfiguration struct {
	//RestClient can be used to tune certain aspects of the k8s client in use.
	RestClient *RESTClientConfiguration `json:"restClient,omitempty"`
}

// KubeVirtConfiguration holds all kubevirt configurations
type KubeVirtConfiguration struct {
	CPUModel               string                  `json:"cpuModel,omitempty"`
	CPURequest             *resource.Quantity      `json:"cpuRequest,omitempty"`
	DeveloperConfiguration *DeveloperConfiguration `json:"developerConfiguration,omitempty"`
	EmulatedMachines       []string                `json:"emulatedMachines,omitempty"`
	ImagePullPolicy        k8sv1.PullPolicy        `json:"imagePullPolicy,omitempty"`
	MigrationConfiguration *MigrationConfiguration `json:"migrations,omitempty"`
	MachineType            string                  `json:"machineType,omitempty"`
	NetworkConfiguration   *NetworkConfiguration   `json:"network,omitempty"`
	OVMFPath               string                  `json:"ovmfPath,omitempty"`
	SELinuxLauncherType    string                  `json:"selinuxLauncherType,omitempty"`
	DefaultRuntimeClass    string                  `json:"defaultRuntimeClass,omitempty"`
	SMBIOSConfig           *SMBiosConfiguration    `json:"smbios,omitempty"`

	// EvictionStrategy defines at the cluster level if the VirtualMachineInstance should be
	// migrated instead of shut-off in case of a node drain. If the VirtualMachineInstance specific
	// field is set it overrides the cluster level one.
	EvictionStrategy *EvictionStrategy `json:"evictionStrategy,omitempty"`

	// deprecated
	SupportedGuestAgentVersions    []string                          `json:"supportedGuestAgentVersions,omitempty"`
	MemBalloonStatsPeriod          *uint32                           `json:"memBalloonStatsPeriod,omitempty"`
	PermittedHostDevices           *PermittedHostDevices             `json:"permittedHostDevices,omitempty"`
	MediatedDevicesConfiguration   *MediatedDevicesConfiguration     `json:"mediatedDevicesConfiguration,omitempty"`
	MinCPUModel                    string                            `json:"minCPUModel,omitempty"`
	ObsoleteCPUModels              map[string]bool                   `json:"obsoleteCPUModels,omitempty"`
	VirtualMachineInstancesPerNode *int                              `json:"virtualMachineInstancesPerNode,omitempty"`
	APIConfiguration               *ReloadableComponentConfiguration `json:"apiConfiguration,omitempty"`
	WebhookConfiguration           *ReloadableComponentConfiguration `json:"webhookConfiguration,omitempty"`
	ControllerConfiguration        *ReloadableComponentConfiguration `json:"controllerConfiguration,omitempty"`
	HandlerConfiguration           *ReloadableComponentConfiguration `json:"handlerConfiguration,omitempty"`
}

type SMBiosConfiguration struct {
	Manufacturer string `json:"manufacturer,omitempty"`
	Product      string `json:"product,omitempty"`
	Version      string `json:"version,omitempty"`
	Sku          string `json:"sku,omitempty"`
	Family       string `json:"family,omitempty"`
}

// MigrationConfiguration holds migration options
type MigrationConfiguration struct {
	NodeDrainTaintKey                 *string            `json:"nodeDrainTaintKey,omitempty"`
	ParallelOutboundMigrationsPerNode *uint32            `json:"parallelOutboundMigrationsPerNode,omitempty"`
	ParallelMigrationsPerCluster      *uint32            `json:"parallelMigrationsPerCluster,omitempty"`
	AllowAutoConverge                 *bool              `json:"allowAutoConverge,omitempty"`
	BandwidthPerMigration             *resource.Quantity `json:"bandwidthPerMigration,omitempty"`
	CompletionTimeoutPerGiB           *int64             `json:"completionTimeoutPerGiB,omitempty"`
	ProgressTimeout                   *int64             `json:"progressTimeout,omitempty"`
	UnsafeMigrationOverride           *bool              `json:"unsafeMigrationOverride,omitempty"`
	AllowPostCopy                     *bool              `json:"allowPostCopy,omitempty"`
	DisableTLS                        *bool              `json:"disableTLS,omitempty"`
	Network                           *string            `json:"network,omitempty"`
}

// DiskVerification holds container disks verification limits
type DiskVerification struct {
	MemoryLimit *resource.Quantity `json:"memoryLimit"`
}

// DeveloperConfiguration holds developer options
type DeveloperConfiguration struct {
	FeatureGates           []string          `json:"featureGates,omitempty"`
	LessPVCSpaceToleration int               `json:"pvcTolerateLessSpaceUpToPercent,omitempty"`
	MinimumReservePVCBytes uint64            `json:"minimumReservePVCBytes,omitempty"`
	MemoryOvercommit       int               `json:"memoryOvercommit,omitempty"`
	NodeSelectors          map[string]string `json:"nodeSelectors,omitempty"`
	// UseEmulation can be set to true to allow fallback to software emulation
	// in case hardware-assisted emulation is not available.
	UseEmulation       bool `json:"useEmulation,omitempty"`
	CPUAllocationRatio int  `json:"cpuAllocationRatio,omitempty"`
	// Allow overriding the automatically determined minimum TSC frequency of the cluster
	// and fixate the minimum to this frequency.
	MinimumClusterTSCFrequency *int64            `json:"minimumClusterTSCFrequency,omitempty"`
	DiskVerification           *DiskVerification `json:"diskVerification,omitempty"`
	LogVerbosity               *LogVerbosity     `json:"logVerbosity,omitempty"`
}

// LogVerbosity sets log verbosity level of  various components
type LogVerbosity struct {
	VirtAPI        uint `json:"virtAPI,omitempty"`
	VirtController uint `json:"virtController,omitempty"`
	VirtHandler    uint `json:"virtHandler,omitempty"`
	VirtLauncher   uint `json:"virtLauncher,omitempty"`
	VirtOperator   uint `json:"virtOperator,omitempty"`
	// NodeVerbosity represents a map of nodes with a specific verbosity level
	NodeVerbosity map[string]uint `json:"nodeVerbosity,omitempty"`
}

// PermittedHostDevices holds information about devices allowed for passthrough
type PermittedHostDevices struct {
	// +listType=atomic
	PciHostDevices []PciHostDevice `json:"pciHostDevices,omitempty"`
	// +listType=atomic
	MediatedDevices []MediatedHostDevice `json:"mediatedDevices,omitempty"`
}

// PciHostDevice represents a host PCI device allowed for passthrough
type PciHostDevice struct {
	// The vendor_id:product_id tuple of the PCI device
	PCIVendorSelector string `json:"pciVendorSelector"`
	// The name of the resource that is representing the device. Exposed by
	// a device plugin and requested by VMs. Typically of the form
	// vendor.com/product_nameThe name of the resource that is representing
	// the device. Exposed by a device plugin and requested by VMs.
	// Typically of the form vendor.com/product_name
	ResourceName string `json:"resourceName"`
	// If true, KubeVirt will leave the allocation and monitoring to an
	// external device plugin
	ExternalResourceProvider bool `json:"externalResourceProvider,omitempty"`
}

// MediatedHostDevice represents a host mediated device allowed for passthrough
type MediatedHostDevice struct {
	MDEVNameSelector         string `json:"mdevNameSelector"`
	ResourceName             string `json:"resourceName"`
	ExternalResourceProvider bool   `json:"externalResourceProvider,omitempty"`
}

// MediatedDevicesConfiguration holds information about MDEV types to be defined, if available
type MediatedDevicesConfiguration struct {
	// +listType=atomic
	MediatedDevicesTypes []string `json:"mediatedDevicesTypes,omitempty"`
	// +optional
	// +listType=atomic
	NodeMediatedDeviceTypes []NodeMediatedDeviceTypesConfig `json:"nodeMediatedDeviceTypes,omitempty"`
}

// NodeMediatedDeviceTypesConfig holds information about MDEV types to be defined in a specifc node that matches the NodeSelector field.
// +k8s:openapi-gen=true
type NodeMediatedDeviceTypesConfig struct {
	// NodeSelector is a selector which must be true for the vmi to fit on a node.
	// Selector which must match a node's labels for the vmi to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	NodeSelector map[string]string `json:"nodeSelector"`
	// +listType=atomic
	MediatedDevicesTypes []string `json:"mediatedDevicesTypes"`
}

// NetworkConfiguration holds network options
type NetworkConfiguration struct {
	NetworkInterface                  string `json:"defaultNetworkInterface,omitempty"`
	PermitSlirpInterface              *bool  `json:"permitSlirpInterface,omitempty"`
	PermitBridgeInterfaceOnPodNetwork *bool  `json:"permitBridgeInterfaceOnPodNetwork,omitempty"`
}

// GuestAgentPing configures the guest-agent based ping probe
type GuestAgentPing struct {
}

type ProfilerResult struct {
	PprofData map[string][]byte `json:"pprofData,omitempty"`
}

type ClusterProfilerResults struct {
	ComponentResults map[string]ProfilerResult `json:"componentResults"`
	Continue         string                    `json:"continue,omitempty"`
}

type ClusterProfilerRequest struct {
	LabelSelector string `json:"labelSelector,omitempty"`
	Continue      string `json:"continue,omitempty"`
	PageSize      int64  `json:"pageSize"`
}

// FlavorMatcher references a flavor that is used to fill fields in the VMI template.
type FlavorMatcher struct {
	// Name is the name of the VirtualMachineFlavor or VirtualMachineClusterFlavor
	Name string `json:"name"`

	// Kind specifies which flavor resource is referenced.
	// Allowed values are: "VirtualMachineFlavor" and "VirtualMachineClusterFlavor".
	// If not specified, "VirtualMachineClusterFlavor" is used by default.
	//
	// +optional
	Kind string `json:"kind,omitempty"`
}

// PreferenceMatcher references a set of preference that is used to fill fields in the VMI template.
type PreferenceMatcher struct {
	// Name is the name of the VirtualMachinePreference or VirtualMachineClusterPreference
	Name string `json:"name"`

	// Kind specifies which preference resource is referenced.
	// Allowed values are: "VirtualMachinePreference" and "VirtualMachineClusterPreference".
	// If not specified, "VirtualMachineClusterPreference" is used by default.
	//
	// +optional
	Kind string `json:"kind,omitempty"`
}
