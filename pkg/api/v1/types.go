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

//go:generate swagger-doc
//go:generate deepcopy-gen -i . --go-header-file ../../../hack/boilerplate/boilerplate.go.txt
//go:generate defaulter-gen -i . --go-header-file ../../../hack/boilerplate/boilerplate.go.txt
//go:generate openapi-gen -i . --output-package=kubevirt.io/kubevirt/pkg/api/v1  --go-header-file ../../../hack/boilerplate/boilerplate.go.txt

/*
 ATTENTION: Rerun code generators when comments on structs or fields are modified.
*/

import (
	"encoding/json"
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/datavolumecontroller/v1alpha1"
	"kubevirt.io/kubevirt/pkg/precond"
)

// GroupName is the group name use in this package
const GroupName = "kubevirt.io"
const SubresourceGroupName = "subresources.kubevirt.io"

const DefaultGracePeriodSeconds int64 = 30

// GroupVersion is group version used to register these objects
var GroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha2"}

// GroupVersion is group version used to register these objects
var SubresourceGroupVersion = schema.GroupVersion{Group: SubresourceGroupName, Version: "v1alpha2"}

// GroupVersionKind
var VirtualMachineInstanceGroupVersionKind = schema.GroupVersionKind{Group: GroupName, Version: GroupVersion.Version, Kind: "VirtualMachineInstance"}

var VirtualMachineInstanceReplicaSetGroupVersionKind = schema.GroupVersionKind{Group: GroupName, Version: GroupVersion.Version, Kind: "VirtualMachineInstanceReplicaSet"}

var VirtualMachineInstancePresetGroupVersionKind = schema.GroupVersionKind{Group: GroupName, Version: GroupVersion.Version, Kind: "VirtualMachineInstancePreset"}

var VirtualMachineGroupVersionKind = schema.GroupVersionKind{Group: GroupName, Version: GroupVersion.Version, Kind: "VirtualMachine"}

var VirtualMachineInstanceMigrationGroupVersionKind = schema.GroupVersionKind{Group: GroupName, Version: GroupVersion.Version, Kind: "VirtualMachineInstanceMigration"}

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&VirtualMachineInstance{},
		&VirtualMachineInstanceList{},
		&metav1.ListOptions{},
		&metav1.DeleteOptions{},
		&VirtualMachineInstanceReplicaSet{},
		&VirtualMachineInstanceReplicaSetList{},
		&VirtualMachineInstancePreset{},
		&VirtualMachineInstancePresetList{},
		&VirtualMachineInstanceMigration{},
		&VirtualMachineInstanceMigrationList{},
		&metav1.GetOptions{},
		&VirtualMachine{},
		&VirtualMachineList{},
	)
	scheme.AddKnownTypes(metav1.Unversioned,
		&metav1.Status{},
	)
	return nil
}

var (
	Scheme         = runtime.NewScheme()
	Codecs         = serializer.NewCodecFactory(Scheme)
	ParameterCodec = runtime.NewParameterCodec(Scheme)
	SchemeBuilder  = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme    = SchemeBuilder.AddToScheme
)

func init() {
	AddToScheme(Scheme)
	AddToScheme(scheme.Scheme)
}

// VirtualMachineInstance is *the* VirtualMachineInstance Definition. It represents a virtual machine in the runtime environment of kubernetes.
// ---
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachineInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// VirtualMachineInstance Spec contains the VirtualMachineInstance specification.
	Spec VirtualMachineInstanceSpec `json:"spec,omitempty" valid:"required"`
	// Status is the high level overview of how the VirtualMachineInstance is doing. It contains information available to controllers and users.
	Status VirtualMachineInstanceStatus `json:"status,omitempty"`
}

// VirtualMachineInstanceList is a list of VirtualMachines
// ---
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachineInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	ListMeta        metav1.ListMeta          `json:"metadata,omitempty"`
	Items           []VirtualMachineInstance `json:"items"`
}

