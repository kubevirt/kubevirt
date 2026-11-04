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

package virthandler

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/hypervisor"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("claimDeviceOwnership", func() {
	var controller *BaseController
	var path string

	BeforeEach(func() {
		diskutils.MockDefaultOwnershipManager()

		config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
		controller = &BaseController{
			clusterConfig:      config,
			hypervisorNodeInfo: hypervisor.NewHypervisorNodeInformation(""),
		}

		path = GinkgoT().TempDir()
	})

	Context("with a generic device", func() {
		const device = "device"
		BeforeEach(func() {
			Expect(os.Mkdir(filepath.Join(path, "dev"), 0o755)).To(Succeed())
			f, err := os.Create(filepath.Join(path, "dev", device))
			Expect(err).ToNot(HaveOccurred())
			Expect(f.Close()).To(Succeed())
		})

		It("should succeed", func() {
			p, err := safepath.JoinAndResolveWithRelativeRoot("/", path)
			Expect(err).ToNot(HaveOccurred())
			Expect(controller.claimDeviceOwnership(p, device)).To(Succeed())
		})

		DescribeTable("should return an error if the device doesn't exist", func(softEmulation bool) {
			kv := &v1.KubeVirtConfiguration{
				DeveloperConfiguration: &v1.DeveloperConfiguration{
					UseEmulation: softEmulation,
				},
			}
			config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(kv)
			controller.clusterConfig = config

			p, err := safepath.JoinAndResolveWithRelativeRoot("/", path)
			Expect(err).ToNot(HaveOccurred())
			Expect(controller.claimDeviceOwnership(p, "noexist")).To(HaveOccurred())
		},
			Entry("with software emulation enabled", true),
			Entry("with software emulation disable", false),
		)
	})

	DescribeTable("iwith kvm device not existing should", func(softEmulation bool) {
		kv := &v1.KubeVirtConfiguration{
			DeveloperConfiguration: &v1.DeveloperConfiguration{
				UseEmulation: softEmulation,
			},
		}
		config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(kv)
		controller.clusterConfig = config

		p, err := safepath.JoinAndResolveWithRelativeRoot("/", path)
		Expect(err).ToNot(HaveOccurred())
		err = controller.claimDeviceOwnership(p, "kvm")
		if softEmulation {
			Expect(err).NotTo(HaveOccurred())
		} else {
			Expect(err).To(HaveOccurred())
		}
	},
		Entry("failed with software emulation enabled", true),
		Entry("succeed with software emulation disable", false),
	)
})
