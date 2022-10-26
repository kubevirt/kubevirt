// NOTE: Boilerplate only.  Ignore this file.

// package v1beta1 contains API Schema definitions for the hco vbeta1 API group
// +k8s:deepcopy-gen=package,register
// +groupName=hco.kubevirt.io
package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	hcoutils "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: hcoutils.APIVersionGroup, Version: hcoutils.APIVersionBeta}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	// AddToScheme tbd
	AddToScheme = SchemeBuilder.AddToScheme
)
