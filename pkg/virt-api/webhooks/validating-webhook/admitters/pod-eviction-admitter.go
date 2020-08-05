package admitters

import (
	"fmt"

	"k8s.io/api/admission/v1beta1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
		return &v1beta1.AdmissionResponse{Allowed: true}
	}

	launcher, err := admitter.VirtClient.CoreV1().Pods(ar.Request.Namespace).Get(ar.Request.Name, metav1.GetOptions{})
	if err != nil {
		// we don't want to block the request because we were not able to find the pod
		return &v1beta1.AdmissionResponse{Allowed: true}
	}

	if value, exists := launcher.GetLabels()[virtv1.AppLabel]; !exists || value != "virt-launcher" {
		// this is not a virt-launcher app so there's no reason to block it
		return &v1beta1.AdmissionResponse{Allowed: true}
	}

	updatedLauncher := launcher.DeepCopy()
	updatedLauncher.Status.Conditions = append(updatedLauncher.Status.Conditions, k8sv1.PodCondition{
		Type:               virtv1.LauncherMarkedForEviction,
		Reason:             "MarkedForEviction",
		Message:            "An eviction request on this pod was intercepted",
		LastTransitionTime: metav1.Now(),
		Status:             k8sv1.ConditionTrue,
	})

	if _, err = admitter.VirtClient.CoreV1().Pods(ar.Request.Namespace).UpdateStatus(updatedLauncher); err != nil {
		return &v1beta1.AdmissionResponse{
			Allowed: false,
			Result: &v1.Status{
				Message: fmt.Sprintf("could not migrate VMI: %s for evicted launcher: %s. Error: %s",
					launcher.Labels[virtv1.DomainAnnotation], launcher.GetName(), err.Error()),
			},
		}
	}

	return &v1beta1.AdmissionResponse{
		Allowed: false,
		Result: &v1.Status{
			Message: "virt-launcher eviction will be handled by KubeVirt",
			Code:    429,
		},
	}
}