// VirtualMachineInstanceSpec is a description of a VirtualMachineInstance.
// ---
// +k8s:openapi-gen=true
type VirtualMachineInstanceSpec struct {
	// Specification of the desired behavior of the VirtualMachineInstance on the host.
	Domain DomainSpec `json:"domain"`
	// NodeSelector is a selector which must be true for the vmi to fit on a node.
	// Selector which must match a node's labels for the vmi to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// If affinity is specifies, obey all the affinity rules
	Affinity *k8sv1.Affinity `json:"affinity,omitempty"`
	// If toleration is specified, obey all the toleration rules.
	Tolerations []k8sv1.Toleration `json:"tolerations,omitempty"`
	// Grace period observed after signalling a VirtualMachineInstance to stop after which the VirtualMachineInstance is force terminated.
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`
	// List of volumes that can be mounted by disks belonging to the vmi.
	Volumes []Volume `json:"volumes,omitempty"`
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
}

// VirtualMachineInstanceStatus represents information about the status of a VirtualMachineInstance. Status may trail the actual
// state of a system.
// ---
// +k8s:openapi-gen=true
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
	// Interfaces represent the details of available network interfaces.
	Interfaces []VirtualMachineInstanceNetworkInterface `json:"interfaces,omitempty"`
	// Represents the status of a live migration
	MigrationState *VirtualMachineInstanceMigrationState `json:"migrationState,omitempty"`
}

// Required to satisfy Object interface
func (v *VirtualMachineInstance) GetObjectKind() schema.ObjectKind {
	return &v.TypeMeta
}

// Required to satisfy ObjectMetaAccessor interface
func (v *VirtualMachineInstance) GetObjectMeta() metav1.Object {
	return &v.ObjectMeta
}

func (v *VirtualMachineInstance) IsReady() bool {
	// TODO once we support a ready condition, use it instead
	return v.IsRunning()
}

func (v *VirtualMachineInstance) IsScheduling() bool {
	return v.Status.Phase == Scheduling
}

func (v *VirtualMachineInstance) IsScheduled() bool {
	return v.Status.Phase == Scheduled
}

func (v *VirtualMachineInstance) IsRunning() bool {
	return v.Status.Phase == Running
}

func (v *VirtualMachineInstance) IsFinal() bool {
	return v.Status.Phase == Failed || v.Status.Phase == Succeeded
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

// Required to satisfy Object interface
func (vl *VirtualMachineInstanceList) GetObjectKind() schema.ObjectKind {
	return &vl.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (vl *VirtualMachineInstanceList) GetListMeta() meta.List {
	return &vl.ListMeta
}

func (v *VirtualMachineInstance) UnmarshalJSON(data []byte) error {
	type VMICopy VirtualMachineInstance
	tmp := VMICopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := VirtualMachineInstance(tmp)
	*v = tmp2
	return nil
}

func (vl *VirtualMachineInstanceList) UnmarshalJSON(data []byte) error {
	type VMIListCopy VirtualMachineInstanceList
	tmp := VMIListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := VirtualMachineInstanceList(tmp)
	*vl = tmp2
	return nil
}

func (v *VirtualMachineInstance) MarshalBinary() (data []byte, err error) {
	return json.Marshal(*v)
}

func (v *VirtualMachineInstance) UnmarshalBinary(data []byte) error {
	return v.UnmarshalJSON(data)
}

// ---
// +k8s:openapi-gen=true
type VirtualMachineInstanceConditionType string

// These are valid conditions of VMIs.
const (
	// VMIReady means the pod is able to service requests and should be added to the
	// load balancing pools of all matching services.
	VirtualMachineInstanceReady VirtualMachineInstanceConditionType = "Ready"

	// If there happens any error while trying to synchronize the VirtualMachineInstance with the Domain,
	// this is reported as false.
	VirtualMachineInstanceSynchronized VirtualMachineInstanceConditionType = "Synchronized"
)

// ---
// +k8s:openapi-gen=true
type VirtualMachineInstanceCondition struct {
	Type               VirtualMachineInstanceConditionType `json:"type"`
	Status             k8sv1.ConditionStatus               `json:"status"`
	LastProbeTime      metav1.Time                         `json:"lastProbeTime,omitempty"`
	LastTransitionTime metav1.Time                         `json:"lastTransitionTime,omitempty"`
	Reason             string                              `json:"reason,omitempty"`
	Message            string                              `json:"message,omitempty"`
}

// The migration phase indicates that the job has completed
func (m *VirtualMachineInstanceMigration) IsFinal() bool {
	return m.Status.Phase == MigrationFailed || m.Status.Phase == MigrationSucceeded
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

// ---
// +k8s:openapi-gen=true
type VirtualMachineInstanceNetworkInterface struct {
	// IP address of a Virtual Machine interface
	IP string `json:"ipAddress,omitempty"`
	// Hardware address of a Virtual Machine interface
	MAC string `json:"mac,omitempty"`
}

type VirtualMachineInstanceMigrationState struct {
	// The time the migration action began
	StartTimestamp *metav1.Time `json:"startTimestamp,omitempty"`
	// The time the migration action ended
	EndTimestamp *metav1.Time `json:"endTimestamp,omitempty"`
	// The Target Node has seen the Domain Start Event
	TargetNodeDomainDetected bool `json:"targetNodeDomainDetected,omitempty"`
	// The address of the target node to use for the migration
	TargetNodeAddress string `json:"targetNodeAddress,omitempty"`
	// The target node that the VMI is moving to
	TargetNode string `json:"targetNode,omitempty"`
	// The source node that the VMI originated on
	SourceNode string `json:"sourceNode,omitempty"`
	// Indicates the migration completed
	Completed bool `json:"completed,omitempty"`
	// Indicates that the migration failed
	Failed bool `json:"failed,omitempty"`
	// The VirtualMachineInstanceMigration object associated with this migration
	MigrationUID types.UID `json:"migrationUid,omitempty"`
}

// VirtualMachineInstancePhase is a label for the condition of a VirtualMachineInstance at the current time.
// ---
// +k8s:openapi-gen=true
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
	// This label marks resources that belong to KubeVirt. An optional value
	// may indicate which specific KubeVirt component a resource belongs to.
	AppLabel string = "kubevirt.io"
	// This annotation is used to match virtual machine instances represented as
	// libvirt XML domains with their pods. Among other things, the annotation is
	// used to detect virtual machines with dead pods. Used on Pod.
	DomainAnnotation string = "kubevirt.io/domain"
	// Represents the name of the migration job this target pod is associated with
	MigrationJobNameAnnotation string = "kubevirt.io/migrationJobName"
	// This label is used to match virtual machine instance IDs with pods.
	// Similar to kubevirt.io/domain. Used on Pod.
	CreatedByLabel string = "kubevirt.io/created-by"
	// This label is used to indicate that this pod is the target of a migration job.
	MigrationJobLabel string = "kubevirt.io/migrationJobUID"
	// This annotation defines which KubeVirt component owns the resource. Used
	// on Pod.
	OwnedByAnnotation string = "kubevirt.io/owned-by"
	// This label describes which cluster node runs the virtual machine
	// instance. Needed because with CRDs we can't use field selectors. Used on
	// VirtualMachineInstance.
	NodeNameLabel string = "kubevirt.io/nodeName"
	// This label describes which cluster node runs the target Pod for a Virtual
	// Machine Instance migration job. Needed because with CRDs we can't use field
	// selectors. Used on VirtualMachineInstance.
	MigrationTargetNodeNameLabel string = "kubevirt.io/migrationTargetNodeName"
	// This label declares whether a particular node is available for
	// scheduling virtual machine instances on it. Used on Node.
	NodeSchedulable string = "kubevirt.io/schedulable"
	// This annotation is regularly updated by virt-handler to help determine
	// if a particular node is alive and hence should be available for new
	// virtual machine instance scheduling. Used on Node.
	VirtHandlerHeartbeat string = "kubevirt.io/heartbeat"

	VirtualMachineInstanceFinalizer string = "foregroundDeleteVirtualMachine"
	CPUManager                      string = "cpumanager"
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

// ---
// +k8s:openapi-gen=true
type SyncEvent string

const (
	Created         SyncEvent = "Created"
	Deleted         SyncEvent = "Deleted"
	PresetFailed    SyncEvent = "PresetFailed"
	Override        SyncEvent = "Override"
	Started         SyncEvent = "Started"
	ShuttingDown    SyncEvent = "ShuttingDown"
	Stopped         SyncEvent = "Stopped"
	PreparingTarget SyncEvent = "PreparingTarget"
	Migrating       SyncEvent = "Migrating"
	Migrated        SyncEvent = "Migrated"
	SyncFailed      SyncEvent = "SyncFailed"
	Resumed         SyncEvent = "Resumed"
)

func (s SyncEvent) String() string {
	return string(s)
}

func NewMinimalVMI(vmiName string) *VirtualMachineInstance {
	return NewMinimalVMIWithNS(k8sv1.NamespaceDefault, vmiName)
}

func NewMinimalVMIWithNS(namespace string, vmiName string) *VirtualMachineInstance {
	precond.CheckNotEmpty(vmiName)
	vmi := NewVMIReferenceFromNameWithNS(namespace, vmiName)
	vmi.Spec = VirtualMachineInstanceSpec{Domain: NewMinimalDomainSpec()}
	vmi.TypeMeta = metav1.TypeMeta{
		APIVersion: GroupVersion.String(),
		Kind:       "VirtualMachineInstance",
	}
	return vmi
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
// ---
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachineInstanceReplicaSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// VirtualMachineInstance Spec contains the VirtualMachineInstance specification.
	Spec VirtualMachineInstanceReplicaSetSpec `json:"spec,omitempty" valid:"required"`
	// Status is the high level overview of how the VirtualMachineInstance is doing. It contains information available to controllers and users.
	Status VirtualMachineInstanceReplicaSetStatus `json:"status,omitempty"`
}

// VMIList is a list of VMIs
// ---
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachineInstanceReplicaSetList struct {
	metav1.TypeMeta `json:",inline"`
	ListMeta        metav1.ListMeta                    `json:"metadata,omitempty"`
	Items           []VirtualMachineInstanceReplicaSet `json:"items"`
}

// ---
// +k8s:openapi-gen=true
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

// ---
// +k8s:openapi-gen=true
type VirtualMachineInstanceReplicaSetStatus struct {
	// Total number of non-terminated pods targeted by this deployment (their labels match the selector).
	// +optional
	Replicas int32 `json:"replicas,omitempty" protobuf:"varint,2,opt,name=replicas"`

	// The number of ready replicas for this replica set.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty" protobuf:"varint,4,opt,name=readyReplicas"`

	Conditions []VirtualMachineInstanceReplicaSetCondition `json:"conditions,omitempty" optional:"true"`
}

