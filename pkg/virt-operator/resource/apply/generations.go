package apply

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	k6tv1 "kubevirt.io/api/core/v1"

	appsv1 "k8s.io/api/apps/v1"

	operatorsv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	policyv1 "k8s.io/api/policy/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	admissionRegistrationGroup = "admissionregistration.k8s.io"
	appsGroup                  = "apps"
	deploymentsResource        = "deployments"
	daemonsetsResource         = "daemonsets"
)

func getGroupResource(required runtime.Object) (group, resource string, err error) {
	switch required.(type) {
	case *extv1.CustomResourceDefinition:
		group = "apiextensions.k8s.io/v1"
		resource = "customresourcedefinitions"
	case *admissionregistrationv1.MutatingWebhookConfiguration:
		group = admissionRegistrationGroup
		resource = "mutatingwebhookconfigurations"
	case *admissionregistrationv1.ValidatingWebhookConfiguration:
		group = admissionRegistrationGroup
		resource = "validatingwebhookconfigurations"
	case *policyv1.PodDisruptionBudget:
		group = appsGroup
		resource = "poddisruptionbudgets"
	case *appsv1.Deployment:
		group = appsGroup
		resource = deploymentsResource
	case *appsv1.DaemonSet:
		group = appsGroup
		resource = daemonsetsResource
	default:
		err = fmt.Errorf("resource type is not known")
		return group, resource, err
	}

	return group, resource, err
}

func GetExpectedGeneration(required runtime.Object, previousGenerations []k6tv1.GenerationStatus) int64 {
	group, resource, err := getGroupResource(required)
	if err != nil {
		return -1
	}

	operatorGenerations := toOperatorGenerations(previousGenerations)

	meta := required.(v1.Object)
	gr := schema.GroupResource{Group: group, Resource: resource}
	generation := resourcemerge.GenerationFor(
		operatorGenerations, gr, meta.GetNamespace(), meta.GetName())
	if generation == nil {
		return -1
	}

	return generation.LastGeneration
}

func SetGeneration(generations *[]k6tv1.GenerationStatus, actual runtime.Object) {
	if actual == nil {
		return
	}

	group, resource, err := getGroupResource(actual)
	if err != nil {
		return
	}

	operatorGenerations := toOperatorGenerations(*generations)
	meta := actual.(v1.Object)

	resourcemerge.SetGeneration(&operatorGenerations, operatorsv1.GenerationStatus{
		Group:          group,
		Resource:       resource,
		Namespace:      meta.GetNamespace(),
		Name:           meta.GetName(),
		LastGeneration: meta.GetGeneration(),
	})

	newGenerations := toAPIGenerations(operatorGenerations)
	*generations = newGenerations
}

func toOperatorGeneration(generation k6tv1.GenerationStatus) operatorsv1.GenerationStatus {
	return operatorsv1.GenerationStatus{
		Group:          generation.Group,
		Resource:       generation.Resource,
		Namespace:      generation.Namespace,
		Name:           generation.Name,
		LastGeneration: generation.LastGeneration,
		Hash:           generation.Hash,
	}
}

func toAPIGeneration(generation operatorsv1.GenerationStatus) k6tv1.GenerationStatus {
	return k6tv1.GenerationStatus{
		Group:          generation.Group,
		Resource:       generation.Resource,
		Namespace:      generation.Namespace,
		Name:           generation.Name,
		LastGeneration: generation.LastGeneration,
		Hash:           generation.Hash,
	}
}

func toOperatorGenerations(generations []k6tv1.GenerationStatus) (operatorGenerations []operatorsv1.GenerationStatus) {
	for _, generation := range generations {
		operatorGenerations = append(operatorGenerations, toOperatorGeneration(generation))
	}
	return operatorGenerations
}

func toAPIGenerations(generations []operatorsv1.GenerationStatus) (apiGenerations []k6tv1.GenerationStatus) {
	for _, generation := range generations {
		apiGenerations = append(apiGenerations, toAPIGeneration(generation))
	}
	return apiGenerations
}
