package admitters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	k8scorev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	virtv1 "kubevirt.io/api/core/v1"
	kubevirt "kubevirt.io/client-go/kubevirt"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/util/migrations"
	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type PodEvictionAdmitter struct {
	clusterConfig *virtconfig.ClusterConfig
	kubeClient    kubernetes.Interface
	virtClient    kubevirt.Interface
}

func NewPodEvictionAdmitter(clusterConfig *virtconfig.ClusterConfig, kubeClient kubernetes.Interface, virtClient kubevirt.Interface) *PodEvictionAdmitter {
	return &PodEvictionAdmitter{
		clusterConfig: clusterConfig,
		kubeClient:    kubeClient,
		virtClient:    virtClient,
	}
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

func (admitter *PodEvictionAdmitter) Admit(ctx context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	pod, err := admitter.kubeClient.CoreV1().Pods(ar.Request.Namespace).Get(ctx, ar.Request.Name, metav1.GetOptions{})
	if err != nil {
		return validating_webhooks.NewPassingAdmissionResponse()
	}

	if !isVirtLauncher(pod) || isCompleted(pod) {
		return validating_webhooks.NewPassingAdmissionResponse()
	}

	vmiName, exists := pod.GetAnnotations()[virtv1.DomainAnnotation]
	if !exists {
		return validating_webhooks.NewPassingAdmissionResponse()
	}

	vmi, err := admitter.virtClient.KubevirtV1().VirtualMachineInstances(ar.Request.Namespace).Get(ctx, vmiName, metav1.GetOptions{})
	if err != nil {
		return denied(fmt.Sprintf("kubevirt failed getting the vmi: %s", err.Error()))
	}

	evictionStrategy := migrations.VMIEvictionStrategy(admitter.clusterConfig, vmi)
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
		err := admitter.markVMI(ctx, vmi.Namespace, vmi.Name, vmi.Status.NodeName, isDryRun(ar))
		if err != nil {
			// As with the previous case, it is up to the user to issue a retry.
			return denied(fmt.Sprintf("kubevirt failed marking the vmi for eviction: %s", err.Error()))
		}

		return denied(fmt.Sprintf("Eviction triggered evacuation of VMI \"%s/%s\"", vmi.Namespace, vmi.Name))
	}

	// We can let the request go through because the pod is protected by a PDB if the VMI wants to be live-migrated on
	// eviction. Otherwise, we can just evict it.
	return validating_webhooks.NewPassingAdmissionResponse()
}

func (admitter *PodEvictionAdmitter) markVMI(ctx context.Context, vmiNamespace, vmiName, nodeName string, dryRun bool) error {
	patchBytes, err := patch.New(patch.WithAdd("/status/evacuationNodeName", nodeName)).GeneratePayload()
	if err != nil {
		return err
	}

	var patchOptions metav1.PatchOptions
	if dryRun {
		patchOptions.DryRun = []string{metav1.DryRunAll}
	}

	_, err = admitter.
		virtClient.
		KubevirtV1().
		VirtualMachineInstances(vmiNamespace).
		Patch(ctx,
			vmiName,
			types.JSONPatchType,
			patchBytes,
			patchOptions,
		)

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
