package services

import (
	"fmt"
	"strings"

	k8sv1 "k8s.io/api/core/v1"
)

func GetComputeContainer(pod *k8sv1.Pod) *k8sv1.Container {
	for _, container := range pod.Spec.Containers {
		if strings.HasSuffix(container.Name, "compute") {
			return &container
		}
	}
	return nil
}

func GetPodCPULimitsCount(pod *k8sv1.Pod) (int64, error) {
	cc := GetComputeContainer(pod)
	if cc == nil {
		return 0, fmt.Errorf("Could not find VMI compute container")
	}

	cpuLimit, ok := cc.Resources.Limits[k8sv1.ResourceCPU]
	if !ok {
		return 0, fmt.Errorf("Could not find dedicated CPU limit in VMI compute container")
	}
	return cpuLimit.Value(), nil
}

func GetPodMemoryRequests(pod *k8sv1.Pod) (string, error) {
	cc := GetComputeContainer(pod)
	if cc == nil {
		return "", fmt.Errorf("Could not find VMI compute container")
	}

	memReq, ok := cc.Resources.Requests[k8sv1.ResourceMemory]
	if !ok {
		return "", fmt.Errorf("Could not find memory request in VMI compute container")
	}

	if hugePagesReq, ok := cc.Resources.Requests[k8sv1.ResourceHugePagesPrefix+"2Mi"]; ok {
		memReq.Add(hugePagesReq)
	}

	if hugePagesReq, ok := cc.Resources.Requests[k8sv1.ResourceHugePagesPrefix+"1Gi"]; ok {
		memReq.Add(hugePagesReq)
	}

	return memReq.String(), nil
}
