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
 */

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

var (
	onceIPv4            sync.Once
	clusterSupportsIpv4 bool
	errIPv4             error
	onceIPv6            sync.Once
	clusterSupportsIpv6 bool
	errIPv6             error
)

func DualStack() (bool, error) {
	supportsIpv4, err := SupportsIpv4()
	if err != nil {
		return false, err
	}

	supportsIpv6, err := SupportsIpv6()
	if err != nil {
		return false, err
	}
	return supportsIpv4 && supportsIpv6, nil
}

func SupportsIpv4() (bool, error) {
	onceIPv4.Do(func() {
		clusterSupportsIpv4, errIPv4 = clusterAnswersIPCondition(netutils.IsIPv4String)
	})
	return clusterSupportsIpv4, errIPv4
}

func SupportsIpv6() (bool, error) {
	onceIPv6.Do(func() {
		clusterSupportsIpv6, errIPv6 = clusterAnswersIPCondition(netutils.IsIPv6String)
	})
	return clusterSupportsIpv6, errIPv6
}

func clusterAnswersIPCondition(ipCondition func(ip string) bool) (bool, error) {
	// grab us some neat kubevirt pod; let's say virt-handler is our target.
	targetPodType := "virt-handler"
	virtHandlerPod, err := getPodByKubeVirtRole(targetPodType)
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

func getPodByKubeVirtRole(kubevirtPodRole string) (*k8sv1.Pod, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}

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
