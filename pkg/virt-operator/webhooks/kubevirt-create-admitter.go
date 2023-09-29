package webhooks

import (
	"fmt"

	"kubevirt.io/client-go/log"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	webhooks "kubevirt.io/kubevirt/pkg/util/webhooks"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
)

func NewKubeVirtCreateAdmitter(client kubecli.KubevirtClient) *kubeVirtCreateAdmitter {
	return &kubeVirtCreateAdmitter{
		client: client,
	}
}

type kubeVirtCreateAdmitter struct {
	client kubecli.KubevirtClient
}

func (k *kubeVirtCreateAdmitter) Admit(review *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	log.Log.Info("Trying to create KV")
	if resp := webhooks.ValidateSchema(v1.KubeVirtGroupVersionKind, review.Request.Object.Raw); resp != nil {
		return resp
	}
	//TODO: Do we want semantic validation

	// Best effort
	list, err := k.client.KubeVirt(k8sv1.NamespaceAll).List(&metav1.ListOptions{})
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}
	if len(list.Items) == 0 {
		fmt.Println("Allowed to create KV")
		return webhookutils.NewPassingAdmissionResponse()
	}
	return webhooks.ToAdmissionResponseError(fmt.Errorf("Kubevirt is already created"))
}
