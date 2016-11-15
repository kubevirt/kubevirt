package api

import (
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/meta"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
)

type VM struct {
	unversioned.TypeMeta
	ObjectMeta api.ObjectMeta
	Spec       VMSpec
	Status     VMStatus
}

type VMList struct {
	unversioned.TypeMeta
	unversioned.ListMeta
	VMs []VM
}

type VMSpec struct {
	NodeSelector map[string]string
}

type VMStatus struct {
	NodeName   string
	Conditions []VMCondition
	Phase      VMPhase
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
	LastProbeTime      unversioned.Time    `json:"lastProbeTime,omitempty"`
	LastTransitionTime unversioned.Time    `json:"lastTransitionTime,omitempty"`
	Reason             string              `json:"reason,omitempty"`
	Message            string              `json:"message,omitempty"`
}
