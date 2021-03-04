package apply

import (
	"fmt"

	operatorsv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)


type NamespaceNameGenerationGetter interface {
	GetNamespace() string
	GetName() string
	GetGeneration() int64
}

func getResourceInfo(required NamespaceNameGenerationGetter) (namespace string, name string, group string, resource string, generation int64, err error) {

	switch required.(type) {
	case *admissionregistrationv1.MutatingWebhookConfiguration:
		group = "admissionregistration.k8s.io"
		resource = "mutatingwebhookconfigurations"
	case *admissionregistrationv1.ValidatingWebhookConfiguration:
		group = "admissionregistration.k8s.io"
		resource = "validatingwebhookconfigurations"
	case *v1beta1.PodDisruptionBudget:
		group = "apps"
		resource = "poddisruptionbudget"
	default:
		err = fmt.Errorf("resource type is not known")
		return
	}

	namespace = required.GetNamespace()
	name = required.GetName()
	generation = required.GetGeneration()

	return
}

func GetExpectedGeneration(required NamespaceNameGenerationGetter, previousGenerations []operatorsv1.GenerationStatus) int64 {
	namespace, name, group, resource, _, err := getResourceInfo(required)
	if err != nil {
		return -1
	}

	generation := resourcemerge.GenerationFor(previousGenerations, schema.GroupResource{Group: group, Resource: resource}, namespace, name)
	if generation == nil {
		return -1
	}

	return generation.LastGeneration
}

func SetGeneration(generations *[]operatorsv1.GenerationStatus, actual NamespaceNameGenerationGetter) {
	if actual == nil {
		return
	}

	namespace, name, group, resource, generation, err := getResourceInfo(actual)
	if err != nil {
		return
	}

	resourcemerge.SetGeneration(generations, operatorsv1.GenerationStatus{
		Group:          group,
		Resource:       resource,
		Namespace:      namespace,
		Name:           name,
		LastGeneration: generation,
	})
}
