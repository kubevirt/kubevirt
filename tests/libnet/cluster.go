package libnet

import (
	"context"
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	netutils "k8s.io/utils/net"

	v1 "kubevirt.io/client-go/apis/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests/flags"
)

func IsClusterDualStack(virtClient kubecli.KubevirtClient) (bool, error) {
	// grab us some neat kubevirt pod; let's say virt-handler is our target.
	targetPodType := "virt-handler"
	virtHandlerPod, err := getPodByKubeVirtRole(virtClient, targetPodType)
	if err != nil {
		return false, err
	}

	hasMultipleIPAddrs := len(virtHandlerPod.Status.PodIPs) > 1
	if !hasMultipleIPAddrs {
		return false, nil
	}

	for _, ip := range virtHandlerPod.Status.PodIPs {
		if netutils.IsIPv6String(ip.IP) {
			return true, nil
		}
	}
	return false, nil
}

func getPodByKubeVirtRole(virtClient kubecli.KubevirtClient, kubevirtPodRole string) (*k8sv1.Pod, error) {
	labelSelectorValue := fmt.Sprintf("%s = %s", v1.AppLabel, kubevirtPodRole)
	pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelectorValue})
	if err != nil {
		return nil, fmt.Errorf("could not filter virt-handler pods: %v", err)
	}
	if len(pods.Items) <= 0 {
		return nil, fmt.Errorf("could not find virt-handler pods on the system")
	}
	return &pods.Items[0], nil
}
