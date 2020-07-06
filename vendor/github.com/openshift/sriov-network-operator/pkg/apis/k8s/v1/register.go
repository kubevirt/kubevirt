// NOTE: Boilerplate only.  Ignore this file.

// Package v1 contains API Schema definitions for the k8s v1 API group
// +k8s:deepcopy-gen=package,register
// +groupName=k8s.cni.cncf.io
package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "k8s.cni.cncf.io", Version: "v1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)
