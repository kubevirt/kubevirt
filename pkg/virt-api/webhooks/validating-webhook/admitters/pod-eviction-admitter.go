package admitters

import (
	"context"
	"fmt"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type PodEvictionAdmitter struct {
	ClusterConfig *virtconfig.ClusterConfig
	VirtClient    kubecli.KubevirtClient
}

func (admitter *PodEvictionAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if !admitter.ClusterConfig.LiveMigrationEnabled() {
		return validating_webhooks.NewPassingAdmissionResponse()
	}

	launcher, err := admitter.VirtClient.CoreV1().Pods(ar.Request.Namespace).Get(context.Background(), ar.Request.Name, metav1.GetOptions{})
	if err != nil {
		return validating_webhooks.NewPassingAdmissionResponse()
	}

	if value, exists := launcher.GetLabels()[virtv1.AppLabel]; !exists || value != "virt-launcher" {
		return validating_webhooks.NewPassingAdmissionResponse()
	}

	domainName, exists := launcher.GetAnnotations()[virtv1.DomainAnnotation]
	if !exists {
		return validating_webhooks.NewPassingAdmissionResponse()
	}

	vmi, err := admitter.VirtClient.VirtualMachineInstance(ar.Request.Namespace).Get(domainName, &metav1.GetOptions{})
	if err != nil {
		return denied(fmt.Sprintf("kubevirt failed getting the vmi: %s", err.Error()))
	}
	if !vmi.IsEvictable() {
		// we don't act on VMIs without an eviction strategy
		return validating_webhooks.NewPassingAdmissionResponse()
	} else if !vmi.IsMigratable() {
		return denied(fmt.Sprintf(
			"VMI %s is configured with an eviction strategy but is not live-migratable", vmi.Name))
	}

	if !vmi.IsMarkedForEviction() && vmi.Status.NodeName == launcher.Spec.NodeName {
		dryRun := ar.Request.DryRun != nil && *ar.Request.DryRun == true
		err := admitter.markVMI(ar, vmi, dryRun)
		if err != nil {
			// As with the previous case, it is up to the user to issue a retry.
			return denied(fmt.Sprintf("kubevirt failed marking the vmi for eviction: %s", err.Error()))
		}
	}

	// We can let the request go through because the pod is protected by a PDB if the VMI wants to be live-migrated on
	// eviction. Otherwise, we can just evict it.
	return validating_webhooks.NewPassingAdmissionResponse()
}

func (admitter *PodEvictionAdmitter) markVMI(ar *admissionv1.AdmissionReview, vmi *virtv1.VirtualMachineInstance, dryRun bool) (err error) {
	vmiCopy := vmi.DeepCopy()
	vmiCopy.Status.EvacuationNodeName = vmi.Status.NodeName
	if !dryRun {
		_, err = admitter.VirtClient.VirtualMachineInstance(ar.Request.Namespace).Update(vmiCopy)
	}
	return err
}

func denied(message string) *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{
		Allowed: false,
		Result: &metav1.Status{
			Message: message,
			Code:    http.StatusTooManyRequests,
		},
	}
}
