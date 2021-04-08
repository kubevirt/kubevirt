package admitters

import (
	admissionv1 "k8s.io/api/admission/v1"

	webhooks2 "kubevirt.io/kubevirt/pkg/virt-api/webhooks"

	"kubevirt.io/kubevirt/pkg/util/webhooks"
)

type StatusAdmitter struct {
	VmsAdmitter *VMsAdmitter
}

func (s *StatusAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if resp := webhooks.ValidateStatus(ar.Request.Object.Raw); resp != nil {
		return resp
	}

	if webhooks.ValidateRequestResource(ar.Request.Resource, webhooks2.VirtualMachineGroupVersionResource.Group, webhooks2.VirtualMachineGroupVersionResource.Resource) {
		return s.VmsAdmitter.AdmitStatus(ar)
	}

	reviewResponse := admissionv1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}
