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

/*
 ATTENTION: Rerun code generators when comments on structs or fields are modified.
*/

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/jeevatkm/go-model"
	"github.com/satori/go.uuid"

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

// GroupVersion is group version used to register these objects
var GroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha1"}

// GroupVersionKind
var VirtualMachineGroupVersionKind = schema.GroupVersionKind{Group: GroupName, Version: GroupVersion.Version, Kind: "VirtualMachine"}

var MigrationGroupVersionKind = schema.GroupVersionKind{Group: GroupName, Version: GroupVersion.Version, Kind: "Migration"}

var VMReplicaSetGroupVersionKind = schema.GroupVersionKind{Group: GroupName, Version: GroupVersion.Version, Kind: "VirtualMachineReplicaSet"}

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
		&Spice{},
		&Migration{},
		&MigrationList{},
		&VirtualMachineReplicaSet{},
		&VirtualMachineReplicaSetList{},
		&metav1.GetOptions{},
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

	// TODO the whole mapping registration can be done be an automatic process with reflection
	model.AddConversion((*uuid.UUID)(nil), (*string)(nil), func(in reflect.Value) (reflect.Value, error) {
		return reflect.ValueOf(in.Interface().(uuid.UUID).String()), nil
	})
	model.AddConversion((*string)(nil), (*uuid.UUID)(nil), func(in reflect.Value) (reflect.Value, error) {
		return reflect.ValueOf(uuid.FromStringOrNil(in.String())), nil
	})
}

// VirtualMachine is *the* VM Definition. It represents a virtual machine in the runtime environment of kubernetes.
type VirtualMachine struct {
	metav1.TypeMeta `json:",inline"`
	ObjectMeta      metav1.ObjectMeta `json:"metadata,omitempty"`
	// VM Spec contains the VM specification.
	Spec VMSpec `json:"spec,omitempty" valid:"required"`
	// Status is the high level overview of how the VM is doing. It contains information available to controllers and users.
	Status VMStatus `json:"status"`
}

func (in *VirtualMachine) DeepCopyInto(out *VirtualMachine) {
	err := model.Copy(out, in)
	if err != nil {
		panic(err)
	}
	return
}

func (in *VirtualMachine) DeepCopy() *VirtualMachine {
	if in == nil {
		return nil
	}
	out := new(VirtualMachine)
	in.DeepCopyInto(out)
	return out
}

func (in *VirtualMachine) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// VirtualMachineList is a list of VirtualMachines
type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	ListMeta        metav1.ListMeta  `json:"metadata,omitempty"`
	Items           []VirtualMachine `json:"items"`
}

func (in *VirtualMachineList) DeepCopyInto(out *VirtualMachineList) {
	err := model.Copy(out, in)
	if err != nil {
		panic(err)
	}
	return
}

func (in *VirtualMachineList) DeepCopy() *VirtualMachineList {
	if in == nil {
		return nil
	}
	out := new(VirtualMachineList)
	in.DeepCopyInto(out)
	return out
}

