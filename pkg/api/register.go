package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

// SchemeGroupVersion is used server side to register internal objects
var SchemeGroupVersion = schema.GroupVersion{Group: v1.GroupName, Version: runtime.APIVersionInternal}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns back a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&v1.VirtualMachine{},
		&v1.VirtualMachineList{},
		&metav1.ListOptions{},
		&metav1.DeleteOptions{},
		&v1.Spice{},
		&v1.Migration{},
		&v1.MigrationList{},
		&v1.VirtualMachineReplicaSet{},
		&v1.VirtualMachineReplicaSetList{},
		&metav1.GetOptions{},
	)
	return nil
}