// ---
// +k8s:openapi-gen=true
type VirtualMachineInstanceReplicaSetCondition struct {
	Type               VirtualMachineInstanceReplicaSetConditionType `json:"type"`
	Status             k8sv1.ConditionStatus                         `json:"status"`
	LastProbeTime      metav1.Time                                   `json:"lastProbeTime,omitempty"`
	LastTransitionTime metav1.Time                                   `json:"lastTransitionTime,omitempty"`
	Reason             string                                        `json:"reason,omitempty"`
	Message            string                                        `json:"message,omitempty"`
}

// ---
// +k8s:openapi-gen=true
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

// ---
// +k8s:openapi-gen=true
type VirtualMachineInstanceTemplateSpec struct {
	ObjectMeta metav1.ObjectMeta `json:"metadata,omitempty"`
	// VirtualMachineInstance Spec contains the VirtualMachineInstance specification.
	Spec VirtualMachineInstanceSpec `json:"spec,omitempty" valid:"required"`
}

// Required to satisfy Object interface
func (v *VirtualMachineInstanceReplicaSet) GetObjectKind() schema.ObjectKind {
	return &v.TypeMeta
}

// Required to satisfy ObjectMetaAccessor interface
func (v *VirtualMachineInstanceReplicaSet) GetObjectMeta() metav1.Object {
	return &v.ObjectMeta
}

