package apply

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	k6tv1 "kubevirt.io/api/core/v1"

	appsv1 "k8s.io/api/apps/v1"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	policyv1 "k8s.io/api/policy/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getGroupResource(required runtime.Object) (group string, resource string, err error) {

	switch required.(type) {
	case *extv1.CustomResourceDefinition:
		group = "apiextensions.k8s.io/v1"
		resource = "customresourcedefinitions"
	case *admissionregistrationv1.MutatingWebhookConfiguration:
		group = "admissionregistration.k8s.io"
		resource = "mutatingwebhookconfigurations"
	case *admissionregistrationv1.ValidatingWebhookConfiguration:
		group = "admissionregistration.k8s.io"
		resource = "validatingwebhookconfigurations"
	case *policyv1.PodDisruptionBudget:
		group = "apps"
		resource = "poddisruptionbudgets"
	case *appsv1.Deployment:
		group = "apps"
		resource = "deployments"
	case *appsv1.DaemonSet:
		group = "apps"
		resource = "daemonsets"
	default:
		err = fmt.Errorf("resource type is not known")
		return
	}

	return
}

func GetExpectedGeneration(required runtime.Object, previousGenerations []k6tv1.GenerationStatus) int64 {
	group, resource, err := getGroupResource(required)
	if err != nil {
		return -1
	}

	meta := required.(v1.Object)
	generation := generationFor(previousGenerations,
		group,
		resource,
		meta.GetNamespace(),
		meta.GetName(),
	)
	if generation == nil {
		return -1
	}

	return generation.LastGeneration
}

func SetGeneration(generations *[]k6tv1.GenerationStatus, actual runtime.Object) {
	if generations == nil || actual == nil {
		return
	}

	group, resource, err := getGroupResource(actual)
	if err != nil {
		return
	}

	meta := actual.(v1.Object)
	newGeneration := k6tv1.GenerationStatus{
		Group:          group,
		Resource:       resource,
		Namespace:      meta.GetNamespace(),
		Name:           meta.GetName(),
		LastGeneration: meta.GetGeneration(),
	}

	existingGeneration := generationFor(*generations,
		newGeneration.Group,
		newGeneration.Resource,
		newGeneration.Namespace,
		newGeneration.Name,
	)

	if existingGeneration == nil {
		*generations = append(*generations, newGeneration)
		return
	}

	existingGeneration.LastGeneration = newGeneration.LastGeneration
	existingGeneration.Hash = newGeneration.Hash
}

func generationFor(generations []k6tv1.GenerationStatus, group, resource, namespace, name string) *k6tv1.GenerationStatus {
	for i := range generations {
		curr := &generations[i]
		if curr.Namespace == namespace &&
			curr.Name == name &&
			curr.Group == group &&
			curr.Resource == resource {

			return curr
		}
	}

	return nil
}
