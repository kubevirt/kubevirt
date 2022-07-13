package cluster

import (
	"context"
	"fmt"
	"sync"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	netutils "k8s.io/utils/net"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/flags"
)

var onceIPv4 sync.Once
var clusterSupportsIpv4 bool
var errIPv4 error
var onceIPv6 sync.Once
var clusterSupportsIpv6 bool
var errIPv6 error

func DualStack(virtClient kubecli.KubevirtClient) (bool, error) {
	supportsIpv4, err := SupportsIpv4(virtClient)
	if err != nil {
		return false, err
	}

	supportsIpv6, err := SupportsIpv6(virtClient)
	if err != nil {
		return false, err
	}
	return supportsIpv4 && supportsIpv6, nil
}

func SupportsIpv4(virtClient kubecli.KubevirtClient) (bool, error) {
	onceIPv4.Do(func() {
		clusterSupportsIpv4, errIPv4 = clusterAnswersIPCondition(virtClient, netutils.IsIPv4String)
	})
	return clusterSupportsIpv4, errIPv4
}

func SupportsIpv6(virtClient kubecli.KubevirtClient) (bool, error) {
	onceIPv6.Do(func() {
		clusterSupportsIpv6, errIPv6 = clusterAnswersIPCondition(virtClient, netutils.IsIPv6String)
	})
	return clusterSupportsIpv6, errIPv6
}

func clusterAnswersIPCondition(virtClient kubecli.KubevirtClient, ipCondition func(ip string) bool) (bool, error) {
	// grab us some neat kubevirt pod; let's say virt-handler is our target.
	targetPodType := "virt-handler"
	virtHandlerPod, err := getPodByKubeVirtRole(virtClient, targetPodType)
	if err != nil {
		return false, err
	}

	for _, ip := range virtHandlerPod.Status.PodIPs {
		if ipCondition(ip.IP) {
			return true, nil
		}
	}
	return false, nil
}

func getPodByKubeVirtRole(virtClient kubecli.KubevirtClient, kubevirtPodRole string) (*k8sv1.Pod, error) {
	labelSelectorValue := fmt.Sprintf("%s = %s", v1.AppLabel, kubevirtPodRole)
	pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(
		context.Background(),
		metav1.ListOptions{LabelSelector: labelSelectorValue},
	)
	if err != nil {
		return nil, fmt.Errorf("could not filter virt-handler pods: %v", err)
	}
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("could not find virt-handler pods on the system")
	}
	return &pods.Items[0], nil
}
