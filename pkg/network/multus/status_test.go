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

package multus_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8scorev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	"kubevirt.io/kubevirt/pkg/network/multus"
)

var _ = Describe("Network Status", func() {
	Context("NetworkStatusesByPodIfaceName", func() {
		It("Should map network statuses by pod interface name with multiple networks", func() {
			networkStatuses := []networkv1.NetworkStatus{
				{Name: "cluster-default", Interface: "eth0"},
				{Name: "meganet", Interface: "pod7e0055a6880"},
			}

			expected := map[string]networkv1.NetworkStatus{
				"eth0":           {Name: "cluster-default", Interface: "eth0"},
				"pod7e0055a6880": {Name: "meganet", Interface: "pod7e0055a6880"},
			}

			Expect(multus.NetworkStatusesByPodIfaceName(networkStatuses)).To(Equal(expected))
		})
	})

	Context("NetworkStatusesFromPod", func() {
		DescribeTable("should return an empty slice", func(podAnnotations map[string]string) {
			Expect(multus.NetworkStatusesFromPod(newStubPod(podAnnotations))).To(BeEmpty())
		},
			Entry("when network status annotation is missing", map[string]string{}),
			Entry("when network status annotation is empty", map[string]string{networkv1.NetworkStatusAnnot: ""}),
			Entry("when network status annotation is illegal", map[string]string{networkv1.NetworkStatusAnnot: "not a valid JSON array"}),
		)

		It("Should return a valid network status slice", func() {
			const multusNetworkStatusWithPrimaryAndSecondaryNets = `[` +
				`{"name":"k8s-pod-network","ips":["10.244.196.146","fd10:244::c491"],"default":true,"dns":{}},` +
				`{"name":"meganet","interface":"pod7e0055a6880","mac":"8a:37:d9:e7:0f:18","dns":{}}` +
				`]`

			annotations := map[string]string{networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryAndSecondaryNets}

			expectedResult := []networkv1.NetworkStatus{
				{
					Name:    "k8s-pod-network",
					IPs:     []string{"10.244.196.146", "fd10:244::c491"},
					Default: true,
					DNS:     networkv1.DNS{},
				},
				{
					Name:      "meganet",
					Interface: "pod7e0055a6880",
					Mac:       "8a:37:d9:e7:0f:18",
					Default:   false,
					DNS:       networkv1.DNS{},
				},
			}

			Expect(multus.NetworkStatusesFromPod(newStubPod(annotations))).To(Equal(expectedResult))
		})
	})

	Context("LookupPodPrimaryIfaceName", func() {
		const (
			defaultPrimaryPodIfaceName = "eth0"
			customPrimaryPodIfaceName  = "custom-iface"
		)

		DescribeTable("should return empty string", func(networkStatuses []networkv1.NetworkStatus) {
			Expect(multus.LookupPodPrimaryIfaceName(networkStatuses)).To(BeEmpty())
		},
			Entry("when network status is nil", nil),
			Entry("when network status is empty", []networkv1.NetworkStatus{}),
			Entry("when network status does not report interface name",
				[]networkv1.NetworkStatus{
					{Name: "k8s-pod-network", Default: true},
				},
			),
			Entry("when network status does not report a default interface name",
				[]networkv1.NetworkStatus{
					{Name: "net1", Interface: "", Default: false},
					{Name: "net2", Interface: "pod12345", Default: false},
				},
			),
		)

		DescribeTable("Should return the primary pod interface name", func(networkStatuses []networkv1.NetworkStatus, expectedResult string) {
			Expect(multus.LookupPodPrimaryIfaceName(networkStatuses)).To(Equal(expectedResult))
		},
			Entry("When the primary pod interface name is reported and its the default value",
				[]networkv1.NetworkStatus{
					{Name: "k8s-pod-network", Interface: defaultPrimaryPodIfaceName, Default: true},
					{Name: "some-net", Interface: "pod123456", Default: false},
				},
				defaultPrimaryPodIfaceName,
			),
			Entry("When the primary pod interface name is reported and it has the custom value",
				[]networkv1.NetworkStatus{
					{Name: "k8s-pod-network", Interface: defaultPrimaryPodIfaceName, Default: false},
					{Name: "k8s-pod-network", Interface: customPrimaryPodIfaceName, Default: true},
					{Name: "some-net", Interface: "net1", Default: false},
				},
				customPrimaryPodIfaceName,
			),
		)
	})
})

func newStubPod(annotations map[string]string) *k8scorev1.Pod {
	return &k8scorev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
		},
	}
}
