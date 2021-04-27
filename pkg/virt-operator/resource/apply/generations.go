package apply

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	appsv1 "k8s.io/api/apps/v1"

	operatorsv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/api/policy/v1beta1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	case *v1beta1.PodDisruptionBudget:
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

func GetExpectedGeneration(required runtime.Object, previousGenerations []operatorsv1.GenerationStatus) int64 {
	group, resource, err := getGroupResource(required)
	if err != nil {
		return -1
	}

	meta := required.(v1.Object)
	generation := resourcemerge.GenerationFor(previousGenerations, schema.GroupResource{Group: group, Resource: resource}, meta.GetNamespace(), meta.GetName())
	if generation == nil {
		return -1
	}

	return generation.LastGeneration
}

func SetGeneration(generations *[]operatorsv1.GenerationStatus, actual runtime.Object) {
	if actual == nil {
		return
	}

	group, resource, err := getGroupResource(actual)
	if err != nil {
		return
	}

	meta := actual.(v1.Object)

	resourcemerge.SetGeneration(generations, operatorsv1.GenerationStatus{
		Group:          group,
		Resource:       resource,
		Namespace:      meta.GetNamespace(),
		Name:           meta.GetName(),
		LastGeneration: meta.GetGeneration(),
	})
}
