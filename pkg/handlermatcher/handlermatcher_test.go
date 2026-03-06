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

package handlermatcher

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
)

func TestHandlerMatcher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Handler Matcher Suite")
}

var _ = Describe("MatchVMIToHandlerPool", func() {
	gpuPool := v1.VirtHandlerPoolConfig{
		Name:              "gpu-pool",
		VirtLauncherImage: "registry.example.com/virt-launcher:gpu",
		NodeSelector:      map[string]string{"nvidia.com/gpu.product": "Tesla-T4"},
		Selector: v1.VirtHandlerPoolSelector{
			DeviceNames: []string{"nvidia.com/TU104GL_Tesla_T4"},
		},
	}

	labelPool := v1.VirtHandlerPoolConfig{
		Name:              "secure-pool",
		VirtLauncherImage: "registry.example.com/virt-launcher:hardened",
		NodeSelector:      map[string]string{"security-zone": "restricted"},
		Selector: v1.VirtHandlerPoolSelector{
			VMLabels: &v1.VirtHandlerPoolVMLabels{
				MatchLabels: map[string]string{"workload-class": "secure"},
			},
		},
	}

	bothPool := v1.VirtHandlerPoolConfig{
		Name:              "both-pool",
		VirtLauncherImage: "registry.example.com/virt-launcher:both",
		NodeSelector:      map[string]string{"special": "true"},
		Selector: v1.VirtHandlerPoolSelector{
			DeviceNames: []string{"nvidia.com/TU104GL_Tesla_T4"},
			VMLabels: &v1.VirtHandlerPoolVMLabels{
				MatchLabels: map[string]string{"workload-class": "secure"},
			},
		},
	}

	pools := []v1.VirtHandlerPoolConfig{gpuPool, labelPool}

	newVMI := func(gpus []v1.GPU, hostDevices []v1.HostDevice, labels map[string]string) *v1.VirtualMachineInstance {
		return &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs:        gpus,
						HostDevices: hostDevices,
					},
				},
			},
		}
	}

	DescribeTable("should match VMI to pool",
		func(vmi *v1.VirtualMachineInstance, pools []v1.VirtHandlerPoolConfig, expectedPoolName string) {
			pool := MatchVMIToHandlerPool(pools, vmi)
			if expectedPoolName == "" {
				Expect(pool).To(BeNil())
			} else {
				Expect(pool).NotTo(BeNil())
				Expect(pool.Name).To(Equal(expectedPoolName))
			}
		},
		Entry("GPU device match",
			newVMI(
				[]v1.GPU{{Name: "gpu1", DeviceName: "nvidia.com/TU104GL_Tesla_T4"}},
				nil, nil,
			),
			pools, "gpu-pool",
		),
		Entry("HostDevice match",
			newVMI(
				nil,
				[]v1.HostDevice{{Name: "dev1", DeviceName: "nvidia.com/TU104GL_Tesla_T4"}},
				nil,
			),
			pools, "gpu-pool",
		),
		Entry("label match",
			newVMI(nil, nil, map[string]string{"workload-class": "secure"}),
			pools, "secure-pool",
		),
		Entry("no match - wrong device",
			newVMI(
				[]v1.GPU{{Name: "gpu1", DeviceName: "intel.com/fpga"}},
				nil, nil,
			),
			pools, "",
		),
		Entry("no match - wrong label",
			newVMI(nil, nil, map[string]string{"workload-class": "standard"}),
			pools, "",
		),
		Entry("no match - empty VMI",
			newVMI(nil, nil, nil),
			pools, "",
		),
		Entry("first match wins - device matches first pool",
			newVMI(
				[]v1.GPU{{Name: "gpu1", DeviceName: "nvidia.com/TU104GL_Tesla_T4"}},
				nil,
				map[string]string{"workload-class": "secure"},
			),
			pools, "gpu-pool",
		),
		Entry("OR semantics - device matches in combined pool",
			newVMI(
				[]v1.GPU{{Name: "gpu1", DeviceName: "nvidia.com/TU104GL_Tesla_T4"}},
				nil, nil,
			),
			[]v1.VirtHandlerPoolConfig{bothPool}, "both-pool",
		),
		Entry("OR semantics - label matches in combined pool",
			newVMI(nil, nil, map[string]string{"workload-class": "secure"}),
			[]v1.VirtHandlerPoolConfig{bothPool}, "both-pool",
		),
		Entry("empty pool list",
			newVMI(
				[]v1.GPU{{Name: "gpu1", DeviceName: "nvidia.com/TU104GL_Tesla_T4"}},
				nil, nil,
			),
			nil, "",
		),
		Entry("label match requires all labels",
			newVMI(nil, nil, map[string]string{"workload-class": "secure", "other": "val"}),
			pools, "secure-pool",
		),
		Entry("label match fails on partial",
			newVMI(nil, nil, map[string]string{"other": "val"}),
			[]v1.VirtHandlerPoolConfig{{
				Name:              "multi-label",
				VirtLauncherImage: "img",
				NodeSelector:      map[string]string{"k": "v"},
				Selector: v1.VirtHandlerPoolSelector{
					VMLabels: &v1.VirtHandlerPoolVMLabels{
						MatchLabels: map[string]string{"a": "1", "b": "2"},
					},
				},
			}},
			"",
		),
	)
})

var _ = Describe("GetLauncherImageForVMI", func() {
	It("should return pool launcher image when matched", func() {
		pools := []v1.VirtHandlerPoolConfig{{
			Name:              "gpu-pool",
			VirtLauncherImage: "registry.example.com/virt-launcher:gpu",
			NodeSelector:      map[string]string{"k": "v"},
			Selector: v1.VirtHandlerPoolSelector{
				DeviceNames: []string{"nvidia.com/TU104GL_Tesla_T4"},
			},
		}}
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{Name: "gpu1", DeviceName: "nvidia.com/TU104GL_Tesla_T4"}},
					},
				},
			},
		}
		Expect(GetLauncherImageForVMI(pools, vmi, "default:latest")).To(Equal("registry.example.com/virt-launcher:gpu"))
	})

	It("should return default image when no pool matches", func() {
		vmi := &v1.VirtualMachineInstance{}
		Expect(GetLauncherImageForVMI(nil, vmi, "default:latest")).To(Equal("default:latest"))
	})

	It("should return default image when pool matches but has no launcher override", func() {
		pools := []v1.VirtHandlerPoolConfig{{
			Name:             "handler-only",
			VirtHandlerImage: "registry.example.com/virt-handler:custom",
			NodeSelector:     map[string]string{"k": "v"},
			Selector: v1.VirtHandlerPoolSelector{
				DeviceNames: []string{"nvidia.com/TU104GL_Tesla_T4"},
			},
		}}
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{Name: "gpu1", DeviceName: "nvidia.com/TU104GL_Tesla_T4"}},
					},
				},
			},
		}
		Expect(GetLauncherImageForVMI(pools, vmi, "default:latest")).To(Equal("default:latest"))
	})
})
