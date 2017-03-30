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
	kubeapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/meta"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apimachinery/announced"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/runtime/schema"
	"k8s.io/client-go/pkg/types"
	"kubevirt.io/kubevirt/pkg/api"
	"kubevirt.io/kubevirt/pkg/mapper"
	"kubevirt.io/kubevirt/pkg/precond"
)

// GroupName is the group name use in this package
const GroupName = "kubevirt.io"

// GroupVersion is group version used to register these objects
var GroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha1"}

// GroupVersionKind
var GroupVersionKind = schema.GroupVersionKind{Group: GroupName, Version: GroupVersion.Version, Kind: "VM"}

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&VM{},
		&VMList{},
		&kubeapi.ListOptions{},
		&v1.ListOptions{},
		&kubeapi.DeleteOptions{},
		&v1.DeleteOptions{},
		&Spice{},
		&Migration{},
		&MigrationList{},
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
			ImportPrefix:           "kubevirt.io/kubevirt/pgk/api/v1",
		},
		announced.VersionToSchemeFunc{
			GroupVersion.Version: SchemeBuilder.AddToScheme,
		},
	).Announce().RegisterAndEnable(); err != nil {
		panic(err)
	}

	// TODO the whole mapping registration can be done be an automatic process with reflection
	model.AddConversion((*uuid.UUID)(nil), (*string)(nil), func(in reflect.Value) (reflect.Value, error) {
		return reflect.ValueOf(in.Interface().(uuid.UUID).String()), nil
	})
	model.AddConversion((*string)(nil), (*uuid.UUID)(nil), func(in reflect.Value) (reflect.Value, error) {
		return reflect.ValueOf(uuid.FromStringOrNil(in.String())), nil
	})

	mapper.AddConversion(&VMSpec{}, &api.VMSpec{})
	mapper.AddConversion(&VMCondition{}, &api.VMCondition{})
	mapper.AddConversion(&VMStatus{}, &api.VMStatus{})
	mapper.AddConversion(&v1.ObjectMeta{}, &kubeapi.ObjectMeta{})
}

type VM struct {
	metav1.TypeMeta `json:",inline"`
	ObjectMeta      v1.ObjectMeta `json:"metadata,omitempty"`
	Spec            VMSpec        `json:"spec,omitempty" valid:"required"`
	Status          VMStatus      `json:"status"`
}

// VMList is a list of VMs
type VMList struct {
	metav1.TypeMeta `json:",inline"`
	ListMeta        metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VM            `json:"items"`
}

