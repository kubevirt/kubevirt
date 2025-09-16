/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package admitters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	k8scorev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
	switch {
	case isHotplugPod(pod):
		return admitter.admitHotplugPod(ctx, pod)
	case isVirtLauncher(pod) && !isCompleted(pod):
		return admitter.admitLauncherPod(ctx, ar, pod)
	}
	return validating_webhooks.NewPassingAdmissionResponse()
}

func (admitter *PodEvictionAdmitter) admitHotplugPod(ctx context.Context, pod *k8scorev1.Pod) *admissionv1.AdmissionResponse {
	ownerPod, err := admitter.kubeClient.CoreV1().Pods(pod.Namespace).Get(ctx, pod.OwnerReferences[0].Name, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return denied(fmt.Sprintf("failed getting owner for hotplug pod: %v", err))
		}
		return validating_webhooks.NewPassingAdmissionResponse()
	}
	if !isVirtLauncher(ownerPod) || isCompleted(ownerPod) {
		return validating_webhooks.NewPassingAdmissionResponse()
	}
	vmiName, exists := ownerPod.GetAnnotations()[virtv1.DomainAnnotation]
	if !exists {
		return validating_webhooks.NewPassingAdmissionResponse()
	}

	_, err = admitter.virtClient.KubevirtV1().VirtualMachineInstances(pod.Namespace).Get(ctx, vmiName, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return denied(fmt.Sprintf("kubevirt failed getting the vmi: %v", err))
		}
		return validating_webhooks.NewPassingAdmissionResponse()
	}

	return denied(fmt.Sprintf("cannot evict hotplug pod: %s associated with running vmi: %s in namespace %s", pod.Name, vmiName, pod.Namespace))
}

func (admitter *PodEvictionAdmitter) admitLauncherPod(ctx context.Context, ar *admissionv1.AdmissionReview, pod *k8scorev1.Pod) *admissionv1.AdmissionResponse {
	vmiName, exists := pod.GetAnnotations()[virtv1.DomainAnnotation]
	if !exists {
		return validating_webhooks.NewPassingAdmissionResponse()
	}

	vmi, err := admitter.virtClient.KubevirtV1().VirtualMachineInstances(ar.Request.Namespace).Get(ctx, vmiName, metav1.GetOptions{})
	if err != nil {
		return denied(fmt.Sprintf("kubevirt failed getting the vmi: %v", err))
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

	if !markForEviction {
		return validating_webhooks.NewPassingAdmissionResponse()
	}

	// This message format is expected from descheduler.
	const evictionFmt = "Eviction triggered evacuation of VMI \"%s/%s\""
	if vmi.IsMarkedForEviction() {
		return denied(fmt.Sprintf("Evacuation in progress: "+evictionFmt, vmi.Namespace, vmi.Name))
	}
	if vmi.Status.NodeName != pod.Spec.NodeName {
		return denied("Eviction request for target Pod")
	}
	err = admitter.markVMI(ctx, vmi.Namespace, vmi.Name, pod.Spec.NodeName, isDryRun(ar))
	if err != nil {
		// As with the previous case, it is up to the user to issue a retry.
		return denied(fmt.Sprintf("kubevirt failed marking the vmi for eviction: %v", err))
	}
	return denied(fmt.Sprintf(evictionFmt, vmi.Namespace, vmi.Name))
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

func isHotplugPod(pod *k8scorev1.Pod) bool {
	return pod.Labels[virtv1.AppLabel] == "hotplug-disk" && len(pod.OwnerReferences) == 1
}

func isCompleted(pod *k8scorev1.Pod) bool {
	return pod.Status.Phase == k8scorev1.PodFailed || pod.Status.Phase == k8scorev1.PodSucceeded
}
