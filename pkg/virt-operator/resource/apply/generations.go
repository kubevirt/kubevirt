package apply

import (
	operatorsv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func ExpectedMutatingWebhookConfigurationGeneration(required *admissionregistrationv1beta1.MutatingWebhookConfiguration, previousGenerations []operatorsv1.GenerationStatus) int64 {
	generation := resourcemerge.GenerationFor(previousGenerations, schema.GroupResource{Group: "admissionregistration.k8s.io", Resource: "mutatingwebhookconfigurations"}, required.Namespace, required.Name)
	if generation != nil {
		return generation.LastGeneration
	}

	return -1
}

func SetMutatingWebhookConfigurationGeneration(generations *[]operatorsv1.GenerationStatus, actual *admissionregistrationv1beta1.MutatingWebhookConfiguration) {
	if actual == nil {
		return
	}
	resourcemerge.SetGeneration(generations, operatorsv1.GenerationStatus{
		Group:          "admissionregistration.k8s.io",
		Resource:       "mutatingwebhookconfigurations",
		Namespace:      actual.Namespace,
		Name:           actual.Name,
		LastGeneration: actual.ObjectMeta.Generation,
	})
}

func ExpectedValidatingWebhookConfigurationGeneration(required *admissionregistrationv1beta1.ValidatingWebhookConfiguration, previousGenerations []operatorsv1.GenerationStatus) int64 {
	generation := resourcemerge.GenerationFor(previousGenerations, schema.GroupResource{Group: "admissionregistration.k8s.io", Resource: "validatingwebhookconfigurations"}, required.Namespace, required.Name)
	if generation != nil {
		return generation.LastGeneration
	}

	return -1
}

func SetValidatingWebhookConfigurationGeneration(generations *[]operatorsv1.GenerationStatus, actual *admissionregistrationv1beta1.ValidatingWebhookConfiguration) {
	if actual == nil {
		return
	}
	resourcemerge.SetGeneration(generations, operatorsv1.GenerationStatus{
		Group:          "admissionregistration.k8s.io",
		Resource:       "validatingwebhookconfigurations",
		Namespace:      actual.Namespace,
		Name:           actual.Name,
		LastGeneration: actual.ObjectMeta.Generation,
	})
}