func (v *VirtualMachineInstanceReplicaSet) UnmarshalJSON(data []byte) error {
	type VMIReplicaSetCopy VirtualMachineInstanceReplicaSet
	tmp := VMIReplicaSetCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := VirtualMachineInstanceReplicaSet(tmp)
	*v = tmp2
	return nil
}

func (vl *VirtualMachineInstanceReplicaSetList) UnmarshalJSON(data []byte) error {
	type VMIReplicaSetListCopy VirtualMachineInstanceReplicaSetList
	tmp := VMIReplicaSetListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := VirtualMachineInstanceReplicaSetList(tmp)
	*vl = tmp2
	return nil
}

// Required to satisfy Object interface
func (vl *VirtualMachineInstanceReplicaSetList) GetObjectKind() schema.ObjectKind {
	return &vl.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (vl *VirtualMachineInstanceReplicaSetList) GetListMeta() meta.List {
	return &vl.ListMeta
}

// VirtualMachineInstanceMigration represents the object tracking a VMI's migration
// to another host in the cluster
// ---
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachineInstanceMigration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VirtualMachineInstanceMigrationSpec   `json:"spec,omitempty" valid:"required"`
	Status            VirtualMachineInstanceMigrationStatus `json:"status,omitempty"`
}

// Required to satisfy Object interface
func (v *VirtualMachineInstanceMigration) GetObjectKind() schema.ObjectKind {
	return &v.TypeMeta
}

