package admitters

import (
	"k8s.io/api/admission/v1beta1"

	webhooks2 "kubevirt.io/kubevirt/pkg/virt-api/webhooks"

	"kubevirt.io/kubevirt/pkg/util/webhooks"
)

type StatusAdmitter struct {
	vmsAdmitter *VMsAdmitter
}

func (s *StatusAdmitter) Admit(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	if resp := webhooks.ValidateStatus(ar.Request.Object.Raw); resp != nil {
		return resp
	}

	if webhooks.ValidateRequestResource(ar.Request.Resource, webhooks2.VirtualMachineGroupVersionResource.Group, webhooks2.VirtualMachineGroupVersionResource.Resource) {
		return s.vmsAdmitter.AdmitStatus(ar)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}