// VMSpec is a description of a VM
type VMSpec struct {
	Domain *DomainSpec `json:"domain,omitempty"`
	// If labels are specified, only nodes marked with all of these labels are considered when scheduling the VM.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

// VMStatus represents information about the status of a VM. Status may trail the actual
// state of a system.
type VMStatus struct {
	NodeName          string        `json:"nodeName,omitempty"`
	MigrationNodeName string        `json:"migrationNodeName,omitempty"`
	Conditions        []VMCondition `json:"conditions,omitempty"`
	Phase             VMPhase       `json:"phase"`
}

// Required to satisfy Object interface
func (v *VM) GetObjectKind() schema.ObjectKind {
	return &v.TypeMeta
}

// Required to satisfy ObjectMetaAccessor interface
func (v *VM) GetObjectMeta() meta.Object {
	return &v.ObjectMeta
}

// Required to satisfy Object interface
func (vl *VMList) GetObjectKind() schema.ObjectKind {
	return &vl.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (vl *VMList) GetListMeta() metav1.List {
	return &vl.ListMeta
}

func (v *VM) UnmarshalJSON(data []byte) error {
	type VMCopy VM
	tmp := VMCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := VM(tmp)
	*v = tmp2
	return nil
}

func (vl *VMList) UnmarshalJSON(data []byte) error {
	type VMListCopy VMList
	tmp := VMListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := VMList(tmp)
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
	Type               VMConditionType    `json:"type"`
	Status             v1.ConditionStatus `json:"status"`
	LastProbeTime      metav1.Time        `json:"lastProbeTime,omitempty"`
	LastTransitionTime metav1.Time        `json:"lastTransitionTime,omitempty"`
	Reason             string             `json:"reason,omitempty"`
	Message            string             `json:"message,omitempty"`
}

// VMPhase is a label for the condition of a VM at the current time.
type VMPhase string

// These are the valid statuses of pods.
const (
	//When a VM Object is first initialized and no phase has been set.
	VmPhaseUnset VMPhase = ""
	// VMPending means the VM has been accepted by the system.
	// Either a target pod does not yet exist or a target Pod exists but is not yet scheduled and in running state.
	Scheduling VMPhase = "Scheduling"
	// A target pod was scheduled and the system saw that Pod in runnig state.
	// Here is where the responsibility of virt-controller ends and virt-handler takes over.
	Pending VMPhase = "Pending"
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

func NewVM(name string, uid types.UID) *VM {
	return &VM{
		Spec: VMSpec{},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			UID:       uid,
			Namespace: kubeapi.NamespaceDefault,
		},
		Status: VMStatus{},
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       GroupVersionKind.Kind,
		},
	}
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

func NewMinimalVM(vmName string) *VM {
	precond.CheckNotEmpty(vmName)
	vm := NewVMReferenceFromName(vmName)
	vm.Spec = VMSpec{Domain: NewMinimalDomainSpec(vmName)}
	vm.TypeMeta = metav1.TypeMeta{
		APIVersion: GroupVersion.String(),
		Kind:       "VM",
	}
	return vm
}

// TODO Namespace could be different, also store it somewhere in the domain, so that we can report deletes on handler startup properly
func NewVMReferenceFromName(name string) *VM {
	vm := &VM{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: kubeapi.NamespaceDefault,
			SelfLink:  fmt.Sprintf("/apis/%s/namespaces/%s/vms/%s", GroupVersion.String(), kubeapi.NamespaceDefault, name),
		},
	}
	vm.SetGroupVersionKind(schema.GroupVersionKind{Group: GroupVersion.Group, Kind: "VM", Version: GroupVersion.Version})
	return vm
}

type Spice struct {
	metav1.TypeMeta `json:",inline" ini:"-"`
	ObjectMeta      v1.ObjectMeta `json:"metadata,omitempty" ini:"-"`
	Info            SpiceInfo     `json:"info,omitempty" valid:"required" ini:"virt-viewer"`
}

type SpiceInfo struct {
	Type  string `json:"type" ini:"type"`
	Host  string `json:"host" ini:"host"`
	Port  int32  `json:"port" ini:"port"`
	Proxy string `json:"proxy,omitempty" ini:"proxy,omitempty"`
}

func NewSpice(vmName string) *Spice {
	return &Spice{
		Info: SpiceInfo{},
		ObjectMeta: v1.ObjectMeta{
			Name:      vmName,
			Namespace: kubeapi.NamespaceDefault,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       "Spice",
		},
	}
}

//TODO validate that this is correct
func NewMinimalMigration(name string, vmName string) *Migration {
	migration := NewMigrationReferenceFromName(name)
	migration.Spec = MigrationSpec{
		Selector: VMSelector{vmName},
	}
	return migration
}

func NewMigrationReferenceFromName(name string) *Migration {
	migration := &Migration{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: kubeapi.NamespaceDefault,
			SelfLink:  fmt.Sprintf("/apis/%s/namespaces/%s/%s", GroupVersion.String(), kubeapi.NamespaceDefault, name),
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
	ObjectMeta      v1.ObjectMeta   `json:"metadata,omitempty"`
	Spec            MigrationSpec   `json:"spec,omitempty" valid:"required"`
	Status          MigrationStatus `json:"status,omitempty"`
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
	MigrationPending MigrationPhase = "Pending"

	// Migration is actively progressing
	MigrationInProgress MigrationPhase = "InProgress"

	// Migration has completed successfully
	MigrationSucceeded MigrationPhase = "Succeeded"

	// Migration has failed.  The Status structure of the associated Virtual Machine
	// Will indicate whether if the error was fatal.
	MigrationFailed MigrationPhase = "Failed"
)

// MigrationStatus is the last reported status of a VM Migratrion. Status may trail the actual
// state of a migration.
type MigrationStatus struct {
	Phase MigrationPhase `json:"phase,omitempty"`
}

// Required to satisfy ObjectMetaAccessor interface
func (m *Migration) GetObjectMeta() meta.Object {
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

// Required to satisfy Object interface
func (ml *MigrationList) GetObjectKind() schema.ObjectKind {
	return &ml.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (ml *MigrationList) GetListMeta() metav1.List {
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

// Given a VM, create a NodeSelectorTerm with anti-affinity for that VM's node.
// This is useful for the case when a migration away from a node must occur.
func AntiAffinityFromVMNode(vm *VM) *v1.Affinity {
	return antiAffinityFromNode(vm.Status.NodeName)
}

func antiAffinityFromNode(nodeName string) *v1.Affinity {
	selector := v1.NodeSelectorTerm{
		MatchExpressions: []v1.NodeSelectorRequirement{
			{
				Key:      "kubernetes.io/hostname",
				Operator: v1.NodeSelectorOpNotIn,
				Values:   []string{nodeName},
			},
		},
	}
	return &v1.Affinity{
		NodeAffinity: &v1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{NodeSelectorTerms: []v1.NodeSelectorTerm{selector}},
		},
	}
}

// Given a pod and an affinity rule, add the affinity to any others
// associated with the pod (if any). In this context, pod is the destination
// of a migration.
func SetAntiAffinityToPod(pod *v1.Pod, affinity *v1.Affinity) (*v1.Pod, error) {

	newAffinity, err := json.Marshal(affinity)
	if err != nil {
		return nil, err
	}
	pod.Annotations = map[string]string{}
	pod.Annotations["scheduler.alpha.kubernetes.io/affinity"] = string(newAffinity)

	return pod, nil
}
