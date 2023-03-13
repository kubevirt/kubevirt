package util

import v1 "k8s.io/api/core/v1"

// IsPodReady treats the pod as ready to be handed over to virt-handler, as soon as all pods except
// the compute pod are ready.
func IsPodReady(pod *v1.Pod) bool {
	if IsPodDownOrGoingDown(pod) {
		return false
	}

	for _, containerStatus := range pod.Status.ContainerStatuses {
		// The compute container potentially holds a readiness probe for the VMI. Therefore
		// don't wait for the compute container to become ready (the VMI later on will trigger the change to ready)
		// and only check that the container started
		if containerStatus.Name == "compute" {
			if containerStatus.State.Running == nil {
				return false
			}
		} else if containerStatus.Name == "istio-proxy" {
			// When using istio the istio-proxy container will not be ready
			// until there is a service pointing to this pod.
			// We need to start the VM anyway
			if containerStatus.State.Running == nil {
				return false
			}

		} else if containerStatus.Ready == false {
			return false
		}
	}

	return pod.Status.Phase == v1.PodRunning
}

func IsPodDownOrGoingDown(pod *v1.Pod) bool {
	return PodIsDown(pod) || IsComputeContainerDown(pod) || pod.DeletionTimestamp != nil
}

func IsPodFailedOrGoingDown(pod *v1.Pod) bool {
	return isPodFailed(pod) || isComputeContainerFailed(pod) || pod.DeletionTimestamp != nil
}

func IsComputeContainerDown(pod *v1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == "compute" {
			return containerStatus.State.Terminated != nil
		}
	}
	return false
}

func isComputeContainerFailed(pod *v1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == "compute" {
			return containerStatus.State.Terminated != nil && containerStatus.State.Terminated.ExitCode != 0
		}
	}
	return false
}

func PodIsDown(pod *v1.Pod) bool {
	return pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed
}

func isPodFailed(pod *v1.Pod) bool {
	return pod.Status.Phase == v1.PodFailed
}

func PodExists(pod *v1.Pod) bool {
	if pod != nil {
		return true
	}
	return false
}
