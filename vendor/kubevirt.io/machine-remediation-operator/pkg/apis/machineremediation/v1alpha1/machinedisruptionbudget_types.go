package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MachineDisruptionBudgetSpec is a description of a MachineDisruptionBudget.
type MachineDisruptionBudgetSpec struct {
	// An deletion of the machine is allowed if at least "minAvailable" machines selected by
	// "selector" will still be available after the deletion.
	// So for example you can prevent all voluntary deletions by specifying all available nodes.
	// +optional
	MinAvailable *int32 `json:"minAvailable,omitempty" protobuf:"bytes,1,opt,name=minAvailable"`

	// Label query over machines whose deletions are managed by the disruption
	// budget.
	// +optional
	// +kubebuilder:validation:Minimum=0
	Selector *metav1.LabelSelector `json:"selector,omitempty" protobuf:"bytes,2,opt,name=selector"`

	// An deletion is allowed if at most "maxUnavailable" machines selected by
	// "selector" are unavailable after the deletion.
	// For example, one can prevent all voluntary deletions by specifying 0.
	// This is a mutually exclusive setting with "minAvailable".
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxUnavailable *int32 `json:"maxUnavailable,omitempty" protobuf:"bytes,3,opt,name=maxUnavailable"`
}

// MachineDisruptionBudgetStatus represents information about the status of a
// MachineDisruptionBudget. Status may trail the actual state of a system.
type MachineDisruptionBudgetStatus struct {
	// Most recent generation observed when updating this MDB status. MachineDisruptionsAllowed and other
	// status information is valid only if observedGeneration equals to MDB's object generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,1,opt,name=observedGeneration"`

	// DisruptedMachines contains information about machines whose deletion was
	// processed by the API server but has not yet been observed by the MachineDisruptionBudget controller.
	// A machine will be in this map from the time when the API server processed the
	// deletion request to the time when the machine is seen by MDB controller
	// as having been marked for deletion (or after a timeout). The key in the map is the name of the machine
	// and the value is the time when the API server processed the deletion request. If
	// the deletion didn't occur and a machine is still there it will be removed from
	// the list automatically by MachineDisruptionBudget controller after some time.
	// If everything goes smooth this map should be empty for the most of the time.
	// Large number of entries in the map may indicate problems with machines deletions.
	// +optional
	DisruptedMachines map[string]metav1.Time `json:"disruptedMachines,omitempty" protobuf:"bytes,2,rep,name=DisruptedMachines"`

	// Number of machines disruptions that are currently allowed.
	MachineDisruptionsAllowed int32 `json:"disruptionsAllowed" protobuf:"varint,3,opt,name=disruptionsAllowed"`

	// current number of healthy machines
	CurrentHealthy int32 `json:"currentHealthy" protobuf:"varint,4,opt,name=currentHealthy"`

	// minimum desired number of healthy machines
	DesiredHealthy int32 `json:"desiredHealthy" protobuf:"varint,5,opt,name=desiredHealthy"`

	// total number of machines counted by this disruption budget
	Total int32 `json:"total" protobuf:"varint,6,opt,name=total"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineDisruptionBudget is an object to define the max disruption that a collection of machines can experience
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=mdb;mdbs
// +kubebuilder:printcolumn:name="Healthy",type="integer",JSONPath=".status.currentHealthy",description="The number of healthy machines"
// +kubebuilder:printcolumn:name="Total",type="integer",JSONPath=".status.total",description="The total number of machines"
// +kubebuilder:printcolumn:name="Desired",type="integer",JSONPath=".status.desiredHealthy",description="The desired number of healthy machines"
type MachineDisruptionBudget struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the desired behavior of the MachineDisruptionBudget.
	// +optional
	Spec MachineDisruptionBudgetSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	// Most recently observed status of the MachineDisruptionBudget.
	// +optional
	Status MachineDisruptionBudgetStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineDisruptionBudgetList is a collection of MachineDisruptionBudgets.
type MachineDisruptionBudgetList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []MachineDisruptionBudget `json:"items" protobuf:"bytes,2,rep,name=items"`
}
