package api

import (
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/meta"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/runtime/schema"
)

type VM struct {
	metav1.TypeMeta
	ObjectMeta api.ObjectMeta
	Spec       VMSpec
	Status     VMStatus
}

type VMList struct {
	metav1.TypeMeta
	metav1.ListMeta
	VMs []VM
}

// VMSpec is a description of a pod
type VMSpec struct {
	NodeSelector map[string]string
}

// VMStatus represents information about the status of a VM. Status may trail the actual
// state of a system.
type VMStatus struct {
	NodeName   string
	Conditions []VMCondition
	Phase      VMPhase
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

type VMPhase string
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
	Type               VMConditionType     `json:"type"`
	Status             api.ConditionStatus `json:"status"`
	LastProbeTime      metav1.Time         `json:"lastProbeTime,omitempty"`
	LastTransitionTime metav1.Time         `json:"lastTransitionTime,omitempty"`
	Reason             string              `json:"reason,omitempty"`
	Message            string              `json:"message,omitempty"`
}
