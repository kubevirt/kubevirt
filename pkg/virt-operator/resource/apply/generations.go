package apply

import (
	operatorsv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func ExpectedMutatingWebhookConfigurationGeneration(required *admissionregistrationv1.MutatingWebhookConfiguration, previousGenerations []operatorsv1.GenerationStatus) int64 {
	generation := resourcemerge.GenerationFor(previousGenerations, schema.GroupResource{Group: admissionregistrationv1.GroupName, Resource: "mutatingwebhookconfigurations"}, required.Namespace, required.Name)
	if generation != nil {
		return generation.LastGeneration
	}

	return -1
}

func SetMutatingWebhookConfigurationGeneration(generations *[]operatorsv1.GenerationStatus, actual *admissionregistrationv1.MutatingWebhookConfiguration) {
	if actual == nil {
		return
	}
	resourcemerge.SetGeneration(generations, operatorsv1.GenerationStatus{
		Group:          admissionregistrationv1.GroupName,
		Resource:       "mutatingwebhookconfigurations",
		Namespace:      actual.Namespace,
		Name:           actual.Name,
		LastGeneration: actual.ObjectMeta.Generation,
	})
}

func ExpectedValidatingWebhookConfigurationGeneration(required *admissionregistrationv1.ValidatingWebhookConfiguration, previousGenerations []operatorsv1.GenerationStatus) int64 {
	generation := resourcemerge.GenerationFor(previousGenerations, schema.GroupResource{Group: admissionregistrationv1.GroupName, Resource: "validatingwebhookconfigurations"}, required.Namespace, required.Name)
	if generation != nil {
		return generation.LastGeneration
	}

	return -1
}

func SetValidatingWebhookConfigurationGeneration(generations *[]operatorsv1.GenerationStatus, actual *admissionregistrationv1.ValidatingWebhookConfiguration) {
	if actual == nil {
		return
	}
	resourcemerge.SetGeneration(generations, operatorsv1.GenerationStatus{
		Group:          admissionregistrationv1.GroupName,
		Resource:       "validatingwebhookconfigurations",
		Namespace:      actual.Namespace,
		Name:           actual.Name,
		LastGeneration: actual.ObjectMeta.Generation,
	})
}

func SetPodDisruptionBudgetGeneration(generations *[]operatorsv1.GenerationStatus, actual *v1beta1.PodDisruptionBudget) {
	if actual == nil {
		return
	}
	resourcemerge.SetGeneration(generations, operatorsv1.GenerationStatus{
		Group:          "apps",
		Resource:       "poddisruptionbudget",
		Namespace:      actual.Namespace,
		Name:           actual.Name,
		LastGeneration: actual.ObjectMeta.Generation,
	})
}

func ExpectedPodDisruptionBudgetGeneration(required *v1beta1.PodDisruptionBudget, previousGenerations []operatorsv1.GenerationStatus) int64 {
	generation := resourcemerge.GenerationFor(previousGenerations, schema.GroupResource{Group: "apps", Resource: "poddisruptionbudget"}, required.Namespace, required.Name)
	if generation != nil {
		return generation.LastGeneration
	}
	return -1
}