// Required to satisfy ObjectMetaAccessor interface
func (v *VirtualMachineInstanceMigration) GetObjectMeta() metav1.Object {
	return &v.ObjectMeta
}

// VirtualMachineInstanceMigrationList is a list of VirtualMachineMigrations
// ---
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachineInstanceMigrationList struct {
	metav1.TypeMeta `json:",inline"`
	ListMeta        metav1.ListMeta                   `json:"metadata,omitempty"`
	Items           []VirtualMachineInstanceMigration `json:"items"`
}

// Required to satisfy Object interface
func (vl *VirtualMachineInstanceMigrationList) GetObjectKind() schema.ObjectKind {
	return &vl.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (vl *VirtualMachineInstanceMigrationList) GetListMeta() meta.List {
	return &vl.ListMeta
}

// ---
// +k8s:openapi-gen=true
type VirtualMachineInstanceMigrationSpec struct {
	// The name of the VMI to perform the migration on. VMI must exist in the migration objects namespace
	VMIName string `json:"vmiName,omitempty" valid:"required"`
}

// VirtualMachineInstanceMigration reprents information pertaining to a VMI's migration.
// ---
// +k8s:openapi-gen=true
type VirtualMachineInstanceMigrationStatus struct {
	Phase VirtualMachineInstanceMigrationPhase `json:"phase,omitempty"`
}

// VirtualMachineInstanceMigrationPhase is a label for the condition of a VirtualMachineInstanceMigration at the current time.
// ---
// +k8s:openapi-gen=true
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

// ---
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachineInstancePreset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// VirtualMachineInstance Spec contains the VirtualMachineInstance specification.
	Spec VirtualMachineInstancePresetSpec `json:"spec,omitempty" valid:"required"`
}

// Required to satisfy Object interface
func (v *VirtualMachineInstancePreset) GetObjectKind() schema.ObjectKind {
	return &v.TypeMeta
}

// Required to satisfy ObjectMetaAccessor interface
func (v *VirtualMachineInstancePreset) GetObjectMeta() metav1.Object {
	return &v.ObjectMeta
}

// VirtualMachineInstancePresetList is a list of VirtualMachinePresets
// ---
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachineInstancePresetList struct {
	metav1.TypeMeta `json:",inline"`
	ListMeta        metav1.ListMeta                `json:"metadata,omitempty"`
	Items           []VirtualMachineInstancePreset `json:"items"`
}

// ---
// +k8s:openapi-gen=true
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

func (vl *VirtualMachineInstancePresetList) UnmarshalJSON(data []byte) error {
	type VirtualMachinePresetListCopy VirtualMachineInstancePresetList
	tmp := VirtualMachinePresetListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := VirtualMachineInstancePresetList(tmp)
	*vl = tmp2
	return nil
}

