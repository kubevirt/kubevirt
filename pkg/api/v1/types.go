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
	"k8s.io/apimachinery/pkg/apimachinery/announced"
	"k8s.io/apimachinery/pkg/apimachinery/registered"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	"kubevirt.io/kubevirt/pkg/precond"
)

// GroupName is the group name use in this package
const GroupName = "kubevirt.io"
const SubresourceGroupName = "subresources.kubevirt.io"

const DefaultGracePeriodSeconds int64 = 30

// GroupVersion is group version used to register these objects
var GroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha1"}

// GroupVersion is group version used to register these objects
var SubresourceGroupVersion = schema.GroupVersion{Group: SubresourceGroupName, Version: "v1alpha1"}

// GroupVersionKind
var VirtualMachineGroupVersionKind = schema.GroupVersionKind{Group: GroupName, Version: GroupVersion.Version, Kind: "VirtualMachine"}

var VMReplicaSetGroupVersionKind = schema.GroupVersionKind{Group: GroupName, Version: GroupVersion.Version, Kind: "VirtualMachineReplicaSet"}

var VirtualMachinePresetGroupVersionKind = schema.GroupVersionKind{Group: GroupName, Version: GroupVersion.Version, Kind: "VirtualMachinePreset"}

var OfflineVirtualMachineGroupVersionKind = schema.GroupVersionKind{Group: GroupName, Version: GroupVersion.Version, Kind: "OfflineVirtualMachine"}

var (
	groupFactoryRegistry = make(announced.APIGroupFactoryRegistry)
	registry             = registered.NewOrDie(GroupVersion.String())
)

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&VirtualMachine{},
		&VirtualMachineList{},
		&metav1.ListOptions{},
		&metav1.DeleteOptions{},
		&VirtualMachineReplicaSet{},
		&VirtualMachineReplicaSetList{},
		&VirtualMachinePreset{},
		&VirtualMachinePresetList{},
		&metav1.GetOptions{},
		&OfflineVirtualMachine{},
		&OfflineVirtualMachineList{},
	)
	return nil
}

func init() {
	SchemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	if err := announced.NewGroupMetaFactory(
		&announced.GroupMetaFactoryArgs{
			GroupName:              GroupName,
			VersionPreferenceOrder: []string{GroupVersion.Version},
		},
		announced.VersionToSchemeFunc{
			GroupVersion.Version: SchemeBuilder.AddToScheme,
		},
	).Announce(groupFactoryRegistry).RegisterAndEnable(registry, scheme.Scheme); err != nil {
		panic(err)
	}
}

// VirtualMachine is *the* VM Definition. It represents a virtual machine in the runtime environment of kubernetes.
// ---
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// VM Spec contains the VM specification.
	Spec VirtualMachineSpec `json:"spec,omitempty" valid:"required"`
	// Status is the high level overview of how the VM is doing. It contains information available to controllers and users.
	Status VirtualMachineStatus `json:"status,omitempty"`
}

// VirtualMachineList is a list of VirtualMachines
// ---
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	ListMeta        metav1.ListMeta  `json:"metadata,omitempty"`
	Items           []VirtualMachine `json:"items"`
}

