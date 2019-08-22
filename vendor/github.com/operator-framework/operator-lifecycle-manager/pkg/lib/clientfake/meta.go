package clientfake

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage/names"
)

// BuildSelfLink returns a selflink for the given group version, plural, namespace, and name.
func BuildSelfLink(groupVersion, plural, namespace, name string) string {
	if namespace == metav1.NamespaceAll {
		return fmt.Sprintf("/apis/%s/%s/%s", groupVersion, plural, name)
	}
	return fmt.Sprintf("/apis/%s/namespaces/%s/%s/%s", groupVersion, namespace, plural, name)
}

// AddSimpleGeneratedName returns the given object with a simple generated name added to its metadata.
// If a name already exists, there is no GenerateName field set, or there is an issue accessing the object's metadata
// the object is returned unmodified.
func AddSimpleGeneratedName(obj runtime.Object) runtime.Object {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return obj
	}
	if accessor.GetName() == "" && accessor.GetGenerateName() != "" {
		// TODO: for tests, it would be nice to be able to retrieve this name later
		accessor.SetName(names.SimpleNameGenerator.GenerateName(accessor.GetGenerateName()))
	}

	return obj
}

// AddSimpleGeneratedNames returns the list objects with simple generated names added to their metadata.
// If a name already exists, there is no GenerateName field set, or there is an issue accessing the object's metadata
// the object is returned unmodified.
func AddSimpleGeneratedNames(objs ...runtime.Object) []runtime.Object {
	for i, obj := range objs {
		objs[i] = AddSimpleGeneratedName(obj)
	}

	return objs
}
