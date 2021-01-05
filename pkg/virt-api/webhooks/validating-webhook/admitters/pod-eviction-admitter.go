package admitters

import (
	"fmt"
	"net/http"

	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type PodEvictionAdmitter struct {
	ClusterConfig *virtconfig.ClusterConfig
	VirtClient    kubecli.KubevirtClient
}

func (admitter *PodEvictionAdmitter) Admit(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	if !admitter.ClusterConfig.LiveMigrationEnabled() {
		return allowed()
	}

	launcher, err := admitter.VirtClient.CoreV1().Pods(ar.Request.Namespace).Get(ar.Request.Name, metav1.GetOptions{})
	if err != nil {
		return allowed()
	}

	if value, exists := launcher.GetLabels()[virtv1.AppLabel]; !exists || value != "virt-launcher" {
		return allowed()
	}

	domainName, exists := launcher.GetAnnotations()[virtv1.DomainAnnotation]
	if !exists {
		return allowed()
	}

	vmi, err := admitter.VirtClient.VirtualMachineInstance(ar.Request.Namespace).Get(domainName, &metav1.GetOptions{})
	if err != nil {
		return denied(fmt.Sprintf("kubevirt failed getting the vmi: %s", err.Error()))
	}
	if !vmi.IsEvictable() {
		// we don't act on VMIs without an eviction strategy
		return allowed()
	} else if !vmi.IsMigratable() {
		return denied(fmt.Sprintf(
			"VMI %s is configured with an eviction strategy but is not live-migratable", vmi.Name))
	}

	if !vmi.IsMarkedForEviction() {
		err := admitter.markVMI(ar, vmi, err)
		if err != nil {
			// As with the previous case, it is up to the user to issue a retry.
			return denied(fmt.Sprintf("kubevirt failed marking the vmi for eviction: %s", err.Error()))
		}
	}

	// We can let the request go through because the pod is protected by a PDB if the VMI wants to be live-migrated on
	// eviction. Otherwise, we can just evict it.
	return allowed()
}

func (admitter *PodEvictionAdmitter) markVMI(ar *v1beta1.AdmissionReview, vmi *virtv1.VirtualMachineInstance, err error) error {
	vmiCopy := vmi.DeepCopy()
	vmiCopy.Status.EvacuationNodeName = vmi.Status.NodeName
	_, err = admitter.VirtClient.VirtualMachineInstance(ar.Request.Namespace).Update(vmiCopy)
	return err
}

func denied(message string) *v1beta1.AdmissionResponse {
	return &v1beta1.AdmissionResponse{
		Allowed: false,
		Result: &metav1.Status{
			Message: message,
			Code:    http.StatusTooManyRequests,
		},
	}
}

func allowed() *v1beta1.AdmissionResponse {
	return &v1beta1.AdmissionResponse{Allowed: true}
}