// VirtualMachineSpec is a description of a VirtualMachine.
// ---
// +k8s:openapi-gen=true
type VirtualMachineSpec struct {
	// Specification of the desired behavior of the VirtualMachine on the host.
	Domain DomainSpec `json:"domain"`
	// NodeSelector is a selector which must be true for the vm to fit on a node.
	// Selector which must match a node's labels for the vm to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// If affinity is specifies, obey all the affinity rules
	Affinity *Affinity `json:"affinity,omitempty"`
	// Grace period observed after signalling a VM to stop after which the VM is force terminated.
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`
	// List of volumes that can be mounted by disks belonging to the vm.
	Volumes []Volume `json:"volumes,omitempty"`
}

// Affinity groups all the affinity rules related to a VM
// ---
// +k8s:openapi-gen=true
type Affinity struct {
	// Node affinity support
	NodeAffinity *k8sv1.NodeAffinity `json:"nodeAffinity,omitempty"`
}

// VirtualMachineStatus represents information about the status of a VM. Status may trail the actual
// state of a system.
// ---
// +k8s:openapi-gen=true
type VirtualMachineStatus struct {
	// NodeName is the name where the VM is currently running.
	NodeName string `json:"nodeName,omitempty"`
	// Conditions are specific points in VM's pod runtime.
	Conditions []VirtualMachineCondition `json:"conditions,omitempty"`
	// Phase is the status of the VM in kubernetes world. It is not the VM status, but partially correlates to it.
	Phase VMPhase `json:"phase,omitempty"`
	// Interfaces represent the details of available network interfaces.
	Interfaces []VirtualMachineNetworkInterface `json:"interfaces,omitempty"`
}

// Required to satisfy Object interface
func (v *VirtualMachine) GetObjectKind() schema.ObjectKind {
	return &v.TypeMeta
}

// Required to satisfy ObjectMetaAccessor interface
func (v *VirtualMachine) GetObjectMeta() metav1.Object {
	return &v.ObjectMeta
}

func (v *VirtualMachine) IsReady() bool {
	// TODO once we support a ready condition, use it instead
	return v.IsRunning()
}

func (v *VirtualMachine) IsRunning() bool {
	return v.Status.Phase == Running
}

func (v *VirtualMachine) IsFinal() bool {
	return v.Status.Phase == Failed || v.Status.Phase == Succeeded
}

// Required to satisfy Object interface
func (vl *VirtualMachineList) GetObjectKind() schema.ObjectKind {
	return &vl.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (vl *VirtualMachineList) GetListMeta() meta.List {
	return &vl.ListMeta
}

func (v *VirtualMachine) UnmarshalJSON(data []byte) error {
	type VMCopy VirtualMachine
	tmp := VMCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := VirtualMachine(tmp)
	*v = tmp2
	return nil
}

func (vl *VirtualMachineList) UnmarshalJSON(data []byte) error {
	type VMListCopy VirtualMachineList
	tmp := VMListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := VirtualMachineList(tmp)
	*vl = tmp2
	return nil
}

func (v *VirtualMachine) MarshalBinary() (data []byte, err error) {
	return json.Marshal(*v)
}

func (v *VirtualMachine) UnmarshalBinary(data []byte) error {
	return v.UnmarshalJSON(data)
}

// ---
// +k8s:openapi-gen=true
type VirtualMachineConditionType string

// These are valid conditions of VMs.
const (
	// VMReady means the pod is able to service requests and should be added to the
	// load balancing pools of all matching services.
	VirtualMachineReady VirtualMachineConditionType = "Ready"

	// If there happens any error while trying to synchronize the VM with the Domain,
	// this is reported as false.
	VirtualMachineSynchronized VirtualMachineConditionType = "Synchronized"
)

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
type VirtualMachineNetworkInterface struct {
	// IP address of a Virtual Machine interface
	IP string `json:"ipAddress,omitempty"`
	// Hardware address of a Virtual Machine interface
	MAC string `json:"mac,omitempty"`
}

// VMPhase is a label for the condition of a VM at the current time.
// ---
// +k8s:openapi-gen=true
type VMPhase string

// These are the valid statuses of pods.
const (
	//When a VM Object is first initialized and no phase, or Pending is present.
	VmPhaseUnset VMPhase = ""
	// Pending means the VM has been accepted by the system.
	Pending VMPhase = "Pending"
	// Either a target pod does not yet exist or a target Pod exists but is not yet scheduled and in running state.
	Scheduling VMPhase = "Scheduling"
	// A target pod was scheduled and the system saw that Pod in runnig state.
	// Here is where the responsibility of virt-controller ends and virt-handler takes over.
	Scheduled VMPhase = "Scheduled"
	// VMRunning means the pod has been bound to a node and the VM is started.
	Running VMPhase = "Running"
	// VMSucceeded means that the VM stopped voluntarily, e.g. reacted to SIGTERM or shutdown was invoked from
	// inside the VM.
	Succeeded VMPhase = "Succeeded"
	// VMFailed means that associated Pod is in failure state (exited with a non-zero exit code or was stopped by
	// the system).
	Failed VMPhase = "Failed"
	// VMUnknown means that for some reason the state of the VM could not be obtained, typically due
	// to an error in communicating with the host of the VM.
	Unknown VMPhase = "Unknown"
)

const (
	AppLabel      string = "kubevirt.io"
	DomainLabel   string = "kubevirt.io/domain"
	VMUIDLabel    string = "kubevirt.io/vmUID"
	NodeNameLabel string = "kubevirt.io/nodeName"
)

func NewVM(name string, uid types.UID) *VirtualMachine {
	return &VirtualMachine{
		Spec: VirtualMachineSpec{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			UID:       uid,
			Namespace: k8sv1.NamespaceDefault,
		},
		Status: VirtualMachineStatus{},
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       VirtualMachineGroupVersionKind.Kind,
		},
	}
}

// ---
// +k8s:openapi-gen=true
type SyncEvent string

const (
	Created      SyncEvent = "Created"
	Deleted      SyncEvent = "Deleted"
	PresetFailed SyncEvent = "PresetFailed"
	Override     SyncEvent = "Override"
	Started      SyncEvent = "Started"
	ShuttingDown SyncEvent = "ShuttingDown"
	Stopped      SyncEvent = "Stopped"
	SyncFailed   SyncEvent = "SyncFailed"
	Resumed      SyncEvent = "Resumed"
)

func (s SyncEvent) String() string {
	return string(s)
}

func NewMinimalVM(vmName string) *VirtualMachine {
	return NewMinimalVMWithNS(k8sv1.NamespaceDefault, vmName)
}

func NewMinimalVMWithNS(namespace string, vmName string) *VirtualMachine {
	precond.CheckNotEmpty(vmName)
	vm := NewVMReferenceFromNameWithNS(namespace, vmName)
	vm.Spec = VirtualMachineSpec{Domain: NewMinimalDomainSpec()}
	vm.TypeMeta = metav1.TypeMeta{
		APIVersion: GroupVersion.String(),
		Kind:       "VirtualMachine",
	}
	return vm
}

// TODO Namespace could be different, also store it somewhere in the domain, so that we can report deletes on handler startup properly
func NewVMReferenceFromName(name string) *VirtualMachine {
	return NewVMReferenceFromNameWithNS(k8sv1.NamespaceDefault, name)
}

func NewVMReferenceFromNameWithNS(namespace string, name string) *VirtualMachine {
	vm := &VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			SelfLink:  fmt.Sprintf("/apis/%s/namespaces/%s/virtualmachines/%s", GroupVersion.String(), namespace, name),
		},
	}
	vm.SetGroupVersionKind(schema.GroupVersionKind{Group: GroupVersion.Group, Kind: "VM", Version: GroupVersion.Version})
	return vm
}

type VMSelector struct {
	// Name of the VM to migrate
	Name string `json:"name" valid:"required"`
}

// Given a VM, update all NodeSelectorTerms with anti-affinity for that VM's node.
// This is useful for the case when a migration away from a node must occur.
// This method returns the full Affinity structure updated the anti affinity terms
func UpdateAntiAffinityFromVMNode(pod *k8sv1.Pod, vm *VirtualMachine) *k8sv1.Affinity {
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

		term.MatchExpressions = append(term.MatchExpressions, PrepareVMNodeAntiAffinitySelectorRequirement(vm))
		selector.NodeSelectorTerms[idx] = term
	}

	return pod.Spec.Affinity
}

// Given a VM, create a NodeSelectorTerm with anti-affinity for that VM's node.
// This is useful for the case when a migration away from a node must occur.
func PrepareVMNodeAntiAffinitySelectorRequirement(vm *VirtualMachine) k8sv1.NodeSelectorRequirement {
	return k8sv1.NodeSelectorRequirement{
		Key:      "kubernetes.io/hostname",
		Operator: k8sv1.NodeSelectorOpNotIn,
		Values:   []string{vm.Status.NodeName},
	}
}

// VM is *the* VM Definition. It represents a virtual machine in the runtime environment of kubernetes.
// ---
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachineReplicaSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// VM Spec contains the VM specification.
	Spec VMReplicaSetSpec `json:"spec,omitempty" valid:"required"`
	// Status is the high level overview of how the VM is doing. It contains information available to controllers and users.
	Status VMReplicaSetStatus `json:"status,omitempty"`
}

// VMList is a list of VMs
// ---
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachineReplicaSetList struct {
	metav1.TypeMeta `json:",inline"`
	ListMeta        metav1.ListMeta            `json:"metadata,omitempty"`
	Items           []VirtualMachineReplicaSet `json:"items"`
}

// ---
// +k8s:openapi-gen=true
type VMReplicaSetSpec struct {
	// Number of desired pods. This is a pointer to distinguish between explicit
	// zero and not specified. Defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Label selector for pods. Existing ReplicaSets whose pods are
	// selected by this will be the ones affected by this deployment.
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty" valid:"required"`

	// Template describes the pods that will be created.
	Template *VMTemplateSpec `json:"template" valid:"required"`

	// Indicates that the replica set is paused.
	// +optional
	Paused bool `json:"paused,omitempty" protobuf:"varint,7,opt,name=paused"`
}

