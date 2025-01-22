package admitters

import (
	"context"

	admissionv1 "k8s.io/api/admission/v1"

	"kubevirt.io/kubevirt/pkg/util/webhooks"
	webhooks2 "kubevirt.io/kubevirt/pkg/virt-api/webhooks"

	"libvmi" 
)

type StatusAdmitter struct {
	VmsAdmitter *VMsAdmitter
}

func (s *StatusAdmitter) Admit(ctx context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if resp := webhooks.ValidateStatus(ar.Request.Object.Raw); resp != nil {
		return resp
	}
	// Check if the request resource is a VirtualMachine and handle it
	if webhooks.ValidateRequestResource(ar.Request.Resource, webhooks2.VirtualMachineGroupVersionResource.Group, webhooks2.VirtualMachineGroupVersionResource.Resource) {
		vmi := libvmi.New()

		return s.VmsAdmitter.AdmitStatus(ctx, ar)
	}
	reviewResponse := admissionv1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}
