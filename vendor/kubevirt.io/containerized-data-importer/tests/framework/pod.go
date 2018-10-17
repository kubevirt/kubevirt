package framework

import (
	"time"

	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/tests/utils"
)

// CreatePod is a wrapper around utils.CreatePod
func (f *Framework) CreatePod(podDef *k8sv1.Pod) (*k8sv1.Pod, error) {
	return utils.CreatePod(f.K8sClient, f.Namespace.Name, podDef)
}

// DeletePod is a wrapper around utils.DeletePod
func (f *Framework) DeletePod(pod *k8sv1.Pod) error {
	return utils.DeletePod(f.K8sClient, pod, f.Namespace.Name)
}

// WaitTimeoutForPodReady is a wrapper around utils.WaitTimeouotForPodReady
func (f *Framework) WaitTimeoutForPodReady(podName string, timeout time.Duration) error {
	return utils.WaitTimeoutForPodReady(f.K8sClient, podName, f.Namespace.Name, timeout)
}

// WaitTimeoutForPodStatus is a wrapper around utils.WaitTimeouotForPodStatus
func (f *Framework) WaitTimeoutForPodStatus(podName string, status k8sv1.PodPhase, timeout time.Duration) error {
	return utils.WaitTimeoutForPodStatus(f.K8sClient, podName, f.Namespace.Name, status, timeout)
}

// FindPodByPrefix is a wrapper around utils.FindPodByPrefix
func (f *Framework) FindPodByPrefix(prefix string) (*k8sv1.Pod, error) {
	return utils.FindPodByPrefix(f.K8sClient, f.Namespace.Name, prefix, common.CDILabelSelector)
}
