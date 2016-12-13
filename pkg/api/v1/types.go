package v1

//go:generate desc

import (
	"encoding/json"
	"github.com/rmohr/go-model"
	"github.com/satori/go.uuid"
	kubeapi "k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/meta"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/apimachinery/announced"
	"k8s.io/client-go/1.5/pkg/runtime"
	"kubevirt.io/kubevirt/pkg/api"
	"reflect"
)

// GroupName is the group name use in this package
const GroupName = "kubevirt.io"

// GroupVersion is group version used to register these objects
var GroupVersion = unversioned.GroupVersion{Group: GroupName, Version: "v1alpha1"}

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&VM{},
		&VMList{},
		&kubeapi.ListOptions{},
		&kubeapi.DeleteOptions{},
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
	model.AddConversion((*VMSpec)(nil), (*api.VMSpec)(nil), func(in reflect.Value) (reflect.Value, error) {
		out := api.VMSpec{}
		errs := model.Copy(&out, in.Interface())
		if len(errs) > 0 {
			return reflect.ValueOf(out), errs[0]
		}
		return reflect.ValueOf(out), nil
	})
	model.AddConversion((*api.VMSpec)(nil), (*VMSpec)(nil), func(in reflect.Value) (reflect.Value, error) {
		out := VMSpec{}
		errs := model.Copy(&out, in.Interface())
		if len(errs) > 0 {
			return reflect.ValueOf(out), errs[0]
		}
		return reflect.ValueOf(out), nil
	})
	model.AddConversion((*VMSpec)(nil), (*api.VMSpec)(nil), func(in reflect.Value) (reflect.Value, error) {
		out := api.VMSpec{}
		errs := model.Copy(&out, in.Interface())
		if len(errs) > 0 {
			return reflect.ValueOf(out), errs[0]
		}
		return reflect.ValueOf(out), nil
	})
	model.AddConversion((*VMCondition)(nil), (*api.VMCondition)(nil), func(in reflect.Value) (reflect.Value, error) {
		out := api.VMCondition{}
		errs := model.Copy(&out, in.Interface())
		if len(errs) > 0 {
			return reflect.ValueOf(out), errs[0]
		}
		return reflect.ValueOf(out), nil
	})
	model.AddConversion((*api.VMCondition)(nil), (*VMCondition)(nil), func(in reflect.Value) (reflect.Value, error) {
		out := VMCondition{}
		errs := model.Copy(&out, in.Interface())
		if len(errs) > 0 {
			return reflect.ValueOf(out), errs[0]
		}
		return reflect.ValueOf(out), nil
	})
	model.AddConversion((*api.VMStatus)(nil), (*VMStatus)(nil), func(in reflect.Value) (reflect.Value, error) {
		out := VMStatus{}
		errs := model.Copy(&out, in.Interface())
		if len(errs) > 0 {
			return reflect.ValueOf(out), errs[0]
		}
		return reflect.ValueOf(out), nil
	})
	model.AddConversion((*VMStatus)(nil), (*api.VMStatus)(nil), func(in reflect.Value) (reflect.Value, error) {
		out := api.VMStatus{}
		errs := model.Copy(&out, in.Interface())
		if len(errs) > 0 {
			return reflect.ValueOf(out), errs[0]
		}
		return reflect.ValueOf(out), nil
	})
}

type VM struct {
	unversioned.TypeMeta `json:",inline"`
	ObjectMeta           kubeapi.ObjectMeta `json:"metadata,omitempty"`
	Spec                 VMSpec             `json:"spec,omitempty" valid:"required"`
	Status               VMStatus           `json:"status"`
}

// VMList is a list of VMs
type VMList struct {
	unversioned.TypeMeta `json:",inline"`
	ListMeta             unversioned.ListMeta `json:"metadata,omitempty"`
	Items                []VM                 `json:"items"`
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
	NodeName   string        `json:"nodeName,omitempty"`
	Conditions []VMCondition `json:"conditions,omitempty"`
	Phase      VMPhase       `json:"phase"`
}

// Required to satisfy Object interface
func (v *VM) GetObjectKind() unversioned.ObjectKind {
	return &v.TypeMeta
}

// Required to satisfy ObjectMetaAccessor interface
func (v *VM) GetObjectMeta() meta.Object {
	return &v.ObjectMeta
}

// Required to satisfy Object interface
func (vl *VMList) GetObjectKind() unversioned.ObjectKind {
	return &vl.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (vl *VMList) GetListMeta() unversioned.List {
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
	LastProbeTime      unversioned.Time   `json:"lastProbeTime,omitempty"`
	LastTransitionTime unversioned.Time   `json:"lastTransitionTime,omitempty"`
	Reason             string             `json:"reason,omitempty"`
	Message            string             `json:"message,omitempty"`
}

// VMPhase is a label for the condition of a VM at the current time.
type VMPhase string

// These are the valid statuses of pods.
const (
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
	AppLabel      string = "kubevirt.io/app"
	DomainLabel   string = "kubevirt.io/domain"
	UIDLabel      string = "kubevirt.io/vmUID"
	NodeNameLabel string = "kubevirt.io/nodeName"
)
