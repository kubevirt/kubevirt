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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Handler matcher", func() {
	const (
		defaultImage     = "registry.io/kubevirt/virt-launcher:default"
		gpuHandlerImage  = "registry.io/kubevirt/virt-launcher:gpu"
		fpgaHandlerImage = "registry.io/kubevirt/virt-launcher:fpga"
	)

	Context("MatchVMIToAdditionalHandler", func() {
		var (
			vmi                *v1.VirtualMachineInstance
			additionalHandlers []v1.AdditionalVirtHandlerConfig
		)

		BeforeEach(func() {
			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi",
					Namespace: "default",
				},
			}
			additionalHandlers = nil
		})

		It("should return nil when VMI is nil", func() {
			result := MatchVMIToAdditionalHandler(nil, additionalHandlers)
			Expect(result).To(BeNil())
		})

		It("should return nil when additionalHandlers is empty", func() {
			vmi.Spec.NodeSelector = map[string]string{"gpu": "true"}
			result := MatchVMIToAdditionalHandler(vmi, nil)
			Expect(result).To(BeNil())
		})

		It("should return nil when VMI has no node selector", func() {
			additionalHandlers = []v1.AdditionalVirtHandlerConfig{
				{
					Name:              "gpu-handler",
					VirtLauncherImage: gpuHandlerImage,
					NodeSelector:      map[string]string{"gpu": "true"},
				},
			}
			result := MatchVMIToAdditionalHandler(vmi, additionalHandlers)
			Expect(result).To(BeNil())
		})

		It("should return nil when handler has empty node selector", func() {
			vmi.Spec.NodeSelector = map[string]string{"gpu": "true"}
			additionalHandlers = []v1.AdditionalVirtHandlerConfig{
				{
					Name:              "gpu-handler",
					VirtLauncherImage: gpuHandlerImage,
				},
			}
			result := MatchVMIToAdditionalHandler(vmi, additionalHandlers)
			Expect(result).To(BeNil())
		})

		It("should return matching handler when VMI node selector matches exactly", func() {
			vmi.Spec.NodeSelector = map[string]string{"gpu": "true"}
			additionalHandlers = []v1.AdditionalVirtHandlerConfig{
				{
					Name:              "gpu-handler",
					VirtLauncherImage: gpuHandlerImage,
					NodeSelector:      map[string]string{"gpu": "true"},
				},
			}
			result := MatchVMIToAdditionalHandler(vmi, additionalHandlers)
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("gpu-handler"))
		})

		It("should return matching handler when VMI node selector is a superset", func() {
			vmi.Spec.NodeSelector = map[string]string{
				"gpu":    "true",
				"region": "us-west",
			}
			additionalHandlers = []v1.AdditionalVirtHandlerConfig{
				{
					Name:              "gpu-handler",
					VirtLauncherImage: gpuHandlerImage,
					NodeSelector:      map[string]string{"gpu": "true"},
				},
			}
			result := MatchVMIToAdditionalHandler(vmi, additionalHandlers)
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("gpu-handler"))
		})

		It("should return nil when VMI node selector is a subset of handler's", func() {
			vmi.Spec.NodeSelector = map[string]string{"gpu": "true"}
			additionalHandlers = []v1.AdditionalVirtHandlerConfig{
				{
					Name:              "gpu-handler",
					VirtLauncherImage: gpuHandlerImage,
					NodeSelector: map[string]string{
						"gpu":    "true",
						"region": "us-west",
					},
				},
			}
			result := MatchVMIToAdditionalHandler(vmi, additionalHandlers)
			Expect(result).To(BeNil())
		})

		It("should return nil when node selector values don't match", func() {
			vmi.Spec.NodeSelector = map[string]string{"gpu": "false"}
			additionalHandlers = []v1.AdditionalVirtHandlerConfig{
				{
					Name:              "gpu-handler",
					VirtLauncherImage: gpuHandlerImage,
					NodeSelector:      map[string]string{"gpu": "true"},
				},
			}
			result := MatchVMIToAdditionalHandler(vmi, additionalHandlers)
			Expect(result).To(BeNil())
		})

		It("should return the first matching handler when multiple match", func() {
			vmi.Spec.NodeSelector = map[string]string{
				"gpu":  "true",
				"fpga": "true",
			}
			additionalHandlers = []v1.AdditionalVirtHandlerConfig{
				{
					Name:              "gpu-handler",
					VirtLauncherImage: gpuHandlerImage,
					NodeSelector:      map[string]string{"gpu": "true"},
				},
				{
					Name:              "fpga-handler",
					VirtLauncherImage: fpgaHandlerImage,
					NodeSelector:      map[string]string{"fpga": "true"},
				},
			}
			result := MatchVMIToAdditionalHandler(vmi, additionalHandlers)
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("gpu-handler"))
		})
	})

	Context("GetLauncherImageForVMI", func() {
		var (
			vmi                *v1.VirtualMachineInstance
			additionalHandlers []v1.AdditionalVirtHandlerConfig
		)

		BeforeEach(func() {
			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi",
					Namespace: "default",
				},
			}
			additionalHandlers = nil
		})

		It("should return the default image when no additional handlers", func() {
			image := GetLauncherImageForVMI(vmi, additionalHandlers, defaultImage)
			Expect(image).To(Equal(defaultImage))
		})

		It("should return the default image when VMI has no node selector", func() {
			additionalHandlers = []v1.AdditionalVirtHandlerConfig{
				{
					Name:              "gpu-handler",
					VirtLauncherImage: gpuHandlerImage,
					NodeSelector:      map[string]string{"gpu": "true"},
				},
			}
			image := GetLauncherImageForVMI(vmi, additionalHandlers, defaultImage)
			Expect(image).To(Equal(defaultImage))
		})

		It("should return the handler image when VMI matches handler", func() {
			vmi.Spec.NodeSelector = map[string]string{"gpu": "true"}
			additionalHandlers = []v1.AdditionalVirtHandlerConfig{
				{
					Name:              "gpu-handler",
					VirtLauncherImage: gpuHandlerImage,
					NodeSelector:      map[string]string{"gpu": "true"},
				},
			}
			image := GetLauncherImageForVMI(vmi, additionalHandlers, defaultImage)
			Expect(image).To(Equal(gpuHandlerImage))
		})

		It("should return the default image when matching handler has empty VirtLauncherImage", func() {
			vmi.Spec.NodeSelector = map[string]string{"gpu": "true"}
			additionalHandlers = []v1.AdditionalVirtHandlerConfig{
				{
					Name:         "gpu-handler",
					NodeSelector: map[string]string{"gpu": "true"},
				},
			}
			image := GetLauncherImageForVMI(vmi, additionalHandlers, defaultImage)
			Expect(image).To(Equal(defaultImage))
		})

		It("should return the default image when no handler matches", func() {
			vmi.Spec.NodeSelector = map[string]string{"cpu": "intel"}
			additionalHandlers = []v1.AdditionalVirtHandlerConfig{
				{
					Name:              "gpu-handler",
					VirtLauncherImage: gpuHandlerImage,
					NodeSelector:      map[string]string{"gpu": "true"},
				},
			}
			image := GetLauncherImageForVMI(vmi, additionalHandlers, defaultImage)
			Expect(image).To(Equal(defaultImage))
		})
	})
})
