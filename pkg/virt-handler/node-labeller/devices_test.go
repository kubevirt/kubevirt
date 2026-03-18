//go:build amd64 || s390x

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

package nodelabeller

import (
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubevirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/testutils"
	util "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
)

var _ = Describe("Node-labeller devices config", func() {
	var nlController *NodeLabeller

	BeforeEach(func() {
		kv := &kubevirtv1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: kubevirtv1.KubeVirtSpec{
				Configuration: kubevirtv1.KubeVirtConfiguration{
					ObsoleteCPUModels: util.DefaultObsoleteCPUModels,
				},
			},
		}

		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKV(kv)

		nlController = &NodeLabeller{
			nodeClient:              nil,
			clusterConfig:           clusterConfig,
			logger:                  log.DefaultLogger(),
			volumePath:              "testdata",
			domCapabilitiesFileName: "virsh_domcapabilities.xml",
			cpuCounter:              nil,
			hostCPUModel:            hostCPUModel{requiredFeatures: make(map[string]bool)},
			arch:                    newArchLabeller(runtime.GOARCH),
		}
	})

	Context("should return correct <hostdev> capabilities", func() {
		DescribeTable("for IOMMUFD",
			func(isSupported bool, domCapsFileName string) {
				if domCapsFileName != "" {
					nlController.domCapabilitiesFileName = domCapsFileName
				}

				hostDomCapabilities, err := nlController.getDomCapabilities()
				Expect(err).ToNot(HaveOccurred())

				nlController.loadDomHostDevCaps(&hostDomCapabilities.HostDev)
				Expect(nlController.hostDevIOMMUFDSupported).To(Equal(isSupported))
			},
			Entry("when IOMMUFD is reported as not supported", false, "domcapabilities_hostdev_noiommufd.xml"),
			Entry("when IOMMUFD is reported as supported", true, "domcapabilities_hostdev_iommufd.xml"),
		)
	})

	It("make sure proper labels are removed on removeLabellerLabels()", func() {
		node := &k8sv1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Labels: devicesNodeLabels,
			},
		}

		nlController.removeLabellerLabels(node)

		badKey := ""
		for key := range node.Labels {
			for _, labellerPrefix := range nodeLabellerLabels {
				if strings.HasPrefix(key, labellerPrefix) {
					badKey = key
					break
				}
			}
		}
		Expect(badKey).To(BeEmpty())
	})
})

var devicesNodeLabels = map[string]string{
	kubevirtv1.HostDevIOMMUFDLabel: "true",
}
