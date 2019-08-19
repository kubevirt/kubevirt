package apps

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators"
)

var (
	Group = "packages.apps.redhat.com"

	// SchemeGroupVersion is the GroupVersion used to register this object
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: runtime.APIVersionInternal}
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme        = SchemeBuilder.AddToScheme
)

// Kind takes an unqualified kind and returns the group-qualified kind.
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns the group-qualified resource.
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// addKnownTypes adds the set of types defined in this package to the supplied scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	// Add types for each GroupVersion
	scheme.AddKnownTypes(SchemeGroupVersion,
		&operators.PackageManifest{},
		&operators.PackageManifestList{},
	)

	return nil
}
