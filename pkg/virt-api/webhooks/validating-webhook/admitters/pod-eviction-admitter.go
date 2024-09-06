package admitters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/types"

	admissionv1 "k8s.io/api/admission/v1"
	k8scorev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/util/migrations"
	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type PodEvictionAdmitter struct {
	ClusterConfig *virtconfig.ClusterConfig
	VirtClient    kubecli.KubevirtClient
}

func isDryRun(ar *admissionv1.AdmissionReview) bool {
	dryRun := ar.Request.DryRun != nil && *ar.Request.DryRun == true

	if !dryRun {
		evictionObject := policyv1.Eviction{}
		if err := json.Unmarshal(ar.Request.Object.Raw, &evictionObject); err == nil {
			if evictionObject.DeleteOptions != nil && len(evictionObject.DeleteOptions.DryRun) > 0 {
				dryRun = evictionObject.DeleteOptions.DryRun[0] == metav1.DryRunAll
			}
		}
	}
	return dryRun
}

func (admitter *PodEvictionAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	pod, err := admitter.VirtClient.CoreV1().Pods(ar.Request.Namespace).Get(context.Background(), ar.Request.Name, metav1.GetOptions{})
	if err != nil {
		return validating_webhooks.NewPassingAdmissionResponse()
	}

	if !isVirtLauncher(pod) || isCompleted(pod) {
		return validating_webhooks.NewPassingAdmissionResponse()
	}

	domainName, exists := pod.GetAnnotations()[virtv1.DomainAnnotation]
	if !exists {
		return validating_webhooks.NewPassingAdmissionResponse()
	}

	vmi, err := admitter.VirtClient.VirtualMachineInstance(ar.Request.Namespace).Get(context.Background(), domainName, metav1.GetOptions{})
	if err != nil {
		return denied(fmt.Sprintf("kubevirt failed getting the vmi: %s", err.Error()))
	}

	evictionStrategy := migrations.VMIEvictionStrategy(admitter.ClusterConfig, vmi)
	if evictionStrategy == nil {
		// we don't act on VMIs without an eviction strategy
		return validating_webhooks.NewPassingAdmissionResponse()
	}

	markForEviction := false

	switch *evictionStrategy {
	case virtv1.EvictionStrategyLiveMigrate:
		if !vmi.IsMigratable() {
			return denied(fmt.Sprintf("VMI %s is configured with an eviction strategy but is not live-migratable", vmi.Name))
		}
		markForEviction = true
	case virtv1.EvictionStrategyLiveMigrateIfPossible:
		if vmi.IsMigratable() {
			markForEviction = true
		}
	case virtv1.EvictionStrategyExternal:
		markForEviction = true
	}

	if markForEviction && !vmi.IsMarkedForEviction() && vmi.Status.NodeName == pod.Spec.NodeName {
		err := admitter.markVMI(ar, vmi.Name, vmi.Status.NodeName, isDryRun(ar))
		if err != nil {
			// As with the previous case, it is up to the user to issue a retry.
			return denied(fmt.Sprintf("kubevirt failed marking the vmi for eviction: %s", err.Error()))
		}
	}

	// We can let the request go through because the pod is protected by a PDB if the VMI wants to be live-migrated on
	// eviction. Otherwise, we can just evict it.
	return validating_webhooks.NewPassingAdmissionResponse()
}

func (admitter *PodEvictionAdmitter) markVMI(ar *admissionv1.AdmissionReview, vmiName, nodeName string, dryRun bool) (err error) {
	data := fmt.Sprintf(`[{ "op": "add", "path": "/status/evacuationNodeName", "value": "%s" }]`, nodeName)

	if !dryRun {
		_, err = admitter.
			VirtClient.
			VirtualMachineInstance(ar.Request.Namespace).
			Patch(context.Background(),
				vmiName,
				types.JSONPatchType,
				[]byte(data),
				metav1.PatchOptions{})
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

func isVirtLauncher(pod *k8scorev1.Pod) bool {
	return pod.Labels[virtv1.AppLabel] == "virt-launcher"
}

func isCompleted(pod *k8scorev1.Pod) bool {
	return pod.Status.Phase == k8scorev1.PodFailed || pod.Status.Phase == k8scorev1.PodSucceeded
}