// ---
// +k8s:openapi-gen=true
type VMReplicaSetStatus struct {
	// Total number of non-terminated pods targeted by this deployment (their labels match the selector).
	// +optional
	Replicas int32 `json:"replicas,omitempty" protobuf:"varint,2,opt,name=replicas"`

	// The number of ready replicas for this replica set.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty" protobuf:"varint,4,opt,name=readyReplicas"`

	Conditions []VMReplicaSetCondition `json:"conditions,omitempty" optional:"true"`
}

// ---
// +k8s:openapi-gen=true
type VMReplicaSetCondition struct {
	Type               VMReplicaSetConditionType `json:"type"`
	Status             k8sv1.ConditionStatus     `json:"status"`
	LastProbeTime      metav1.Time               `json:"lastProbeTime,omitempty"`
	LastTransitionTime metav1.Time               `json:"lastTransitionTime,omitempty"`
	Reason             string                    `json:"reason,omitempty"`
	Message            string                    `json:"message,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type VMReplicaSetConditionType string

const (
	// VMReplicaSetReplicaFailure is added in a replica set when one of its vms
	// fails to be created due to insufficient quota, limit ranges, pod security policy, node selectors,
	// etc. or deleted due to kubelet being down or finalizers are failing.
	VMReplicaSetReplicaFailure VMReplicaSetConditionType = "ReplicaFailure"

	// VMReplicaSetReplicaPaused is added in a replica set when the replica set got paused by the controller.
	// After this condition was added, it is safe to remove or add vms by hand and adjust the replica count by hand.
	VMReplicaSetReplicaPaused VMReplicaSetConditionType = "ReplicaPaused"
)

// ---
// +k8s:openapi-gen=true
type VMTemplateSpec struct {
	ObjectMeta metav1.ObjectMeta `json:"metadata,omitempty"`
	// VM Spec contains the VM specification.
	Spec VirtualMachineSpec `json:"spec,omitempty" valid:"required"`
}

// Required to satisfy Object interface
func (v *VirtualMachineReplicaSet) GetObjectKind() schema.ObjectKind {
	return &v.TypeMeta
}

// Required to satisfy ObjectMetaAccessor interface
func (v *VirtualMachineReplicaSet) GetObjectMeta() metav1.Object {
	return &v.ObjectMeta
}

func (v *VirtualMachineReplicaSet) UnmarshalJSON(data []byte) error {
	type VMReplicaSetCopy VirtualMachineReplicaSet
	tmp := VMReplicaSetCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := VirtualMachineReplicaSet(tmp)
	*v = tmp2
	return nil
}

func (vl *VirtualMachineReplicaSetList) UnmarshalJSON(data []byte) error {
	type VMReplicaSetListCopy VirtualMachineReplicaSetList
	tmp := VMReplicaSetListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := VirtualMachineReplicaSetList(tmp)
	*vl = tmp2
	return nil
}

// Required to satisfy Object interface
func (vl *VirtualMachineReplicaSetList) GetObjectKind() schema.ObjectKind {
	return &vl.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (vl *VirtualMachineReplicaSetList) GetListMeta() meta.List {
	return &vl.ListMeta
}

// ---
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachinePreset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// VM Spec contains the VM specification.
	Spec VirtualMachinePresetSpec `json:"spec,omitempty" valid:"required"`
}

// Required to satisfy Object interface
func (v *VirtualMachinePreset) GetObjectKind() schema.ObjectKind {
	return &v.TypeMeta
}

// Required to satisfy ObjectMetaAccessor interface
func (v *VirtualMachinePreset) GetObjectMeta() metav1.Object {
	return &v.ObjectMeta
}

// VirtualMachinePresetList is a list of VirtualMachinePresets
// ---
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachinePresetList struct {
	metav1.TypeMeta `json:",inline"`
	ListMeta        metav1.ListMeta        `json:"metadata,omitempty"`
	Items           []VirtualMachinePreset `json:"items"`
}

// ---
// +k8s:openapi-gen=true
type VirtualMachinePresetSpec struct {
	// Selector is a label query over a set of VMs.
	// Required.
	Selector metav1.LabelSelector `json:"selector"`
	// Domain is the same object type as contained in VirtualMachineSpec
	Domain *DomainSpec `json:"domain,omitempty"`
}

func NewVirtualMachinePreset(name string, selector metav1.LabelSelector) *VirtualMachinePreset {
	return &VirtualMachinePreset{
		Spec: VirtualMachinePresetSpec{
			Selector: selector,
			Domain:   &DomainSpec{},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: k8sv1.NamespaceDefault,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       VirtualMachinePresetGroupVersionKind.Kind,
		},
	}
}

func (vl *VirtualMachinePresetList) UnmarshalJSON(data []byte) error {
	type VirtualMachinePresetListCopy VirtualMachinePresetList
	tmp := VirtualMachinePresetListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := VirtualMachinePresetList(tmp)
	*vl = tmp2
	return nil
}

// Required to satisfy Object interface
func (vl *VirtualMachinePresetList) GetObjectKind() schema.ObjectKind {
	return &vl.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (vl *VirtualMachinePresetList) GetListMeta() meta.List {
	return &vl.ListMeta
}

// OfflineVirtualMachine handles the VirtualMachines that are not running
// or are in a stopped state
// The OfflineVirtualMachine contains the template to create the
// VirtualMachine. It also mirrors the running state of the created
// VirtualMachine in its status.
// ---
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type OfflineVirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec contains the specification of VirtualMachine created
	Spec OfflineVirtualMachineSpec `json:"spec,omitempty"`
	// Status holds the current state of the controller and brief information
	// about its associated VirtualMachine
	Status OfflineVirtualMachineStatus `json:"status,omitempty"`
}

// OfflineVirtualMachineList is a list of offlinevirtualmachines
// ---
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type OfflineVirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	// Items is a list of OfflineVirtualMachines
	Items []OfflineVirtualMachine `json:"items"`
}

// OfflineVirtualMachineSpec describes how the proper OfflineVirtualMachine
// should look like
// ---
// +k8s:openapi-gen=true
type OfflineVirtualMachineSpec struct {
	// Running controlls whether the associatied VirtualMachine is created or not
	Running bool `json:"running"`

	// Template is the direct specification of VirtualMachine
	Template *VMTemplateSpec `json:"template"`
}

// OfflineVirtualMachineStatus represents the status returned by the
// controller to describe how the OfflineVirtualMachine is doing
// ---
// +k8s:openapi-gen=true
type OfflineVirtualMachineStatus struct {
	// Hold the state information of the OfflineVirtualMachine and its VirtualMachine
	Conditions []OfflineVirtualMachineCondition `json:"conditions,omitempty" optional:"true"`
}

// GetObjectKind is required to satisfy Object interface
func (v *OfflineVirtualMachine) GetObjectKind() schema.ObjectKind {
	return &v.TypeMeta
}

// GetObjectMeta is required to satisfy ObjectMetaAccessor interface
func (v *OfflineVirtualMachine) GetObjectMeta() metav1.Object {
	return &v.ObjectMeta
}

// OfflineVirtualMachineCondition represents the state of OfflineVirtualMachine
// ---
// +k8s:openapi-gen=true
type OfflineVirtualMachineCondition struct {
	Type               OfflineVirtualMachineConditionType `json:"type"`
	Status             k8sv1.ConditionStatus              `json:"status"`
	LastProbeTime      metav1.Time                        `json:"lastProbeTime,omitempty"`
	LastTransitionTime metav1.Time                        `json:"lastTransitionTime,omitempty"`
	Reason             string                             `json:"reason,omitempty"`
	Message            string                             `json:"message,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type OfflineVirtualMachineConditionType string

const (
	// OfflineVirtualMachineFailure is added in a offline virtual machine when its vm
	// fails to be created due to insufficient quota, limit ranges, pod security policy, node selectors,
	// etc. or deleted due to kubelet being down or finalizers are failing.
	OfflineVirtualMachineFailure OfflineVirtualMachineConditionType = "Failure"

	// OfflineVirtualMachineRunning is added in a offline virtual machine when the VM succesfully runs.
	// After this condition was added, the VM is up and running.
	OfflineVirtualMachineRunning OfflineVirtualMachineConditionType = "Running"
)