func (in *VirtualMachineList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// VMSpec is a description of a VM. Not to be confused with api.DomainSpec in virt-handler.
// It is expected that v1.DomainSpec will be merged into this structure.
type VMSpec struct {
	// Domain is the actual libvirt domain.
	Domain *DomainSpec `json:"domain,omitempty"`
	// If labels are specified, only nodes marked with all of these labels are considered when scheduling the VM.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// If affinity is specifies, obey all the affinity rules
	Affinity *Affinity `json:"affinity,omitempty"`
}

// Affinity groups all the affinity rules related to a VM
type Affinity struct {
	// Host affinity support
	NodeAffinity *k8sv1.NodeAffinity `json:"nodeAffinity,omitempty"`
}

// VMStatus represents information about the status of a VM. Status may trail the actual
// state of a system.
type VMStatus struct {
	// NodeName is the name where the VM is currently running.
	NodeName string `json:"nodeName,omitempty"`
	// MigrationNodeName is the node where the VM is live migrating to.
	MigrationNodeName string `json:"migrationNodeName,omitempty"`
	// Conditions are specific points in VM's pod runtime.
	Conditions []VMCondition `json:"conditions,omitempty"`
	// Phase is the status of the VM in kubernetes world. It is not the VM status, but partially correlates to it.
	Phase VMPhase `json:"phase"`
	// Graphics represent the details of available graphical consoles.
	Graphics []VMGraphics `json:"graphics"`
}

type VMGraphics struct {
	Type string `json:"type"`
	Host string `json:"host"`
	Port int32  `json:"port"`
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
	return v.Status.Phase == Running || v.Status.Phase == Migrating
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

type VMConditionType string

// These are valid conditions of VMs.
const (
	// PodCreated means that the VM request was translated into a Pod which can be scheduled and started by
	// Kubernetes.
	PodCreated VMConditionType = "PodCreated"
	// VMReady means the pod is able to service requests and should be added to the
	// load balancing pools of all matching services.
	VMReady VMConditionType = "Ready"
)

type VMCondition struct {
	Type               VMConditionType       `json:"type"`
	Status             k8sv1.ConditionStatus `json:"status"`
	LastProbeTime      metav1.Time           `json:"lastProbeTime,omitempty"`
	LastTransitionTime metav1.Time           `json:"lastTransitionTime,omitempty"`
	Reason             string                `json:"reason,omitempty"`
	Message            string                `json:"message,omitempty"`
}

// VMPhase is a label for the condition of a VM at the current time.
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
	// VMMigrating means the VM is currently migrated by a controller.
	Migrating VMPhase = "Migrating"
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
	AppLabel          string = "kubevirt.io/app"
	DomainLabel       string = "kubevirt.io/domain"
	VMUIDLabel        string = "kubevirt.io/vmUID"
	NodeNameLabel     string = "kubevirt.io/nodeName"
	MigrationUIDLabel string = "kubevirt.io/migrationUID"
	MigrationLabel    string = "kubevirt.io/migration"
)

func NewVM(name string, uid types.UID) *VirtualMachine {
	return &VirtualMachine{
		Spec: VMSpec{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			UID:       uid,
			Namespace: k8sv1.NamespaceDefault,
		},
		Status: VMStatus{},
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       VirtualMachineGroupVersionKind.Kind,
		},
	}
}

type MigrationEvent string

const (
	StartedVirtualMachineMigration   MigrationEvent = "MigrationStarted"
	SucceededVirtualMachineMigration MigrationEvent = "MigrationSucceeded"
	FailedVirtualMachineMigration    MigrationEvent = "MigrationFailed"
)

func (s MigrationEvent) String() string {
	return string(s)
}

type SyncEvent string

const (
	Created    SyncEvent = "Created"
	Deleted    SyncEvent = "Deleted"
	Started    SyncEvent = "Started"
	Stopped    SyncEvent = "Stopped"
	SyncFailed SyncEvent = "SyncFailed"
	Resumed    SyncEvent = "Resumed"
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
	vm.Spec = VMSpec{Domain: NewMinimalDomainSpec()}
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

type Spice struct {
	metav1.TypeMeta `json:",inline" ini:"-"`
	ObjectMeta      metav1.ObjectMeta `json:"metadata,omitempty" ini:"-"`
	Info            SpiceInfo         `json:"info,omitempty" valid:"required" ini:"virt-viewer"`
}

func (in *Spice) DeepCopyInto(out *Spice) {
	err := model.Copy(out, in)
	if err != nil {
		panic(err)
	}
	return
}

func (in *Spice) DeepCopy() *Spice {
	if in == nil {
		return nil
	}
	out := new(Spice)
	in.DeepCopyInto(out)
	return out
}

func (in *Spice) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

type SpiceInfo struct {
	Type  string `json:"type" ini:"type"`
	Host  string `json:"host" ini:"host"`
	Port  int32  `json:"port" ini:"port"`
	Proxy string `json:"proxy,omitempty" ini:"proxy,omitempty"`
}

func NewSpice(namespace string, vmName string) *Spice {
	return &Spice{
		Info: SpiceInfo{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmName,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       "Spice",
		},
	}
}

//TODO validate that this is correct
func NewMinimalMigration(name string, vmName string) *Migration {
	return NewMinimalMigrationWithNS(k8sv1.NamespaceDefault, name, vmName)
}

func NewMinimalMigrationWithNS(namespace string, name string, vmName string) *Migration {
	migration := NewMigrationReferenceFromName(namespace, name)
	migration.Spec = MigrationSpec{
		Selector: VMSelector{vmName},
	}
	return migration
}

func NewMigrationReferenceFromName(namespace string, name string) *Migration {
	migration := &Migration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			SelfLink:  fmt.Sprintf("/apis/%s/namespaces/%s/%s", GroupVersion.String(), namespace, name),
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       "Migration",
		},
		Status: MigrationStatus{
			Phase: MigrationUnknown,
		},
	}
	return migration
}