// Required to satisfy Object interface
func (vl *VirtualMachineInstancePresetList) GetObjectKind() schema.ObjectKind {
	return &vl.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (vl *VirtualMachineInstancePresetList) GetListMeta() meta.List {
	return &vl.ListMeta
}

// VirtualMachine handles the VirtualMachines that are not running
// or are in a stopped state
// The VirtualMachine contains the template to create the
// VirtualMachineInstance. It also mirrors the running state of the created
// VirtualMachineInstance in its status.
// ---
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec contains the specification of VirtualMachineInstance created
	Spec VirtualMachineSpec `json:"spec,omitempty"`
	// Status holds the current state of the controller and brief information
	// about its associated VirtualMachineInstance
	Status VirtualMachineStatus `json:"status,omitempty"`
}

// VirtualMachineList is a list of virtualmachines
// ---
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	// Items is a list of VirtualMachines
	Items []VirtualMachine `json:"items"`
}

// VirtualMachineSpec describes how the proper VirtualMachine
// should look like
// ---
// +k8s:openapi-gen=true
type VirtualMachineSpec struct {
	// Running controls whether the associatied VirtualMachineInstance is created or not
	Running bool `json:"running"`

	// Template is the direct specification of VirtualMachineInstance
	Template *VirtualMachineInstanceTemplateSpec `json:"template"`

	// dataVolumeTemplates is a list of dataVolumes that the VirtualMachineInstance template can reference.
	// DataVolumes in this list are dynamically created for the VirtualMachine and are tied to the VirtualMachine's life-cycle.
	DataVolumeTemplates []cdiv1.DataVolume `json:"dataVolumeTemplates,omitempty"`
}

// VirtualMachineStatus represents the status returned by the
// controller to describe how the VirtualMachine is doing
// ---
// +k8s:openapi-gen=true
type VirtualMachineStatus struct {
	// Created indicates if the virtual machine is created in the cluster
	Created bool `json:"created,omitempty"`
	// Ready indicates if the virtual machine is running and ready
	Ready bool `json:"ready,omitempty"`
	// Hold the state information of the VirtualMachine and its VirtualMachineInstance
	Conditions []VirtualMachineCondition `json:"conditions,omitempty" optional:"true"`
}

// GetObjectKind is required to satisfy Object interface
func (v *VirtualMachine) GetObjectKind() schema.ObjectKind {
	return &v.TypeMeta
}

// GetObjectMeta is required to satisfy ObjectMetaAccessor interface
func (v *VirtualMachine) GetObjectMeta() metav1.Object {
	return &v.ObjectMeta
}

// VirtualMachineCondition represents the state of VirtualMachine
// ---
// +k8s:openapi-gen=true
type VirtualMachineCondition struct {
	Type               VirtualMachineConditionType `json:"type"`
	Status             k8sv1.ConditionStatus       `json:"status"`
	LastProbeTime      metav1.Time                 `json:"lastProbeTime,omitempty"`
	LastTransitionTime metav1.Time                 `json:"lastTransitionTime,omitempty"`
	Reason             string                      `json:"reason,omitempty"`
	Message            string                      `json:"message,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type VirtualMachineConditionType string

const (
	// VirtualMachineFailure is added in a virtual machine when its vmi
	// fails to be created due to insufficient quota, limit ranges, pod security policy, node selectors,
	// etc. or deleted due to kubelet being down or finalizers are failing.
	VirtualMachineFailure VirtualMachineConditionType = "Failure"
)

// ---
// +k8s:openapi-gen=true
type HostDiskType string

const (
	// if disk does not exist at the given path,
	// a disk image will be created there
	HostDiskExistsOrCreate HostDiskType = "DiskOrCreate"
	// a disk image must exist at given disk path
	HostDiskExists HostDiskType = "Disk"
)

// ---
// +k8s:openapi-gen=true
type DriverCache string

const (
	// CacheNone - I/O from the guest is not cached on the host, but may be kept in a writeback disk cache.
	CacheNone DriverCache = "none"
	// CacheWriteThrough - I/O from the guest is cached on the host but written through to the physical medium.
	CacheWriteThrough DriverCache = "writethrough"
)
