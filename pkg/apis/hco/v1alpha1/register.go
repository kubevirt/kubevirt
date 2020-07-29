// NOTE: Boilerplate only.  Ignore this file.

// Package v1alpha1 contains API Schema definitions for the hco v1alpha1 API group
// +k8s:deepcopy-gen=package,register
// +groupName=hco.kubevirt.io
package v1alpha1

import (
	hcoutils "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: hcoutils.APIVersionGroup, Version: hcoutils.APIVersionAlpha}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	// AddToScheme tbd
	AddToScheme = SchemeBuilder.AddToScheme
)
