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

package services

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

func newClusterConfigWithSerialConsoleLog(disabled bool) *virtconfig.ClusterConfig {
	kvConfig := &v1.KubeVirtConfiguration{
		DeveloperConfiguration: &v1.DeveloperConfiguration{},
	}
	if disabled {
		kvConfig.VirtualMachineOptions = &v1.VirtualMachineOptions{
			DisableSerialConsoleLog: &v1.DisableSerialConsoleLog{},
		}
	}
	config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(kvConfig)
	return config
}

var _ = Describe("Serial Console Log", func() {

	Context("isSerialConsoleLogEnabled", func() {
		It("should return true when no VMI overrides are set and config enables it", func() {
			vmi := &v1.VirtualMachineInstance{}
			config := newClusterConfigWithSerialConsoleLog(false)
			Expect(isSerialConsoleLogEnabled(vmi, config)).To(BeTrue())
		})

		It("should return false when config disables serial console log", func() {
			vmi := &v1.VirtualMachineInstance{}
			config := newClusterConfigWithSerialConsoleLog(true)
			Expect(isSerialConsoleLogEnabled(vmi, config)).To(BeFalse())
		})

		It("should return false when AutoattachSerialConsole is false", func() {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							AutoattachSerialConsole: pointer.P(false),
						},
					},
				},
			}
			config := newClusterConfigWithSerialConsoleLog(false)
			Expect(isSerialConsoleLogEnabled(vmi, config)).To(BeFalse())
		})

		It("should return true when LogSerialConsole is explicitly true", func() {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							LogSerialConsole: pointer.P(true),
						},
					},
				},
			}
			// Even with config disabled, VMI-level override wins
			config := newClusterConfigWithSerialConsoleLog(true)
			Expect(isSerialConsoleLogEnabled(vmi, config)).To(BeTrue())
		})

		It("should return false when LogSerialConsole is explicitly false", func() {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							LogSerialConsole: pointer.P(false),
						},
					},
				},
			}
			config := newClusterConfigWithSerialConsoleLog(false)
			Expect(isSerialConsoleLogEnabled(vmi, config)).To(BeFalse())
		})

		It("should return false when AutoattachSerialConsole is false even if LogSerialConsole is true", func() {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							AutoattachSerialConsole: pointer.P(false),
							LogSerialConsole:        pointer.P(true),
						},
					},
				},
			}
			config := newClusterConfigWithSerialConsoleLog(false)
			Expect(isSerialConsoleLogEnabled(vmi, config)).To(BeFalse())
		})
	})

	Context("resourcesForSerialConsoleLogContainer", func() {
		It("should return default resources when no overrides are configured", func() {
			config := newClusterConfigWithSerialConsoleLog(false)
			resources := resourcesForSerialConsoleLogContainer(false, false, config)

			Expect(resources.Requests[k8sv1.ResourceMemory]).To(Equal(resource.MustParse("35M")))
			Expect(resources.Requests[k8sv1.ResourceCPU]).To(Equal(resource.MustParse("5m")))
			Expect(resources.Limits[k8sv1.ResourceMemory]).To(Equal(resource.MustParse("60M")))
			Expect(resources.Limits[k8sv1.ResourceCPU]).To(Equal(resource.MustParse("15m")))
		})

		It("should equalize requests and limits for dedicated CPUs", func() {
			config := newClusterConfigWithSerialConsoleLog(false)
			resources := resourcesForSerialConsoleLogContainer(true, false, config)

			Expect(resources.Requests[k8sv1.ResourceCPU]).To(Equal(resources.Limits[k8sv1.ResourceCPU]))
			Expect(resources.Requests[k8sv1.ResourceMemory]).To(Equal(resources.Limits[k8sv1.ResourceMemory]))
		})

		It("should equalize requests and limits for guaranteed QOS", func() {
			config := newClusterConfigWithSerialConsoleLog(false)
			resources := resourcesForSerialConsoleLogContainer(false, true, config)

			Expect(resources.Requests[k8sv1.ResourceCPU]).To(Equal(resources.Limits[k8sv1.ResourceCPU]))
			Expect(resources.Requests[k8sv1.ResourceMemory]).To(Equal(resources.Limits[k8sv1.ResourceMemory]))
		})

		It("should not equalize requests and limits when neither dedicated CPU nor guaranteed QOS", func() {
			config := newClusterConfigWithSerialConsoleLog(false)
			resources := resourcesForSerialConsoleLogContainer(false, false, config)

			Expect(resources.Requests[k8sv1.ResourceMemory]).ToNot(Equal(resources.Limits[k8sv1.ResourceMemory]))
			Expect(resources.Requests[k8sv1.ResourceCPU]).ToNot(Equal(resources.Limits[k8sv1.ResourceCPU]))
		})
	})

	Context("generateSerialConsoleLogContainer", func() {
		It("should return nil when serial console log is disabled", func() {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							AutoattachSerialConsole: pointer.P(false),
						},
					},
				},
			}
			config := newClusterConfigWithSerialConsoleLog(false)
			container := generateSerialConsoleLogContainer(vmi, "test-image", config, 1)
			Expect(container).To(BeNil())
		})

		It("should return a valid container when serial console log is enabled", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					UID: "test-uid",
				},
			}
			config := newClusterConfigWithSerialConsoleLog(false)
			container := generateSerialConsoleLogContainer(vmi, "test-image", config, 2)
			Expect(container).ToNot(BeNil())
			Expect(container.Name).To(Equal("guest-console-log"))
			Expect(container.Image).To(Equal("test-image"))
			Expect(container.Command).To(Equal([]string{"/usr/bin/virt-tail"}))
			Expect(container.Args).To(ContainElement("--logfile"))
		})

		It("should set security context with non-root user", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					UID: "test-uid",
				},
			}
			config := newClusterConfigWithSerialConsoleLog(false)
			container := generateSerialConsoleLogContainer(vmi, "test-image", config, 1)
			Expect(container).ToNot(BeNil())
			Expect(container.SecurityContext).ToNot(BeNil())
			Expect(*container.SecurityContext.RunAsNonRoot).To(BeTrue())
			Expect(*container.SecurityContext.AllowPrivilegeEscalation).To(BeFalse())
			Expect(container.SecurityContext.Capabilities.Drop).To(ContainElement(k8sv1.Capability("ALL")))
		})
	})
})