// A Migration is a job that moves a Virtual Machine from one node to another
type Migration struct {
	metav1.TypeMeta `json:",inline"`
	ObjectMeta      metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec            MigrationSpec     `json:"spec,omitempty" valid:"required"`
	Status          MigrationStatus   `json:"status,omitempty"`
}

func (in *Migration) DeepCopyInto(out *Migration) {
	err := model.Copy(out, in)
	if err != nil {
		panic(err)
	}
	return
}

func (in *Migration) DeepCopy() *Migration {
	if in == nil {
		return nil
	}
	out := new(Migration)
	in.DeepCopyInto(out)
	return out
}

func (in *Migration) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// MigrationSpec is a description of a VM Migration
// For example "destinationNodeName": "testvm" will migrate a VM called "testvm" in the namespace "default"
type MigrationSpec struct {
	// Criterias for selecting the VM to migrate.
	// For example
	// selector:
	//   name: testvm
	// will select the VM `testvm` for migration
	Selector VMSelector `json:"selector"`
	// Criteria to use when selecting the destination for the migration
	// for example, to select by the hostname, specify `kubernetes.io/hostname: master`
	// other possible choices include the hardware required to run the vm or
	// or lableing of the nodes to indicate their roles in larger applications.
	// examples:
	// disktype: ssd,
	// randomGenerator: /dev/random,
	// randomGenerator: superfastdevice,
	// app: mysql,
	// licensedForServiceX: true
	// Note that these selectors are additions to the node selectors on the VM itself and they must not exist on the VM.
	// If they are conflicting with the VM, no migration will be started.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

type VMSelector struct {
	// Name of the VM to migrate
	Name string `json:"name" valid:"required"`
}

type MigrationPhase string

const (
	// Create Migration has been called but nothing has been done with it
	MigrationUnknown MigrationPhase = ""

	// Migration has been scheduled but no update on the status has been recorded
	MigrationRunning MigrationPhase = "Running"

	// Migration has completed successfully
	MigrationSucceeded MigrationPhase = "Succeeded"

	// Migration has failed.  The Status structure of the associated Virtual Machine
	// Will indicate whether if the error was fatal.
	MigrationFailed MigrationPhase = "Failed"
)

// MigrationStatus is the last reported status of a VM Migratrion. Status may trail the actual
// state of a migration.
type MigrationStatus struct {
	Phase    MigrationPhase `json:"phase,omitempty"`
	Instance *types.UID     `json:"instance,omitempty"`
}

// Required to satisfy ObjectMetaAccessor interface
func (m *Migration) GetObjectMeta() metav1.Object {
	return &m.ObjectMeta
}

func (m *Migration) UnmarshalJSON(data []byte) error {
	type MigrationCopy Migration
	tmp := MigrationCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := Migration(tmp)
	*m = tmp2
	return nil
}

//A list of Migrations
type MigrationList struct {
	metav1.TypeMeta `json:",inline"`
	ListMeta        metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Migration     `json:"items"`
}

func (in *MigrationList) DeepCopyInto(out *MigrationList) {
	err := model.Copy(out, in)
	if err != nil {
		panic(err)
	}
	return
}

func (in *MigrationList) DeepCopy() *MigrationList {
	if in == nil {
		return nil
	}
	out := new(MigrationList)
	in.DeepCopyInto(out)
	return out
}

func (in *MigrationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// Required to satisfy Object interface
func (ml *MigrationList) GetObjectKind() schema.ObjectKind {
	return &ml.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (ml *MigrationList) GetListMeta() meta.List {
	return &ml.ListMeta
}

func (ml *MigrationList) UnmarshalJSON(data []byte) error {
	type MigrationListCopy MigrationList
	tmp := MigrationListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := MigrationList(tmp)
	*ml = tmp2
	return nil
}

// Host specific data, used by the migration controller to fetch host specific migration information from the target host
type MigrationHostInfo struct {
	Slice      string   `json:"slice"`
	Controller []string `json:"controller"`
	PidNS      string   `json:"pidns"`
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
type VirtualMachineReplicaSet struct {
	metav1.TypeMeta `json:",inline"`
	ObjectMeta      metav1.ObjectMeta `json:"metadata,omitempty"`
	// VM Spec contains the VM specification.
	Spec VMReplicaSetSpec `json:"spec,omitempty" valid:"required"`
	// Status is the high level overview of how the VM is doing. It contains information available to controllers and users.
	Status VMReplicaSetStatus `json:"status"`
}

// VMList is a list of VMs
type VirtualMachineReplicaSetList struct {
	metav1.TypeMeta `json:",inline"`
	ListMeta        metav1.ListMeta            `json:"metadata,omitempty"`
	Items           []VirtualMachineReplicaSet `json:"items"`
}

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

type VMReplicaSetStatus struct {
	// Total number of non-terminated pods targeted by this deployment (their labels match the selector).
	// +optional
	Replicas int32 `json:"replicas,omitempty" protobuf:"varint,2,opt,name=replicas"`

	// The number of ready replicas for this replica set.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty" protobuf:"varint,4,opt,name=readyReplicas"`

	Conditions []VMReplicaSetCondition `json:"conditions"`
}

type VMReplicaSetCondition struct {
	Type               VMReplicaSetConditionType `json:"type"`
	Status             k8sv1.ConditionStatus     `json:"status"`
	LastProbeTime      metav1.Time               `json:"lastProbeTime,omitempty"`
	LastTransitionTime metav1.Time               `json:"lastTransitionTime,omitempty"`
	Reason             string                    `json:"reason,omitempty"`
	Message            string                    `json:"message,omitempty"`
}

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

type VMTemplateSpec struct {
	ObjectMeta metav1.ObjectMeta `json:"metadata,omitempty"`
	// VM Spec contains the VM specification.
	Spec VMSpec `json:"spec,omitempty" valid:"required"`
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

func (in *VirtualMachineReplicaSet) DeepCopyInto(out *VirtualMachineReplicaSet) {
	err := model.Copy(out, in)
	if err != nil {
		panic(err)
	}
	return
}

func (in *VirtualMachineReplicaSet) DeepCopy() *VirtualMachineReplicaSet {
	if in == nil {
		return nil
	}
	out := new(VirtualMachineReplicaSet)
	in.DeepCopyInto(out)
	return out
}

func (in *VirtualMachineReplicaSet) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

func (in *VirtualMachineReplicaSetList) DeepCopyInto(out *VirtualMachineReplicaSetList) {
	err := model.Copy(out, in)
	if err != nil {
		panic(err)
	}
	return
}

func (in *VirtualMachineReplicaSetList) DeepCopy() *VirtualMachineReplicaSetList {
	if in == nil {
		return nil
	}
	out := new(VirtualMachineReplicaSetList)
	in.DeepCopyInto(out)
	return out
}

func (in *VirtualMachineReplicaSetList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}
