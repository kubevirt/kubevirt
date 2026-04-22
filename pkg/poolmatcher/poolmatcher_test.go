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

package poolmatcher

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("PoolMatcher", func() {

	Context("MatchVMIToWorkerPool", func() {

		It("should return nil when no pools are defined", func() {
			vmi := &v1.VirtualMachineInstance{}
			Expect(MatchVMIToWorkerPool(nil, vmi)).To(BeNil())
		})

		It("should return nil when no pool matches", func() {
			pools := []v1.WorkerPoolConfig{
				{
					Name:     "gpu-pool",
					Selector: v1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/A100"}},
				},
			}
			vmi := &v1.VirtualMachineInstance{}
			Expect(MatchVMIToWorkerPool(pools, vmi)).To(BeNil())
		})

		It("should match by GPU deviceName", func() {
			pools := []v1.WorkerPoolConfig{
				{
					Name:     "gpu-pool",
					Selector: v1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/A100"}},
				},
			}
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{
								{DeviceName: "nvidia.com/A100"},
							},
						},
					},
				},
			}
			result := MatchVMIToWorkerPool(pools, vmi)
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("gpu-pool"))
		})

		It("should match by hostDevice deviceName", func() {
			pools := []v1.WorkerPoolConfig{
				{
					Name:     "fpga-pool",
					Selector: v1.WorkerPoolSelector{DeviceNames: []string{"intel.com/fpga"}},
				},
			}
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							HostDevices: []v1.HostDevice{
								{DeviceName: "intel.com/fpga"},
							},
						},
					},
				},
			}
			result := MatchVMIToWorkerPool(pools, vmi)
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("fpga-pool"))
		})

		It("should match by VMLabels", func() {
			pools := []v1.WorkerPoolConfig{
				{
					Name: "label-pool",
					Selector: v1.WorkerPoolSelector{
						VMLabels: &v1.WorkerPoolVMLabels{
							MatchLabels: map[string]string{"workload": "ai"},
						},
					},
				},
			}
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"workload": "ai", "other": "value"},
				},
			}
			result := MatchVMIToWorkerPool(pools, vmi)
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("label-pool"))
		})

		It("should not match when VMI labels are missing required labels", func() {
			pools := []v1.WorkerPoolConfig{
				{
					Name: "label-pool",
					Selector: v1.WorkerPoolSelector{
						VMLabels: &v1.WorkerPoolVMLabels{
							MatchLabels: map[string]string{"workload": "ai", "tier": "gpu"},
						},
					},
				},
			}
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"workload": "ai"},
				},
			}
			Expect(MatchVMIToWorkerPool(pools, vmi)).To(BeNil())
		})

		It("should return first matching pool (first-match-wins)", func() {
			pools := []v1.WorkerPoolConfig{
				{
					Name:     "first",
					Selector: v1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/A100"}},
				},
				{
					Name:     "second",
					Selector: v1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/A100"}},
				},
			}
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{
								{DeviceName: "nvidia.com/A100"},
							},
						},
					},
				},
			}
			result := MatchVMIToWorkerPool(pools, vmi)
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("first"))
		})

		It("should match with OR semantics between deviceNames and vmLabels", func() {
			pools := []v1.WorkerPoolConfig{
				{
					Name: "or-pool",
					Selector: v1.WorkerPoolSelector{
						DeviceNames: []string{"nvidia.com/A100"},
						VMLabels: &v1.WorkerPoolVMLabels{
							MatchLabels: map[string]string{"workload": "ai"},
						},
					},
				},
			}
			// Match by label only (no GPU)
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"workload": "ai"},
				},
			}
			result := MatchVMIToWorkerPool(pools, vmi)
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("or-pool"))
		})

		It("should not match when VMI has no labels and selector requires labels", func() {
			pools := []v1.WorkerPoolConfig{
				{
					Name: "label-pool",
					Selector: v1.WorkerPoolSelector{
						VMLabels: &v1.WorkerPoolVMLabels{
							MatchLabels: map[string]string{"workload": "ai"},
						},
					},
				},
			}
			vmi := &v1.VirtualMachineInstance{}
			Expect(MatchVMIToWorkerPool(pools, vmi)).To(BeNil())
		})
	})

	Context("GetLauncherImageForVMI", func() {

		It("should return default image when no pools match", func() {
			pools := []v1.WorkerPoolConfig{
				{
					Name:              "pool",
					VirtLauncherImage: "custom:latest",
					Selector:          v1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/A100"}},
				},
			}
			vmi := &v1.VirtualMachineInstance{}
			Expect(GetLauncherImageForVMI(pools, vmi, "default:latest")).To(Equal("default:latest"))
		})

		It("should return pool launcher image when pool matches", func() {
			pools := []v1.WorkerPoolConfig{
				{
					Name:              "pool",
					VirtLauncherImage: "custom:latest",
					Selector:          v1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/A100"}},
				},
			}
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{
								{DeviceName: "nvidia.com/A100"},
							},
						},
					},
				},
			}
			Expect(GetLauncherImageForVMI(pools, vmi, "default:latest")).To(Equal("custom:latest"))
		})

		It("should return default image when pool matches but has no launcher image override", func() {
			pools := []v1.WorkerPoolConfig{
				{
					Name:             "pool",
					VirtHandlerImage: "handler:latest",
					Selector:         v1.WorkerPoolSelector{DeviceNames: []string{"nvidia.com/A100"}},
				},
			}
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{
								{DeviceName: "nvidia.com/A100"},
							},
						},
					},
				},
			}
			Expect(GetLauncherImageForVMI(pools, vmi, "default:latest")).To(Equal("default:latest"))
		})
	})
})
